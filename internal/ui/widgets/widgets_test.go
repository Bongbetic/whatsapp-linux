package widgets

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func TestInitials(t *testing.T) {
	if got := Initials("Ada Lovelace"); got != "AL" {
		t.Fatalf("got %q", got)
	}
	if got := Initials(""); got != "?" {
		t.Fatalf("got %q", got)
	}
}

func TestRowText(t *testing.T) {
	m := bridge.Message{Kind: bridge.MsgText, Body: "hi"}
	if RowText(m) != "hi" {
		t.Fatal("text")
	}
	m = bridge.Message{Kind: bridge.MsgRevoked}
	if RowText(m) == "" {
		t.Fatal("revoked needs placeholder")
	}
	m = bridge.Message{Kind: bridge.MsgText, Body: "x", Edited: true}
	if RowText(m) != "x (edited)" {
		t.Fatalf("got %q", RowText(m))
	}
}

func TestRowMetaTicks(t *testing.T) {
	m := bridge.Message{IsFromMe: true, State: bridge.StateRead}
	if got := RowMeta(m); len(got) == 0 {
		t.Fatal("empty meta")
	}
	in := bridge.Message{IsFromMe: false, State: bridge.StateRead}
	if got := RowMeta(in); len(got) == 0 {
		t.Fatal("empty meta inbound")
	}
}
