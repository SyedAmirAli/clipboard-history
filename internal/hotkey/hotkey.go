//go:build linux

// Package hotkey registers a single global keyboard shortcut on X11.
//
// The hotkey string format is a "+"-separated list of modifiers and a
// single key, e.g. "Super+V", "Ctrl+Alt+H", "Ctrl+Shift+,". Modifier
// names are case-insensitive. Recognised modifiers:
//
//	Super, Meta, Mod4 → Mod4Mask  (Linux "Windows" key)
//	Ctrl, Control     → ControlMask
//	Alt, Mod1         → Mod1Mask
//	Shift             → ShiftMask
//
// The trailing token must be a single ASCII alphanumeric or one of a
// small set of named keys ("Space", "Enter", "Tab", "Escape").
package hotkey

import (
	"context"
	"fmt"
	"strings"

	gh "golang.design/x/hotkey"
)

// Manager owns the current registered hotkey and exposes a channel of
// press events. Use Replace() to change the hotkey at runtime — the
// previous registration is unregistered first.
type Manager struct {
	current *gh.Hotkey
	events  chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
}

// New returns an idle Manager. Call Register to install a shortcut.
func New() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		events: make(chan struct{}, 4),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Events returns the channel that receives a struct{}{} value each time
// the registered hotkey is pressed.
func (m *Manager) Events() <-chan struct{} { return m.events }

// Register installs (or replaces) the global hotkey to `spec`.
func (m *Manager) Register(spec string) error {
	mods, key, err := parseSpec(spec)
	if err != nil {
		return err
	}
	hk := gh.New(mods, key)
	if err := hk.Register(); err != nil {
		return fmt.Errorf("register hotkey %q: %w", spec, err)
	}
	if m.current != nil {
		_ = m.current.Unregister()
	}
	m.current = hk
	go m.pump(hk)
	return nil
}

// Close unregisters the active hotkey (if any) and stops event pumping.
func (m *Manager) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.current != nil {
		_ = m.current.Unregister()
		m.current = nil
	}
}

func (m *Manager) pump(hk *gh.Hotkey) {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-hk.Keydown():
			select {
			case m.events <- struct{}{}:
			default: // never block; one queued press is enough
			}
		}
	}
}

func parseSpec(spec string) ([]gh.Modifier, gh.Key, error) {
	parts := strings.Split(spec, "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("hotkey %q: expected at least one modifier + a key", spec)
	}
	var mods []gh.Modifier
	for _, p := range parts[:len(parts)-1] {
		m, ok := modifierFor(strings.TrimSpace(p))
		if !ok {
			return nil, 0, fmt.Errorf("hotkey %q: unknown modifier %q", spec, p)
		}
		mods = append(mods, m)
	}
	key, ok := keyFor(strings.TrimSpace(parts[len(parts)-1]))
	if !ok {
		return nil, 0, fmt.Errorf("hotkey %q: unsupported key %q", spec, parts[len(parts)-1])
	}
	return mods, key, nil
}

// On Linux, golang.design/x/hotkey only exposes raw X11 modifier masks
// (Mod1..Mod5) plus ModCtrl/ModShift. We map the friendly names to the
// correct mask: Super/Meta/Win → Mod4, Alt → Mod1.
func modifierFor(name string) (gh.Modifier, bool) {
	switch strings.ToLower(name) {
	case "super", "meta", "mod4", "win":
		return gh.Mod4, true
	case "ctrl", "control":
		return gh.ModCtrl, true
	case "alt", "mod1":
		return gh.Mod1, true
	case "shift":
		return gh.ModShift, true
	}
	return 0, false
}

func keyFor(name string) (gh.Key, bool) {
	if len(name) == 1 {
		c := name[0]
		switch {
		case c >= 'a' && c <= 'z':
			return letterKey(c - 'a' + 'A')
		case c >= 'A' && c <= 'Z':
			return letterKey(c)
		case c >= '0' && c <= '9':
			return digitKey(c)
		}
	}
	switch strings.ToLower(name) {
	case "space":
		return gh.KeySpace, true
	case "enter", "return":
		return gh.KeyReturn, true
	case "tab":
		return gh.KeyTab, true
	case "escape", "esc":
		return gh.KeyEscape, true
	}
	return 0, false
}

func letterKey(c byte) (gh.Key, bool) {
	switch c {
	case 'A':
		return gh.KeyA, true
	case 'B':
		return gh.KeyB, true
	case 'C':
		return gh.KeyC, true
	case 'D':
		return gh.KeyD, true
	case 'E':
		return gh.KeyE, true
	case 'F':
		return gh.KeyF, true
	case 'G':
		return gh.KeyG, true
	case 'H':
		return gh.KeyH, true
	case 'I':
		return gh.KeyI, true
	case 'J':
		return gh.KeyJ, true
	case 'K':
		return gh.KeyK, true
	case 'L':
		return gh.KeyL, true
	case 'M':
		return gh.KeyM, true
	case 'N':
		return gh.KeyN, true
	case 'O':
		return gh.KeyO, true
	case 'P':
		return gh.KeyP, true
	case 'Q':
		return gh.KeyQ, true
	case 'R':
		return gh.KeyR, true
	case 'S':
		return gh.KeyS, true
	case 'T':
		return gh.KeyT, true
	case 'U':
		return gh.KeyU, true
	case 'V':
		return gh.KeyV, true
	case 'W':
		return gh.KeyW, true
	case 'X':
		return gh.KeyX, true
	case 'Y':
		return gh.KeyY, true
	case 'Z':
		return gh.KeyZ, true
	}
	return 0, false
}

func digitKey(c byte) (gh.Key, bool) {
	switch c {
	case '0':
		return gh.Key0, true
	case '1':
		return gh.Key1, true
	case '2':
		return gh.Key2, true
	case '3':
		return gh.Key3, true
	case '4':
		return gh.Key4, true
	case '5':
		return gh.Key5, true
	case '6':
		return gh.Key6, true
	case '7':
		return gh.Key7, true
	case '8':
		return gh.Key8, true
	case '9':
		return gh.Key9, true
	}
	return 0, false
}
