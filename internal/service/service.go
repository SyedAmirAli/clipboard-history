// Package service is the bridge between the Go backend and the Wails
// frontend. Every method on Service is exposed to JavaScript via the
// Wails Bind mechanism.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/clipboard"
	"clipd/internal/db"
	"clipd/internal/vault"
	"clipd/internal/x11hint"
)

// EventNewItem asks the frontend to refresh its list from the database.
const EventNewItem = "clipboard:new-item"

// EventCleared is published after the user clears the history.
const EventCleared = "clipboard:cleared"

// EventVaultChanged is published when vault setup, lock state, or entries change.
const EventVaultChanged = "vault:changed"

const (
	vaultSettingsKey      = "private_vault"
	vaultInactivityPeriod = 5 * time.Minute
)

// HotkeySetter abstracts the hotkey manager so the service can update
// the registered shortcut when settings change.
type HotkeySetter interface {
	Register(spec string) error
}

// Service is bound onto the Wails runtime. All methods are JS-callable.
type Service struct {
	store        *db.Store
	hotkeyMgr    HotkeySetter
	visible      atomic.Bool
	ctx          context.Context
	settingsLog  atomic.Value // last loaded clipboard.Settings, for hotkey diffing
	vaultMu      sync.Mutex
	vaultKey     []byte
	vaultExpiry  time.Time
	pendingSetup *vault.SetupBundle
	backupOnce   sync.Once
}

// New constructs a Service with all wiring dependencies.
func New(store *db.Store, hk HotkeySetter) *Service {
	return &Service{store: store, hotkeyMgr: hk}
}

// AttachContext is called from Wails OnStartup so subsequent Show/Hide
// operations can target the runtime.
func (s *Service) AttachContext(ctx context.Context) {
	s.ctx = ctx
	s.startBackupLoop()
}

// Context exposes the Wails context for collaborators (e.g. tray) that
// need to emit events or control the window from outside the JS bridge.
func (s *Service) Context() context.Context { return s.ctx }

func (s *Service) emitNewItem() {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, EventNewItem)
}

func (s *Service) emitVaultChanged() {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, EventVaultChanged)
}

// ----- Bound methods (called from JS) -----

// ListItems returns up to `limit` history entries optionally filtered by
// a case-insensitive substring on text content. Pinned items come first.
func (s *Service) ListItems(filter string, limit int) ([]clipboard.Item, error) {
	if limit <= 0 {
		limit = 200
	}
	items, err := s.store.List(filter, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []clipboard.Item{}
	}
	return items, nil
}

// writeToClipboard puts a stored item onto the Wayland clipboard.
func (s *Service) writeToClipboard(id int64) error {
	ct, text, blob, err := s.store.GetForPaste(id)
	if err != nil {
		return err
	}
	switch ct {
	case clipboard.ContentTypeText:
		return clipboard.WriteText(text)
	case clipboard.ContentTypeImage:
		return clipboard.WriteImagePNG(blob)
	default:
		return fmt.Errorf("unknown content_type: %s", ct)
	}
}

// CopyItem writes the stored item onto the clipboard but leaves the popup
// open and does not auto-paste. Used by the inline/context "Copy" action so
// the user can grab an entry without the window closing.
func (s *Service) CopyItem(id int64) error {
	return s.writeToClipboard(id)
}

// PasteItem writes the stored item back onto the X11 CLIPBOARD selection,
// hides the popup, and (if enabled) auto-pastes into the focused window so
// the user doesn't even need to press Ctrl+V. This is the row-click / Enter
// "use this item" action.
func (s *Service) PasteItem(id int64) error {
	if err := s.writeToClipboard(id); err != nil {
		return err
	}
	s.HidePopup()

	// Auto-paste: after the popup hides and focus returns to the user's
	// previous window, synthesise Ctrl+V so the value drops straight into the
	// field they were editing. Done asynchronously with a short delay to let
	// the WM hand focus back and xclip take ownership of the selection.
	if s.CurrentSettings().AutoPaste {
		go func() {
			time.Sleep(150 * time.Millisecond)
			if err := clipboard.SendPaste(); err != nil {
				log.Printf("auto-paste: %v", err)
			}
		}()
	}
	return nil
}

