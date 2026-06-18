//go:build windows

package clipboard

import (
	"syscall"
	"unsafe"
)

// All user32 DLL handle (`user32`) is declared in paster_windows.go.
var (
	procFindWindowW    = user32.NewProc("FindWindowW")
	procGetWindowLongW = user32.NewProc("GetWindowLongW")
	procSetWindowLongW = user32.NewProc("SetWindowLongW")
)

const (
	wsExToolWindow = 0x00000080 // hides a window from the taskbar / Alt-Tab
	wsExAppWindow  = 0x00040000 // forces a taskbar button for a top-level window
)

// gwlExStyle is GWL_EXSTYLE (-20). It's a var (not const) so the negative value
// can be converted to uintptr at runtime for the syscall (a constant conversion
// would overflow uintptr at compile time).
var gwlExStyle = -20

// EnsureTaskbarButton forces the top-level window with the given title to own a
// normal taskbar button: it sets WS_EX_APPWINDOW and clears WS_EX_TOOLWINDOW.
// Without this, Wails' frameless window minimises to the legacy minimised-window
// placeholder in the bottom-left of the desktop (a stray icon) instead of the
// taskbar. Call it while the window is hidden (just before showing) so the
// taskbar registers the button when the window is next shown.
func EnsureTaskbarButton(title string) {
	hwnd := findWindowByTitle(title)
	if hwnd == 0 {
		return
	}
	ex, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlExStyle))
	newEx := (ex &^ uintptr(wsExToolWindow)) | uintptr(wsExAppWindow)
	if newEx != ex {
		procSetWindowLongW.Call(hwnd, uintptr(gwlExStyle), newEx)
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
