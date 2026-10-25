package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if v, err := db.Version(ctx); err != nil || v != 2 {
		t.Fatalf("version=%d err=%v, want 2", v, err)
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

func TestSearchFTS(t *testing.T) {
	db, ctx := openTestDB(t)
	chat := bridge.JID{User: "s", Server: "s.whatsapp.net"}
	if err := db.UpsertChat(ctx, bridge.Chat{JID: chat, Kind: bridge.ChatDM}); err != nil {
		t.Fatal(err)
	}
	mk := func(id, body string) bridge.Message {
		return bridge.Message{Chat: chat, ID: id, Sender: chat, Ts: time.Unix(100, 0), Kind: bridge.MsgText, Body: body, State: bridge.StateSent}
	}
	for _, m := range []bridge.Message{mk("1", "hello world"), mk("2", "goodbye world"), mk("3", "hello again")} {
		if err := db.UpsertMessage(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.SearchMessages(ctx, "hello", "", 10)
	if err != nil || len(got) != 2 {
		t.Fatalf("search=%+v err=%v", got, err)
	}
	scoped, err := db.SearchMessages(ctx, "world", chat.String(), 10)
	if err != nil || len(scoped) != 2 {
		t.Fatalf("scoped=%+v err=%v", scoped, err)
	}
	if empty, err := db.SearchMessages(ctx, "   ", "", 10); err != nil || len(empty) != 0 {
		t.Fatalf("blank=%+v err=%v", empty, err)
	}
	// Edit re-indexes via trigger.
	edited := mk("1", "totally different")
	if err := db.UpsertMessage(ctx, edited); err != nil {
		t.Fatal(err)
	}
	if again, _ := db.SearchMessages(ctx, "hello", "", 10); len(again) != 1 {
		t.Fatalf("reindex=%+v", again)
	}
}

func TestBackupAndExport(t *testing.T) {
	db, ctx := openTestDB(t)
	chat := bridge.Chat{JID: bridge.JID{User: "e", Server: "s.whatsapp.net"}, Kind: bridge.ChatDM, Title: "Ex"}
	if err := db.UpsertChat(ctx, chat); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertMessage(ctx, bridge.Message{Chat: chat.JID, ID: "m1", Sender: chat.JID, Ts: time.Unix(10, 0), Kind: bridge.MsgText, Body: "hello", State: bridge.StateRead}); err != nil {
		t.Fatal(err)
	}
	var txt strings.Builder
	if err := db.ExportChat(ctx, chat.JID.String(), "txt", &txt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt.String(), "hello") || !strings.Contains(txt.String(), "Ex") {
		t.Fatalf("txt=%q", txt.String())
	}
	var js strings.Builder
	if err := db.ExportChat(ctx, chat.JID.String(), "json", &js); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages []struct {
			Body string `json:"body"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(js.String()), &decoded); err != nil || len(decoded.Messages) != 1 {
		t.Fatalf("json=%q err=%v", js.String(), err)
	}
	if err := db.ExportChat(ctx, chat.JID.String(), "xml", &js); err == nil {
		t.Fatal("bad format must fail")
	}
	bak := filepath.Join(t.TempDir(), "backup.db")
	if err := db.BackupTo(bak); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(bak)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestChatStateSync(t *testing.T) {
	db, ctx := openTestDB(t)
	jid := "w@s.whatsapp.net"
	if err := db.UpsertChat(ctx, bridge.Chat{JID: bridge.JID{User: "w", Server: "s.whatsapp.net"}, Kind: bridge.ChatDM}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertMessage(ctx, bridge.Message{Chat: bridge.JID{User: "w", Server: "s.whatsapp.net"}, ID: "m1", Sender: bridge.JID{User: "w", Server: "s.whatsapp.net"}, Ts: time.Unix(1, 0), Kind: bridge.MsgText, Body: "x", State: bridge.StateSent}); err != nil {
		t.Fatal(err)
	}
	if err := db.SetChatArchived(ctx, jid, true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetChatPinned(ctx, jid, true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetChatMuted(ctx, jid, 99); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetChat(ctx, jid)
	if err != nil || !got.Archived || !got.Pinned || got.MutedUntil != 99 {
		t.Fatalf("flags=%+v err=%v", got, err)
	}
	if err := db.SetMessageStarred(ctx, jid, "m1", true); err != nil {
		t.Fatal(err)
	}
	msgs, _ := db.ListMessages(ctx, jid, 10, 0, "")
	if len(msgs) != 1 || !msgs[0].Starred {
		t.Fatalf("star=%+v", msgs)
	}
	if err := db.DeleteMessage(ctx, jid, "m1"); err != nil {
		t.Fatal(err)
	}
	if msgs, _ := db.ListMessages(ctx, jid, 10, 0, ""); len(msgs) != 0 {
		t.Fatal("delete-for-me")
	}
	if err := db.ClearChat(ctx, jid); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteChat(ctx, jid); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetChat(ctx, jid); err == nil {
		t.Fatal("chat should be gone")
	}
}

func TestUpgradeAutoBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	// Simulate a v1 database: drop v2 objects, reset version.
	raw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{`DROP TRIGGER messages_ai`, `DROP TRIGGER messages_ad`, `DROP TRIGGER messages_au`, `DROP TABLE messages_fts`, `PRAGMA user_version=1`} {
		if _, err := raw.db.Exec(stmt); err != nil {
			raw.Close()
			t.Fatal(err)
		}
	}
	raw.Close()
	// Reopen: hook must back up, then re-apply 0002.
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if v, _ := db2.Version(context.Background()); v != 2 {
		t.Fatalf("version=%d", v)
	}
	if _, err := os.Stat(path + ".pre-migrate-v1"); err != nil {
		t.Fatal("pre-migrate backup missing")
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
