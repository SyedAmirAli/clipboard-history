// Package x11hint nudges clipd's own X11 (XWayland) window so GNOME stops
// flashing the dock/taskbar and popping "<app> is ready" toasts, and so clipd
// keeps no taskbar entry at all.
//
// Root cause: clipd's window is summoned programmatically and GNOME's
// focus-stealing prevention briefly flags it "demands attention" on map. That
// flag is what GNOME Shell's windowAttentionHandler turns into the "<app> is
// ready" toast and the dock-icon pulse. Crucially, that handler *ignores
// skip-taskbar windows* — so if the window is marked skip-taskbar BEFORE it
// first maps, the toast never fires, there's no dock/taskbar entry to blink,
// and the attention flag is moot. Verified on GNOME/XWayland: a skip-taskbar
// state set while the window is still hidden survives the first map.
//
// This only acts under X11/XWayland (DISPLAY set); on a native-Wayland session
// it is a no-op. We shell out to xdotool/xprop because Wails exposes no handle
// to its GtkWindow. Those tools are listed as recommended package deps.
package x11hint

import (
	"os"
	"os/exec"
	"strings"
	"time"
)

func underX11() bool { return os.Getenv("DISPLAY") != "" }

// findWindows returns every X11 window clipd owns. `search --class clipd`
// matches both the managed toplevel and the embedded WebKit child, and which
// one the compositor treats as the taskbar window isn't guaranteed, so we mark
// them all.
func findWindows() []string {
	out, err := exec.Command("xdotool", "search", "--class", "clipd").Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// mark sets the skip-taskbar/skip-pager utility hints on one window.
func mark(id string) {
	_ = exec.Command("xprop", "-id", id, "-f", "_NET_WM_WINDOW_TYPE", "32a",
		"-set", "_NET_WM_WINDOW_TYPE", "_NET_WM_WINDOW_TYPE_UTILITY").Run()
	_ = exec.Command("xprop", "-id", id, "-f", "_NET_WM_STATE", "32a",
		"-set", "_NET_WM_STATE", "_NET_WM_STATE_SKIP_TASKBAR,_NET_WM_STATE_SKIP_PAGER").Run()
}

// markAll marks every clipd window; returns true once at least one was found.
func markAll() bool {
	ids := findWindows()
	for _, id := range ids {
		mark(id)
	}
	return len(ids) > 0
}

// SuppressTaskbar marks clipd's window skip-taskbar as early as possible —
// while it is still hidden at startup, BEFORE its first map — so GNOME's
// windowAttentionHandler ignores it (no "is ready" toast) and it never gets a
// taskbar/dock entry to blink. It runs in the background: it retries until the
// window exists (created hidden, usually within ~0.5s), then re-applies a few
// times to cover the first map. No-op off X11.
func SuppressTaskbar() {
	if !underX11() {
		return
	}
	go func() {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if markAll() {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
		// Re-apply briefly in case the first map (or a hint reset) clears it.
		for i := 0; i < 6; i++ {
			time.Sleep(400 * time.Millisecond)
			markAll()
		}
	}()
}

// Nudge re-marks the window (covering any later hide/show re-map) and activates
// it so the popup gets keyboard focus immediately. Called shortly after each
// show. The skip-taskbar mark from SuppressTaskbar already prevents the toast;
// this just keeps focus behaviour right. No-op off X11.
func Nudge() {
	if !underX11() {
		return
	}
	ids := findWindows()
	for _, id := range ids {
		mark(id)
	}
	if n := len(ids); n > 0 {
		_ = exec.Command("xdotool", "windowactivate", ids[n-1]).Run()
	}
}
