//go:build linux

package clipboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Start launches the poll loop for Linux (X11/Wayland).
func (w *Watcher) Start(ctx context.Context) error {
	if missing := MissingTools(w.server); len(missing) > 0 {
		return fmt.Errorf("missing %s clipboard tools: %s (install %s)",
			w.server, strings.Join(missing, ", "), installHint(w.server))
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	w.cancel = cancel
	go w.loop(ctx)
	return nil
}

func (w *Watcher) loop(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.tick(ctx)
		}
	}
}

func (w *Watcher) tick(ctx context.Context) {
	targets, _ := readTargets(ctx, w.server)
	if hasTarget(targets, "image/png") {
		img, err := readImage(ctx, w.server)
		if err == nil && len(img) > 0 {
			h := hashBytes(img)
			if w.swapHash(h) {
				return
			}
			w.publish(Event{ContentType: ContentTypeImage, ImagePNG: img, Hash: h})
			return
		}
	}
	text, err := readText(ctx, w.server)
	if err != nil || text == "" {
		return
	}
	h := hashString(text)
	if w.swapHash(h) {
		return
	}
	w.publish(Event{ContentType: ContentTypeText, Text: text, Hash: h})
}

// hasTarget reports whether the X11 TARGETS list contains the given mime.
func hasTarget(targets []string, mime string) bool {
	for _, t := range targets {
		if strings.EqualFold(strings.TrimSpace(t), mime) {
			return true
		}
	}
	return false
}

func readTargets(ctx context.Context, srv Server) ([]string, error) {
	var out []byte
	var err error
	if srv == ServerWayland {
		out, err = runCmd(ctx, "wl-paste", "--list-types")
	} else {
		out, err = runCmd(ctx, "xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func readText(ctx context.Context, srv Server) (string, error) {
	var out []byte
	var err error
	if srv == ServerWayland {
		out, err = runCmd(ctx, "wl-paste", "--no-newline")
	} else {
		out, err = runCmd(ctx, "xclip", "-selection", "clipboard", "-o")
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readImage(ctx context.Context, srv Server) ([]byte, error) {
	if srv == ServerWayland {
		return runCmd(ctx, "wl-paste", "--no-newline", "--type", "image/png")
	}
	return runCmd(ctx, "xclip", "-selection", "clipboard", "-t", "image/png", "-o")
}

// installHint returns the apt package(s) that provide the clipboard tools.
func installHint(s Server) string {
	if s == ServerWayland {
		return "'sudo apt install wl-clipboard'"
	}
	return "'sudo apt install xclip'"
}

// runCmd executes a short-lived command, capturing stdout.
func runCmd(ctx context.Context, name string, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// xclip returns non-zero for "no selection of that type" — treat as empty.
			return nil, nil
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}
