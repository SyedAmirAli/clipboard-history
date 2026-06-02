// Tool genicon generates a minimal 32x32 clipboard glyph PNG for use
// as the system tray icon. Run from the repo root with:
//
//	go run ./tools/genicon > internal/tray/icon.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

const (
	size = 32
)

func main() {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	fg := color.NRGBA{R: 0xE6, G: 0xE6, B: 0xE6, A: 0xFF}
	clip := color.NRGBA{R: 0xBC, G: 0xBC, B: 0xBC, A: 0xFF}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0})
		}
	}

	// Clipboard body (rounded-rectangle approximation)
	bodyX0, bodyY0, bodyX1, bodyY1 := 6, 8, 26, 30
	for y := bodyY0; y < bodyY1; y++ {
		for x := bodyX0; x < bodyX1; x++ {
			img.SetNRGBA(x, y, fg)
		}
	}
	// Top clip
	for y := 4; y < 9; y++ {
		for x := 12; x < 20; x++ {
			img.SetNRGBA(x, y, clip)
		}
	}
	// Three "lines" representing text rows
	row := color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	drawHLine(img, 10, 14, 22, row)
	drawHLine(img, 10, 19, 22, row)
	drawHLine(img, 10, 24, 18, row)

	enc := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(os.Stdout, img); err != nil {
		panic(err)
	}
}

func drawHLine(img *image.NRGBA, x0, y, x1 int, c color.NRGBA) {
	for x := x0; x < x1; x++ {
		img.SetNRGBA(x, y, c)
		img.SetNRGBA(x, y+1, c)
	}
}
