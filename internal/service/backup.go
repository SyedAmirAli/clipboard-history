package service

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/clipboard"
)

const lastBackupKey = "last_backup_at"

// backupWindow is how far back the daily backup reaches for non-pinned items.
const backupWindow = 24 * time.Hour

// vaultBackupEntry is one vault row exactly as stored: the payload is
// AES-GCM-encrypted with a key derived from the vault PIN/password, so the
// backup exposes nothing readable and clipd never has to persist a secret.
type vaultBackupEntry struct {
	ID          int64  `json:"id"`
	ContentType string `json:"contentType"`
	Payload     string `json:"payload"` // base64
	Nonce       string `json:"nonce"`   // base64
	ContentHash string `json:"contentHash"`
	CreatedAt   int64  `json:"createdAt"`
	LastUsedAt  int64  `json:"lastUsedAt"`
}

type backupManifest struct {
	CreatedAt      int64  `json:"createdAt"`
	WindowFromUnix int64  `json:"windowFromUnix"`
	TextItems      int    `json:"textItems"`
	ImageItems     int    `json:"imageItems"`
	PinnedIncluded bool   `json:"pinnedIncluded"`
	VaultEntries   int    `json:"vaultEntries"`
	VaultNote      string `json:"vaultNote,omitempty"`
	CleanedAfter   bool   `json:"cleanedAfter"`
	App            string `json:"app"`
}

// RunBackupNow performs a backup immediately using the saved backup settings
// and returns the path of the written zip. Called from the settings UI's
// "Backup now" button and by the daily scheduler.
func (s *Service) RunBackupNow() (string, error) {
	path, err := s.runBackup(s.CurrentSettings())
	if err != nil {
		return "", err
	}
	return path, nil
}

// ChooseBackupDir opens a native directory picker for the backup destination.
func (s *Service) ChooseBackupDir() (string, error) {
	if s.ctx == nil {
		return "", errors.New("window not ready")
	}
	return wailsruntime.OpenDirectoryDialog(s.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Choose backup folder",
		DefaultDirectory: defaultDialogDir(s.CurrentSettings().BackupDir),
	})
}

// runBackup writes the backup zip and applies the post-backup clean when
// configured. It never touches pinned items or the vault when cleaning.
func (s *Service) runBackup(settings clipboard.Settings) (string, error) {
	if settings.BackupDir == "" {
		return "", errors.New("no backup folder configured — choose one in Settings → Daily Backup")
	}
	if err := os.MkdirAll(settings.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup folder: %w", err)
	}
	now := time.Now()
	cutoff := now.Add(-backupWindow).Unix()
	items, err := s.store.ListBackupItems(cutoff, settings.BackupIncludePinned)
	if err != nil {
		return "", err
	}

	var vaultEntries []vaultBackupEntry
	if settings.BackupIncludeVault {
		rows, err := s.store.ListVaultEntries()
		if err != nil {
			return "", err
		}
		for _, row := range rows {
			vaultEntries = append(vaultEntries, vaultBackupEntry{
				ID:          row.ID,
				ContentType: string(row.ContentType),
				Payload:     base64.StdEncoding.EncodeToString(row.Payload),
				Nonce:       base64.StdEncoding.EncodeToString(row.Nonce),
				ContentHash: row.ContentHash,
				CreatedAt:   row.CreatedAt,
				LastUsedAt:  row.LastUsedAt,
			})
		}
	}
	if len(items) == 0 && len(vaultEntries) == 0 {
		return "", errors.New("nothing to back up in the last 24 hours")
	}

	path := uniquePath(filepath.Join(settings.BackupDir,
		now.Format("clipd-backup-20060102-150405.zip")))
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create backup zip: %w", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	writeEntry := func(name string, modified time.Time, data []byte) error {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: modified,
		})
		if err != nil {
			return fmt.Errorf("zip entry %s: %w", name, err)
		}
		_, err = w.Write(data)
		return err
	}

	textN, imageN := 0, 0
	for _, it := range items {
		if it.Item.ContentType == clipboard.ContentTypeImage {
			if len(it.ImageBlob) == 0 {
				continue
			}
			imageN++
			name := fmt.Sprintf("images/%03d-%s", imageN, downloadFilename(it))
			if err := writeEntry(name, time.Unix(it.Item.CreatedAt, 0), it.ImageBlob); err != nil {
				return "", err
			}
		} else {
			textN++
			name := fmt.Sprintf("texts/%03d-%s", textN, downloadFilename(it))
			if err := writeEntry(name, time.Unix(it.Item.CreatedAt, 0), []byte(it.Item.TextContent)); err != nil {
				return "", err
			}
		}
	}

	manifest := backupManifest{
		CreatedAt:      now.Unix(),
		WindowFromUnix: cutoff,
		TextItems:      textN,
		ImageItems:     imageN,
		PinnedIncluded: settings.BackupIncludePinned,
		VaultEntries:   len(vaultEntries),
		CleanedAfter:   settings.BackupCleanAfter,
		App:            "clipd",
	}
	if len(vaultEntries) > 0 {
		manifest.VaultNote = "vault/entries.json holds AES-GCM-encrypted payloads as stored by clipd; they can only be decrypted with the vault PIN/password."
		data, err := json.MarshalIndent(vaultEntries, "", "  ")
		if err != nil {
			return "", fmt.Errorf("encode vault entries: %w", err)
		}
		if err := writeEntry("vault/entries.json", now, data); err != nil {
			return "", err
		}
	}
	mdata, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	if err := writeEntry("manifest.json", now, mdata); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("finish backup zip: %w", err)
	}

	_ = s.store.SetSetting(lastBackupKey, strconv.FormatInt(now.Unix(), 10))

	if settings.BackupCleanAfter {
		if err := s.store.ClearAll(true); err != nil {
			log.Printf("backup: post-backup clean failed: %v", err)
		} else if s.ctx != nil {
			wailsruntime.EventsEmit(s.ctx, EventCleared)
		}
	}
	return path, nil
}

