// Package status renders Status (WL-4.3): own status row with post
// composer, contacts' rings list fed by status-broadcast messages,
// and a minimal viewer with reply box.
package status

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// View is the status page widget.
type View struct {
	Box     *gtk.Box
	list    *gtk.ListBox
	db      *appstore.DB
	eng     bridge.Engine
	onReply func(to bridge.JID, text string)
}

// New builds the page. onReply sends a reply into the sender's DM.
func New(db *appstore.DB, eng bridge.Engine, onReply func(to bridge.JID, text string)) *View {
	v := &View{db: db, eng: eng, onReply: onReply}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 6)
	composer := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Post a text status…")
	composer.Append(entry)
	post := gtk.NewButtonWithLabel("Post")
	post.ConnectClicked(func() {
		text := entry.Buffer().Text()
		if text == "" {
			return
		}
		if err := eng.StatusPost(context.Background(), text); err == nil {
			entry.Buffer().SetText("", -1)
		}
		v.Refresh()
	})
	composer.Append(post)
	v.Box.Append(composer)
	v.list = gtk.NewListBox()
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetChild(v.list)
	v.Box.Append(scroll)
	return v
}

// Refresh reloads recent status-broadcast messages grouped by sender.
func (v *View) Refresh() {
	v.list.RemoveAll()
	msgs, err := v.db.ListMessages(context.Background(), "status@broadcast", 100, 0, "")
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, m := range msgs {
		key := m.Sender.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		m := m
		row := gtk.NewButtonWithLabel("● " + key)
		row.ConnectClicked(func() { v.viewer(m) })
		v.list.Append(row)
	}
}

// viewer pops a minimal status viewer with a reply box.
func (v *View) viewer(m bridge.Message) {
	d := gtk.NewDialog()
	d.SetTitle("Status")
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	box.Append(gtk.NewLabel(widgets.RowText(m)))
	reply := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Reply…")
	reply.Append(entry)
	send := gtk.NewButtonWithLabel("Send")
	send.ConnectClicked(func() {
		if text := entry.Buffer().Text(); text != "" {
			v.onReply(m.Sender, text)
			d.Close()
		}
	})
	reply.Append(send)
	box.Append(reply)
	d.SetChild(box)
	d.Show()
}
