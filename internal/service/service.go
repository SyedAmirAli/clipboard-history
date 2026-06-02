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
	"os"
	"strconv"
	"sync/atomic"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"clipd/internal/clipboard"
	"clipd/internal/db"
	"clipd/internal/thumbnail"
)

// EventNewItem is the Wails event published whenever a new clipboard
// entry is stored. Frontend subscribers can refresh their list.
const EventNewItem = "clipboard:new-item"

// EventCleared is published after the user clears the history.
const EventCleared = "clipboard:cleared"

// HotkeySetter abstracts the hotkey manager so the service can update
// the registered shortcut when settings change.
type HotkeySetter interface {
	Register(spec string) error
}

// WatcherSuppressor lets the service tell the watcher to ignore the
// very next clipboard change matching a known hash — used when pasting
// a history item so it doesn't bounce back through the watcher.
type WatcherSuppressor interface {
	Suppress(hash string)
}

// AutostartManager toggles the freedesktop autostart .desktop entry.
type AutostartManager interface {
	SetEnabled(enabled bool) error
	IsEnabled() (bool, error)
}

// Service is bound onto the Wails runtime. All methods are JS-callable.
type Service struct {
	store       *db.Store
	hotkeyMgr   HotkeySetter
	suppressor  WatcherSuppressor
	autostart   AutostartManager
	visible     atomic.Bool
	ctx         context.Context
	settingsLog atomic.Value // last loaded clipboard.Settings, for hotkey diffing
}

// New constructs a Service with all wiring dependencies.
func New(store *db.Store, hk HotkeySetter, sup WatcherSuppressor, as AutostartManager) *Service {
	return &Service{store: store, hotkeyMgr: hk, suppressor: sup, autostart: as}
}

// AttachContext is called from Wails OnStartup so subsequent Show/Hide
// operations can target the runtime.
func (s *Service) AttachContext(ctx context.Context) { s.ctx = ctx }

// Context exposes the Wails context for collaborators (e.g. tray) that
// need to emit events or control the window from outside the JS bridge.
func (s *Service) Context() context.Context { return s.ctx }

// ----- Clipboard ingestion (called from main, NOT from JS) -----

// IngestText is invoked by the main loop whenever the X11 watcher
// reports a new text payload. It stores the item, enforces the
// max-items cap, and publishes a refresh event to the UI.
func (s *Service) IngestText(text, hash string) error {
	settings := s.CurrentSettings()
	res, err := s.store.AddText(text, hash)
	if err != nil {
		return err
	}
	if res.IsNew {
		_, _ = s.store.TrimToMax(settings.MaxItems)
	}
	s.emitNewItem()
	return nil
}

// IngestImage is invoked by the main loop for image clipboard payloads.
// It auto-rejects images larger than the user-configured cap.
func (s *Service) IngestImage(png []byte, hash string) error {
	settings := s.CurrentSettings()
	if !settings.KeepImages {
		return nil
	}
	if settings.MaxImageMB > 0 && len(png) > settings.MaxImageMB*1024*1024 {
		return fmt.Errorf("image %d bytes exceeds %d MB cap", len(png), settings.MaxImageMB)
	}
	thumb, w, h, err := thumbnail.DataURL(png)
	if err != nil {
		return err
	}
	res, err := s.store.AddImage(png, thumb, hash, w, h)
	if err != nil {
		return err
	}
	if res.IsNew {
		_, _ = s.store.TrimToMax(settings.MaxItems)
	}
	s.emitNewItem()
	return nil
}

func (s *Service) emitNewItem() {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, EventNewItem)
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

// PasteItem writes the stored item back onto the X11 CLIPBOARD selection
// and hides the popup window so the user can immediately Ctrl+V.
func (s *Service) PasteItem(id int64) error {
	ct, text, blob, err := s.store.GetForPaste(id)
	if err != nil {
		return err
	}
	switch ct {
	case clipboard.ContentTypeText:
		h := "t:" + sumHex([]byte(text))
		s.suppressor.Suppress(h)
		if err := clipboard.WriteText(text); err != nil {
			return err
		}
	case clipboard.ContentTypeImage:
		h := "i:" + sumHex(blob)
		s.suppressor.Suppress(h)
		if err := clipboard.WriteImagePNG(blob); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown content_type: %s", ct)
	}
	s.HidePopup()
	return nil
}

// PinItem toggles whether an item should survive eviction.
func (s *Service) PinItem(id int64, pinned bool) error {
	return s.store.SetPinned(id, pinned)
}

// DeleteItem removes a single history entry.
func (s *Service) DeleteItem(id int64) error {
	return s.store.Delete(id)
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
	if s.autostart != nil {
		if v, err := s.autostart.IsEnabled(); err == nil {
			out.Autostart = v
		}
	}
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

// UpdateSettings persists a Settings struct and reapplies side-effects
// (hotkey re-registration, autostart toggle).
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
	if s.hotkeyMgr != nil && in.Hotkey != prev.Hotkey {
		if err := s.hotkeyMgr.Register(in.Hotkey); err != nil {
			return prev, fmt.Errorf("re-register hotkey: %w", err)
		}
	}
	if s.autostart != nil && in.Autostart != prev.Autostart {
		if err := s.autostart.SetEnabled(in.Autostart); err != nil {
			return prev, fmt.Errorf("update autostart: %w", err)
		}
	}
	s.settingsLog.Store(in)
	return in, nil
}

// ----- Window control -----

// ShowPopup brings the popup window to the foreground.
func (s *Service) ShowPopup() {
	if s.ctx == nil {
		return
	}
	wailsruntime.WindowShow(s.ctx)
	wailsruntime.WindowSetAlwaysOnTop(s.ctx, true)
	wailsruntime.WindowCenter(s.ctx)
	s.visible.Store(true)
}

// HidePopup hides the popup window.
func (s *Service) HidePopup() {
	if s.ctx == nil {
		return
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
