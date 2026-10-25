package media

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestCropRotate(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	c := Crop(img, image.Rect(5, 5, 15, 15))
	if c.Bounds().Dx() != 10 || c.Bounds().Dy() != 10 {
		t.Fatalf("crop=%v", c.Bounds())
	}
	r := Rotate90(img)
	if r.Bounds().Dx() != 20 || r.Bounds().Dy() != 40 {
		t.Fatalf("rotate=%v", r.Bounds())
	}
	// Out-of-bounds crop clamps, never panics.
	_ = Crop(img, image.Rect(-99, -99, 999, 999))
}

func TestStroke(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	Stroke(img, []image.Point{{2, 2}, {17, 17}}, color.Black, 3)
	if img.RGBAAt(10, 10) == (color.RGBA{}) {
		t.Fatal("stroke missed the diagonal")
	}
}

func TestStorage(t *testing.T) {
	data := t.TempDir()
	chatDir := filepath.Join(data, "media", "1@s.whatsapp.net")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chatDir, "a.bin"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	u, err := UsageByChat(data)
	if err != nil || u["1@s.whatsapp.net"] != 100 {
		t.Fatalf("usage=%v err=%v", u, err)
	}
	cache := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(cache, "t.jpg"), []byte("x"), 0o644)
	if err := ClearCache(cache); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(cache); len(entries) != 0 {
		t.Fatal("cache not cleared")
	}
}
