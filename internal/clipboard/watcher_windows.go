//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
	user32DLL   = syscall.NewLazyDLL("user32.dll")

	procGetClipboardSequenceNumber = user32DLL.NewProc("GetClipboardSequenceNumber")
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
	cfText        = 1
	cfDIB         = 8
	cfPNG         = 49320
	GMEM_MOVEABLE = 0x0002

	// BITMAPINFOHEADER compression modes.
	biRGB         = 0
	biBitfields   = 3
	biAlphaFields = 6
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

	// GetClipboardSequenceNumber increments on every clipboard change. It is far
	// more reliable than polling the owner window (which misses changes when HWNDs
	// are reused) and needs no clipboard open, so it never contends with Ctrl+C.
	var lastSeq uintptr
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			seq, _, _ := procGetClipboardSequenceNumber.Call()
			if seq == lastSeq {
				continue
			}
			lastSeq = seq
			w.tick(ctx)
		}
	}
}

func (w *Watcher) tick(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Try text first (most common). readTextWindows holds the clipboard only
	// long enough to copy the bytes out, then releases it.
	if text, err := readTextWindows(); err == nil && text != "" {
		h := hashString(text)
		if w.swapHash(h) {
			return
		}
		w.publish(Event{ContentType: ContentTypeText, Text: text, Hash: h})
		return
	}

	// Fall back to image.
	if img, err := readImageWindows(); err == nil && len(img) > 0 {
		h := hashBytes(img)
		if w.swapHash(h) {
			return
		}
		w.publish(Event{ContentType: ContentTypeImage, ImagePNG: img, Hash: h})
	}
}

// readTextWindows copies the clipboard text out under a locked OS thread and a
// short clipboard hold. All heavy work (none, for text) stays outside the hold.
func readTextWindows() (string, error) {
	// The Win32 clipboard is thread-affine: OpenClipboard and CloseClipboard must
	// run on the same OS thread. Pin the goroutine so the runtime can't migrate it
	// mid-sequence, which would strand the clipboard open and break it system-wide.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
		return "", fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	if !isFormatAvailable(cfText) {
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

	size, _, _ := procGlobalSize.Call(handle)
	if size == 0 {
		return "", nil
	}

	cstr := (*[1 << 30]byte)(unsafe.Pointer(lockedPtr))[:int(size):int(size)]
	length := int(size)
	for i := 0; i < len(cstr); i++ {
		if cstr[i] == 0 {
			length = i
			break
		}
	}
	return string(cstr[:length]), nil
}

// readImageWindows copies the raw DIB bytes out under a short, thread-locked
// clipboard hold, then decodes/encodes to PNG *after* releasing the clipboard.
// Keeping the heavy work outside the hold is what prevents the clipboard from
// being stranded open (which previously killed Ctrl+C app-wide).
func readImageWindows() ([]byte, error) {
	raw, err := readDIBRaw()
	if err != nil {
		return nil, err
	}

	img, err := dibToImage(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DIB to image: %w", err)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}
	return pngBuf.Bytes(), nil
}

// readDIBRaw returns a private copy of the clipboard's CF_DIB bytes. The
// clipboard is held only for the memcpy.
func readDIBRaw() ([]byte, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
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

	const maxImageSize = 100 * 1024 * 1024
	if size > maxImageSize {
		return nil, fmt.Errorf("clipboard image too large: %d bytes (max: %d)", size, maxImageSize)
	}

	src := (*[1 << 30]byte)(unsafe.Pointer(lockedPtr))[:int(size):int(size)]
	raw := make([]byte, int(size))
	copy(raw, src)
	return raw, nil
}

// dibToImage decodes a packed DIB (BITMAPINFOHEADER/V4/V5, 24- or 32-bit,
// BI_RGB or BI_BITFIELDS) into an opaque RGBA image. It indexes the byte slice
// directly (no streaming reader) and recovers from any out-of-range panic so a
// malformed DIB can never crash the app.
func dibToImage(dib []byte) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("panic decoding DIB: %v", r)
		}
	}()

	const maxDimension = 20000
	if len(dib) < 40 {
		return nil, fmt.Errorf("DIB data too small for header (%d bytes)", len(dib))
	}

	le := binary.LittleEndian
	biSize := le.Uint32(dib[0:4])
	width := int(int32(le.Uint32(dib[4:8])))
	rawHeight := int(int32(le.Uint32(dib[8:12])))
	bitCount := int(le.Uint16(dib[14:16]))
	compression := le.Uint32(dib[16:20])

	topDown := false
	height := rawHeight
	if height < 0 {
		height = -height
		topDown = true
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid DIB dimensions: %dx%d", width, rawHeight)
	}
	if width > maxDimension || height > maxDimension {
		return nil, fmt.Errorf("DIB dimensions too large: %dx%d (max: %d)", width, height, maxDimension)
	}
	if bitCount != 24 && bitCount != 32 {
		return nil, fmt.Errorf("unsupported DIB bit count: %d", bitCount)
	}

	// Locate the pixel data. For a 40-byte BITMAPINFOHEADER, BI_BITFIELDS color
	// masks (3 or 4 DWORDs) sit between the header and the pixels. For V4/V5
	// headers (>=108 bytes) the masks live inside the header itself.
	pixelOffset := int(biSize)
	if biSize <= 40 {
		switch compression {
		case biRGB:
			pixelOffset = 40
		case biBitfields:
			pixelOffset = 40 + 12
		case biAlphaFields:
			pixelOffset = 40 + 16
		default:
			return nil, fmt.Errorf("unsupported DIB compression: %d", compression)
		}
	} else if compression != biRGB && compression != biBitfields && compression != biAlphaFields {
		return nil, fmt.Errorf("unsupported DIB compression: %d", compression)
	}
	if pixelOffset < 0 || pixelOffset >= len(dib) {
		return nil, fmt.Errorf("pixel offset %d outside DIB data (%d bytes)", pixelOffset, len(dib))
	}

	bytesPerPixel := bitCount / 8
	rowSize := ((width*bitCount + 31) / 32) * 4 // rows are padded to 4-byte boundaries
	pix := dib[pixelOffset:]

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for row := 0; row < height; row++ {
		// DIBs are bottom-up unless the height was negative (top-down).
		srcRow := height - 1 - row
		if topDown {
			srcRow = row
		}
		rowStart := srcRow * rowSize
		if rowStart+width*bytesPerPixel > len(pix) {
			break // truncated data: leave the remaining rows blank rather than panic
		}
		di := dst.PixOffset(0, row)
		for x := 0; x < width; x++ {
			si := rowStart + x*bytesPerPixel
			// DIB stores BGR(A); force the image opaque because clipboard DIB alpha
			// is frequently zero/garbage and would otherwise render fully transparent.
			dst.Pix[di] = pix[si+2]   // R
			dst.Pix[di+1] = pix[si+1] // G
			dst.Pix[di+2] = pix[si]   // B
			dst.Pix[di+3] = 255       // A
			di += 4
		}
	}
	return dst, nil
}

// openClipboardRetry opens the clipboard, retrying briefly to ride out the
// common case where another app holds it open for a moment. Must be called with
// the OS thread already locked.
func openClipboardRetry() bool {
	for i := 0; i < 12; i++ {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func closeClipboard() bool {
	r, _, _ := procCloseClipboard.Call()
	return r != 0
}

func isFormatAvailable(format uint32) bool {
	r, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	return r != 0
}
