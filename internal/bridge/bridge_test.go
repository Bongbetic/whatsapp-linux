package bridge_test

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func TestEventKinds(t *testing.T) {
	evs := []bridge.Event{
		bridge.ConnEvent{}, bridge.QRCode{}, bridge.PairDone{},
		bridge.MessageEvent{}, bridge.ChatEvent{}, bridge.ReceiptEvent{},
		bridge.TypingEvent{}, bridge.SyncProgress{}, bridge.CallEvent{},
		bridge.RefreshEvent{},
	}
	seen := map[string]bool{}
	for _, e := range evs {
		if e.Kind() == "" || seen[e.Kind()] {
			t.Fatalf("bad/duplicate kind %q", e.Kind())
		}
		seen[e.Kind()] = true
	}
}

func TestJIDString(t *testing.T) {
	j := bridge.JID{User: "1", Server: "s.whatsapp.net"}
	if j.String() != "1@s.whatsapp.net" {
		t.Fatalf("got %q", j.String())
	}
}
