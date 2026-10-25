package media

import (
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, image.White)
		}
	}
	p := filepath.Join(t.TempDir(), "src.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func decodeSize(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func TestPrepareImageCapsEdge(t *testing.T) {
	src := writeTestPNG(t, 3200, 800)
	out, err := PrepareImage(src, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	w, h := decodeSize(t, out)
	if w != 1600 || h != 400 {
		t.Fatalf("got %dx%d want 1600x400", w, h)
	}
}

func TestPrepareImageKeepsSmall(t *testing.T) {
	src := writeTestPNG(t, 100, 50)
	out, err := PrepareImage(src, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := decodeSize(t, out); w != 100 || h != 50 {
		t.Fatalf("small image must not upscale: %dx%d", w, h)
	}
}

func TestThumbnail(t *testing.T) {
	src := writeTestPNG(t, 1280, 960)
	out, err := Thumbnail(src, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := decodeSize(t, out); w != 320 {
		t.Fatalf("thumb width=%d", w)
	}
}

func TestNeedsTranscodeProbe(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	mp4 := filepath.Join(t.TempDir(), "clip.mp4")
	cmd := exec.Command(ffmpeg, "-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=64x64:rate=5",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", mp4)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make clip: %v\n%s", err, out)
	}
	need, err := NeedsTranscode(mp4)
	if err != nil || need {
		t.Fatalf("mp4/h264 must stream-copy: need=%v err=%v", need, err)
	}
	// A still image probes fine but is not H.264: transcode path.
	need, err = NeedsTranscode(writeTestPNG(t, 8, 8))
	if err != nil || !need {
		t.Fatalf("png must take transcode path: need=%v err=%v", need, err)
	}
	// Missing file: probe errors.
	if _, err := NeedsTranscode(filepath.Join(t.TempDir(), "nope.mp4")); err == nil {
		t.Fatal("missing file must fail probe")
	}
}
