//go:build windows

package clipboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"runtime"
	"unsafe"
)

// All Win32 DLL and proc declarations are in watcher_windows.go

// WriteText puts the given UTF-8 string onto the system clipboard using Win32 API.
func WriteText(s string) error {
	// Convert Go string to null-terminated bytes before touching the clipboard.
	data := append([]byte(s), 0)

	// The clipboard is thread-affine: open and close must happen on the same OS
	// thread, so pin the goroutine for the whole open/set/close sequence.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procEmptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("failed to empty clipboard")
	}

	hMem, err := allocGlobal(data)
	if err != nil {
		return err
	}

	if result, _, _ := procSetClipboardData.Call(uintptr(cfText), hMem); result == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data")
	}

	// After SetClipboardData succeeds, Windows owns the memory; don't free it.
	return nil
}

// WriteImagePNG puts a PNG image onto the system clipboard as CF_DIB, the format
// understood by virtually every Windows app (Paint, Word, browsers, etc).
func WriteImagePNG(pngBytes []byte) error {
	// Decode and build the DIB *before* opening the clipboard so the clipboard is
	// held for the shortest possible time (heavy work outside the lock).
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	dibData, err := imageToDIB(img)
	if err != nil {
		return fmt.Errorf("failed to convert image to DIB: %w", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procEmptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("failed to empty clipboard")
	}

	hMem, err := allocGlobal(dibData)
	if err != nil {
		return err
	}

	if result, _, _ := procSetClipboardData.Call(uintptr(cfDIB), hMem); result == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data as DIB")
	}

	return nil
}

// allocGlobal copies data into a GMEM_MOVEABLE global block suitable for handing
// to SetClipboardData. The caller owns the returned handle until the set
// succeeds (after which Windows owns it).
func allocGlobal(data []byte) (uintptr, error) {
	hMem, _, _ := procGlobalAlloc.Call(uintptr(GMEM_MOVEABLE), uintptr(len(data)))
	if hMem == 0 {
		return 0, fmt.Errorf("failed to allocate global memory")
	}

	lpMem, _, _ := procGlobalLock.Call(hMem)
	if lpMem == 0 {
		procGlobalFree.Call(hMem)
		return 0, fmt.Errorf("failed to lock global memory")
	}

	dst := (*[1 << 30]byte)(unsafe.Pointer(lpMem))[:len(data):len(data)]
	copy(dst, data)
	procGlobalUnlock.Call(hMem)
	return hMem, nil
}

// imageToDIB encodes an image as a packed 32-bit bottom-up DIB
// (BITMAPINFOHEADER + BGRA pixels). Alpha is forced opaque so pasted images
// never come out transparent in apps that honour the alpha channel.
func imageToDIB(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var buf bytes.Buffer

	hdr := struct {
		Size        uint32
		Width       int32
		Height      int32
		Planes      uint16
		BitCount    uint16
		Compression uint32
		ImageSize   uint32
		XPelsPerM   int32
		YPelsPerM   int32
		ClrUsed     uint32
		ClrImport   uint32
	}{
		Size:      40,
		Width:     int32(width),
		Height:    int32(height),
		Planes:    1,
		BitCount:  32,
		XPelsPerM: 2835,
		YPelsPerM: 2835,
	}

	if err := binary.Write(&buf, binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}

	// 32-bit rows are inherently 4-byte aligned, so no extra padding is needed.
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			r32, g32, b32, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			buf.WriteByte(uint8(b32 >> 8))
			buf.WriteByte(uint8(g32 >> 8))
			buf.WriteByte(uint8(r32 >> 8))
			buf.WriteByte(255)
		}
	}

	return buf.Bytes(), nil
}
