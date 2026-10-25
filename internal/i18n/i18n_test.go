package i18n

import "testing"

func TestFallback(t *testing.T) {
	SetLanguage("xx-none")
	if got := T("login.title"); got != "Link this device" {
		t.Fatalf("fallback broke: %q", got)
	}
	if got := T("missing.key"); got != "missing.key" {
		t.Fatalf("key fallback: %q", got)
	}
}

func TestAddAndSelect(t *testing.T) {
	Add("de", map[string]string{"login.title": "Gerät verknüpfen"})
	SetLanguage("de")
	if got := T("login.title"); got != "Gerät verknüpfen" {
		t.Fatalf("got %q", got)
	}
	SetLanguage("en")
}