// PinItem toggles whether an item should survive eviction.
func (s *Service) PinItem(id int64, pinned bool) error {
	return s.store.SetPinned(id, pinned)
}

// DeleteItem removes a single history entry.
func (s *Service) DeleteItem(id int64) error {
	_, err := s.store.Delete(id)
	return err
}

// ----- Private Vault -----

type VaultStatus struct {
	Configured     bool  `json:"configured"`
	Unlocked       bool  `json:"unlocked"`
	FailedAttempts int   `json:"failedAttempts"`
	LockedUntil    int64 `json:"lockedUntil,omitempty"`
}

type VaultEntryView struct {
	ID          int64                 `json:"id"`
	Title       string                `json:"title,omitempty"`
	ContentType clipboard.ContentType `json:"contentType"`
	Preview     string                `json:"preview"`
	ImageThumb  string                `json:"imageThumb,omitempty"`
	ImageW      int                   `json:"imageW,omitempty"`
	ImageH      int                   `json:"imageH,omitempty"`
	CreatedAt   int64                 `json:"createdAt"`
	LastUsedAt  int64                 `json:"lastUsedAt"`
}

type VaultSecretView struct {
	ContentType clipboard.ContentType `json:"contentType"`
	Text        string                `json:"text,omitempty"`
}

func (s *Service) VaultStatus() (VaultStatus, error) {
	meta, err := s.vaultMetadata()
	if err != nil {
		return VaultStatus{}, err
	}
	return VaultStatus{
		Configured:     meta.Configured,
		Unlocked:       s.vaultUnlocked(),
		FailedAttempts: meta.FailedAttempts,
		LockedUntil:    lockoutUntil(meta),
	}, nil
}

func (s *Service) StartVaultSetup() (vault.SetupBundle, error) {
	bundle, err := vault.NewSetupBundle("Private Vault")
	if err != nil {
		return vault.SetupBundle{}, err
	}
	s.vaultMu.Lock()
	s.pendingSetup = &bundle
	s.vaultMu.Unlock()
	return bundle, nil
}

func (s *Service) ConfirmVaultSetup(pin, confirm, code string) (VaultStatus, error) {
	if strings.TrimSpace(pin) == "" {
		return VaultStatus{}, errors.New("PIN/password is required")
	}
	if pin != confirm {
		return VaultStatus{}, errors.New("PIN/password confirmation does not match")
	}
	s.vaultMu.Lock()
	pending := s.pendingSetup
	s.vaultMu.Unlock()
	if pending == nil {
		return VaultStatus{}, errors.New("vault setup has not been started")
	}
	if !vault.ValidTOTP(pending.Secret, code, time.Now()) {
		return VaultStatus{}, vault.ErrInvalidCode
	}
	meta, key, err := vault.NewMetadata(pin, pending.Secret)
	if err != nil {
		return VaultStatus{}, err
	}
	if err := s.saveVaultMetadata(meta); err != nil {
		return VaultStatus{}, err
	}
	s.vaultMu.Lock()
	s.vaultKey = key
	s.vaultExpiry = time.Now().Add(vaultInactivityPeriod)
	s.pendingSetup = nil
	s.vaultMu.Unlock()
	s.emitVaultChanged()
	return s.VaultStatus()
}

func (s *Service) UnlockVaultWithPIN(pin string) (VaultStatus, error) {
	meta, err := s.vaultMetadata()
	if err != nil {
		return VaultStatus{}, err
	}
	if err := ensureCanAttempt(meta); err != nil {
		return VaultStatus{}, err
	}
	if !meta.VerifyPIN(pin) {
		_ = s.recordVaultFailure(meta)
		return VaultStatus{}, vault.ErrInvalidPIN
	}
	key, err := meta.VaultKey()
	if err != nil {
		return VaultStatus{}, err
	}
	meta.FailedAttempts = 0
	meta.LastFailedAt = 0
	_ = s.saveVaultMetadata(meta)
	s.setVaultKey(key)
	s.emitVaultChanged()
	return s.VaultStatus()
}

