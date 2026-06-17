// Package autostart manages whether clipd launches on login.
// Platform-specific implementations in autostart_windows.go and autostart_linux.go
package autostart

// Manager implements service.AutostartManager.
type Manager struct {
	ExecOverride string
}

// New returns a Manager for the current platform.
func New() *Manager { return &Manager{} }

// SetEnabled and IsEnabled are implemented in platform-specific files.
func (m *Manager) SetEnabled(enabled bool) error {
	panic("SetEnabled must be implemented by platform-specific code")
}

func (m *Manager) IsEnabled() (bool, error) {
	panic("IsEnabled must be implemented by platform-specific code")
}
