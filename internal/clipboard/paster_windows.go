//go:build windows

package clipboard

import (
	"syscall"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procKeyBdEvent = user32.NewProc("keybd_event")
)

// SendPaste synthesises a Ctrl+V key event into the current focused window
// using Windows Win32 API. On Windows, this works for all applications unlike
// the X11/Wayland limitations.
func SendPaste() error {
	const (
		VK_CONTROL = 0x11
		VK_V       = 0x56
		KEYEVENTF_KEYUP = 0x2
	)

	// Press Ctrl down
	keyBdEvent(VK_CONTROL, 0, 0, 0)
	// Press V
	keyBdEvent(VK_V, 0, 0, 0)
	// Release V
	keyBdEvent(VK_V, 0, KEYEVENTF_KEYUP, 0)
	// Release Ctrl
	keyBdEvent(VK_CONTROL, 0, KEYEVENTF_KEYUP, 0)

	return nil
}

func keyBdEvent(bVk, bScan, dwFlags, dwExtraInfo uint32) {
	procKeyBdEvent.Call(
		uintptr(bVk),
		uintptr(bScan),
		uintptr(dwFlags),
		uintptr(dwExtraInfo),
	)
}
