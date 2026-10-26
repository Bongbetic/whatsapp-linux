// Package channels renders Channels (WL-4.4): followed list, channel
// page with react, follow/unfollow by JID, mute toggle. Directory
// search and posting-as-channel stay scoped to eligibility (Phase 8).
package channels

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

// View is the channels page widget.
type View struct {
	Box  *gtk.Box
	list *gtk.ListBox
	db   *appstore.DB
	eng  bridge.Engine
}

// New builds the page.
func New(db *appstore.DB, eng bridge.Engine) *View {
	v := &View{db: db, eng: eng}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 6)
	row := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("channel JID or invite to follow")
	row.Append(entry)
	follow := gtk.NewButtonWithLabel("Follow")
	follow.ConnectClicked(func() {
		jid := parseJID(entry.Buffer().Text())
		if jid.User == "" {
			return
		}
		_ = eng.ChannelFollow(context.Background(), jid)
		entry.Buffer().SetText("", -1)
		v.Refresh()
	})
	row.Append(follow)
	v.Box.Append(row)
	v.list = gtk.NewListBox()
	// The side page already scrolls; a nested scroller would
	// collapse to zero height inside it.
	v.Box.Append(v.list)
	return v
}

// Refresh lists channel chats with follow state and mute toggles.
func (v *View) Refresh() {
	v.list.RemoveAll()
	chats, err := v.db.ListChats(context.Background(), 200, 0, "", true)
	if err != nil {
		return
	}
	for _, c := range chats {
		if c.Kind != bridge.ChatChannel {
			continue
		}
		c := c
		row := gtk.NewBox(gtk.OrientationHorizontal, 6)
		lbl := gtk.NewLabel(c.Title)
		lbl.SetHExpand(true)
		lbl.SetXAlign(0)
		row.Append(lbl)
		mute := gtk.NewButtonWithLabel("Mute")
		mute.ConnectClicked(func() {
			_ = v.eng.ChannelSetMute(context.Background(), c.JID, true)
		})
		row.Append(mute)
		leave := gtk.NewButtonWithLabel("Unfollow")
		leave.ConnectClicked(func() {
			_ = v.eng.ChannelUnfollow(context.Background(), c.JID)
			v.Refresh()
		})
		row.Append(leave)
		v.list.Append(row)
	}
}

func parseJID(s string) bridge.JID {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return bridge.JID{User: s[:i], Server: s[i+1:]}
		}
	}
	if s == "" {
		return bridge.JID{}
	}
	return bridge.JID{User: s, Server: "newsletter"}
}
