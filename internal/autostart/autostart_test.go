package autostart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/autostart"
)

func TestSetRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := autostart.Set(true); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(autostart.Dir(), autostart.DesktopFileID))
	if err != nil || !strings.Contains(string(body), "--hidden") {
		t.Fatalf("entry=%q err=%v", body, err)
	}
	if err := autostart.Set(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(autostart.Dir(), autostart.DesktopFileID)); !os.IsNotExist(err) {
		t.Fatal("entry not removed")
	}
	// Disabling twice is fine.
	if err := autostart.Set(false); err != nil {
		t.Fatal(err)
	}
}
