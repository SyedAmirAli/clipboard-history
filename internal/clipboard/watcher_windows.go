//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

	if !isFormatAvailable(cfDIB) {
		return nil, fmt.Errorf("no image format available")
	}

	handle, _, _ := procGetClipboardData.Call(uintptr(cfDIB))
	if handle == 0 {
		return nil, fmt.Errorf("failed to get clipboard image data")
	}

	lockedPtr, _, _ := procGlobalLock.Call(handle)
	if lockedPtr == 0 {
		return nil, fmt.Errorf("failed to lock clipboard image")
	}
	defer procGlobalUnlock.Call(handle)

	size, _, _ := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("failed to get clipboard image size")
	}

	dibData := (*[1 << 30]byte)(unsafe.Pointer(lockedPtr))[:size:size]

	img, err := dibToImage(dibData)
	if err != nil {
		fmt.Printf("DEBUG: dibToImage failed on %d bytes: %v\n", len(dibData), err)
		return nil, fmt.Errorf("failed to convert DIB to image: %w", err)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	return pngBuf.Bytes(), nil
}

func dibToImage(dibData []byte) (image.Image, error) {
	if len(dibData) < 40 {
		return nil, fmt.Errorf("DIB data too small for header")
	}

	reader := bytes.NewReader(dibData)
	var hdr struct {
		Size      uint32
		Width     int32
		Height    int32
		Planes    uint16
		BitCount  uint16
		Compress  uint32
		ImageSize uint32
		XPelsPerM int32
		YPelsPerM int32
		ClrUsed   uint32
		ClrImport uint32
	}

	if err := binary.Read(reader, binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}

	if hdr.Width <= 0 || hdr.Height <= 0 {
		return nil, fmt.Errorf("invalid DIB dimensions: %dx%d (BitCount: %d)", hdr.Width, hdr.Height, hdr.BitCount)
	}

	fmt.Printf("DEBUG: DIB header - Size:%d Width:%d Height:%d BitCount:%d Compress:%d\n", hdr.Size, hdr.Width, hdr.Height, hdr.BitCount, hdr.Compress)

	height := int(hdr.Height)
	if hdr.Height < 0 {
		height = -height
	}

	rect := image.Rect(0, 0, int(hdr.Width), height)
	dst := image.NewRGBA(rect)

	switch hdr.BitCount {
	case 24, 32:
		if err := decodeDIBPixels(reader, dst, int(hdr.Width), height, int(hdr.BitCount)); err != nil {
			fmt.Printf("DEBUG: decodeDIBPixels failed: %v\n", err)
			return nil, fmt.Errorf("decode pixels (BitCount=%d, %dx%d): %w", hdr.BitCount, hdr.Width, height, err)
		}
	default:
		return nil, fmt.Errorf("unsupported DIB bit count: %d (Width:%d Height:%d)", hdr.BitCount, hdr.Width, hdr.Height)
	}

	return dst, nil
}

func decodeDIBPixels(reader *bytes.Reader, dst *image.RGBA, width, height, bitCount int) error {
	bytesPerPixel := bitCount / 8
	rowSize := ((width*bitCount + 31) / 32) * 4

	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			var b, g, r, a uint8 = 0, 0, 0, 255

			if bitCount == 24 {
				if err := binary.Read(reader, binary.LittleEndian, &b); err != nil {
					return err
				}
				if err := binary.Read(reader, binary.LittleEndian, &g); err != nil {
					return err
				}
				if err := binary.Read(reader, binary.LittleEndian, &r); err != nil {
					return err
				}
			} else if bitCount == 32 {
				if err := binary.Read(reader, binary.LittleEndian, &b); err != nil {
					return err
				}
				if err := binary.Read(reader, binary.LittleEndian, &g); err != nil {
					return err
				}
				if err := binary.Read(reader, binary.LittleEndian, &r); err != nil {
					return err
				}
				if err := binary.Read(reader, binary.LittleEndian, &a); err != nil {
					return err
				}
			}

			dst.SetRGBA(x, y, color.RGBA{r, g, b, a})
		}

		readBytes := width * bytesPerPixel
		padding := rowSize - readBytes
		if padding > 0 {
			_, err := reader.Read(make([]byte, padding))
			if err != nil && err.Error() != "EOF" {
				return err
			}
		}
	}

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

func isFormatAvailable(format uint32) bool {
	r, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	return r != 0
}
