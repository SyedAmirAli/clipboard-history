package clipboard

import (
	"fmt"
	"os/exec"
	"syscall"
)

// SendPaste synthesises a Ctrl+V key event into whatever window currently
// holds keyboard focus, using xdotool. clipd calls this right after it copies
// a selected history item and hides itself, so the value lands directly in the
// field the user was editing — no manual Ctrl+V needed.
//
// It is best-effort: the clipboard has already been written by the time this
// runs, so if xdotool is missing we just return an error for the caller to log
// and the user can still paste manually.
func SendPaste() error {
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