func (s *Service) UnlockVaultWithCode(code string) (VaultStatus, error) {
	meta, key, err := s.verifyVaultCode(code)
	if err != nil {
		return VaultStatus{}, err
	}
	meta.FailedAttempts = 0
	meta.LastFailedAt = 0
	_ = s.saveVaultMetadata(meta)
	s.setVaultKey(key)
	s.emitVaultChanged()
	return s.VaultStatus()
}

func (s *Service) ResetVaultPIN(code, pin, confirm string) (VaultStatus, error) {
	if strings.TrimSpace(pin) == "" {
		return VaultStatus{}, errors.New("PIN/password is required")
	}
	if pin != confirm {
		return VaultStatus{}, errors.New("PIN/password confirmation does not match")
	}
	meta, key, err := s.verifyVaultCode(code)
	if err != nil {
		return VaultStatus{}, err
	}
	meta, err = meta.WithNewPIN(pin)
	if err != nil {
		return VaultStatus{}, err
	}
	if err := s.saveVaultMetadata(meta); err != nil {
		return VaultStatus{}, err
	}
	s.setVaultKey(key)
	s.emitVaultChanged()
	return s.VaultStatus()
}

func (s *Service) LockVault() error {
	s.vaultMu.Lock()
	s.vaultKey = nil
	s.vaultExpiry = time.Time{}
	s.vaultMu.Unlock()
	s.emitVaultChanged()
	return nil
}

func (s *Service) ListVaultItems() ([]VaultEntryView, error) {
	key, err := s.requireVaultKey()
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListVaultEntries()
	if err != nil {
		return nil, err
	}
	out := make([]VaultEntryView, 0, len(rows))
	for _, row := range rows {
		plain, err := vault.OpenEntry(key, row.Payload, row.Nonce)
		if err != nil {
			return nil, err
		}
		out = append(out, VaultEntryView{
			ID:          row.ID,
			Title:       plain.Title,
			ContentType: clipboard.ContentType(plain.ContentType),
			Preview:     vaultPreview(plain),
			ImageThumb:  plain.ImageThumb,
			ImageW:      plain.ImageW,
			ImageH:      plain.ImageH,
			CreatedAt:   row.CreatedAt,
			LastUsedAt:  row.LastUsedAt,
		})
	}
	return out, nil
}

func (s *Service) MoveItemToVault(id int64) error {
	key, err := s.requireVaultKey()
	if err != nil {
		return err
	}
	full, err := s.store.GetFull(id)
	if err != nil {
		return err
	}
	plain := vault.PlainEntry{
		ContentType: string(full.Item.ContentType),
		Text:        full.Item.TextContent,
		ImagePNG:    full.ImageBlob,
		ImageThumb:  full.Item.ImageThumb,
		ImageW:      full.Item.ImageW,
		ImageH:      full.Item.ImageH,
	}
	payload, nonce, err := vault.SealEntry(key, plain)
	if err != nil {
		return err
	}
	if _, err := s.store.AddVaultEntry(full.Item.ContentType, payload, nonce, full.Item.ContentHash); err != nil {
		return err
	}
	if _, err := s.store.Delete(id); err != nil {
		return err
	}
	s.emitNewItem()
	s.emitVaultChanged()
	return nil
}

func (s *Service) CopyVaultItem(id int64) error {
	key, err := s.requireVaultKey()
	if err != nil {
		return err
	}
	row, err := s.store.GetVaultEntry(id)
	if err != nil {
		return err
	}
	plain, err := vault.OpenEntry(key, row.Payload, row.Nonce)
	if err != nil {
		return err
	}
	switch clipboard.ContentType(plain.ContentType) {
	case clipboard.ContentTypeText:
		return clipboard.WriteText(plain.Text)
	case clipboard.ContentTypeImage:
		return clipboard.WriteImagePNG(plain.ImagePNG)
	default:
		return fmt.Errorf("unknown vault content_type: %s", plain.ContentType)
	}
}

