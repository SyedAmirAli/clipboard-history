// Package autostart manages a freedesktop.org autostart .desktop entry
// at $XDG_CONFIG_HOME/autostart/clipd.desktop so the app launches on
// the user's next login.
package autostart

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"clipd/internal/config"
)

// Default path of the installed binary on Debian/Ubuntu (via our .deb).
const defaultExec = "/usr/bin/clipd"

// desktopTemplate is the body written to ~/.config/autostart/clipd.desktop
// when autostart is enabled. The fields here mirror the production
// .desktop file shipped in the .deb.
const desktopTemplate = `[Desktop Entry]
Type=Application
Name=clipd
GenericName=Clipboard History
Comment=Lightweight clipboard history with pinning and image support
Exec=%s
Icon=clipd
Terminal=false
Categories=Utility;
StartupNotify=false
X-GNOME-Autostart-enabled=true
`

// Manager implements service.AutostartManager.
type Manager struct {
	// ExecOverride lets tests / dev mode use a non-default binary path.
	// Empty means use defaultExec or the running binary when applicable.
	ExecOverride string
}

// New returns a Manager that writes to the user's autostart directory.
func New() *Manager { return &Manager{} }

// SetEnabled writes or removes the autostart .desktop entry.
func (m *Manager) SetEnabled(enabled bool) error {
	path, err := config.AutostartFilePath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	exec := m.execPath()
	body := []byte(formatDesktop(exec))
	return os.WriteFile(path, body, 0o644)
}

// IsEnabled returns true when the autostart entry exists.
func (m *Manager) IsEnabled() (bool, error) {
	path, err := config.AutostartFilePath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
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

func formatDesktop(execPath string) string {
	return replaceFirst(desktopTemplate, "%s", execPath)
}

// replaceFirst is a tiny helper to avoid pulling in fmt for a single sub.
func replaceFirst(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
