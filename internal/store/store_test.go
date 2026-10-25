package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func openTestDB(t *testing.T) (*DB, context.Context) {
	t.Helper()
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if v, err := db.Version(ctx); err != nil || v != 1 {
		t.Fatalf("version=%d err=%v, want 1", v, err)
	}
	// Migrating twice is a noop (WL-1.2).
	if err := db.migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return db, ctx
}

func TestChatMessageRoundTrip(t *testing.T) {
	db, ctx := openTestDB(t)
	chat := bridge.Chat{JID: bridge.JID{User: "123", Server: "s.whatsapp.net"}, Kind: bridge.ChatDM, Title: "T", LastTs: 42}
	if err := db.UpsertChat(ctx, chat); err != nil {
		t.Fatal(err)
	}
	msg := bridge.Message{Chat: chat.JID, ID: "m1", Sender: chat.JID, Ts: time.Unix(42, 0), Kind: bridge.MsgText, Body: "hi", State: bridge.StateSent}
	if err := db.UpsertMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateMessageState(ctx, chat.JID.String(), "m1", bridge.StateRead); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetChat(ctx, chat.JID.String())
	if err != nil || got.Title != "T" {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	msgs, err := db.ListMessages(ctx, chat.JID.String(), 10, 0, "")
	if err != nil || len(msgs) != 1 || msgs[0].State != bridge.StateRead || msgs[0].Body != "hi" {
		t.Fatalf("msgs=%+v err=%v", msgs, err)
	}
	chats, err := db.ListChats(ctx, 10, 0, "", false)
	if err != nil || len(chats) != 1 {
		t.Fatalf("chats=%+v err=%v", chats, err)
	}
	// Keyset second page is empty.
	next, err := db.ListChats(ctx, 10, chats[0].LastTs, chat.JID.String(), false)
	if err != nil || len(next) != 0 {
		t.Fatalf("second page=%+v err=%v", next, err)
	}
}

func TestContactGroupPollCallKV(t *testing.T) {
	db, ctx := openTestDB(t)
	c := bridge.Contact{JID: bridge.JID{User: "9", Server: "s.whatsapp.net"}, PushName: "P"}
	if err := db.UpsertContact(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := db.SetBlocked(ctx, c.JID.String(), true); err != nil {
		t.Fatal(err)
	}
	cs, err := db.ListContacts(ctx)
	if err != nil || len(cs) != 1 || !cs[0].Blocked {
		t.Fatalf("contacts=%+v err=%v", cs, err)
	}
	g := "g@g.us"
	if err := db.SetGroupMembers(ctx, g, []bridge.GroupMember{{User: c.JID, Role: bridge.RoleAdmin}}); err != nil {
		t.Fatal(err)
	}
	ms, err := db.GroupMembers(ctx, g)
	if err != nil || len(ms) != 1 || ms[0].Role != bridge.RoleAdmin {
		t.Fatalf("members=%+v err=%v", ms, err)
	}
	if err := db.SavePoll(ctx, g, "p1", "Q?", `["a","b"]`, 1); err != nil {
		t.Fatal(err)
	}
	if err := db.SavePollVote(ctx, g, "p1", c.JID.String(), `["a"]`, 7); err != nil {
		t.Fatal(err)
	}
	if err := db.LogCall(ctx, "c1", g, "in", "audio", 8, 60, "answered"); err != nil {
		t.Fatal(err)
	}
	calls, err := db.ListCalls(ctx, 10)
	if err != nil || len(calls) != 1 || calls[0].DurationS != 60 {
		t.Fatalf("calls=%+v err=%v", calls, err)
	}
	if err := db.SetDownload(ctx, g, "m9", "queued", 0, 100); err != nil {
		t.Fatal(err)
	}
	if err := db.KVPut(ctx, "k", "v"); err != nil {
		t.Fatal(err)
	}
	if v, err := db.KVGet(ctx, "k"); err != nil || v != "v" {
		t.Fatalf("kv=%q err=%v", v, err)
	}
}
