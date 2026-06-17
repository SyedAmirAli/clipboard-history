//go:build windows

package clipboard

import (
	"sync"
	"syscall"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procKeyBdEvent            = user32.NewProc("keybd_event")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
)

var (
	lastFocusedWindowMu sync.Mutex
	lastFocusedWindow   uintptr
)

// SaveFocusedWindow stores the currently focused window so it can be restored
// before pasting. Call this before showing the clipboard popup.
func SaveFocusedWindow() {
	lastFocusedWindowMu.Lock()
	defer lastFocusedWindowMu.Unlock()

	hWnd, _, _ := procGetForegroundWindow.Call()
	if hWnd != 0 {
		lastFocusedWindow = hWnd
	}
}

// SendPaste synthesises a Ctrl+V key event after restoring the previously
// focused window. This ensures the paste goes to the user's original
// application, not the clipboard history popup.
func SendPaste() error {
	const (
		VK_CONTROL = 0x11
		VK_V       = 0x56
		KEYEVENTF_KEYUP = 0x2
	)

	// Restore focus to the previously focused window
	lastFocusedWindowMu.Lock()
	hWnd := lastFocusedWindow
	lastFocusedWindowMu.Unlock()

	if hWnd != 0 {
		procSetForegroundWindow.Call(hWnd)
	}

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