func (s *Service) RevealVaultItem(id int64) (VaultSecretView, error) {
	key, err := s.requireVaultKey()
	if err != nil {
		return VaultSecretView{}, err
	}
	row, err := s.store.GetVaultEntry(id)
	if err != nil {
		return VaultSecretView{}, err
	}
	plain, err := vault.OpenEntry(key, row.Payload, row.Nonce)
	if err != nil {
		return VaultSecretView{}, err
	}
	if clipboard.ContentType(plain.ContentType) != clipboard.ContentTypeText {
		return VaultSecretView{ContentType: clipboard.ContentType(plain.ContentType)}, nil
	}
	return VaultSecretView{
		ContentType: clipboard.ContentType(plain.ContentType),
		Text:        plain.Text,
	}, nil
}

func (s *Service) UpdateVaultItemTitle(id int64, title string) error {
	key, err := s.requireVaultKey()
	if err != nil {
		return err
	}
	row, err := s.store.ReadVaultEntry(id)
	if err != nil {
		return err
	}
	plain, err := vault.OpenEntry(key, row.Payload, row.Nonce)
	if err != nil {
		return err
	}
	plain.Title = normalizeVaultTitle(title)
	payload, nonce, err := vault.SealEntry(key, plain)
	if err != nil {
		return err
	}
	if err := s.store.UpdateVaultEntryPayload(id, payload, nonce); err != nil {
		return err
	}
	s.emitVaultChanged()
	return nil
}

func (s *Service) DeleteVaultItem(id int64) error {
	if _, err := s.requireVaultKey(); err != nil {
		return err
	}
	if err := s.store.DeleteVaultEntry(id); err != nil {
		return err
	}
	s.emitVaultChanged()
	return nil
}

// ClearAll wipes history. When keepPinned is true, pinned rows survive.
func (s *Service) ClearAll(keepPinned bool) error {
	if err := s.store.ClearAll(keepPinned); err != nil {
		return err
	}
	if s.ctx != nil {
		wailsruntime.EventsEmit(s.ctx, EventCleared)
	}
	return nil
}

// GetSettings returns the persisted settings, falling back to defaults
// for any keys that have never been set.
func (s *Service) GetSettings() (clipboard.Settings, error) {
	def := clipboard.DefaultSettings()
	out := def
	get := func(k, fb string) string {
		v, _ := s.store.GetSetting(k, fb)
		return v
	}
	out.Hotkey = get("hotkey", def.Hotkey)
	out.Theme = get("theme", def.Theme)
	out.MaxItems = atoiOr(get("max_items", strconv.Itoa(def.MaxItems)), def.MaxItems)
	out.MaxImageMB = atoiOr(get("max_image_mb", strconv.Itoa(def.MaxImageMB)), def.MaxImageMB)
	out.KeepImages = get("keep_images", boolStr(def.KeepImages)) == "1"
	out.HideOnBlur = get("hide_on_blur", boolStr(def.HideOnBlur)) == "1"
	out.LaunchAtTop = get("launch_at_top", boolStr(def.LaunchAtTop)) == "1"
	out.WindowFrame = get("window_frame", boolStr(def.WindowFrame)) == "1"
	out.AutoPaste = get("auto_paste", boolStr(def.AutoPaste)) == "1"
	out.DownloadDir = get("download_dir", def.DownloadDir)
	out.PinnedOnTop = get("pinned_on_top", boolStr(def.PinnedOnTop)) == "1"
	out.ShowMemory = get("show_memory", boolStr(def.ShowMemory)) == "1"
	out.RememberPosition = get("remember_position", boolStr(def.RememberPosition)) == "1"
	out.BackupEnabled = get("backup_enabled", boolStr(def.BackupEnabled)) == "1"
	out.BackupTime = get("backup_time", def.BackupTime)
	out.BackupDir = get("backup_dir", def.BackupDir)
	out.BackupIncludeVault = get("backup_include_vault", boolStr(def.BackupIncludeVault)) == "1"
	out.BackupIncludePinned = get("backup_include_pinned", boolStr(def.BackupIncludePinned)) == "1"
	out.BackupCleanAfter = get("backup_clean_after", boolStr(def.BackupCleanAfter)) == "1"
	s.settingsLog.Store(out)
	return out, nil
}