// ----- daily scheduler -----

// startBackupLoop runs the daily scheduler. Checked once a minute: when the
// configured slot for today has passed and no backup has run since that slot,
// one is performed. This also covers catch-up — if the machine was off at the
// scheduled time, the first check after startup triggers the missed backup.
func (s *Service) startBackupLoop() {
	s.backupOnce.Do(func() {
		go func() {
			// Small startup delay so DB/settings are warm before a
			// potential catch-up run fires.
			time.Sleep(15 * time.Second)
			for {
				s.maybeRunScheduledBackup(time.Now())
				time.Sleep(time.Minute)
			}
		}()
	})
}

func (s *Service) maybeRunScheduledBackup(now time.Time) {
	settings := s.CurrentSettings()
	if !settings.BackupEnabled || settings.BackupDir == "" {
		return
	}
	lastRaw, _ := s.store.GetSetting(lastBackupKey, "0")
	last, _ := strconv.ParseInt(lastRaw, 10, 64)
	if !backupDue(now, settings.BackupTime, time.Unix(last, 0)) {
		return
	}
	path, err := s.runBackup(settings)
	if err != nil {
		log.Printf("scheduled backup failed: %v", err)
		// Record the attempt so a persistent failure (e.g. unplugged
		// backup drive) retries tomorrow instead of every minute.
		_ = s.store.SetSetting(lastBackupKey, strconv.FormatInt(now.Unix(), 10))
		notify("clipd backup failed", err.Error())
		return
	}
	log.Printf("scheduled backup written: %s", path)
	notify("clipd", "Daily backup saved to "+path)
}

// backupDue reports whether a backup should run: today's slot (slotHHMM on
// now's date) has passed and the last backup predates that slot.
func backupDue(now time.Time, slotHHMM string, lastRun time.Time) bool {
	t, err := time.Parse("15:04", slotHHMM)
	if err != nil {
		return false
	}
	slot := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if now.Before(slot) {
		return false
	}
	return lastRun.Before(slot)
}

// notify shows a desktop notification, best-effort (libnotify-bin is a
// recommended package dependency, not a hard one).
func notify(title, body string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return
	}
	_ = exec.Command("notify-send", "--app-name=clipd", title, body).Run()
}
