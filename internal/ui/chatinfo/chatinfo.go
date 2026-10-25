// Package chatinfo renders the right-side chat-info drawer (WL-2.6):
// photo, about, members list for groups, mute/ephemeral/block
// actions. Group management (add/remove/promote/invites) and the
// safety-numbers dialog land at WL-4.1/4.6; the drawer already
// exposes the hooks.
package chatinfo

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Callbacks wires drawer actions to the app shell.
type Callbacks struct {
	OnMute      func(chat bridge.JID, hours int64)
	OnBlock     func(chat bridge.JID, block bool)
	OnGroupEdit func(chat bridge.JID) // WL-4.1 member management
}

// View is the chat-info drawer widget.
type View struct {
	Box   *gtk.Box
	title *gtk.Label
	about *gtk.Label
	cb    Callbacks
	chat  bridge.Chat
}

// New builds the drawer.
func New(cb Callbacks) *View {
	v := &View{cb: cb}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 8)
	v.title = gtk.NewLabel("")
	v.title.AddCSSClass("title-2")
	v.Box.Append(v.title)
	v.about = gtk.NewLabel("")
	v.about.AddCSSClass("dim-label")
	v.Box.Append(v.about)

	row := gtk.NewBox(gtk.OrientationHorizontal, 6)
	mute := gtk.NewButtonWithLabel("Mute 8h")
	mute.ConnectClicked(func() { v.cb.OnMute(v.chat.JID, 8) })
	row.Append(mute)
	block := gtk.NewButtonWithLabel("Block")
	block.ConnectClicked(func() { v.cb.OnBlock(v.chat.JID, true) })
	row.Append(block)
	grp := gtk.NewButtonWithLabel("Members…")
	grp.ConnectClicked(func() { v.cb.OnGroupEdit(v.chat.JID) })
	row.Append(grp)
	v.Box.Append(row)
	return v
}

// ShowChat loads a chat into the drawer.
func (v *View) ShowChat(c bridge.Chat, contact bridge.Contact) {
	v.chat = c
	v.title.SetText(c.Title)
	about := contact.About
	if about == "" && c.Kind == bridge.ChatGroup {
		about = "Group • members at WL-4.1"
	}
	v.about.SetText(about)
}
