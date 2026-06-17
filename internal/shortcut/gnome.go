// Package shortcut handles global keyboard shortcut registration.
// On Linux: registers with GNOME/Wayland via gsettings.
// On Windows: hotkeys are handled by the hotkey manager directly.
// Platform-specific implementations in shortcut_linux.go and shortcut_windows.go.
package shortcut
