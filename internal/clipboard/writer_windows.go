//go:build windows

package clipboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"unsafe"
)

// All Win32 DLL and proc declarations are in watcher_windows.go

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

// WriteImagePNG puts a PNG image onto the system clipboard.
// Writes PNG directly as CF_DIB since it's the most compatible format.
func WriteImagePNG(pngBytes []byte) error {
	// Decode PNG and convert to DIB format with proper row padding
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	dibData, err := imageToDIB(img)
	if err != nil {
		return fmt.Errorf("failed to convert image to DIB: %w", err)
	}

	if !openClipboard() {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if ok, _, _ := procEmptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("failed to empty clipboard")
	}

	// Try setting as CF_DIB first (most compatible with Windows apps)
	hMem, _, _ := procGlobalAlloc.Call(uintptr(GMEM_MOVEABLE), uintptr(len(dibData)))
	if hMem == 0 {
		return fmt.Errorf("failed to allocate global memory for DIB")
	}

	lpMem, _, _ := procGlobalLock.Call(hMem)
	if lpMem == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to lock global memory")
	}

	copy((*[1 << 30]byte)(unsafe.Pointer(lpMem))[:len(dibData)], dibData)
	procGlobalUnlock.Call(hMem)

	if result, _, _ := procSetClipboardData.Call(uintptr(cfDIB), hMem); result == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data as DIB: result=%d", result)
	}

	return nil
}

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

	// DIB rows are stored bottom-to-top, each row padded to 4-byte boundary
	rowSize := ((width*32 + 31) / 32) * 4 // Calculate row stride in bytes
	padding := rowSize - (width * 4)

	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			r32, g32, b32, a32 := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r := uint8(r32 >> 8)
			g := uint8(g32 >> 8)
			b := uint8(b32 >> 8)
			a := uint8(a32 >> 8)

			buf.WriteByte(b)
			buf.WriteByte(g)
			buf.WriteByte(r)
			buf.WriteByte(a)
		}
		// Add padding to align row to 4-byte boundary
		for p := 0; p < padding; p++ {
			buf.WriteByte(0)
		}
	}

	return buf.Bytes(), nil
}
