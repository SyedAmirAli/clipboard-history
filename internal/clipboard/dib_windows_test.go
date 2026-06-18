//go:build windows

package clipboard

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// buildBITMAPINFOHEADER writes a 40-byte BITMAPINFOHEADER.
func buildBITMAPINFOHEADER(width, height int32, bitCount uint16, compression uint32) []byte {
	hdr := make([]byte, 40)
	le := binary.LittleEndian
	le.PutUint32(hdr[0:4], 40)
	le.PutUint32(hdr[4:8], uint32(width))
	le.PutUint32(hdr[8:12], uint32(height))
	le.PutUint16(hdr[12:14], 1) // planes
	le.PutUint16(hdr[14:16], bitCount)
	le.PutUint32(hdr[16:20], compression)
	return hdr
}

// TestDIBRoundTrip verifies imageToDIB -> dibToImage reproduces the original
// pixels (alpha forced opaque). This is the exact path used when an image is
// written to the clipboard and then read back.
func TestDIBRoundTrip(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})     // red
	src.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})     // green
	src.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})     // blue
	src.SetRGBA(1, 1, color.RGBA{255, 255, 255, 255}) // white

	dib, err := imageToDIB(src)
	if err != nil {
		t.Fatalf("imageToDIB: %v", err)
	}

	got, err := dibToImage(dib)
	if err != nil {
		t.Fatalf("dibToImage: %v", err)
	}

	if got.Bounds() != src.Bounds() {
		t.Fatalf("bounds: got %v want %v", got.Bounds(), src.Bounds())
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			wr, wg, wb, _ := src.At(x, y).RGBA()
			gr, gg, gb, ga := got.At(x, y).RGBA()
			if gr != wr || gg != wg || gb != wb || ga != 0xffff {
				t.Errorf("pixel (%d,%d): got RGBA(%d,%d,%d,%d) want RGB(%d,%d,%d) opaque",
					x, y, gr>>8, gg>>8, gb>>8, ga>>8, wr>>8, wg>>8, wb>>8)
			}
		}
	}
}

// TestDIBBitfields verifies a 32-bit BI_BITFIELDS DIB — the format modern
// Windows screenshot tools produce — decodes correctly. Previously these were
// rejected outright, which is why screenshots failed to appear in history.
func TestDIBBitfields(t *testing.T) {
	const biBitfieldsCompression = 3
	dib := buildBITMAPINFOHEADER(2, 2, 32, biBitfieldsCompression)

	// Three RGB color masks follow the header for BI_BITFIELDS.
	masks := make([]byte, 12)
	binary.LittleEndian.PutUint32(masks[0:4], 0x00FF0000)  // red
	binary.LittleEndian.PutUint32(masks[4:8], 0x0000FF00)  // green
	binary.LittleEndian.PutUint32(masks[8:12], 0x000000FF) // blue
	dib = append(dib, masks...)

	// Pixels are bottom-up, stored BGRA. Bottom row first.
	// Bottom row (image y=1): blue, white. Top row (image y=0): red, green.
	bgra := func(b, g, r, a byte) []byte { return []byte{b, g, r, a} }
	dib = append(dib, bgra(255, 0, 0, 0)...)     // blue   (alpha 0 on purpose)
	dib = append(dib, bgra(255, 255, 255, 0)...) // white
	dib = append(dib, bgra(0, 0, 255, 0)...)     // red
	dib = append(dib, bgra(0, 255, 0, 0)...)     // green

	got, err := dibToImage(dib)
	if err != nil {
		t.Fatalf("dibToImage(BI_BITFIELDS): %v", err)
	}

	want := map[[2]int]color.RGBA{
		{0, 0}: {255, 0, 0, 255},
		{1, 0}: {0, 255, 0, 255},
		{0, 1}: {0, 0, 255, 255},
		{1, 1}: {255, 255, 255, 255},
	}
	for pos, w := range want {
		gr, gg, gb, ga := got.At(pos[0], pos[1]).RGBA()
		if gr>>8 != uint32(w.R) || gg>>8 != uint32(w.G) || gb>>8 != uint32(w.B) || ga>>8 != 255 {
			t.Errorf("pixel %v: got RGBA(%d,%d,%d,%d) want %v (opaque)",
				pos, gr>>8, gg>>8, gb>>8, ga>>8, w)
		}
	}
}

// TestDIBMalformedNoPanic ensures a truncated/garbage DIB returns an error
// instead of crashing the app (the decoder runs under panic recovery).
func TestDIBMalformedNoPanic(t *testing.T) {
	cases := [][]byte{
		nil,
		{1, 2, 3},
		buildBITMAPINFOHEADER(2, 2, 32, 0), // header only, no pixels
		buildBITMAPINFOHEADER(100000, 100000, 32, 0),
		buildBITMAPINFOHEADER(2, 2, 16, 0), // unsupported bit count
	}
	for i, dib := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("case %d panicked: %v", i, r)
				}
			}()
			// We only require that it does not panic. An error or a best-effort
			// image are both acceptable for malformed input.
			_, _ = dibToImage(dib)
		}()
	}
}
