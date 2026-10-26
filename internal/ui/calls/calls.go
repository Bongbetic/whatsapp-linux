// Package calls renders the call log page (WL-5.3, §9 ui/calls):
// call log list from the calls table, new-call picker, call-link
// create/copy (links need the fork — picker reports that honestly).
package calls

import (
	"context"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

// View is the calls page widget.
type View struct {
	Box  *gtk.Box
	list *gtk.ListBox
	db   *appstore.DB
	eng  bridge.Engine
}

// New builds the page. onCall places a call via the fork (WL-5.1).
func New(db *appstore.DB, eng bridge.Engine, onCall func(chat bridge.JID, video bool)) *View {
	v := &View{db: db, eng: eng}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 6)
	row := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("phone number to call")
	row.Append(entry)
	voice := gtk.NewButtonWithLabel("Voice call")
	voice.ConnectClicked(func() { onCall(jidOf(entry.Buffer().Text()), false) })
	row.Append(voice)
	video := gtk.NewButtonWithLabel("Video call")
	video.ConnectClicked(func() { onCall(jidOf(entry.Buffer().Text()), true) })
	row.Append(video)
	link := gtk.NewButtonWithLabel("Copy call link")
	link.SetTooltipText("call links land at WL-5.5")
	row.Append(link)
	v.Box.Append(row)
	v.list = gtk.NewListBox()
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetChild(v.list)
	v.Box.Append(scroll)
	return v
}

// Refresh reloads the call log (WL-5.3).
func (v *View) Refresh() {
	v.list.RemoveAll()
	rows, err := v.db.ListCalls(context.Background(), 100)
	if err != nil {
		return
	}
	for _, c := range rows {
		dir := "←"
		if c.Direction == "out" {
			dir = "→"
		}
		v.list.Append(gtk.NewLabel(dir + " " + c.Chat.String() + " " + c.Kind + " " +
			time.Unix(c.StartedTs, 0).Format("02/01 15:04") + " " + c.Outcome))
	}
}

func jidOf(phone string) bridge.JID {
	if phone == "" {
		return bridge.JID{}
	}
	return bridge.JID{User: phone, Server: "s.whatsapp.net"}
}
