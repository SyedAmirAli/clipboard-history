//go:build windows

package clipboard

import (
	"syscall"
	"unsafe"
)

// The user32 DLL handle (`user32`) is declared in paster_windows.go.
var (
	procFindWindowW      = user32.NewProc("FindWindowW")
	procGetWindowLongW   = user32.NewProc("GetWindowLongW")
	procSetWindowLongW   = user32.NewProc("SetWindowLongW")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW  = user32.NewProc("GetMonitorInfoW")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

const (
	monitorDefaultToNearest = 0x00000002
	hwndTop                 = 0
	swpNoSize               = 0x0001
	swpShowWindow           = 0x0040
)

const (
	wsExToolWindow = 0x00000080 // hides a window from the taskbar / Alt-Tab
	wsExAppWindow  = 0x00040000 // forces a taskbar button for a top-level window
	wsSysMenu      = 0x00080000 // window has a system menu (enables taskbar right-click menu)
	wsMinimizeBox  = 0x00020000 // enables "Minimize" in that menu
)

// gwlStyle (GWL_STYLE) and gwlExStyle (GWL_EXSTYLE) are negative window-long
// indices. They're vars (not consts) so the negative values can be converted to
// uintptr at runtime — a constant conversion would overflow uintptr at compile
// time.
var (
	gwlStyle   = -16
	gwlExStyle = -20
)

// SetTaskbarVisible toggles whether the top-level window with the given title
// owns a normal taskbar button. When visible it sets WS_EX_APPWINDOW (and a
// system menu so the taskbar right-click works) and clears WS_EX_TOOLWINDOW;
// when hidden it does the reverse, turning the window into a tool-window popup
// with no taskbar entry. Call it while the window is hidden (just before
// showing) so the taskbar registers the change on show.
func SetTaskbarVisible(title string, visible bool) {
	hwnd := findWindowByTitle(title)
	if hwnd == 0 {
		return
	}

	ex, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlExStyle))
	var newEx uintptr
	if visible {
		newEx = (ex &^ uintptr(wsExToolWindow)) | uintptr(wsExAppWindow)
	} else {
		newEx = (ex &^ uintptr(wsExAppWindow)) | uintptr(wsExToolWindow)
	}
	if newEx != ex {
		procSetWindowLongW.Call(hwnd, uintptr(gwlExStyle), newEx)
	}

	if visible {
		// Give the frameless window a system menu so right-clicking its taskbar
		// button shows the standard window menu (Restore / Minimize / Close).
		style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyle))
		newStyle := style | uintptr(wsSysMenu) | uintptr(wsMinimizeBox)
		if newStyle != style {
			procSetWindowLongW.Call(hwnd, uintptr(gwlStyle), newStyle)
		}
	}
}

func findWindowByTitle(title string) uintptr {
	p, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(p)))
	return hwnd
}

// ShowAtCursor moves the window with the given title to the monitor the cursor
// is on — placed at the cursor, clamped so it stays fully within that monitor's
// work area — and raises it to the foreground. Used when summoning the popup so
// it appears where the user is looking, which matters on multi-monitor setups.
func ShowAtCursor(title string) {
	hwnd := findWindowByTitle(title)
	if hwnd == 0 {
		return
	}

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	var rc rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top

	work := cursorMonitorWork(pt)
	x, y := pt.X, pt.Y
	if x+w > work.Right {
		x = work.Right - w
	}
	if y+h > work.Bottom {
		y = work.Bottom - h
	}
	if x < work.Left {
		x = work.Left
	}
	if y < work.Top {
		y = work.Top
	}

	procSetWindowPos.Call(hwnd, hwndTop, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpShowWindow)
	procSetForegroundWindow.Call(hwnd)
}

// cursorMonitorWork returns the work area (screen minus taskbar) of the monitor
// containing pt. On failure it returns a very large rect so position clamping
// becomes a no-op.
func cursorMonitorWork(pt point) rect {
	packed := uintptr(uint32(pt.X)) | (uintptr(uint32(pt.Y)) << 32)
	hmon, _, _ := procMonitorFromPoint.Call(packed, monitorDefaultToNearest)
	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if ok, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return rect{Left: -1 << 15, Top: -1 << 15, Right: 1 << 15, Bottom: 1 << 15}
	}
	return mi.RcWork
}

// IsForegroundWindow reports whether the window with the given title is the
// current foreground (active) window.
func IsForegroundWindow(title string) bool {
	hwnd := findWindowByTitle(title)
	if hwnd == 0 {
		return false
	}
	fg, _, _ := procGetForegroundWindow.Call()
	return fg == hwnd
}
