//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"  // register GIF decoder for clipboard image validation
	_ "image/jpeg" // register JPEG decoder for clipboard image validation
	"image/png"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
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
	procRegisterClipboardFormatW   = user32DLL.NewProc("RegisterClipboardFormatW")
)

const (
	cfText        = 1
	cfDIB         = 8
	cfDIBV5       = 17
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
			w.tickSafely(ctx)
		}
	}
}

// tickSafely runs one poll, recovering from any panic so a malformed clipboard
// payload can never crash the whole app. The stack is logged so it stays
// diagnosable.
func (w *Watcher) tickSafely(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("clipboard watcher: recovered from panic: %v\n%s", r, debug.Stack())
		}
	}()
	w.tick(ctx)
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
	img, err := readImageWindows()
	if err != nil {
		// Previously this was silent, so image-read failures looked like "nothing
		// happened". Log them so a failing screenshot is diagnosable.
		log.Printf("clipboard: image read failed: %v", err)
		return
	}
	if len(img) > 0 {
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

// pngFormatIDs lazily registers the "PNG" and "image/png" clipboard formats that
// modern apps (Qt/Flameshot, browsers, Office) use, and returns their IDs.
var (
	pngFormatOnce sync.Once
	pngFormatIDs  []uint32
)

func registerPNGFormats() {
	// Different apps expose copied images under different registered names. We try
	// the common ones; the bytes are decoded with image.Decode (PNG/JPEG/GIF),
	// so any of these that carries a standard encoded image will be captured.
	for _, name := range []string{"PNG", "image/png", "image/x-png", "image/jpeg", "JFIF", "image/gif", "GIF"} {
		if p, err := syscall.UTF16PtrFromString(name); err == nil {
			if id, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(p))); id != 0 {
				pngFormatIDs = append(pngFormatIDs, uint32(id))
			}
		}
	}
}

func pngFormats() []uint32 {
	pngFormatOnce.Do(registerPNGFormats)
	return pngFormatIDs
}

// rawImage is one candidate clipboard image payload.
type rawImage struct {
	data  []byte
	isPNG bool   // true: already PNG bytes; false: a packed DIB
	label string // for diagnostics
}

// readImageWindows reads whatever image the clipboard holds and returns PNG
// bytes. It collects every supported format (CF_DIB, CF_DIBV5, registered PNG)
// while the clipboard is briefly open, then decodes them after release and
// returns the first that succeeds — so a quirk in one format can't lose an image
// that another format carries cleanly.
func readImageWindows() ([]byte, error) {
	candidates, err := readClipboardImages()
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, c := range candidates {
		if c.isPNG {
			if _, _, e := image.Decode(bytes.NewReader(c.data)); e != nil {
				lastErr = fmt.Errorf("%s decode: %w", c.label, e)
				continue
			}
			return c.data, nil
		}
		img, e := dibToImage(c.data)
		if e != nil {
			lastErr = fmt.Errorf("%s decode: %w", c.label, e)
			continue
		}
		var pngBuf bytes.Buffer
		if e := png.Encode(&pngBuf, img); e != nil {
			lastErr = fmt.Errorf("%s encode: %w", c.label, e)
			continue
		}
		return pngBuf.Bytes(), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no supported image format on clipboard")
	}
	return nil, lastErr
}

// readClipboardImages opens the clipboard once (thread-pinned) and copies out the
// raw bytes of every supported image format present, in priority order.
func readClipboardImages() ([]rawImage, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
		return nil, fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard()

	pngIDs := pngFormats()
	// One-line diagnostic so a failing screenshot reveals exactly what the source
	// app put on the clipboard.
	log.Printf("clipboard: image change; DIB=%v DIBV5=%v PNG=%v",
		isFormatAvailable(cfDIB), isFormatAvailable(cfDIBV5), anyFormatAvailable(pngIDs))

	var out []rawImage
	// Prefer DIB/DIBV5 (lossless); fall back to PNG.
	if isFormatAvailable(cfDIB) {
		if b, e := readGlobalFormat(cfDIB); e == nil && len(b) > 0 {
			out = append(out, rawImage{data: b, isPNG: false, label: "CF_DIB"})
		}
	}
	if isFormatAvailable(cfDIBV5) {
		if b, e := readGlobalFormat(cfDIBV5); e == nil && len(b) > 0 {
			out = append(out, rawImage{data: b, isPNG: false, label: "CF_DIBV5"})
		}
	}
	for _, id := range pngIDs {
		if isFormatAvailable(id) {
			if b, e := readGlobalFormat(id); e == nil && len(b) > 0 {
				out = append(out, rawImage{data: b, isPNG: true, label: "PNG"})
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no supported image format on clipboard")
	}
	return out, nil
}

// readGlobalFormat copies the HGLOBAL behind a clipboard format into a private
// slice. Must be called with the clipboard already open.
func readGlobalFormat(format uint32) ([]byte, error) {
	handle, _, _ := procGetClipboardData.Call(uintptr(format))
	if handle == 0 {
		return nil, fmt.Errorf("GetClipboardData(%d) returned null", format)
	}
	lockedPtr, _, _ := procGlobalLock.Call(handle)
	if lockedPtr == 0 {
		return nil, fmt.Errorf("GlobalLock failed for format %d", format)
	}
	defer procGlobalUnlock.Call(handle)

	size, _, _ := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("GlobalSize 0 for format %d", format)
	}
	const maxImageSize = 100 * 1024 * 1024
	if size > maxImageSize {
		return nil, fmt.Errorf("clipboard image too large: %d bytes (max: %d)", size, maxImageSize)
	}

	src := (*[1 << 30]byte)(unsafe.Pointer(lockedPtr))[:int(size):int(size)]
	out := make([]byte, int(size))
	copy(out, src)
	return out, nil
}

func anyFormatAvailable(ids []uint32) bool {
	for _, id := range ids {
		if isFormatAvailable(id) {
			return true
		}
	}
	return false
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
