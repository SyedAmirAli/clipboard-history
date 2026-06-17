//go:build linux

package clipboard

import (
	"fmt"
	"os/exec"
	"syscall"
)

// SaveFocusedWindow is a no-op on Linux.
// On Windows, this stores the focused window before showing the popup.
func SaveFocusedWindow() {
	// Linux doesn't need focus restoration as xdotool handles it
}

// SendPaste synthesises a Ctrl+V key event into the focused window using xdotool.
// On Wayland, this is a no-op because Wayland forbids synthetic input injection.
func SendPaste() error {
	// Wayland forbids arbitrary clients from synthesising input into other
	// windows. The item is already on the clipboard, so we skip auto-paste
	// and let the user press Ctrl+V themselves.
	if IsWayland() {
		return nil
	}

	path, err := exec.LookPath("xdotool")
	if err != nil {
		return fmt.Errorf("xdotool not found in PATH: install it with 'sudo apt install xdotool' to enable auto-paste")
	}

	// --clearmodifiers ensures any modifier the user is still physically
	// holding (e.g. the Super of the open-shortcut) doesn't corrupt the chord.
	cmd := exec.Command(path, "key", "--clearmodifiers", "ctrl+v")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xdotool: %w", err)
	}
	go func() { _ = cmd.Wait() }() // best-effort reap
	return nil
}