// CurrentSettings returns the most-recently loaded settings, or the
// defaults if nothing has been cached yet. Used by ingestion paths that
// shouldn't hit SQLite for every clipboard change.
func (s *Service) CurrentSettings() clipboard.Settings {
	if v, ok := s.settingsLog.Load().(clipboard.Settings); ok {
		return v
	}
	v, _ := s.GetSettings()
	return v
}

// UpdateSettings persists a Settings struct and reapplies GUI side-effects.
func (s *Service) UpdateSettings(in clipboard.Settings) (clipboard.Settings, error) {
	prev, _ := s.GetSettings()
	if in.MaxItems <= 0 {
		in.MaxItems = prev.MaxItems
	}
	if in.MaxImageMB <= 0 {
		in.MaxImageMB = prev.MaxImageMB
	}
	if in.Hotkey == "" {
		in.Hotkey = prev.Hotkey
	}
	if in.Theme == "" {
		in.Theme = prev.Theme
	}
	put := func(k, v string) error { return s.store.SetSetting(k, v) }
	if err := put("hotkey", in.Hotkey); err != nil {
		return prev, err
	}
	if err := put("theme", in.Theme); err != nil {
		return prev, err
	}
	if err := put("max_items", strconv.Itoa(in.MaxItems)); err != nil {
		return prev, err
	}
	if err := put("max_image_mb", strconv.Itoa(in.MaxImageMB)); err != nil {
		return prev, err
	}
	if err := put("keep_images", boolStr(in.KeepImages)); err != nil {
		return prev, err
	}
	if err := put("hide_on_blur", boolStr(in.HideOnBlur)); err != nil {
		return prev, err
	}
	if err := put("launch_at_top", boolStr(in.LaunchAtTop)); err != nil {
		return prev, err
	}
	if err := put("window_frame", boolStr(in.WindowFrame)); err != nil {
		return prev, err
	}
	if err := put("auto_paste", boolStr(in.AutoPaste)); err != nil {
		return prev, err
	}
	// Empty is a valid value here: it means "ask where to save every time".
	if err := put("download_dir", in.DownloadDir); err != nil {
		return prev, err
	}
	if err := put("pinned_on_top", boolStr(in.PinnedOnTop)); err != nil {
		return prev, err
	}
	if err := put("show_memory", boolStr(in.ShowMemory)); err != nil {
		return prev, err
	}
	if err := put("remember_position", boolStr(in.RememberPosition)); err != nil {
		return prev, err
	}
	// Backup time must be a valid HH:MM; keep the previous value otherwise.
	if _, err := time.Parse("15:04", in.BackupTime); err != nil {
		in.BackupTime = prev.BackupTime
	}
	if err := put("backup_enabled", boolStr(in.BackupEnabled)); err != nil {
		return prev, err
	}
	if err := put("backup_time", in.BackupTime); err != nil {
		return prev, err
	}
	if err := put("backup_dir", in.BackupDir); err != nil {
		return prev, err
	}
	if err := put("backup_include_vault", boolStr(in.BackupIncludeVault)); err != nil {
		return prev, err
	}
	if err := put("backup_include_pinned", boolStr(in.BackupIncludePinned)); err != nil {
		return prev, err
	}
	if err := put("backup_clean_after", boolStr(in.BackupCleanAfter)); err != nil {
		return prev, err
	}
	if s.hotkeyMgr != nil && in.Hotkey != prev.Hotkey {
		if err := s.hotkeyMgr.Register(in.Hotkey); err != nil {
			return prev, fmt.Errorf("re-register hotkey: %w", err)
		}
	}
	s.settingsLog.Store(in)
	return in, nil
}

func (s *Service) vaultMetadata() (vault.Metadata, error) {
	raw, err := s.store.GetSetting(vaultSettingsKey, "")
	if err != nil {
		return vault.Metadata{}, err
	}
	return vault.DecodeMetadata(raw)
}

func (s *Service) saveVaultMetadata(meta vault.Metadata) error {
	raw, err := vault.EncodeMetadata(meta)
	if err != nil {
		return err
	}
	return s.store.SetSetting(vaultSettingsKey, raw)
}

