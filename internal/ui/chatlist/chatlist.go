// Package chatlist renders the chat list pane (WL-2.2): header with
// title + new-chat, filter pills (All/Unread/Groups), search, and
// rich rows (avatar, name, preview, time, unread pill, mute/pin).
package chatlist

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// Filter modes for the pill row.
const (
	FilterAll    = "all"
	FilterUnread = "unread"
	FilterGroups = "groups"
)

// View is the chat-list pane widget.
type View struct {
	Box      *gtk.Box
	list     *gtk.ListBox
	search   *gtk.Entry
	chats    []bridge.Chat
	filtered []bridge.Chat
	onOpen   func(bridge.Chat)
	onNew    func()
	preview  func(bridge.Chat) string
	cache    map[string]string
	showArch bool
	mode     string
	pills    map[string]*gtk.Button
}

// New builds the pane. onOpen fires on Enter/click, onNew on the
// compose button; preview resolves one row's snippet (may hit the
// store — results are cached per SetChats).
func New(onOpen func(bridge.Chat), onNew func()) *View {
	v := &View{onOpen: onOpen, onNew: onNew, mode: FilterAll, pills: map[string]*gtk.Button{}, cache: map[string]string{}}

	head := gtk.NewBox(gtk.OrientationHorizontal, 8)
	title := gtk.NewLabel("Chats")
	title.AddCSSClass("title-1")
	title.SetHExpand(true)
	title.SetXAlign(0)
	head.Append(title)
	newBtn := gtk.NewButtonFromIconName("list-add-symbolic")
	newBtn.SetTooltipText("New chat (Ctrl+N)")
	newBtn.ConnectClicked(func() { v.onNew() })
	head.Append(newBtn)
	v.Box = gtk.NewBox(gtk.OrientationVertical, 6)
	v.Box.Append(head)

	v.search = gtk.NewEntry()
	v.search.SetPlaceholderText("Search chats (Ctrl+F)")
	buf := v.search.Buffer()
	buf.ConnectInsertedText(func(uint, string, uint) { v.applyFilter() })
	buf.ConnectDeletedText(func(uint, uint) { v.applyFilter() })
	v.Box.Append(v.search)

	prow := gtk.NewBox(gtk.OrientationHorizontal, 6)
	for _, m := range []struct{ mode, label string }{
		{FilterAll, "All"}, {FilterUnread, "Unread"}, {FilterGroups, "Groups"},
	} {
		m := m
		b := gtk.NewButtonWithLabel(m.label)
		b.AddCSSClass("filter-pill")
		b.ConnectClicked(func() { v.SetMode(m.mode) })
		v.pills[m.mode] = b
		prow.Append(b)
	}
	v.Box.Append(prow)
	v.paintPills()

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	v.list = gtk.NewListBox()
	v.list.AddCSSClass("chat-list")
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

// SetPreview installs the snippet resolver (app.db LastMessage).
func (v *View) SetPreview(fn func(bridge.Chat) string) { v.preview = fn }

// SetMode switches the pill filter.
func (v *View) SetMode(mode string) {
	v.mode = mode
	v.paintPills()
	v.applyFilter()
}

func (v *View) paintPills() {
	for mode, b := range v.pills {
		if mode == v.mode {
			b.AddCSSClass("active")
		} else {
			b.RemoveCSSClass("active")
		}
	}
}

// SetChats replaces the list contents (newest-first from store).
func (v *View) SetChats(chats []bridge.Chat) {
	v.chats = chats
	v.cache = map[string]string{}
	v.applyFilter()
}

func (v *View) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(v.search.Buffer().Text()))
	v.filtered = v.filtered[:0]
	for _, c := range v.chats {
		if c.Archived != v.showArch {
			continue
		}
		switch v.mode {
		case FilterUnread:
			if c.Unread == 0 {
				continue
			}
		case FilterGroups:
			if c.Kind != bridge.ChatGroup {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) {
			continue
		}
		v.filtered = append(v.filtered, c)
	}
	v.list.RemoveAll()
	for _, c := range v.filtered {
		v.list.Append(v.chatRow(c))
	}
	v.list.InvalidateFilter()
}

// snippet caches the LastMessage preview per chat row.
func (v *View) snippet(c bridge.Chat) string {
	if s, ok := v.cache[c.JID.String()]; ok {
		return s
	}
	s := ""
	if v.preview != nil {
		s = v.preview(c)
	}
	v.cache[c.JID.String()] = s
	return s
}

// SetShowArchived toggles the archived folder view.
func (v *View) SetShowArchived(show bool) {
	v.showArch = show
	v.applyFilter()
}

// FocusSearch moves keyboard focus to the filter entry (Ctrl+F).
func (v *View) FocusSearch() { v.search.GrabFocus() }

// ClearSearch resets the filter (Esc).
func (v *View) ClearSearch() { v.search.Buffer().SetText("", -1) }

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

func (v *View) chatRow(c bridge.Chat) *gtk.Box {
	row := gtk.NewBox(gtk.OrientationHorizontal, 10)
	row.Append(widgets.AvatarSized(c.Title, 48))
	mid := gtk.NewBox(gtk.OrientationVertical, 2)
	mid.SetHExpand(true)
	title := gtk.NewLabel("")
	if c.Unread > 0 {
		title.SetMarkup(`<b>` + escape(c.Title) + `</b>`)
	} else {
		title.SetText(c.Title)
	}
	title.SetXAlign(0)
	title.SetEllipsize(pango.EllipsizeEnd)
	mid.Append(title)
	last := gtk.NewLabel(v.snippet(c))
	last.AddCSSClass("dim-label")
	last.SetXAlign(0)
	last.SetEllipsize(pango.EllipsizeEnd)
	mid.Append(last)
	row.Append(mid)
	right := gtk.NewBox(gtk.OrientationVertical, 2)
	ts := gtk.NewLabel(stamp(c.LastTs))
	ts.AddCSSClass("dim-label")
	right.Append(ts)
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
		m := gtk.NewLabel(marks)
		right.Append(m)
	}
	row.Append(right)
	return row
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func stamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	if now.Year() == t.Year() && now.YearDay() == t.YearDay() {
		return t.Format("15:04")
	}
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Mon")
	}
	return t.Format("02/01/06")
}
