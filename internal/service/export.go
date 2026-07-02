package service

import (
	"archive/zip"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	aeszip "github.com/alexmullins/zip"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/clipboard"
	"clipd/internal/db"
	"clipd/internal/vault"
)

// GetItemImage returns the full-resolution image of an item as a PNG data
// URL. The list payload only carries a small thumbnail; the preview modal
// calls this lazily for the item being previewed.
func (s *Service) GetItemImage(id int64) (string, error) {
	full, err := s.store.GetFull(id)
	if err != nil {
		return "", err
	}
	if full.Item.ContentType != clipboard.ContentTypeImage || len(full.ImageBlob) == 0 {
		return "", errors.New("item has no image data")
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(full.ImageBlob), nil
}

// ChooseDownloadDir opens a native directory picker and returns the selected
// path ("" when the user cancels). The settings modal persists it.
func (s *Service) ChooseDownloadDir() (string, error) {
	if s.ctx == nil {
		return "", errors.New("window not ready")
	}
	return wailsruntime.OpenDirectoryDialog(s.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Choose download folder",
		DefaultDirectory: defaultDialogDir(s.CurrentSettings().DownloadDir),
	})
}

// DownloadItem saves one history item to disk. When a download folder is
// configured in settings the file is written there directly; otherwise a
// native save dialog asks where to put it (like a browser download). Returns
// the saved path, or "" when the user cancelled the dialog.
func (s *Service) DownloadItem(id int64) (string, error) {
	full, err := s.store.GetFull(id)
	if err != nil {
		return "", err
	}
	name := downloadFilename(full)
	dir := s.CurrentSettings().DownloadDir
	if dir == "" {
		if s.ctx == nil {
			return "", errors.New("window not ready")
		}
		path, err := wailsruntime.SaveFileDialog(s.ctx, wailsruntime.SaveDialogOptions{
			Title:            "Save clipboard item",
			DefaultDirectory: defaultDialogDir(""),
			DefaultFilename:  name,
		})
		if err != nil || path == "" {
			return "", err
		}
		return path, writeItemFile(path, full)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create download folder: %w", err)
	}
	path := uniquePath(filepath.Join(dir, name))
	return path, writeItemFile(path, full)
}

// ExportAllZip writes every history item into a single zip archive — text
// entries under texts/, images under images/. The user picks the zip location
// via a save dialog (defaulting to the configured download folder). Returns
// the saved path, or "" when cancelled.
func (s *Service) ExportAllZip() (string, error) {
	if s.ctx == nil {
		return "", errors.New("window not ready")
	}
	items, err := s.store.ListAllFull()
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", errors.New("history is empty — nothing to export")
	}
	path, err := wailsruntime.SaveFileDialog(s.ctx, wailsruntime.SaveDialogOptions{
		Title:            "Export clipboard history as ZIP",
		DefaultDirectory: defaultDialogDir(s.CurrentSettings().DownloadDir),
		DefaultFilename:  time.Now().Format("clipd-export-20060102-150405.zip"),
	})
	if err != nil || path == "" {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	textN, imageN := 0, 0
	for _, it := range items {
		var entry string
		var data []byte
		switch it.Item.ContentType {
		case clipboard.ContentTypeImage:
			if len(it.ImageBlob) == 0 {
				continue
			}
			imageN++
			entry = fmt.Sprintf("images/%03d-%s", imageN, downloadFilename(it))
			data = it.ImageBlob
		default:
			textN++
			entry = fmt.Sprintf("texts/%03d-%s", textN, downloadFilename(it))
			data = []byte(it.Item.TextContent)
		}
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     entry,
			Method:   zip.Deflate,
			Modified: time.Unix(it.Item.CreatedAt, 0),
		})
		if err != nil {
			return "", fmt.Errorf("zip entry %s: %w", entry, err)
		}
		if _, err := w.Write(data); err != nil {
			return "", fmt.Errorf("zip write %s: %w", entry, err)
		}
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("finish zip: %w", err)
	}
	return path, nil
}

