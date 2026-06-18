//go:build windows

package clipboard

import (
	"syscall"
	"unsafe"
)

// The user32 DLL handle (`user32`) is declared in paster_windows.go.
var (
	procFindWindowW    = user32.NewProc("FindWindowW")
	procGetWindowLongW = user32.NewProc("GetWindowLongW")
	procSetWindowLongW = user32.NewProc("SetWindowLongW")
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
