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
	"strconv"
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

// findToplevel returns clipd's managed toplevel X window — the one carrying
// WM_STATE (the embedded WebKit child has none).
func findToplevel() string {
	for _, id := range findWindows() {
		out, err := exec.Command("xprop", "-id", id, "WM_STATE").Output()
		if err == nil && strings.Contains(string(out), "WM_STATE(WM_STATE)") {
			return id
		}
	}
	return ""
}

// WindowPosition reads the toplevel's absolute position in raw X11
// coordinates. We deliberately use xdotool instead of GTK here: with mutter's
// xwayland-native-scaling / scale-monitor-framebuffer features, GTK's idea of
// the coordinate space diverges from X11's, but xdotool read + move round-trip
// in the same space exactly.
func WindowPosition() (int, int, bool) {
	if !underX11() {
		return 0, 0, false
	}
	id := findToplevel()
	if id == "" {
		return 0, 0, false
	}
	out, err := exec.Command("xdotool", "getwindowgeometry", "--shell", id).Output()
	if err != nil {
		return 0, 0, false
	}
	var x, y int
	var okX, okY bool
	for _, line := range strings.Split(string(out), "\n") {
		if v, found := strings.CutPrefix(line, "X="); found {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				x, okX = n, true
			}
		}
		if v, found := strings.CutPrefix(line, "Y="); found {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				y, okY = n, true
			}
		}
	}
	return x, y, okX && okY
}

// MoveWindow places the toplevel at x,y in raw X11 coordinates (the same
// space WindowPosition reads), so a saved position restores exactly.
func MoveWindow(x, y int) bool {
	if !underX11() {
		return false
	}
	id := findToplevel()
	if id == "" {
		return false
	}
	return exec.Command("xdotool", "windowmove", id,
		strconv.Itoa(x), strconv.Itoa(y)).Run() == nil
}

// Nudge re-marks the window (covering any later hide/show re-map) and activates
// it so the popup gets keyboard focus immediately. Called shortly after each
// show. The skip-taskbar mark from SuppressTaskbar already prevents the toast;
// this just keeps focus behaviour right. No-op off X11.
//
// markSkip controls whether the skip-taskbar/utility hints are (re)applied:
// popup mode wants them, but windowed mode must NOT get them — a utility
// skip-taskbar window can't be minimised on GNOME, which is what broke the
// yellow titlebar button.
func Nudge(markSkip bool) {
	if !underX11() {
		return
	}
	ids := findWindows()
	if markSkip {
		for _, id := range ids {
			mark(id)
		}
	}
	if n := len(ids); n > 0 {
		_ = exec.Command("xdotool", "windowactivate", ids[n-1]).Run()
	}
}
