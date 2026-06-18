// Package autostart manages whether clipd launches on login.
// The implementation lives in autostart_windows.go (Windows registry).
package autostart

// Manager implements service.AutostartManager.
type Manager struct {
	ExecOverride string
}

// New returns a Manager for the current platform.
func New() *Manager { return &Manager{} }
