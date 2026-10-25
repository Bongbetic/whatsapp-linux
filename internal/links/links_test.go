package links_test

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/links"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		phone string
		text  string
	}{
		{"whatsapp://send?phone=+15551234567&text=hi%20there", "15551234567", "hi there"},
		{"https://wa.me/15551234567?text=hello", "15551234567", "hello"},
		{"https://wa.me/15551234567", "15551234567", ""},
		{"tel:+15551234567", "15551234567", ""},
	}
	for _, c := range cases {
		got, err := links.Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) err=%v", c.in, err)
			continue
		}
		if got.Phone != c.phone || got.Text != c.text || got.Kind != "chat" {
			t.Errorf("Parse(%q)=%+v want phone=%q text=%q", c.in, got, c.phone, c.text)
		}
	}
	if _, err := links.Parse("https://example.com/x"); err == nil {
		t.Error("expected error for unsupported link")
	}
	if _, err := links.Parse("https://wa.me/"); err == nil {
		t.Error("expected error for wa.me without phone")
	}
}

// FuzzParseLink: parser must never panic; valid schemes yield chat.
func FuzzParseLink(f *testing.F) {
	for _, seed := range []string{
		"whatsapp://send?phone=123&text=hi",
		"https://wa.me/15551234567?text=x",
		"tel:+15551234567",
		"https://example.com/",
		"",
		"://",
		"whatsapp://send?phone=",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got, err := links.Parse(in)
		if err != nil {
			return
		}
		if got.Kind != "chat" || got.Phone == "" {
			t.Fatalf("bad intent %+v for %q", got, in)
		}
		for _, r := range got.Phone {
			if r < '0' || r > '9' {
				t.Fatalf("non-digit phone %q", got.Phone)
			}
		}
	})
}
