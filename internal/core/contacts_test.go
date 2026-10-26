package core

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

func openCoreTestDB(t *testing.T) (*appstore.DB, context.Context) {
	t.Helper()
	db, err := appstore.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, context.Background()
}

func TestDisplayNamePrefersContactName(t *testing.T) {
	if got := displayName("Book", "Push", "123"); got != "Book" {
		t.Fatalf("got %q", got)
	}
	if got := displayName("", "Push", "123"); got != "Push" {
		t.Fatalf("got %q", got)
	}
	if got := displayName("", "", "123"); got != "123" {
		t.Fatalf("got %q", got)
	}
}

// Live-account regression: contact rows must merge push names and
// phone-book names without clobbering, and DM titles prefer the book.
func TestContactMergeAndDMTitle(t *testing.T) {
	db, ctx := openCoreTestDB(t)
	e := &Engine{db: db}
	jid := bridge.JID{User: "555", Server: "s.whatsapp.net"}
	notePushName(ctx, db, jid, "Pushy")
	noteContactAction(ctx, db, jid, "Real Name")
	// A later push-name change must not clobber the phone-book name.
	notePushName(ctx, db, jid, "NewPush")
	got, err := db.GetContact(ctx, jid.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.ContactName != "Real Name" || got.PushName != "NewPush" {
		t.Fatalf("got %+v", got)
	}
	if title := e.dmTitle(ctx, jid, ""); title != "Real Name" {
		t.Fatalf("title=%q", title)
	}
	if title := e.dmTitle(ctx, bridge.JID{User: "nobody", Server: "s.whatsapp.net"}, ""); title != "+nobody" {
		t.Fatalf("fallback=%q", title)
	}
}

func TestLooksBare(t *testing.T) {
	chat := bridge.JID{User: "123", Server: "s.whatsapp.net"}
	if !looksBare("", chat) || !looksBare(chat.String(), chat) || !looksBare("+123", chat) {
		t.Fatal("bare titles not detected")
	}
	if looksBare("Alice", chat) {
		t.Fatal("real name flagged as bare")
	}
}
