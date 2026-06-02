package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"clipd/internal/clipboard"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// AddTextResult describes the outcome of an Add* call so callers can
// know whether the operation created a new row, bumped an existing one
// to the top, or was a no-op (current top item is identical).
type AddTextResult struct {
	ID       int64
	IsNew    bool
	WasNoop  bool
	WasBump  bool
	WasPinned bool
}

// AddText inserts a text item or bumps the existing row with the same
// content hash to the top of the list (updating last_used_at).
func (s *Store) AddText(text, hash string) (AddTextResult, error) {
	now := time.Now().Unix()
	var existingID int64
	var pinned int
	err := s.db.QueryRow(`SELECT id, pinned FROM items WHERE content_hash = ?`, hash).Scan(&existingID, &pinned)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, ierr := s.db.Exec(
			`INSERT INTO items (content_type, text_content, content_hash, pinned, created_at, last_used_at)
			 VALUES (?, ?, ?, 0, ?, ?)`,
			clipboard.ContentTypeText, text, hash, now, now,
		)
		if ierr != nil {
			return AddTextResult{}, fmt.Errorf("insert text item: %w", ierr)
		}
		id, _ := res.LastInsertId()
		return AddTextResult{ID: id, IsNew: true}, nil
	case err != nil:
		return AddTextResult{}, fmt.Errorf("lookup hash: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE items SET last_used_at = ? WHERE id = ?`, now, existingID); err != nil {
		return AddTextResult{}, fmt.Errorf("bump last_used_at: %w", err)
	}
	return AddTextResult{ID: existingID, WasBump: true, WasPinned: pinned == 1}, nil
}

// AddImage inserts an image item (raw PNG blob + a small data-URL thumbnail)
// or bumps an existing duplicate hash to the top.
func (s *Store) AddImage(blob []byte, thumbDataURL, hash string, w, h int) (AddTextResult, error) {
	now := time.Now().Unix()
	var existingID int64
	var pinned int
	err := s.db.QueryRow(`SELECT id, pinned FROM items WHERE content_hash = ?`, hash).Scan(&existingID, &pinned)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, ierr := s.db.Exec(
			`INSERT INTO items (content_type, image_blob, image_thumb, image_w, image_h, content_hash, pinned, created_at, last_used_at)
			 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
			clipboard.ContentTypeImage, blob, thumbDataURL, w, h, hash, now, now,
		)
		if ierr != nil {
			return AddTextResult{}, fmt.Errorf("insert image item: %w", ierr)
		}
		id, _ := res.LastInsertId()
		return AddTextResult{ID: id, IsNew: true}, nil
	case err != nil:
		return AddTextResult{}, fmt.Errorf("lookup hash: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE items SET last_used_at = ? WHERE id = ?`, now, existingID); err != nil {
		return AddTextResult{}, fmt.Errorf("bump last_used_at: %w", err)
	}
	return AddTextResult{ID: existingID, WasBump: true, WasPinned: pinned == 1}, nil
}

// List returns up to `limit` items optionally filtered by case-insensitive
// substring on text_content. Pinned items are returned first, then by
// most-recently-used.
func (s *Store) List(filter string, limit int) ([]clipboard.Item, error) {
	if limit <= 0 {
		limit = 200
	}
	args := []any{}
	where := ""
	filter = strings.TrimSpace(filter)
	if filter != "" {
		where = "WHERE text_content LIKE ? COLLATE NOCASE"
		args = append(args, "%"+filter+"%")
	}
	query := fmt.Sprintf(`
		SELECT id, content_type, COALESCE(text_content,''), COALESCE(image_thumb,''),
		       image_w, image_h, content_hash, pinned, created_at, last_used_at
		FROM items
		%s
		ORDER BY pinned DESC, last_used_at DESC
		LIMIT ?
	`, where)
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()
	var out []clipboard.Item
	for rows.Next() {
		var it clipboard.Item
		var pinned int
		if err := rows.Scan(&it.ID, &it.ContentType, &it.TextContent, &it.ImageThumb,
			&it.ImageW, &it.ImageH, &it.ContentHash, &pinned, &it.CreatedAt, &it.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		it.Pinned = pinned == 1
		it.Preview = makePreview(it)
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetForPaste returns the full row needed to write an item back to the
// clipboard (including the image blob). The image_blob is loaded lazily
// here, not in the normal List(), to avoid sending megabytes over the
// JS bridge for the UI.
func (s *Store) GetForPaste(id int64) (clipboard.ContentType, string, []byte, error) {
	var ct clipboard.ContentType
	var text sql.NullString
	var blob []byte
	err := s.db.QueryRow(
		`SELECT content_type, text_content, image_blob FROM items WHERE id = ?`, id,
	).Scan(&ct, &text, &blob)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	if err != nil {
		return "", "", nil, err
	}
	now := time.Now().Unix()
	if _, err := s.db.Exec(`UPDATE items SET last_used_at = ? WHERE id = ?`, now, id); err != nil {
		return "", "", nil, fmt.Errorf("bump last_used_at: %w", err)
	}
	return ct, text.String, blob, nil
}

// SetPinned toggles the pinned flag for a row.
func (s *Store) SetPinned(id int64, pinned bool) error {
	v := 0
	if pinned {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE items SET pinned = ? WHERE id = ?`, v, id)
	return err
}

// Delete removes a single row by id and returns the deleted row's content
// hash (empty if the row didn't exist). The caller can use the hash to
// suppress the watcher so a deleted item still sitting on the clipboard
// isn't immediately re-ingested.
func (s *Store) Delete(id int64) (string, error) {
	var hash string
	_ = s.db.QueryRow(`SELECT content_hash FROM items WHERE id = ?`, id).Scan(&hash)
	_, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	return hash, err
}

// ClearAll removes all rows. When keepPinned is true, pinned items are
// preserved.
func (s *Store) ClearAll(keepPinned bool) error {
	if keepPinned {
		_, err := s.db.Exec(`DELETE FROM items WHERE pinned = 0`)
		return err
	}
	_, err := s.db.Exec(`DELETE FROM items`)
	return err
}

// TrimToMax enforces the user's max-items policy by deleting the oldest
// non-pinned rows beyond maxItems. Pinned items are always preserved
// and do not count toward the limit.
func (s *Store) TrimToMax(maxItems int) (int, error) {
	if maxItems <= 0 {
		return 0, nil
	}
	res, err := s.db.Exec(`
		DELETE FROM items
		WHERE pinned = 0
		  AND id IN (
		    SELECT id FROM items
		    WHERE pinned = 0
		    ORDER BY last_used_at DESC
		    LIMIT -1 OFFSET ?
		  )
	`, maxItems)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// GetSetting reads a single settings key, returning fallback when absent.
func (s *Store) GetSetting(key, fallback string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

// SetSetting upserts a single settings key/value.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// makePreview produces a short, single-line summary suitable for the UI list.
func makePreview(it clipboard.Item) string {
	if it.ContentType == clipboard.ContentTypeImage {
		if it.ImageW > 0 && it.ImageH > 0 {
			return fmt.Sprintf("Image %dx%d", it.ImageW, it.ImageH)
		}
		return "Image"
	}
	s := strings.ReplaceAll(it.TextContent, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
