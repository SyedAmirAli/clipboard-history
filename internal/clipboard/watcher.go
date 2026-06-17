package clipboard

import (
	"context"
	"sync"
	"time"
)

// Watcher polls the system clipboard and emits events whenever a new,
// distinct payload (text or image/png) appears. Platform-specific implementations
// (watcher_linux.go, watcher_windows.go) handle the actual clipboard access.
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

// NewWatcher returns a Watcher with the given poll interval.
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

// Start launches the poll loop. Platform-specific implementation required.
func (w *Watcher) Start(ctx context.Context) error {
	// Implemented in watcher_linux.go or watcher_windows.go
	panic("Start must be implemented by platform-specific code")
}

// Stop signals the watcher to exit and waits briefly for cleanup.
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// Suppress sets the watcher's last-seen hash so that the next change
// matching this hash is treated as a no-op.
func (w *Watcher) Suppress(hash string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastHash = hash
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
