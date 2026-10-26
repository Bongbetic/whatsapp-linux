package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/store"
)

func dm(u string) bridge.JID  { return bridge.JID{User: u, Server: "s.whatsapp.net"} }
func grp(u string) bridge.JID { return bridge.JID{User: u, Server: "g.us"} }

type seedChat struct {
	jid    bridge.JID
	name   string
	last   string
	ago    time.Duration
	unread int
	pinned bool
	muted  bool
}

func seed(ctx context.Context, db *store.DB) error {
	now := time.Now()
	day := 24 * time.Hour
	chats := []seedChat{
		{grp("120363000000000001"), "Design Team", "pushed the new rail icons", 4 * time.Minute, 2, true, false},
		{dm("919800000001"), "Mum", "Call me when you are free", 25 * time.Minute, 1, true, false},
		{dm("919800000002"), "Rahul Sharma", "Sounds good, see you then", 80 * time.Minute, 0, false, false},
		{dm("919800000003"), "Priya", "Sent the photos", 3 * time.Hour, 3, false, false},
		{grp("120363000000000002"), "Weekend Trek", "who is bringing the tent?", 5 * time.Hour, 12, false, true},
		{dm("919800000004"), "Ankit", "Haha yes", day + time.Hour, 0, false, false},
		{grp("120363000000000003"), "Office", "standup moved to 11", day + 4*time.Hour, 0, false, true},
		{dm("919800000005"), "Neha Gupta", "Thanks!", 3 * day, 0, false, false},
		{dm("919800000006"), "Dad", "Reached home safely", 5 * day, 0, false, false},
		{dm("919800000007"), "Bongbetic Support", "Your ticket has been resolved", 20 * day, 0, false, false},
	}
	for _, c := range chats {
		kind := bridge.ChatDM
		if c.jid.Server == "g.us" {
			kind = bridge.ChatGroup
		} else if err := db.UpsertContact(ctx, bridge.Contact{JID: c.jid, ContactName: c.name, Phone: "+" + c.jid.User}); err != nil {
			return err
		}
		var muted int64
		if c.muted {
			muted = -1
		}
		ts := now.Add(-c.ago)
		id := "last-" + c.jid.User
		if err := db.UpsertChat(ctx, bridge.Chat{JID: c.jid, Kind: kind, Title: c.name, Unread: c.unread, Pinned: c.pinned, MutedUntil: muted, LastMsgID: id, LastTs: ts.Unix()}); err != nil {
			return err
		}
		sender := c.jid
		if kind == bridge.ChatGroup {
			sender = dm("919811110001")
		}
		if err := db.UpsertMessage(ctx, bridge.Message{Chat: c.jid, ID: id, Sender: sender, Ts: ts, Kind: bridge.MsgText, Body: c.last, State: bridge.StateRead}); err != nil {
			return err
		}
	}

	// Group senders (not in the chat list).
	people := map[string]string{"919811110001": "Rahul Sharma", "919811110002": "Aisha Khan", "919811110003": "Vikram"}
	for u, n := range people {
		if err := db.UpsertContact(ctx, bridge.Contact{JID: dm(u), ContactName: n, Phone: "+" + u}); err != nil {
			return err
		}
	}
	rahul, aisha, vik := dm("919811110001"), dm("919811110002"), dm("919811110003")
	me := dm("919899999999")
	g := chats[0].jid
	yday := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, now.Location()).Add(-day)
	type m struct {
		from  bridge.JID
		min   int // minutes after yday 17:30; >=1440 means relative to today below
		body  string
		state string
	}
	msgs := []m{
		{rahul, 0, "Morning all! Sharing the updated mockups for the chat list.", ""},
		{aisha, 3, "Looks clean. The spacing on the rows feels much better.", ""},
		{me, 7, "Agreed. I will start on the header bar tomorrow.", bridge.StateRead},
		{vik, 12, "Can we try a slightly bigger avatar?", ""},
		{me, 15, "Sure, 48px?", bridge.StateRead},
		{vik, 16, "Yes", ""},
		{me, 40, "Here is the plan for tomorrow:\n1. Header bar and connection dot\n2. Navigation rail icons\n3. Dark mode pass on the conversation view\nShout if I missed anything.", bridge.StateDelivered},
		{aisha, 55, "Perfect, thanks!", ""},
	}
	for i, x := range msgs {
		mm := bridge.Message{Chat: g, ID: fmt.Sprintf("g-%02d", i), Sender: x.from, IsFromMe: x.from == me, Ts: yday.Add(time.Duration(x.min) * time.Minute), Kind: bridge.MsgText, Body: x.body, State: x.state}
		if err := db.UpsertMessage(ctx, mm); err != nil {
			return err
		}
	}
	today := []m{
		{rahul, 0, "Good morning! Did everyone see the new palette?", ""},
		{me, 0, "Yes, the red accent works nicely.", bridge.StateSent},
		{aisha, 0, "Dark mode looks great too.", ""},
		{me, 0, "Thanks! Will push the header bar changes shortly.", bridge.StateRead},
		{rahul, 0, "Pushed the new rail icons, have a look.", ""},
		{aisha, 0, "Nice, checking now", ""},
	}
	// Last two incoming are the unread ones.
	for i, x := range today {
		ts := now.Add(-time.Duration(len(today)-i)*5*time.Minute - time.Minute)
		mm := bridge.Message{Chat: g, ID: fmt.Sprintf("g-t%02d", i), Sender: x.from, IsFromMe: x.from == me, Ts: ts, Kind: bridge.MsgText, Body: x.body, State: x.state}
		if err := db.UpsertMessage(ctx, mm); err != nil {
			return err
		}
	}

	// Call log.
	calls := []struct {
		id, dir, kind, outcome string
		to                     bridge.JID
		ago                    time.Duration
		dur                    int64
	}{
		{"c1", "in", "audio", "missed", chats[1].jid, 2 * time.Hour, 0},
		{"c2", "out", "video", "answered", chats[2].jid, day + 2*time.Hour, 754},
		{"c3", "in", "audio", "answered", chats[3].jid, 3 * day, 95},
	}
	for _, c := range calls {
		if err := db.LogCall(ctx, c.id, c.to.String(), c.dir, c.kind, now.Add(-c.ago).Unix(), c.dur, c.outcome); err != nil {
			return err
		}
	}
	return nil
}