// ExportVaultZip exports every private-vault entry into a password-protected
// (WinZip AES) zip archive. The caller supplies the vault PIN/password: it is
// verified against the vault metadata first — proving the exporter owns the
// vault — and then used as the zip password, so the archive opens with the
// same credentials as the vault itself. Returns the saved path, or "" when
// the user cancels the save dialog.
//
// Note: AES-encrypted zips need a capable extractor (7-Zip, p7zip, WinRAR…);
// the legacy `unzip` tool cannot read them.
func (s *Service) ExportVaultZip(pin string) (string, error) {
	if s.ctx == nil {
		return "", errors.New("window not ready")
	}
	if strings.TrimSpace(pin) == "" {
		return "", errors.New("vault PIN/password is required")
	}
	meta, err := s.vaultMetadata()
	if err != nil {
		return "", err
	}
	if !meta.Configured {
		return "", errors.New("private vault is not configured")
	}
	if err := ensureCanAttempt(meta); err != nil {
		return "", err
	}
	if !meta.VerifyPIN(pin) {
		_ = s.recordVaultFailure(meta)
		return "", vault.ErrInvalidPIN
	}
	key, err := meta.VaultKey()
	if err != nil {
		return "", err
	}
	rows, err := s.store.ListVaultEntries()
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", errors.New("private vault is empty — nothing to export")
	}

	path, err := wailsruntime.SaveFileDialog(s.ctx, wailsruntime.SaveDialogOptions{
		Title:            "Export Private Vault as encrypted ZIP",
		DefaultDirectory: defaultDialogDir(s.CurrentSettings().DownloadDir),
		DefaultFilename:  time.Now().Format("clipd-vault-20060102-150405.zip"),
	})
	if err != nil || path == "" {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer f.Close()
	zw := aeszip.NewWriter(f)
	textN, imageN := 0, 0
	for _, row := range rows {
		plain, err := vault.OpenEntry(key, row.Payload, row.Nonce)
		if err != nil {
			return "", fmt.Errorf("decrypt vault entry %d: %w", row.ID, err)
		}
		ts := time.Unix(row.CreatedAt, 0).Format("20060102-150405")
		var entry string
		var data []byte
		if clipboard.ContentType(plain.ContentType) == clipboard.ContentTypeImage {
			imageN++
			entry = fmt.Sprintf("images/%03d-%s%s.png", imageN, safeTitle(plain.Title, "vault-image"), "-"+ts)
			data = plain.ImagePNG
		} else {
			textN++
			entry = fmt.Sprintf("texts/%03d-%s%s.txt", textN, safeTitle(plain.Title, "vault-text"), "-"+ts)
			data = []byte(plain.Text)
		}
		w, err := zw.Encrypt(entry, pin)
		if err != nil {
			return "", fmt.Errorf("zip entry %s: %w", entry, err)
		}
		if _, err := w.Write(data); err != nil {
			return "", fmt.Errorf("zip write %s: %w", entry, err)
		}
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("finish zip: %w", err)
	}
	return path, nil
}

// safeTitle turns a vault entry title into a filesystem/zip-safe slug,
// falling back when the title is empty.
func safeTitle(title, fallback string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return fallback
	}
	var b strings.Builder
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

// MinimizeWindow minimises the window in windowed mode. In popup mode there
// is no taskbar entry to restore a minimised window from, so it hides the
// popup instead (the hotkey/tray brings it back).
func (s *Service) MinimizeWindow() {
	if s.ctx == nil {
		return
	}
	if s.CurrentSettings().WindowFrame {
		wailsruntime.WindowMinimise(s.ctx)
		return
	}
	s.HidePopup()
}

// ----- helpers -----

// downloadFilename builds a safe, descriptive filename for one item, e.g.
// clipd-text-42-20260702-153000.txt / clipd-image-43-20260702-153000.png.
func downloadFilename(it db.FullItem) string {
	kind, ext := "text", ".txt"
	if it.Item.ContentType == clipboard.ContentTypeImage {
		kind, ext = "image", ".png"
	}
	ts := time.Unix(it.Item.CreatedAt, 0).Format("20060102-150405")
	return fmt.Sprintf("clipd-%s-%d-%s%s", kind, it.Item.ID, ts, ext)
}

// writeItemFile writes the item payload (UTF-8 text or PNG bytes) to path.
func writeItemFile(path string, it db.FullItem) error {
	data := []byte(it.Item.TextContent)
	if it.Item.ContentType == clipboard.ContentTypeImage {
		if len(it.ImageBlob) == 0 {
			return errors.New("item has no image data")
		}
		data = it.ImageBlob
	}
	return os.WriteFile(path, data, 0o644)
}

// uniquePath appends " (n)" before the extension until the path is free, so
// direct-to-folder downloads never overwrite an existing file.
func uniquePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	for i := 1; ; i++ {
		p := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}

// defaultDialogDir picks a sensible starting directory for dialogs: the
// configured download folder if set, else ~/Downloads, else the home dir.
func defaultDialogDir(configured string) string {
	if configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if dl := filepath.Join(home, "Downloads"); dirExists(dl) {
		return dl
	}
	return home
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
