// Package thumbnail produces small base64-encoded PNG previews from full
// images. It uses a hand-written box-averaging downscaler so we don't
// have to depend on golang.org/x/image — keeps the binary lean.
package thumbnail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
)

// MaxDim is the longest edge (in pixels) of generated thumbnails.
const MaxDim = 160

// DataURL takes raw image bytes (PNG, JPEG, GIF, or BMP) and returns a small
// PNG thumbnail encoded as a "data:image/png;base64,..." URL suitable for
// an <img src>. The returned width/height are the natural dimensions of the
// source image (NOT the thumbnail), which the UI uses for display metadata.
// If the input is not PNG, it's automatically converted to PNG for storage.
func DataURL(imgBytes []byte) (dataURL string, srcW, srcH int, err error) {
	src, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", 0, 0, fmt.Errorf("decode image: %w", err)
	}
	b := src.Bounds()
	srcW, srcH = b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return "", srcW, srcH, fmt.Errorf("empty image")
	}
	dstW, dstH := fitWithin(srcW, srcH, MaxDim)
	thumb := downscale(src, dstW, dstH)
	var buf bytes.Buffer
	enc := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, thumb); err != nil {
		return "", srcW, srcH, fmt.Errorf("encode thumb: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), srcW, srcH, nil
}

// ToPNG converts any image format (PNG, JPEG, GIF, BMP) to PNG bytes.
// If already PNG, returns the input unchanged. Used to normalize image
// formats for clipboard storage.
func ToPNG(imgBytes []byte) ([]byte, error) {
	src, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	if format == "png" {
		return imgBytes, nil
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		return nil, fmt.Errorf("encode to PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// fitWithin returns dimensions for an image scaled so the longest edge
// is at most maxDim, preserving the aspect ratio. If the source is
// already smaller than maxDim, the original dimensions are returned.
func fitWithin(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w >= h {
		return maxDim, max(1, h*maxDim/w)
	}
	return max(1, w*maxDim/h), maxDim
}

// downscale produces an RGBA thumbnail using simple box averaging. This
// is good enough for ~128px previews without pulling in image/draw.
func downscale(src image.Image, dstW, dstH int) *image.RGBA {
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		y0 := b.Min.Y + y*srcH/dstH
		y1 := b.Min.Y + (y+1)*srcH/dstH
		if y1 == y0 {
			y1 = y0 + 1
		}
		for x := 0; x < dstW; x++ {
			x0 := b.Min.X + x*srcW/dstW
			x1 := b.Min.X + (x+1)*srcW/dstW
			if x1 == x0 {
				x1 = x0 + 1
			}
			var rSum, gSum, bSum, aSum, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					r, g, b8, a := src.At(sx, sy).RGBA()
					rSum += uint64(r >> 8)
					gSum += uint64(g >> 8)
					bSum += uint64(b8 >> 8)
					aSum += uint64(a >> 8)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = byte(rSum / n)
			dst.Pix[i+1] = byte(gSum / n)
			dst.Pix[i+2] = byte(bSum / n)
			dst.Pix[i+3] = byte(aSum / n)
		}
	}
	return dst
}
