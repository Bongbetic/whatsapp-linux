package logx_test

import (
	"strings"
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/logx"
)

func TestScrubJID(t *testing.T) {
	in := "msg from 1234567890@s.whatsapp.net arrived"
	out := logx.Scrub(in)
	if strings.Contains(out, "1234567890") {
		t.Fatalf("JID leaked: %q", out)
	}
	if !strings.Contains(out, "@s.whatsapp.net") {
		t.Fatalf("server part lost: %q", out)
	}
	// Stable: same input hashes the same way.
	if again := logx.Scrub(in); again != out {
		t.Fatalf("unstable scrub: %q vs %q", out, again)
	}
}

func TestScrubPhone(t *testing.T) {
	out := logx.Scrub("call +1 555 123 4567 now")
	if strings.Contains(out, "555 123 4567") {
		t.Fatalf("phone leaked: %q", out)
	}
}

func TestScrubShortNumbersPass(t *testing.T) {
	in := "retry 3 of 5"
	if out := logx.Scrub(in); out != in {
		t.Fatalf("short numbers must pass through: %q", out)
	}
}
