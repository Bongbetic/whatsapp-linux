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
