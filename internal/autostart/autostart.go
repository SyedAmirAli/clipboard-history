// Package autostart manages whether clipd launches on login.
//
// The .deb installs a system-wide entry at /etc/xdg/autostart/clipd.desktop,
// so clipd autostarts for every user out of the box. The in-app toggle then
// layers a per-user file at $XDG_CONFIG_HOME/autostart/clipd.desktop on top:
// freedesktop precedence means the user entry overrides the system one, so we
// disable autostart by writing a user entry with Hidden=true, and re-enable by
// removing it again. When no system entry exists (e.g. a dev build run from
// source) the user entry is a normal autostart file instead.
package autostart

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"clipd/internal/config"
)

// Default path of the installed binary on Debian/Ubuntu (via our .deb).
const defaultExec = "/usr/bin/clipd"

// systemEntry is where the .deb installs the system-wide autostart file.
const systemEntry = "/etc/xdg/autostart/clipd.desktop"

// desktopTemplate is the body written to ~/.config/autostart/clipd.desktop.
// Exec uses `start` so login launches the daemon (a bare `clipd` would
// toggle the window of an already-running instance). Hidden is set to true
// to suppress a system-wide entry, false for a plain user autostart.
const desktopTemplate = `[Desktop Entry]
Type=Application
Name=clipd
GenericName=Clipboard History
Comment=Lightweight clipboard history with pinning and image support
Exec=%s start
Icon=clipd
Terminal=false
Categories=Utility;
StartupNotify=false
X-GNOME-Autostart-enabled=true
Hidden=%t
`

// Manager implements service.AutostartManager.
type Manager struct {
	// ExecOverride lets tests / dev mode use a non-default binary path.
	// Empty means use defaultExec or the running binary when applicable.
	ExecOverride string
}

// New returns a Manager that writes to the user's autostart directory.
func New() *Manager { return &Manager{} }

// SetEnabled turns login autostart on or off, reconciling the per-user
// entry against the system-wide one installed by the .deb.
func (m *Manager) SetEnabled(enabled bool) error {
	path, err := config.AutostartFilePath()
	if err != nil {
		return err
	}
	if enabled {
		// With a system entry present, autostart is already the default —
		// just drop any user override that was disabling it. Without one
		// (dev build), write a plain user autostart entry.
		if systemEntryExists() {
			return removeIfExists(path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(formatDesktop(m.execPath(), false)), 0o644)
	}
	// Disabling: override the system entry with a Hidden user entry, or —
	// if there's no system entry — simply remove the user one.
	if systemEntryExists() {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(formatDesktop(m.execPath(), true)), 0o644)
	}
	return removeIfExists(path)
}

// IsEnabled reports whether clipd will autostart on the next login. A user
// entry takes precedence (Hidden=true means disabled); with no user entry,
// autostart is on iff the system-wide entry exists.
func (m *Manager) IsEnabled() (bool, error) {
	path, err := config.AutostartFilePath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		return !strings.Contains(string(data), "Hidden=true"), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return systemEntryExists(), nil
}

func systemEntryExists() bool {
	_, err := os.Stat(systemEntry)
	return err == nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) execPath() string {
	if m.ExecOverride != "" {
		return m.ExecOverride
	}
	if _, err := os.Stat(defaultExec); err == nil {
		return defaultExec
	}
	if p, err := os.Executable(); err == nil {
		if abs, aerr := filepath.Abs(p); aerr == nil {
			return abs
		}
		return p
	}
	return defaultExec
}

func formatDesktop(execPath string, hidden bool) string {
	return fmt.Sprintf(desktopTemplate, execPath, hidden)
}
