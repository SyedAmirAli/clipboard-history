package x11hint

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Popup positioning: place the popup near the mouse pointer, fully visible on
// the monitor the pointer is on. The pointer is the best available proxy for
// "the focused input field" — a global caret position simply isn't obtainable
// across apps (least of all on Wayland), but the user's pointer is almost
// always at or near the field they just clicked into. This also solves
// multi-monitor: the popup opens on whichever display the pointer is on.
//
// All coordinates live in the X11 coordinate space (xdotool, xrandr and the
// GTK-X11 window all share it), so this only applies when clipd's own window
// runs under X11/XWayland — on a native-Wayland window the compositor owns
// placement and we fall back to centering.

type rect struct{ x, y, w, h int }

func (r rect) contains(px, py int) bool {
	return px >= r.x && px < r.x+r.w && py >= r.y && py < r.y+r.h
}

// CanPosition reports whether clipd's window position can be controlled at
// all: the window must be an X11 window (native X session, or forced
// GDK_BACKEND=x11 under XWayland) and xdotool must exist to query the pointer.
func CanPosition() bool {
	if !underX11() {
		return false
	}
	if os.Getenv("XDG_SESSION_TYPE") != "x11" && os.Getenv("GDK_BACKEND") != "x11" {
		return false // window is Wayland-native; gtk_window_move is a no-op
	}
	_, err := exec.LookPath("xdotool")
	return err == nil
}

// PopupPosition computes a top-left position for a w×h popup near the mouse
// pointer, guaranteed to fit entirely on the pointer's monitor. Returns
// ok=false when the pointer or monitor layout can't be determined.
func PopupPosition(w, h int) (int, int, bool) {
	px, py, ok := pointerPos()
	if !ok {
		return 0, 0, false
	}
	mon, ok := monitorAt(px, py)
	if !ok {
		return 0, 0, false
	}
	x, y := placeInMonitor(px, py, w, h, mon)
	return x, y, true
}

// placeInMonitor is the pure placement logic: prefer just below-right of the
// pointer (like a context menu); flip above when there's no room below, slide
// left when there's no room on the right, and always clamp fully inside mon.
func placeInMonitor(px, py, w, h int, mon rect) (int, int) {
	const margin = 12  // gap from monitor edges
	const offsetX = 14 // preferred offset right of the pointer
	const offsetY = 18 // preferred offset below the pointer

	x := px + offsetX
	y := py + offsetY

	// Not enough room below → flip fully above the pointer.
	if y+h > mon.y+mon.h-margin {
		y = py - h - offsetY
	}
	// Not enough room on the right → align the popup's right edge leftwards.
	if x+w > mon.x+mon.w-margin {
		x = mon.x + mon.w - w - margin
	}
	// Final clamp: never off the monitor's top/left (also covers popups
	// larger than the space above after a flip).
	if x < mon.x+margin {
		x = mon.x + margin
	}
	if y < mon.y+margin {
		y = mon.y + margin
	}
	// And re-clamp the bottom in case the flip pushed us past the top and
	// the clamp above moved us back down over the bottom edge.
	if y+h > mon.y+mon.h-margin {
		y = mon.y + mon.h - h - margin
		if y < mon.y+margin {
			y = mon.y + margin
		}
	}
	return x, y
}

// pointerPos returns the global pointer position via xdotool.
func pointerPos() (int, int, bool) {
	out, err := exec.Command("xdotool", "getmouselocation", "--shell").Output()
	if err != nil {
		return 0, 0, false
	}
	x, okX := shellVar(string(out), "X")
	y, okY := shellVar(string(out), "Y")
	return x, y, okX && okY
}

func shellVar(s, name string) (int, bool) {
	for _, line := range strings.Split(s, "\n") {
		if v, found := strings.CutPrefix(line, name+"="); found {
			n, err := strconv.Atoi(strings.TrimSpace(v))
			return n, err == nil
		}
	}
	return 0, false
}

// monitorGeomRe matches the geometry field of `xrandr --listactivemonitors`
// output lines, e.g. " 0: +*eDP-1 1920/309x1080/174+0+0  eDP-1".
var monitorGeomRe = regexp.MustCompile(`(\d+)/\d+x(\d+)/\d+\+(\d+)\+(\d+)`)

// monitorAt returns the geometry of the monitor containing the point. When
// xrandr is unavailable or no monitor matches, it falls back to the whole
// X display so the popup is at least clamped somewhere sane.
func monitorAt(px, py int) (rect, bool) {
	for _, m := range activeMonitors() {
		if m.contains(px, py) {
			return m, true
		}
	}
	return displayGeometry()
}

func activeMonitors() []rect {
	out, err := exec.Command("xrandr", "--listactivemonitors").Output()
	if err != nil {
		return nil
	}
	var monitors []rect
	for _, line := range strings.Split(string(out), "\n") {
		m := monitorGeomRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		x, _ := strconv.Atoi(m[3])
		y, _ := strconv.Atoi(m[4])
		if w > 0 && h > 0 {
			monitors = append(monitors, rect{x: x, y: y, w: w, h: h})
		}
	}
	return monitors
}

func displayGeometry() (rect, bool) {
	out, err := exec.Command("xdotool", "getdisplaygeometry").Output()
	if err != nil {
		return rect{}, false
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return rect{}, false
	}
	w, err1 := strconv.Atoi(fields[0])
	h, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return rect{}, false
	}
	return rect{x: 0, y: 0, w: w, h: h}, true
}
