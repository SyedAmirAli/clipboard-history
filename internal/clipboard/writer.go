package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// WriteText puts the given UTF-8 string onto the system clipboard, using
// xclip on X11 and wl-copy on Wayland.
//
// The tool stays alive in the background after it acquires the selection;
// we Start it in a new session group so it survives clipd's lifecycle and
// the calling goroutine returns immediately.
func WriteText(s string) error {
	if IsWayland() {
		return spawnWith("wl-copy", nil, []byte(s))
	}
	return spawnWith("xclip", []string{"-selection", "clipboard"}, []byte(s))
}

// WriteImagePNG puts a PNG image onto the system clipboard so it can be
// pasted into image-aware applications (browsers, image editors, office
// suites, etc.).
func WriteImagePNG(png []byte) error {
	if IsWayland() {
		return spawnWith("wl-copy", []string{"--type", "image/png"}, png)
	}
	return spawnWith("xclip", []string{"-selection", "clipboard", "-t", "image/png"}, png)
}

// spawnWith starts a clipboard-owner process (xclip or wl-copy), feeds it
// the payload on stdin, and detaches it into a new session so it survives
// clipd's lifecycle. We don't Wait on it synchronously — it lingers as the
// selection owner until another process takes over; the OS reaps it then.
func spawnWith(name string, args []string, payload []byte) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found in PATH: install it with %s", name, installHint(DetectServer()))
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	go func() {
		_ = cmd.Wait() // best-effort reap; ignored if killed externally
	}()
	return nil
}