func (s *Service) vaultUnlocked() bool {
	s.vaultMu.Lock()
	defer s.vaultMu.Unlock()
	if len(s.vaultKey) == 0 {
		return false
	}
	if time.Now().After(s.vaultExpiry) {
		s.vaultKey = nil
		s.vaultExpiry = time.Time{}
		return false
	}
	return true
}

func (s *Service) setVaultKey(key []byte) {
	s.vaultMu.Lock()
	defer s.vaultMu.Unlock()
	s.vaultKey = append([]byte(nil), key...)
	s.vaultExpiry = time.Now().Add(vaultInactivityPeriod)
}

func (s *Service) requireVaultKey() ([]byte, error) {
	s.vaultMu.Lock()
	defer s.vaultMu.Unlock()
	if len(s.vaultKey) == 0 || time.Now().After(s.vaultExpiry) {
		s.vaultKey = nil
		s.vaultExpiry = time.Time{}
		return nil, errors.New("private vault is locked")
	}
	s.vaultExpiry = time.Now().Add(vaultInactivityPeriod)
	return append([]byte(nil), s.vaultKey...), nil
}

func (s *Service) verifyVaultCode(code string) (vault.Metadata, []byte, error) {
	meta, err := s.vaultMetadata()
	if err != nil {
		return vault.Metadata{}, nil, err
	}
	if !meta.Configured {
		return vault.Metadata{}, nil, errors.New("private vault is not configured")
	}
	if err := ensureCanAttempt(meta); err != nil {
		return vault.Metadata{}, nil, err
	}
	secret, err := meta.TOTPSecret()
	if err != nil {
		return vault.Metadata{}, nil, err
	}
	if !vault.ValidTOTP(secret, code, time.Now()) {
		_ = s.recordVaultFailure(meta)
		return vault.Metadata{}, nil, vault.ErrInvalidCode
	}
	key, err := meta.VaultKey()
	if err != nil {
		return vault.Metadata{}, nil, err
	}
	return meta, key, nil
}

func (s *Service) recordVaultFailure(meta vault.Metadata) error {
	meta.FailedAttempts++
	meta.LastFailedAt = time.Now().Unix()
	return s.saveVaultMetadata(meta)
}

func ensureCanAttempt(meta vault.Metadata) error {
	until := lockoutUntil(meta)
	if until > time.Now().Unix() {
		return fmt.Errorf("too many failed attempts; try again later")
	}
	return nil
}

func lockoutUntil(meta vault.Metadata) int64 {
	if meta.FailedAttempts < 5 || meta.LastFailedAt == 0 {
		return 0
	}
	return meta.LastFailedAt + 60
}

func vaultPreview(entry vault.PlainEntry) string {
	switch clipboard.ContentType(entry.ContentType) {
	case clipboard.ContentTypeImage:
		if entry.ImageW > 0 && entry.ImageH > 0 {
			return fmt.Sprintf("Image %dx%d", entry.ImageW, entry.ImageH)
		}
		return "Image"
	default:
		s := strings.ReplaceAll(entry.Text, "\r", " ")
		s = strings.ReplaceAll(s, "\n", " ")
		s = strings.ReplaceAll(s, "\t", " ")
		s = strings.Join(strings.Fields(s), " ")
		if len(s) > 80 {
			s = s[:80] + "..."
		}
		return s
	}
}

