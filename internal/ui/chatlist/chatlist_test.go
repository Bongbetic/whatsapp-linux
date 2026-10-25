package chatlist

import (
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func TestFilter(t *testing.T) {
	chats := []bridge.Chat{
		{Title: "Alice"}, {Title: "Bob", Archived: true}, {Title: "alice cooper"},
	}
	got := Filter(chats, "alice", false)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	arch := Filter(chats, "", true)
	if len(arch) != 1 || arch[0].Title != "Bob" {
		t.Fatalf("got %+v", arch)
	}
}
