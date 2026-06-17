//go:build windows

package shortcut

import (
	"fmt"
)

// Available reports whether shortcut installation is available on Windows.
// Windows hotkeys are already registered in the hotkey manager, so this is not needed.
func Available() bool {
	return false
}

// EnsureInstalled is a no-op on Windows.
// Hotkey registration happens in the hotkey manager.
func EnsureInstalled(spec string) error {
	return fmt.Errorf("shortcut installation not needed on Windows; hotkey already registered")
}

// Remove is a no-op on Windows.
func Remove() error {
	return nil
}
