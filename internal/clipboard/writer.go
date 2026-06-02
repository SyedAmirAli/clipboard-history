package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// WriteText puts the given UTF-8 string onto the X11 CLIPBOARD selection.
//
// xclip stays alive in the background after it acquires the selection;
// we Start it in a new session group so it survives clipd's lifecycle
// and the calling goroutine returns immediately.
func WriteText(s string) error {
	return spawnXclipWith([]string{"-selection", "clipboard"}, []byte(s))
}

// WriteImagePNG puts a PNG image onto the X11 CLIPBOARD selection so it
// can be pasted into image-aware applications (browsers, image editors,
// office suites, etc.).
func WriteImagePNG(png []byte) error {
	return spawnXclipWith([]string{"-selection", "clipboard", "-t", "image/png"}, png)
}

func spawnXclipWith(args []string, payload []byte) error {
	if _, err := exec.LookPath("xclip"); err != nil {
		return fmt.Errorf("xclip not found in PATH: install it with 'sudo apt install xclip'")
	}
	cmd := exec.Command("xclip", args...)
	cmd.Stdin = bytes.NewReader(payload)
	// Detach into a new session so xclip survives even if clipd quits and
	// so we don't accidentally wait on it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xclip: %w", err)
	}
	// Intentionally do NOT Wait — xclip keeps running as the selection
	// owner. The OS will reap it once another process takes the selection.
	go func() {
		_ = cmd.Wait() // best-effort reap; ignored if killed externally
	}()
	return nil
}
