// Package chatlist renders the chat list pane (WL-2.2): avatar,
// title, last message, timestamp, unread pill, mute/pin markers,
// live search filter, archived toggle, keyboard navigation.
package chatlist

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// View is the chat-list pane widget.
type View struct {
	Box      *gtk.Box
	list     *gtk.ListBox
	search   *gtk.Entry
	chats    []bridge.Chat
	filtered []bridge.Chat
	onOpen   func(bridge.Chat)
	showArch bool
}

// New builds the pane. onOpen fires on Enter/click.
func New(onOpen func(bridge.Chat)) *View {
	v := &View{onOpen: onOpen}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)

	v.search = gtk.NewEntry()
	v.search.SetPlaceholderText("Search chats (Ctrl+F)")
	buf := v.search.Buffer()
	buf.ConnectInsertedText(func(uint, string, uint) { v.applyFilter() })
	buf.ConnectDeletedText(func(uint, uint) { v.applyFilter() })
	v.Box.Append(v.search)

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	v.list = gtk.NewListBox()
	v.list.SetFilterFunc(func(row *gtk.ListBoxRow) bool {
		i := row.Index()
		return i >= 0 && i < len(v.filtered)
	})
	v.list.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		if i := row.Index(); i >= 0 && i < len(v.filtered) {
			v.onOpen(v.filtered[i])
		}
	})
	scroll.SetChild(v.list)
	v.Box.Append(scroll)
	return v
}

// SetChats replaces the list contents (newest-first from store).
func (v *View) SetChats(chats []bridge.Chat) {
	v.chats = chats
	v.applyFilter()
}

func (v *View) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(v.search.Buffer().Text()))
	v.filtered = v.filtered[:0]
	for _, c := range v.chats {
		if c.Archived != v.showArch {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) {
			continue
		}
		v.filtered = append(v.filtered, c)
	}
	v.list.RemoveAll()
	for _, c := range v.filtered {
		v.list.Append(chatRow(c))
	}
	v.list.InvalidateFilter()
}

// SetShowArchived toggles the archived folder view.
func (v *View) SetShowArchived(show bool) {
	v.showArch = show
	v.applyFilter()
}

// Filter is the pure chat-filter predicate (headless-testable).
func Filter(chats []bridge.Chat, query string, showArch bool) []bridge.Chat {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []bridge.Chat
	for _, c := range chats {
		if c.Archived != showArch {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func chatRow(c bridge.Chat) *gtk.Box {
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.Append(widgets.Avatar(c.Title))
	mid := gtk.NewBox(gtk.OrientationVertical, 0)
	mid.SetHExpand(true)
	title := gtk.NewLabel(c.Title)
	title.SetXAlign(0)
	mid.Append(title)
	last := gtk.NewLabel(lastLine(c))
	last.AddCSSClass("dim-label")
	last.SetXAlign(0)
	mid.Append(last)
	row.Append(mid)
	right := gtk.NewBox(gtk.OrientationVertical, 0)
	right.Append(gtk.NewLabel(stamp(c.LastTs)))
	if c.Unread > 0 {
		pill := gtk.NewLabel(fmt.Sprint(c.Unread))
		pill.AddCSSClass("unread-pill")
		right.Append(pill)
	}
	marks := ""
	if c.Pinned {
		marks += "📌"
	}
	if c.MutedUntil != 0 {
		marks += "🔇"
	}
	if marks != "" {
		right.Append(gtk.NewLabel(marks))
	}
	row.Append(right)
	return row
}

func lastLine(c bridge.Chat) string {
	if c.LastMsgID == "" {
		return "No messages yet"
	}
	return c.LastMsgID
}

func stamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	if time.Since(t) < 24*time.Hour {
		return t.Format("15:04")
	}
	return t.Format("02/01")
}
