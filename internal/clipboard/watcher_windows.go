//go:build windows

package clipboard

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
	user32DLL   = syscall.NewLazyDLL("user32.dll")

	procGetClipboardOwner          = user32DLL.NewProc("GetClipboardOwner")
	procGetClipboardData           = user32DLL.NewProc("GetClipboardData")
	procGlobalSize                 = kernel32DLL.NewProc("GlobalSize")
	procIsClipboardFormatAvailable = user32DLL.NewProc("IsClipboardFormatAvailable")
	procSetClipboardData           = user32DLL.NewProc("SetClipboardData")
	procEmptyClipboard             = user32DLL.NewProc("EmptyClipboard")
	procGlobalAlloc                = kernel32DLL.NewProc("GlobalAlloc")
	procGlobalFree                 = kernel32DLL.NewProc("GlobalFree")
	procGlobalLock                 = kernel32DLL.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32DLL.NewProc("GlobalUnlock")
	procOpenClipboard              = user32DLL.NewProc("OpenClipboard")
	procCloseClipboard             = user32DLL.NewProc("CloseClipboard")
)

const (
	cfText = 1
	cfDIB  = 8
	cfPNG  = 49320
	GMEM_MOVEABLE = 0x0002
)

func hashString(s string) string {
	h := [32]byte{}
	for i, c := range s {
		h[i%32] ^= byte(c)
	}
	return fmt.Sprintf("%x", h[:])
}

func hashBytes(data []byte) string {
	h := [32]byte{}
	for i, b := range data {
		h[i%32] ^= b
	}
	return fmt.Sprintf("%x", h[:])
}

// Start launches the poll loop for Windows.
func (w *Watcher) Start(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	w.cancel = cancel
	go w.loop(ctx)
	return nil
}

func (w *Watcher) loop(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()

	var lastOwner uintptr
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Poll clipboard owner to detect changes
			if owner, _, _ := procGetClipboardOwner.Call(); owner != lastOwner || owner == 0 {
				lastOwner = owner
				w.tick(ctx)
			}
		}
	}
}

func (w *Watcher) tick(ctx context.Context) {
	// Try to read text first (most common)
	text, err := readTextWindows(ctx)
	if err == nil && text != "" {
		h := hashString(text)
		if w.swapHash(h) {
			return
		}
		w.publish(Event{ContentType: ContentTypeText, Text: text, Hash: h})
		return
	}

	// Fall back to image
	if img, err := readImageWindows(ctx); err == nil && len(img) > 0 {
		h := hashBytes(img)
		if w.swapHash(h) {
			return
		}
		w.publish(Event{ContentType: ContentTypeImage, ImagePNG: img, Hash: h})
		return
	}
}

func readTextWindows(ctx context.Context) (string, error) {
	if !openClipboard() {
		return "", fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procIsClipboardFormatAvailable.Call(uintptr(cfText)); ok == 0 {
		return "", nil
	}

	handle, _, _ := procGetClipboardData.Call(uintptr(cfText))
	if handle == 0 {
		return "", fmt.Errorf("failed to get clipboard data")
	}

	lockedPtr, _, _ := procGlobalLock.Call(handle)
	if lockedPtr == 0 {
		return "", fmt.Errorf("failed to lock global handle")
	}
	defer procGlobalUnlock.Call(handle)

	// Convert Windows string to Go string (null-terminated)
	cstr := (*[1 << 20]byte)(unsafe.Pointer(lockedPtr))[:]
	var length int
	for i := 0; i < len(cstr); i++ {
		if cstr[i] == 0 {
			length = i
			break
		}
	}

	return string(cstr[:length]), nil
}

func readImageWindows(ctx context.Context) ([]byte, error) {
	if !openClipboard() {
		return nil, fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	// Try DIB (Device-Independent Bitmap) format - most common for images
	if isFormatAvailable(cfDIB) {
		// For now, we don't support DIB conversion
		// TODO: Implement proper DIB to PNG conversion
		return nil, fmt.Errorf("image format not yet supported on Windows")
	}

	return nil, fmt.Errorf("no image format available")
}

func openClipboard() bool {
	r, _, _ := procOpenClipboard.Call(0)
	return r != 0
}

func closeClipboard() bool {
	r, _, _ := procCloseClipboard.Call()
	return r != 0
}

func isFormatAvailable(format uint32) bool {
	r, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	return r != 0
}
