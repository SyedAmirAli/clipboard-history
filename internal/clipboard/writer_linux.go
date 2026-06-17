//go:build linux

package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// WriteText puts the given UTF-8 string onto the system clipboard using xclip or wl-copy.
func WriteText(s string) error {
	if IsWayland() {
		return spawnWith("wl-copy", nil, []byte(s))
	}
	return spawnWith("xclip", []string{"-selection", "clipboard"}, []byte(s))
}

// WriteImagePNG puts a PNG image onto the system clipboard.
func WriteImagePNG(png []byte) error {
	if IsWayland() {
		return spawnWith("wl-copy", []string{"--type", "image/png"}, png)
	}
	return spawnWith("xclip", []string{"-selection", "clipboard", "-t", "image/png"}, png)
}

// spawnWith starts a clipboard-owner process (xclip or wl-copy), feeds it
// the payload on stdin, and detaches it into a new session.
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
		_ = cmd.Wait() // best-effort reap
	}()
	return nil
}

// installHint returns the package name to install clipboard tools.
func installHint(s Server) string {
	if s == ServerWayland {
		return "'sudo apt install wl-clipboard'"
	}
	return "'sudo apt install xclip'"
}
