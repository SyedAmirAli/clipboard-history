//go:build linux

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

// Default path of the installed binary on Debian/Ubuntu (via the .deb).
const defaultExec = "/usr/bin/clipd"

// systemEntry is where the .deb installs the system-wide autostart file.
const systemEntry = "/etc/xdg/autostart/clipd.desktop"

// desktopTemplate is the body written to ~/.config/autostart/clipd.desktop.
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

// SetEnabled turns login autostart on or off for Linux.
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

// IsEnabled reports whether clipd will autostart on the next login.
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
