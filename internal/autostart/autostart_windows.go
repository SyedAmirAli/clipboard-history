//go:build windows

package autostart

import (
	"fmt"
	"os"
	"os/exec"
)

// Manager implements autostart management for Windows using the Registry via reg.exe.
// Registry path: HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run
type Manager struct {
	ExecOverride string // Override for tests
}

// New returns a Manager for Windows autostart.
func New() *Manager { return &Manager{} }

// SetEnabled turns login autostart on or off by adding/removing a registry entry.
func (m *Manager) SetEnabled(enabled bool) error {
	regPath := `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`
	execPath := m.execPath()

	if enabled {
		// Add registry entry: clipd = <path to executable>
		cmd := exec.Command("reg", "add", regPath, "/v", "clipd", "/d", execPath, "/f")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add autostart registry entry: %w", err)
		}
		return nil
	}

	// Remove registry entry
	cmd := exec.Command("reg", "delete", regPath, "/v", "clipd", "/f")
	if err := cmd.Run(); err != nil {
		// Ignore errors when deleting (entry may not exist)
		return nil
	}
	return nil
}

// IsEnabled reports whether clipd will autostart on the next login.
func (m *Manager) IsEnabled() (bool, error) {
	regPath := `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`

	// Query the registry using reg.exe
	cmd := exec.Command("reg", "query", regPath, "/v", "clipd")
	output, err := cmd.CombinedOutput()

	// If command succeeds and output contains "clipd", it's enabled
	if err == nil && len(output) > 0 {
		return true, nil
	}

	return false, nil
}

func (m *Manager) execPath() string {
	if m.ExecOverride != "" {
		return m.ExecOverride
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "clipd"
}
