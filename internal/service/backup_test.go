package service

import (
	"archive/zip"
	"path/filepath"
	"testing"
	"time"

	"clipd/internal/clipboard"
)

func TestBackupDue(t *testing.T) {
	loc := time.Local
	day := time.Date(2026, 7, 2, 0, 0, 0, 0, loc)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }

	cases := []struct {
		name    string
		now     time.Time
		slot    string
		lastRun time.Time
		want    bool
	}{
		{"before today's slot", at(1, 30), "02:00", at(-22, 0), false},
		{"slot just passed, last run yesterday", at(2, 1), "02:00", at(-22, 0), true},
		{"already ran after today's slot", at(9, 0), "02:00", at(2, 5), false},
		{"catch-up: started at 9am, last run two days ago", at(9, 0), "02:00", day.AddDate(0, 0, -2), true},
		{"never ran before", at(2, 0), "02:00", time.Unix(0, 0), true},
		{"invalid slot string", at(9, 0), "junk", time.Unix(0, 0), false},
		{"exactly at slot", at(2, 0), "02:00", at(-22, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := backupDue(tc.now, tc.slot, tc.lastRun); got != tc.want {
				t.Errorf("backupDue(%v, %q, %v) = %v, want %v", tc.now, tc.slot, tc.lastRun, got, tc.want)
			}
		})
	}
}

func TestRunBackupWritesZipAndCleans(t *testing.T) {
	svc, store := newTestService(t)
	setupVault(t, svc, "1234")

	if _, err := store.AddText("recent item", "t:recent"); err != nil {
		t.Fatal(err)
	}
	res, err := store.AddText("pinned item", "t:pinned")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetPinned(res.ID, true); err != nil {
		t.Fatal(err)
	}
	// A vault entry so vault/entries.json is exercised too.
	full, err := store.AddText("secret text", "t:secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MoveItemToVault(full.ID); err != nil {
		t.Fatalf("move to vault: %v", err)
	}

	dir := t.TempDir()
	settings := clipboard.DefaultSettings()
	settings.BackupDir = dir
	settings.BackupIncludeVault = true
	settings.BackupIncludePinned = true
	settings.BackupCleanAfter = true

	path, err := svc.runBackup(settings)
	if err != nil {
		t.Fatalf("runBackup: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("backup written to %s, want dir %s", path, dir)
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["manifest.json"] {
		t.Error("manifest.json missing from backup")
	}
	if !names["vault/entries.json"] {
		t.Error("vault/entries.json missing from backup")
	}
	textEntries := 0
	for name := range names {
		if filepath.Dir(name) == "texts" {
			textEntries++
		}
	}
	if textEntries != 2 { // recent + pinned
		t.Errorf("expected 2 text entries, got %d", textEntries)
	}

	// Clean-after: non-pinned gone, pinned survives, vault untouched.
	items, err := store.List("", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].Pinned {
		t.Errorf("after clean expected only the pinned item, got %+v", items)
	}
	vaultRows, err := store.ListVaultEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(vaultRows) != 1 {
		t.Errorf("vault should be untouched, got %d rows", len(vaultRows))
	}
}
