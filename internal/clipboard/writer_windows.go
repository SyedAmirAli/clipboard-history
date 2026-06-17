//go:build windows

package clipboard

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
	user32DLL   = syscall.NewLazyDLL("user32.dll")

	procSetClipboardData  = user32DLL.NewProc("SetClipboardData")
	procEmptyClipboard    = user32DLL.NewProc("EmptyClipboard")
	procGlobalAlloc       = kernel32DLL.NewProc("GlobalAlloc")
	procGlobalFree        = kernel32DLL.NewProc("GlobalFree")
	procGlobalLock        = kernel32DLL.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32DLL.NewProc("GlobalUnlock")
	procOpenClipboard     = user32DLL.NewProc("OpenClipboard")
	procCloseClipboard    = user32DLL.NewProc("CloseClipboard")
)

const (
	GMEM_MOVEABLE = 0x0002
	cfText        = 1
	cfPNG         = 49320
)

// WriteText puts the given UTF-8 string onto the system clipboard using Win32 API.
func WriteText(s string) error {
	if !openClipboard() {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procEmptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("failed to empty clipboard")
	}

	// Convert Go string to null-terminated UTF-8 bytes
	data := append([]byte(s), 0)

	// Allocate global memory
	hMem, _, _ := procGlobalAlloc.Call(uintptr(GMEM_MOVEABLE), uintptr(len(data)))
	if hMem == 0 {
		return fmt.Errorf("failed to allocate global memory")
	}

	// Lock and copy data
	lpMem, _, _ := procGlobalLock.Call(hMem)
	if lpMem == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to lock global memory")
	}

	copy((*[1 << 30]byte)(unsafe.Pointer(lpMem))[:len(data)], data)
	procGlobalUnlock.Call(hMem)

	// Set clipboard data
	if result, _, _ := procSetClipboardData.Call(uintptr(cfText), hMem); result == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data")
	}

	// Note: After SetClipboardData succeeds, Windows owns the memory, don't free it
	return nil
}

func openClipboard() bool {
	r, _, _ := procOpenClipboard.Call(0)
	return r != 0
}

func closeClipboard() bool {
	r, _, _ := procCloseClipboard.Call()
	return r != 0
}

// WriteImagePNG puts a PNG image onto the system clipboard.
func WriteImagePNG(png []byte) error {
	if !openClipboard() {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procEmptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("failed to empty clipboard")
	}

	// For now, just put it as a PNG format if available
	// Full DIB/BITMAP support would be more complex
	if cfPNG == 0 {
		return fmt.Errorf("PNG clipboard format not available")
	}

	// Allocate global memory
	hMem, _, _ := procGlobalAlloc.Call(uintptr(GMEM_MOVEABLE), uintptr(len(png)))
	if hMem == 0 {
		return fmt.Errorf("failed to allocate global memory")
	}

	// Lock and copy data
	lpMem, _, _ := procGlobalLock.Call(hMem)
	if lpMem == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to lock global memory")
	}

	copy((*[1 << 30]byte)(unsafe.Pointer(lpMem))[:len(png)], png)
	procGlobalUnlock.Call(hMem)

	// Set clipboard data
	if result, _, _ := procSetClipboardData.Call(uintptr(cfPNG), hMem); result == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data")
	}

	return nil
}
