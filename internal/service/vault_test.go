package service

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"clipd/internal/db"
	"clipd/internal/vault"
)

func newTestService(t *testing.T) (*Service, *db.Store) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "clipd.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	svc := New(store, nil)
	return svc, store
}

func setupVault(t *testing.T, svc *Service, pin string) {
	t.Helper()
	bundle, err := svc.StartVaultSetup()
	if err != nil {
		t.Fatalf("start setup: %v", err)
	}
	code, err := vault.TOTPCode(bundle.Secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	status, err := svc.ConfirmVaultSetup(pin, pin, code)
	if err != nil {
		t.Fatalf("confirm setup: %v", err)
	}
	if !status.Configured || !status.Unlocked {
		t.Fatalf("unexpected setup status: %+v", status)
	}
}

func TestVaultSetupRequiresMatchingPINAndValidCode(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()

	bundle, err := svc.StartVaultSetup()
	if err != nil {
		t.Fatalf("start setup: %v", err)
	}
	if _, err := svc.ConfirmVaultSetup("1234", "9999", "000000"); err == nil {
		t.Fatal("expected mismatched PIN confirmation to fail")
	}
	if _, err := svc.ConfirmVaultSetup("1234", "1234", "000000"); err == nil {
		t.Fatal("expected invalid authenticator code to fail")
	}
	code, err := vault.TOTPCode(bundle.Secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	status, err := svc.ConfirmVaultSetup("1234", "1234", code)
	if err != nil {
		t.Fatalf("confirm setup: %v", err)
	}
	if !status.Configured || !status.Unlocked {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestVaultUnlockWithPINAndAuthenticatorCode(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "old-pin")

	if err := svc.LockVault(); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if _, err := svc.UnlockVaultWithPIN("old-pin"); err != nil {
		t.Fatalf("unlock with pin: %v", err)
	}
	if err := svc.LockVault(); err != nil {
		t.Fatalf("lock: %v", err)
	}
	meta, err := svc.vaultMetadata()
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	secret, err := meta.TOTPSecret()
	if err != nil {
		t.Fatalf("totp secret: %v", err)
	}
	code, err := vault.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if _, err := svc.UnlockVaultWithCode(code); err != nil {
		t.Fatalf("unlock with code: %v", err)
	}
}

func TestVaultResetPINInvalidatesOldPIN(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "old-pin")

	meta, err := svc.vaultMetadata()
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	secret, err := meta.TOTPSecret()
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	code, err := vault.TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if _, err := svc.ResetVaultPIN(code, "new-pin", "new-pin"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	_ = svc.LockVault()
	if _, err := svc.UnlockVaultWithPIN("old-pin"); err == nil {
		t.Fatal("expected old PIN to fail after reset")
	}
	if _, err := svc.UnlockVaultWithPIN("new-pin"); err != nil {
		t.Fatalf("new PIN should unlock: %v", err)
	}
}

func TestMoveToVaultEncryptsAndRemovesNormalHistory(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "pin")

	added, err := store.AddText("secret text", vault.HashText("secret text"))
	if err != nil {
		t.Fatalf("add text: %v", err)
	}
	if err := svc.MoveItemToVault(added.ID); err != nil {
		t.Fatalf("move to vault: %v", err)
	}
	normal, err := svc.ListItems("", 20)
	if err != nil {
		t.Fatalf("list normal: %v", err)
	}
	if len(normal) != 0 {
		t.Fatalf("expected normal history to be empty, got %d", len(normal))
	}
	vaultItems, err := svc.ListVaultItems()
	if err != nil {
		t.Fatalf("list vault: %v", err)
	}
	if len(vaultItems) != 1 || vaultItems[0].Preview != "secret text" {
		t.Fatalf("unexpected vault items: %+v", vaultItems)
	}
	raw := store.Raw().QueryRow(`SELECT payload FROM vault_entries LIMIT 1`)
	var payload []byte
	if err := raw.Scan(&payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if string(payload) == "secret text" {
		t.Fatal("vault payload was stored in plaintext")
	}
}

func TestMoveToVaultFailsWhenLocked(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "pin")
	added, err := store.AddText("secret text", vault.HashText("secret text"))
	if err != nil {
		t.Fatalf("add text: %v", err)
	}
	_ = svc.LockVault()
	if err := svc.MoveItemToVault(added.ID); err == nil {
		t.Fatal("expected locked move to fail")
	}
	normal, err := svc.ListItems("", 20)
	if err != nil {
		t.Fatalf("list normal: %v", err)
	}
	if len(normal) != 1 {
		t.Fatalf("locked move should leave normal history intact, got %d", len(normal))
	}
}

func TestRevealVaultItemReturnsText(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "pin")

	added, err := store.AddText("secret text", vault.HashText("secret text"))
	if err != nil {
		t.Fatalf("add text: %v", err)
	}
	if err := svc.MoveItemToVault(added.ID); err != nil {
		t.Fatalf("move to vault: %v", err)
	}
	vaultItems, err := svc.ListVaultItems()
	if err != nil {
		t.Fatalf("list vault: %v", err)
	}
	if len(vaultItems) != 1 {
		t.Fatalf("expected one vault item, got %d", len(vaultItems))
	}
	revealed, err := svc.RevealVaultItem(vaultItems[0].ID)
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if revealed.Text != "secret text" {
		t.Fatalf("unexpected revealed text: %q", revealed.Text)
	}
}

func TestUpdateVaultItemTitlePersistsEncryptedTitle(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "pin")

	added, err := store.AddText("secret text", vault.HashText("secret text"))
	if err != nil {
		t.Fatalf("add text: %v", err)
	}
	if err := svc.MoveItemToVault(added.ID); err != nil {
		t.Fatalf("move to vault: %v", err)
	}
	vaultItems, err := svc.ListVaultItems()
	if err != nil {
		t.Fatalf("list vault: %v", err)
	}
	if len(vaultItems) != 1 {
		t.Fatalf("expected one vault item, got %d", len(vaultItems))
	}

	if err := svc.UpdateVaultItemTitle(vaultItems[0].ID, "  GitHub token\nprimary  "); err != nil {
		t.Fatalf("update title: %v", err)
	}
	vaultItems, err = svc.ListVaultItems()
	if err != nil {
		t.Fatalf("list vault after title update: %v", err)
	}
	if got, want := vaultItems[0].Title, "GitHub token primary"; got != want {
		t.Fatalf("unexpected title %q, want %q", got, want)
	}
	revealed, err := svc.RevealVaultItem(vaultItems[0].ID)
	if err != nil {
		t.Fatalf("reveal after title update: %v", err)
	}
	if revealed.Text != "secret text" {
		t.Fatalf("unexpected revealed text after title update: %q", revealed.Text)
	}

	raw := store.Raw().QueryRow(`SELECT payload FROM vault_entries WHERE id = ?`, vaultItems[0].ID)
	var payload []byte
	if err := raw.Scan(&payload); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if bytes.Contains(payload, []byte("GitHub")) || bytes.Contains(payload, []byte("secret text")) {
		t.Fatal("vault payload exposed plaintext title or secret")
	}
}

func TestVaultAutoLocksAfterInactivity(t *testing.T) {
	svc, store := newTestService(t)
	defer store.Close()
	setupVault(t, svc, "pin")

	svc.vaultMu.Lock()
	svc.vaultExpiry = time.Now().Add(-time.Second)
	svc.vaultMu.Unlock()
	status, err := svc.VaultStatus()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Unlocked {
		t.Fatal("expected vault to be locked after inactivity expiry")
	}
}
