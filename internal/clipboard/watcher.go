package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Watcher polls the system clipboard and emits events whenever a new,
// distinct payload (text or image/png) appears. It drives the clipboard
// through external CLI tools — `xclip` on X11, `wl-paste` on Wayland —
// selected at construction time by DetectServer.
//
// We deliberately avoid linking against libx11 / libxfixes via cgo so
// the produced binary stays pure-Go and small. The tools are declared as
// runtime dependencies in the .deb package.
type Watcher struct {
	interval time.Duration
	server   Server

	mu       sync.Mutex
	lastHash string
	events   chan Event
	cancel   context.CancelFunc
}

// Event is published on each detected clipboard change.
type Event struct {
	ContentType ContentType
	Text        string
	ImagePNG    []byte
	Hash        string
}

// NewWatcher returns a Watcher with the given poll interval. A typical
// good value is 400-700 ms — fast enough to feel instant, slow enough
// to be invisible on CPU graphs.
func NewWatcher(interval time.Duration) *Watcher {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	return &Watcher{
		interval: interval,
		server:   DetectServer(),
		events:   make(chan Event, 16),
	}
}

// Events returns the read-only channel of clipboard change events.
func (w *Watcher) Events() <-chan Event { return w.events }

// Start launches the poll loop. It returns immediately. Call Stop to
// terminate it. Start is safe to call once per Watcher.
func (w *Watcher) Start(ctx context.Context) error {
	if missing := MissingTools(w.server); len(missing) > 0 {
		return fmt.Errorf("missing %s clipboard tools: %s (install %s)",
			w.server, strings.Join(missing, ", "), installHint(w.server))
	}
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.loop(ctx)
	return nil
}

// Stop signals the watcher to exit and waits briefly for cleanup.
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// Suppress sets the watcher's last-seen hash so that the next change
// matching this hash is treated as a no-op. The caller uses this right
// before programmatically writing to the clipboard (e.g. when pasting
// a stored history item) to avoid re-saving its own write.
func (w *Watcher) Suppress(hash string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastHash = hash
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

func (w *Watcher) swapHash(h string) (same bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastHash == h {
		return true
	}
	w.lastHash = h
	return false
}

func (w *Watcher) publish(e Event) {
	select {
	case w.events <- e:
	default: // drop if consumer is slow; we'll catch the next change
	}
}

// hashString returns a stable hex SHA-256 hash, prefixed with a content-type tag
// so identical bytes never collide between text and image entries.
func hashString(s string) string { return "t:" + sumBytes([]byte(s)) }
func hashBytes(b []byte) string  { return "i:" + sumBytes(b) }

func sumBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
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
		// No --type: wl-paste auto-negotiates the best text flavour and
		// handles charset conversion. --no-newline avoids the trailing \n
		// it would otherwise append, matching xclip's raw output.
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

// installHint returns the apt package(s) that provide the clipboard tools
// for a given server, used in user-facing "not installed" messages.
func installHint(s Server) string {
	if s == ServerWayland {
		return "'sudo apt install wl-clipboard'"
	}
	return "'sudo apt install xclip'"
}

// runCmd executes a short-lived command, capturing stdout. It is tolerant
// of xclip's typical non-zero exits when the clipboard is empty/unsupported.
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
