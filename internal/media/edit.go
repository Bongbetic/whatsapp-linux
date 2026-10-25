package media

import (
	"errors"
	"image"
	"image/color"
)

// Image editor ops v1 (WL-3.5): crop, rotate-90, freehand strokes.
// Text annotation needs font rasterization (no in-tree rasterizer
// without a new dependency) — AnnotateText reserves the hook and
// reports ErrTextUnsupported until WL-3.5b.

// ErrTextUnsupported is returned by AnnotateText until a font
// rasterizer lands (WL-3.5b).
var ErrTextUnsupported = errors.New("text annotation unsupported until WL-3.5b")

// Crop returns the sub-image rect (clamped to bounds).
func Crop(img image.Image, r image.Rectangle) image.Image {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return img
	}
	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	if s, ok := img.(subImager); ok {
		return s.SubImage(r)
	}
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.Set(x-r.Min.X, y-r.Min.Y, img.At(x, y))
		}
	}
	return dst
}

// Rotate90 returns img rotated 90° clockwise.
func Rotate90(img image.Image) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// Stroke draws a polyline of width w in c (freehand draw, WL-3.5).
func Stroke(img *image.RGBA, pts []image.Point, c color.Color, w int) {
	for i := 1; i < len(pts); i++ {
		line(img, pts[i-1], pts[i], c, w)
	}
}

func line(img *image.RGBA, a, b image.Point, c color.Color, w int) {
	dx, dy := b.X-a.X, b.Y-a.Y
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		dot(img, a, c, w)
		return
	}
	for i := 0; i <= steps; i++ {
		dot(img, image.Point{X: a.X + dx*i/steps, Y: a.Y + dy*i/steps}, c, w)
	}
}

func dot(img *image.RGBA, p image.Point, c color.Color, w int) {
	r := w / 2
	for y := p.Y - r; y <= p.Y+r; y++ {
		for x := p.X - r; x <= p.X+r; x++ {
			if (image.Point{X: x, Y: y}).In(img.Bounds()) {
				img.Set(x, y, c)
			}
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// AnnotateText reserves text annotation (WL-3.5b).
func AnnotateText(_ *image.RGBA, _ string, _ image.Point) error {
	return ErrTextUnsupported
}
