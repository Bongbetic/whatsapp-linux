package widgets

import (
	"strings"
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

func TestRowTicks(t *testing.T) {
	if ticks, read := RowTicks(bridge.Message{IsFromMe: true, State: bridge.StateRead}); ticks != "✓✓" || !read {
		t.Fatalf("read: got %q %v", ticks, read)
	}
	if ticks, read := RowTicks(bridge.Message{IsFromMe: true, State: bridge.StateDelivered}); ticks != "✓✓" || read {
		t.Fatalf("delivered: got %q %v", ticks, read)
	}
	if ticks, read := RowTicks(bridge.Message{IsFromMe: true, State: bridge.StateSent}); ticks != "✓" || read {
		t.Fatalf("sent: got %q %v", ticks, read)
	}
	if ticks, _ := RowTicks(bridge.Message{State: bridge.StateRead}); ticks != "" {
		t.Fatal("incoming must have no ticks")
	}
}

func TestChatPreview(t *testing.T) {
	m := bridge.Message{IsFromMe: true, Kind: bridge.MsgText, Body: "hello"}
	if got := ChatPreview(m, ""); got != "You: hello" {
		t.Fatalf("got %q", got)
	}
	m2 := bridge.Message{Kind: bridge.MsgVoice}
	if got := ChatPreview(m2, "Ann"); !strings.HasPrefix(got, "Ann: ") {
		t.Fatalf("got %q", got)
	}
	if SenderColor("a@s.whatsapp.net") == "" {
		t.Fatal("sender color must be non-empty")
	}
}
