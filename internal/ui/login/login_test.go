package login

import "testing"

func TestBitmap(t *testing.T) {
	mods, err := Bitmap("test-qr-payload-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) == 0 || len(mods) != len(mods[0]) {
		t.Fatal("QR bitmap must be non-empty square")
	}
	// Deterministic for the same payload.
	again, _ := Bitmap("test-qr-payload-123")
	if len(again) != len(mods) {
		t.Fatal("unstable bitmap")
	}
}

func TestQRLayoutFitsAllocation(t *testing.T) {
	for _, size := range [][2]int{{347, 280}, {280, 347}, {280, 280}} {
		const modules = 73
		x, y, cell := qrLayout(size[0], size[1], modules)
		if cell < 1 || x < 0 || y < 0 || x+modules*cell > size[0] || y+modules*cell > size[1] {
			t.Fatalf("QR clipped in %v: origin=(%d,%d), cell=%d", size, x, y, cell)
		}
	}
}
