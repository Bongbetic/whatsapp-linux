package settings_test

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/settings"
)

func TestDefaults(t *testing.T) {
	s := settings.New()
	if !s.GetBool(settings.KeyAutostart) || !s.GetBool(settings.KeyCloseToTray) {
		t.Fatal("bad bool defaults")
	}
	if s.GetBool(settings.KeyStartHidden) || s.GetBool(settings.KeyAutoVideo) {
		t.Fatal("false defaults flipped")
	}
	if s.GetFloat(settings.KeyFontScale) != 1.0 {
		t.Fatal("font scale default")
	}
	if s.GetInt(settings.KeyWindowW) != 1180 || s.GetInt(settings.KeyWindowH) != 760 {
		t.Fatal("window defaults")
	}
}

func TestSetGet(t *testing.T) {
	s := settings.New()
	s.Set(settings.KeyFontScale, 1.5)
	if s.GetFloat(settings.KeyFontScale) != 1.5 {
		t.Fatal("set/get float")
	}
	s.Set(settings.KeyDownloadDir, "/tmp/x")
	if s.GetString(settings.KeyDownloadDir) != "/tmp/x" {
		t.Fatal("set/get string")
	}
}
