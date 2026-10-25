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
