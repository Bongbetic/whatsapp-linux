package media

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrepareImage reads src and writes a send-ready JPEG (quality 80,
// max edge 1600 px) unless asDocument (§12). Returns the output path.
func PrepareImage(src, dstDir string, asDocument bool) (string, error) {
	if asDocument {
		return src, nil
	}
	img, err := decode(src)
	if err != nil {
		return "", err
	}
	img = fit(img, 1600)
	out := filepath.Join(dstDir, base(src)+".jpg")
	if err := writeJPEG(out, img, 80); err != nil {
		return "", err
	}
	return out, nil
}

// Thumbnail writes a 320 px JPEG preview into the cache dir (§12).
func Thumbnail(src, dstDir string) (string, error) {
	img, err := decode(src)
	if err != nil {
		return "", err
	}
	out := filepath.Join(dstDir, base(src)+".thumb.jpg")
	return out, writeJPEG(out, fit(img, 320), 70)
}

// StickerWebP converts src to a 512×512 WebP sticker (pad/cover via
// ffmpeg; animated sources stay animated when already WebP, §12).
func StickerWebP(src, dst string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg missing for sticker convert: %w", err)
	}
	cmd := exec.Command(ffmpeg, "-y", "-i", src, "-vf",
		"scale=512:512:force_original_aspect_ratio=increase,crop=512:512",
		"-vcodec", "libwebp", "-lossless", "0", "-q", "80", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sticker convert: %v\n%s", err, out)
	}
	return nil
}

// NeedsTranscode reports whether src is already MP4/H.264 (streaming
// copy) or must be transcoded (§12).
func NeedsTranscode(src string) (bool, error) {
	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		return true, nil // unknown: transcode to be safe
	}
	out, err := exec.Command(probe, "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name,format_name", "-of", "csv=p=0", src).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("probe: %v\n%s", err, out)
	}
	s := strings.TrimSpace(string(out))
	return !(strings.Contains(s, "h264") || strings.Contains(s, "mp4")), nil
}

// TranscodeMP4 converts src to MP4 (H.264+AAC) at dst (§12).
func TranscodeMP4(src, dst string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg missing for transcode: %w", err)
	}
	cmd := exec.Command(ffmpeg, "-y", "-i", src, "-c:v", "libx264", "-preset", "veryfast",
		"-c:a", "aac", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("transcode: %v\n%s", err, out)
	}
	return nil
}

func decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func writeJPEG(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

// fit scales img so its longest edge is at most max (cover down only).
func fit(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	long := w
	if h > long {
		long = h
	}
	if long <= max {
		return img
	}
	nw, nh := w*max/long, h*max/long
	if nw == 0 {
		nw = 1
	}
	if nh == 0 {
		nh = 1
	}
	return scale(img, nw, nh)
}

// scale is a small bilinear resampler (stdlib has none).
func scale(src image.Image, nw, nh int) *image.RGBA {
	if nw < 2 {
		nw = 2
	}
	if nh < 2 {
		nh = 2
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := float64(y) * float64(b.Dy()-1) / float64(nh-1)
		y0, y1, fy := int(sy), int(sy)+1, sy-float64(int(sy))
		if y1 >= b.Dy() {
			y1 = b.Dy() - 1
		}
		for x := 0; x < nw; x++ {
			sx := float64(x) * float64(b.Dx()-1) / float64(nw-1)
			x0, x1, fx := int(sx), int(sx)+1, sx-float64(int(sx))
			if x1 >= b.Dx() {
				x1 = b.Dx() - 1
			}
			var c [4]float64
			pts := [4][2]int{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}}
			ws := [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
			for i, p := range pts {
				r, g, bl, a := src.At(b.Min.X+p[0], b.Min.Y+p[1]).RGBA()
				c[0] += float64(r) * ws[i]
				c[1] += float64(g) * ws[i]
				c[2] += float64(bl) * ws[i]
				c[3] += float64(a) * ws[i]
			}
			dst.SetRGBA(x, y, rgba(c))
		}
	}
	return dst
}

func rgba(c [4]float64) color.RGBA {
	return color.RGBA{
		R: uint8(c[0] / 257),
		G: uint8(c[1] / 257),
		B: uint8(c[2] / 257),
		A: uint8(c[3] / 257),
	}
}

func base(path string) string {
	name := filepath.Base(path)
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return name
}
