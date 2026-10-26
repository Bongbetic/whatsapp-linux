// Package calls renders the call log page (WL-5.3, §9 ui/calls):
// call log list from the calls table, new-call picker, call-link
// create/copy (links need the fork — picker reports that honestly).
package calls

import (
	"context"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// View is the calls page widget.
type View struct {
	Box  *gtk.Box
	list *gtk.ListBox
	db   *appstore.DB
	eng  bridge.Engine
	name func(bridge.JID) string
}

// New builds the page. onCall places a call via the fork (WL-5.1);
// name resolves a JID to the contact's display name.
func New(db *appstore.DB, eng bridge.Engine, name func(bridge.JID) string, onCall func(chat bridge.JID, video bool)) *View {
	v := &View{db: db, eng: eng, name: name}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 6)
	entry := gtk.NewEntry()
	entry.AddCSSClass("search-pill")
	entry.SetPlaceholderText("Phone number to call")
	entry.SetIconFromIconName(gtk.EntryIconPrimary, "wl-calls-symbolic")
	v.Box.Append(entry)
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.AddCSSClass("filter-row")
	voice := gtk.NewButtonWithLabel("Voice call")
	voice.AddCSSClass("filter-pill")
	voice.ConnectClicked(func() { onCall(jidOf(entry.Buffer().Text()), false) })
	row.Append(voice)
	video := gtk.NewButtonWithLabel("Video call")
	video.AddCSSClass("filter-pill")
	video.ConnectClicked(func() { onCall(jidOf(entry.Buffer().Text()), true) })
	row.Append(video)
	link := gtk.NewButtonWithLabel("Copy call link")
	link.AddCSSClass("filter-pill")
	link.SetTooltipText("call links land at WL-5.5")
	row.Append(link)
	v.Box.Append(row)
	v.Box.Append(sectionLabel("Recent"))
	v.list = gtk.NewListBox()
	v.list.AddCSSClass("chat-list")
	v.list.SetSelectionMode(gtk.SelectionNone)
	v.Box.Append(v.list)
	return v
}

func sectionLabel(text string) *gtk.Label {
	l := gtk.NewLabel(text)
	l.AddCSSClass("section-title")
	l.SetXAlign(0)
	return l
}

// Refresh reloads the call log (WL-5.3).
func (v *View) Refresh() {
	v.list.RemoveAll()
	rows, err := v.db.ListCalls(context.Background(), 100)
	if err != nil {
		return
	}
	for _, c := range rows {
		v.list.Append(v.callRow(c))
	}
}

// callRow lays a log entry out like a chat row: avatar, name, and a
// direction/time line that turns red for missed calls.
func (v *View) callRow(c bridge.CallLog) *gtk.Box {
	title := c.Chat.String()
	if v.name != nil {
		title = v.name(c.Chat)
	}
	row := gtk.NewBox(gtk.OrientationHorizontal, 14)
	row.Append(widgets.AvatarFor(title, false, 44))
	text := gtk.NewBox(gtk.OrientationVertical, 2)
	text.SetHExpand(true)
	text.SetVAlign(gtk.AlignCenter)
	nameL := gtk.NewLabel(title)
	nameL.AddCSSClass("chat-row-name")
	nameL.SetXAlign(0)
	nameL.SetEllipsize(pango.EllipsizeEnd)
	text.Append(nameL)
	dir := "↙ Incoming"
	if c.Direction == "out" {
		dir = "↗ Outgoing"
	}
	if c.Outcome == "missed" || c.Outcome == "rejected" {
		dir = "↙ Missed"
		nameL.AddCSSClass("negative")
	}
	sub := gtk.NewLabel(dir + " · " + time.Unix(c.StartedTs, 0).Format("02/01 15:04"))
	sub.AddCSSClass("chat-row-preview")
	sub.SetXAlign(0)
	text.Append(sub)
	row.Append(text)
	icon := "wl-calls-symbolic"
	if c.Kind == "video" {
		icon = "wl-video-symbolic"
	}
	kind := gtk.NewImageFromIconName(icon)
	kind.AddCSSClass("chat-row-mark")
	row.Append(kind)
	return row
}

func jidOf(phone string) bridge.JID {
	if phone == "" {
		return bridge.JID{}
	}
	return bridge.JID{User: phone, Server: "s.whatsapp.net"}
}
