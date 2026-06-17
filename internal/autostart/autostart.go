// Package autostart manages whether clipd launches on login.
// Platform-specific implementations in autostart_windows.go and autostart_linux.go
package autostart

// Manager implements service.AutostartManager.
type Manager struct {
	ExecOverride string
}

// New returns a Manager for the current platform.
func New() *Manager { return &Manager{} }
