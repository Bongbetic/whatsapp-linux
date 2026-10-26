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
	Box       *gtk.Box
	list      *gtk.ListBox
	search    *gtk.Entry
	chats     []bridge.Chat
	filtered  []bridge.Chat
	onOpen    func(bridge.Chat)
	onNew     func()
	preview   func(bridge.Chat) string
	cache     map[string]string
	showArch  bool
	mode      string
	pills     map[string]*gtk.Button
	pillRow   *gtk.Box
	back      *gtk.Button
	title     *gtk.Label
	archBtn   *gtk.Button
	archCnt   int
	archCount *gtk.Label
	selected  string // JID of the open chat, kept highlighted across refreshes
}

// New builds the pane. onOpen fires on Enter/click, onNew on the
// compose button; preview resolves one row's snippet (may hit the
// store — results are cached per SetChats).
func New(onOpen func(bridge.Chat), onNew func()) *View {
	v := &View{onOpen: onOpen, onNew: onNew, mode: FilterAll, pills: map[string]*gtk.Button{}, cache: map[string]string{}}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	v.Box.AddCSSClass("side-panel")

	head := gtk.NewBox(gtk.OrientationHorizontal, 4)
	head.AddCSSClass("pane-header")
	v.back = gtk.NewButtonFromIconName("wl-back-symbolic")
	v.back.AddCSSClass("icon-btn")
	v.back.SetTooltipText("Back to chats")
	v.back.SetVisible(false)
	v.back.ConnectClicked(func() { v.SetShowArchived(false) })
	head.Append(v.back)
	v.title = gtk.NewLabel("Chats")
	v.title.AddCSSClass("pane-title")
	v.title.SetHExpand(true)
	v.title.SetXAlign(0)
	head.Append(v.title)
	newBtn := gtk.NewButtonFromIconName("wl-new-chat-symbolic")
	newBtn.AddCSSClass("icon-btn")
	newBtn.SetTooltipText("New chat (Ctrl+N)")
	newBtn.ConnectClicked(func() { v.onNew() })
	head.Append(newBtn)
	v.Box.Append(head)

	v.search = gtk.NewEntry()
	v.search.AddCSSClass("search-pill")
	v.search.SetPlaceholderText("Search chats")
	v.search.SetIconFromIconName(gtk.EntryIconPrimary, "wl-search-symbolic")
	buf := v.search.Buffer()
	buf.ConnectInsertedText(func(uint, string, uint) { v.applyFilter() })
	buf.ConnectDeletedText(func(uint, uint) { v.applyFilter() })
	v.Box.Append(v.search)

	v.pillRow = gtk.NewBox(gtk.OrientationHorizontal, 8)
	v.pillRow.AddCSSClass("filter-row")
	for _, m := range []struct{ mode, label string }{
		{FilterAll, "All"}, {FilterUnread, "Unread"}, {FilterGroups, "Groups"},
	} {
		m := m
		b := gtk.NewButtonWithLabel(m.label)
		b.AddCSSClass("filter-pill")
		b.ConnectClicked(func() { v.SetMode(m.mode) })
		v.pills[m.mode] = b
		v.pillRow.Append(b)
	}
	v.Box.Append(v.pillRow)
	v.paintPills()

	// "Archived" entry above the list, like the reference client.
	v.archBtn = gtk.NewButton()
	v.archBtn.AddCSSClass("drawer-action")
	v.archBtn.AddCSSClass("archived-row")
	arch := gtk.NewBox(gtk.OrientationHorizontal, 22)
	archIcon := gtk.NewImageFromIconName("wl-archive-symbolic")
	archIcon.AddCSSClass("chat-row-mark")
	arch.Append(archIcon)
	archLbl := gtk.NewLabel("Archived")
	archLbl.SetHExpand(true)
	archLbl.SetXAlign(0)
	arch.Append(archLbl)
	v.archCount = gtk.NewLabel("")
	v.archCount.AddCSSClass("chat-row-time")
	arch.Append(v.archCount)
	v.archBtn.SetChild(arch)
	v.archBtn.ConnectClicked(func() { v.SetShowArchived(true) })
	v.archBtn.SetVisible(false)
	v.Box.Append(v.archBtn)

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.list = gtk.NewListBox()
	v.list.AddCSSClass("chat-list")
	v.list.SetFilterFunc(func(row *gtk.ListBoxRow) bool {
		i := row.Index()
		return i >= 0 && i < len(v.filtered)
	})
	v.list.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		if i := row.Index(); i >= 0 && i < len(v.filtered) {
			v.selected = v.filtered[i].JID.String()
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
	for i, c := range v.filtered {
		v.list.Append(v.chatRow(c))
		if c.JID.String() == v.selected {
			v.list.SelectRow(v.list.RowAtIndex(i))
		}
	}
	v.list.InvalidateFilter()
	v.archCnt = 0
	for _, c := range v.chats {
		if c.Archived {
			v.archCnt++
		}
	}
	v.archCount.SetText(fmt.Sprint(v.archCnt))
	v.archBtn.SetVisible(!v.showArch && v.archCnt > 0 && q == "")
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
	v.back.SetVisible(show)
	v.pillRow.SetVisible(!show)
	if show {
		v.title.SetText("Archived")
	} else {
		v.title.SetText("Chats")
	}
	v.applyFilter()
}

// SetSelected highlights the open chat's row (empty clears).
func (v *View) SetSelected(jid string) {
	v.selected = jid
	v.list.UnselectAll()
	for i, c := range v.filtered {
		if c.JID.String() == jid {
			v.list.SelectRow(v.list.RowAtIndex(i))
		}
	}
}

// UnreadChats counts unarchived chats with unread messages (rail badge).
func (v *View) UnreadChats() int {
	n := 0
	for _, c := range v.chats {
		if !c.Archived && c.Unread > 0 {
			n++
		}
	}
	return n
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
	row := gtk.NewBox(gtk.OrientationHorizontal, 14)
	row.Append(widgets.AvatarFor(c.Title, c.Kind == bridge.ChatGroup, 49))
	mid := gtk.NewBox(gtk.OrientationVertical, 3)
	mid.SetHExpand(true)
	mid.SetVAlign(gtk.AlignCenter)

	top := gtk.NewBox(gtk.OrientationHorizontal, 6)
	title := gtk.NewLabel(c.Title)
	title.AddCSSClass("chat-row-name")
	title.SetHExpand(true)
	title.SetXAlign(0)
	title.SetEllipsize(pango.EllipsizeEnd)
	top.Append(title)
	ts := gtk.NewLabel(stamp(c.LastTs))
	ts.AddCSSClass("chat-row-time")
	if c.Unread > 0 {
		ts.AddCSSClass("unread")
	}
	top.Append(ts)
	mid.Append(top)

	bottom := gtk.NewBox(gtk.OrientationHorizontal, 6)
	last := gtk.NewLabel(v.snippet(c))
	last.AddCSSClass("chat-row-preview")
	last.SetHExpand(true)
	last.SetXAlign(0)
	last.SetEllipsize(pango.EllipsizeEnd)
	bottom.Append(last)
	if c.MutedUntil != 0 {
		bottom.Append(mark("wl-muted-symbolic", "Muted"))
	}
	if c.Unread > 0 {
		pill := gtk.NewLabel(fmt.Sprint(c.Unread))
		pill.AddCSSClass("unread-pill")
		pill.SetVAlign(gtk.AlignCenter)
		bottom.Append(pill)
	}
	if c.Pinned {
		bottom.Append(mark("wl-pin-symbolic", "Pinned"))
	}
	mid.Append(bottom)
	row.Append(mid)
	return row
}

// mark is a small status icon at the end of a row (pinned, muted).
func mark(icon, tip string) *gtk.Image {
	img := gtk.NewImageFromIconName(icon)
	img.AddCSSClass("chat-row-mark")
	img.SetTooltipText(tip)
	return img
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
