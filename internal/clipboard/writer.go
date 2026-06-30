package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// WriteText puts the given UTF-8 string onto the system clipboard via wl-copy.
//
// The tool stays alive in the background after it acquires the selection;
// we Start it in a new session group so it survives clipd's lifecycle and
// the calling goroutine returns immediately.
func WriteText(s string) error {
	return spawnWith("wl-copy", nil, []byte(s))
}

// WriteImagePNG puts a PNG image onto the system clipboard so it can be
// pasted into image-aware applications (browsers, image editors, office
// suites, etc.).
func WriteImagePNG(png []byte) error {
	return spawnWith("wl-copy", []string{"--type", "image/png"}, png)
}

// spawnWith starts the wl-copy clipboard-owner process, feeds it the payload
// on stdin, and detaches it into a new session so it survives clipd's
// lifecycle. We don't Wait on it synchronously — it lingers as the selection
// owner until another process takes over; the OS reaps it then.
func spawnWith(name string, args []string, payload []byte) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found in PATH: install it with %s", name, installHint())
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