func normalizeVaultTitle(title string) string {
	title = strings.ReplaceAll(title, "\r", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\t", " ")
	title = strings.Join(strings.Fields(title), " ")
	if runes := []rune(title); len(runes) > 120 {
		title = string(runes[:120])
	}
	return title
}

// ----- Window control -----

// ShowPopup brings the popup window to the foreground.
func (s *Service) ShowPopup() {
	if s.ctx == nil {
		return
	}
	clipboard.RememberPasteTarget()
	wailsruntime.WindowShow(s.ctx)
	wailsruntime.WindowUnminimise(s.ctx) // restore if minimised to the taskbar
	// In windowed mode the user owns the window position and stacking, so we
	// don't force always-on-top or re-center on every summon. In popup mode
	// we keep the classic Windows+V behaviour: float on top — reopening at
	// the position it was last closed from (saved in HidePopup), which also
	// keeps it on the monitor the user last used it on. Falls back to
	// centering when disabled or before any position has been saved.
	if settings := s.CurrentSettings(); !settings.WindowFrame {
		wailsruntime.WindowSetAlwaysOnTop(s.ctx, true)
		if !settings.RememberPosition || !s.restoreLastPosition() {
			wailsruntime.WindowCenter(s.ctx)
		}
	}
	// A programmatic show gets no input timestamp, so GNOME denies focus and
	// flags the window "demands attention" — which pulses the dock icon and
	// pops a "ready" toast every time. Activating it (under XWayland) once it's
	// mapped clears that flag and hands the popup keyboard focus.
	go func() {
		time.Sleep(120 * time.Millisecond)
		// Only re-apply the skip-taskbar/utility hints in popup mode; in
		// windowed mode they would strip the taskbar entry and break minimise.
		settings := s.CurrentSettings()
		x11hint.Nudge(!settings.WindowFrame)
		// Re-apply the restored position once the window is definitely
		// mapped — a move issued in the same tick as the first map can be
		// dropped by the window manager.
		if !settings.WindowFrame && settings.RememberPosition {
			s.restoreLastPosition()
		}
	}()
	// The clipboard watcher runs in the separate clipd-watch process, so this GUI
	// doesn't learn about new items live. Refresh the list every time the popup
	// is shown so it reflects whatever the worker has captured.
	s.emitNewItem()
	s.visible.Store(true)
}

// restoreLastPosition moves the popup to wherever it was last closed from
// (saved by HidePopup). Returns false when no position has been saved yet so
// the caller can fall back to centering.
//
// Save and restore both go through xdotool (raw X11 coordinates) rather than
// GTK: with mutter's xwayland-native-scaling, GTK's coordinate space diverges
// from X11's (observed: X halved, Y unchanged), while the xdotool read+move
// pair round-trips exactly.
func (s *Service) restoreLastPosition() bool {
	xs, _ := s.store.GetSetting("popup_x", "")
	ys, _ := s.store.GetSetting("popup_y", "")
	if xs == "" || ys == "" {
		return false
	}
	x, errX := strconv.Atoi(xs)
	y, errY := strconv.Atoi(ys)
	if errX != nil || errY != nil {
		return false
	}
	return x11hint.MoveWindow(x, y)
}

// saveLastPosition persists the popup's current position so the next summon
// reopens it exactly where the user left it (survives restarts via the DB).
// Skips saving when the position can't be read (e.g. no X11 window) so a
// previously saved good position is never overwritten with garbage.
func (s *Service) saveLastPosition() {
	x, y, ok := x11hint.WindowPosition()
	if !ok {
		return
	}
	_ = s.store.SetSetting("popup_x", strconv.Itoa(x))
	_ = s.store.SetSetting("popup_y", strconv.Itoa(y))
}

// HidePopup hides the popup window.
func (s *Service) HidePopup() {
	if s.ctx == nil {
		return
	}
	// Capture the position while the window is still mapped — this is what
	// "reopen where I closed it" restores from.
	if !s.CurrentSettings().WindowFrame {
		s.saveLastPosition()
	}
	wailsruntime.WindowHide(s.ctx)
	s.visible.Store(false)
}

// TogglePopup is the canonical entrypoint for the hotkey and the tray.
func (s *Service) TogglePopup() {
	if s.visible.Load() {
		s.HidePopup()
	} else {
		s.ShowPopup()
	}
}

// IsVisible returns the current popup visibility flag. The JS side uses
// this to keep its own state in sync (e.g. after a manual close).
func (s *Service) IsVisible() bool { return s.visible.Load() }

// SystemInfo returns a small map of host facts used by the UI for
// session-aware messaging (e.g. "you're on Wayland — hotkey disabled").
func (s *Service) SystemInfo() map[string]string {
	return map[string]string{
		"sessionType": os.Getenv("XDG_SESSION_TYPE"),
		"desktop":     os.Getenv("XDG_CURRENT_DESKTOP"),
	}
}

// ----- helpers -----

func atoiOr(s string, fb int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fb
	}
	return n
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func sumHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Ensure unused imports are referenced if features change.
var _ = errors.New
