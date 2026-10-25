package notify

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func TestBodyPreview(t *testing.T) {
	m := bridge.Message{Kind: bridge.MsgText, Body: "secret text"}
	if got := bodyPreview(m, true); got != "secret text" {
		t.Fatalf("got %q", got)
	}
	if got := bodyPreview(m, false); got != "New message" {
		t.Fatalf("preview off leaked: %q", got)
	}
	if got := bodyPreview(bridge.Message{Kind: bridge.MsgVoice}, true); got != "🎙 Voice message" {
		t.Fatalf("got %q", got)
	}
}
