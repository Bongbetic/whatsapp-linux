// Package convo renders one open conversation (WL-2.2/2.3): header
// with presence, message list, reply banner, and composer with
// Enter-to-send, typing debounce, and draft callbacks.
package convo

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/emoji"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// Callbacks wires the view to the app shell.
type Callbacks struct {
	OnSend   func(chat bridge.JID, text, quotedID string)
	OnTyping func(chat bridge.JID, typing bool)
	OnDraft  func(chat bridge.JID, text string)
	OnAttach func(chat bridge.JID)
	OnCall   func(chat bridge.JID, video bool)
	OnBack   func() // narrow single-pane: return to the chat list
	OnMic    func() // mic button (toggles voice recording in the shell)
	OnInfo   func() // header click: toggle the contact-info drawer
	// NameFor resolves a sender JID to a display name (group bubbles).
	NameFor func(bridge.JID) string
}

// View is the conversation pane widget.
type View struct {
	Box         *gtk.Box
	stack       *gtk.Stack
	back        *gtk.Button
	avatarBox   *gtk.Box
	avatar      gtk.Widgetter
	name        *gtk.Label
	presence    *gtk.Label
	list        *gtk.ListBox
	scroll      *gtk.ScrolledWindow
	stick       bool // keep the list scrolled to the newest message
	stickQueued bool // a scroll-to-bottom is pending on idle
	composer    *gtk.TextView
	placeholder *gtk.Label
	sendBtn     *gtk.Button
	micBtn      *gtk.Button
	emojiRev    *gtk.Revealer
	replyRev    *gtk.Revealer
	replyLbl    *gtk.Label
	findRev     *gtk.Revealer
	findEntry   *gtk.Entry
	chat        bridge.JID
	kind        string
	hasChat     bool
	replyTo     string
	cb          Callbacks
	enterSend   bool
	typingOn    bool
	recording   bool
	rows        map[string]*gtk.Box
	first       map[string]bool // message id → starts a sender run
	lastRun     string          // run key of the last appended message
	msgs        []bridge.Message
	order       []string
	findIdx     int
}

// New builds the pane. enterSend follows settings enter-to-send.
func New(cb Callbacks, enterSend bool) *View {
	v := &View{cb: cb, enterSend: enterSend, rows: map[string]*gtk.Box{}, first: map[string]bool{}}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	v.Box.AddCSSClass("convo")
	v.stack = gtk.NewStack()
	v.stack.SetVExpand(true)
	v.Box.Append(v.stack)
	v.stack.AddNamed(intro(), "intro")
	page := gtk.NewBox(gtk.OrientationVertical, 0)
	v.stack.AddNamed(page, "chat")

	head := gtk.NewBox(gtk.OrientationHorizontal, 4)
	head.AddCSSClass("convo-header")
	v.back = iconButton("wl-back-symbolic", "Back to chats", func() { v.cb.OnBack() })
	v.back.SetVisible(false)
	head.Append(v.back)
	who := gtk.NewBox(gtk.OrientationHorizontal, 12)
	who.SetHExpand(true)
	v.avatarBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	who.Append(v.avatarBox)
	nametag := gtk.NewBox(gtk.OrientationVertical, 1)
	nametag.SetVAlign(gtk.AlignCenter)
	v.name = gtk.NewLabel("")
	v.name.AddCSSClass("convo-name")
	v.name.SetXAlign(0)
	v.name.SetEllipsize(pango.EllipsizeEnd)
	nametag.Append(v.name)
	v.presence = gtk.NewLabel("")
	v.presence.AddCSSClass("convo-presence")
	v.presence.SetXAlign(0)
	v.presence.SetEllipsize(pango.EllipsizeEnd)
	nametag.Append(v.presence)
	who.Append(nametag)
	info := gtk.NewGestureClick()
	info.ConnectReleased(func(int, float64, float64) {
		if v.hasChat && v.cb.OnInfo != nil {
			v.cb.OnInfo()
		}
	})
	who.AddController(info)
	who.SetCursorFromName("pointer")
	who.SetTooltipText("Contact info")
	head.Append(who)
	head.Append(iconButton("wl-video-symbolic", "Video call", func() {
		if v.hasChat {
			v.cb.OnCall(v.chat, true)
		}
	}))
	head.Append(iconButton("wl-calls-symbolic", "Voice call", func() {
		if v.hasChat {
			v.cb.OnCall(v.chat, false)
		}
	}))
	head.Append(iconButton("wl-search-symbolic", "Search (Ctrl+Shift+F)", func() {
		v.ToggleSearch(!v.findRev.RevealChild())
	}))
	head.Append(iconButton("wl-menu-symbolic", "Contact info", func() {
		if v.hasChat && v.cb.OnInfo != nil {
			v.cb.OnInfo()
		}
	}))
	page.Append(head)

	// In-chat search (Ctrl+Shift+F): Enter jumps to the next match.
	v.findRev = gtk.NewRevealer()
	findBar := gtk.NewBox(gtk.OrientationHorizontal, 6)
	findBar.AddCSSClass("find-bar")
	v.findEntry = gtk.NewEntry()
	v.findEntry.SetHExpand(true)
	v.findEntry.SetPlaceholderText("Search messages (Enter = next, Esc = close)")
	v.findEntry.SetIconFromIconName(gtk.EntryIconPrimary, "wl-search-symbolic")
	v.findEntry.ConnectActivate(func() { v.findNext() })
	findKeys := gtk.NewEventControllerKey()
	findKeys.ConnectKeyPressed(func(keyval, _ uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			v.ToggleSearch(false)
			return true
		}
		return false
	})
	v.findEntry.AddController(findKeys)
	findBar.Append(v.findEntry)
	findBar.Append(iconButton("wl-close-symbolic", "Close search", func() { v.ToggleSearch(false) }))
	v.findRev.SetChild(findBar)
	page.Append(v.findRev)

	v.scroll = gtk.NewScrolledWindow()
	v.scroll.SetVExpand(true)
	v.scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.list = gtk.NewListBox()
	v.list.AddCSSClass("chat-bg")
	v.scroll.SetChild(v.list)
	// Open at the newest message and follow new ones while the user
	// is at the bottom, like the reference client.
	adj := v.scroll.VAdjustment()
	// The jump runs on idle: a value set inside the viewport's own
	// adjustment update is overwritten when that update finishes.
	adj.ConnectChanged(func() {
		if !v.stick || v.stickQueued {
			return
		}
		v.stickQueued = true
		glib.IdleAdd(func() {
			v.stickQueued = false
			if v.stick {
				adj.SetValue(adj.Upper() - adj.PageSize())
			}
		})
	})
	adj.ConnectValueChanged(func() {
		v.stick = adj.Value() >= adj.Upper()-adj.PageSize()-48
	})
	page.Append(v.scroll)

	v.replyRev = gtk.NewRevealer()
	v.replyLbl = gtk.NewLabel("")
	v.replyLbl.AddCSSClass("reply-bar")
	v.replyLbl.SetXAlign(0)
	v.replyLbl.SetEllipsize(pango.EllipsizeEnd)
	v.replyRev.SetChild(v.replyLbl)
	page.Append(v.replyRev)

	// Emoji picker drawer (revealer under the list).
	v.emojiRev = gtk.NewRevealer()
	emojiBox := emoji.Picker(func(e string) { v.InsertText(e) })
	emojiBox.AddCSSClass("emoji-drawer")
	v.emojiRev.SetChild(emojiBox)
	page.Append(v.emojiRev)

	tray := gtk.NewBox(gtk.OrientationHorizontal, 0)
	tray.AddCSSClass("composer-tray")
	bar := gtk.NewBox(gtk.OrientationHorizontal, 2)
	bar.AddCSSClass("composer-pill")
	bar.SetHExpand(true)
	tray.Append(bar)
	attach := iconButton("wl-plus-symbolic", "Attach", func() {
		if v.hasChat {
			v.cb.OnAttach(v.chat)
		}
	})
	attach.SetVAlign(gtk.AlignEnd)
	bar.Append(attach)
	emoBtn := iconButton("wl-emoji-symbolic", "Emoji", func() {
		v.emojiRev.SetRevealChild(!v.emojiRev.RevealChild())
	})
	emoBtn.SetVAlign(gtk.AlignEnd)
	bar.Append(emoBtn)

	cscroll := gtk.NewScrolledWindow()
	cscroll.SetHExpand(true)
	cscroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	cscroll.SetPropagateNaturalHeight(true)
	cscroll.SetMaxContentHeight(140)
	v.composer = gtk.NewTextView()
	v.composer.SetWrapMode(gtk.WrapWordChar)
	v.composer.SetAcceptsTab(false)
	v.composer.SetTopMargin(10)
	v.composer.SetBottomMargin(10)
	v.composer.SetLeftMargin(6)
	v.composer.SetRightMargin(6)
	cscroll.SetChild(v.composer)
	field := gtk.NewOverlay()
	field.SetChild(cscroll)
	field.SetHExpand(true)
	v.placeholder = gtk.NewLabel("Type a message")
	v.placeholder.AddCSSClass("composer-placeholder")
	v.placeholder.SetHAlign(gtk.AlignStart)
	v.placeholder.SetVAlign(gtk.AlignCenter)
	v.placeholder.SetMarginStart(8)
	v.placeholder.SetCanTarget(false)
	field.AddOverlay(v.placeholder)
	bar.Append(field)

	v.micBtn = iconButton("wl-mic-symbolic", "Voice message", func() { v.cb.OnMic() })
	v.micBtn.SetVAlign(gtk.AlignEnd)
	bar.Append(v.micBtn)

	v.sendBtn = gtk.NewButtonFromIconName("wl-send-symbolic")
	v.sendBtn.SetTooltipText("Send")
	v.sendBtn.AddCSSClass("send-btn")
	v.sendBtn.SetVAlign(gtk.AlignEnd)
	v.sendBtn.ConnectClicked(func() { v.submit() })
	bar.Append(v.sendBtn)
	v.swapMicSend()
	page.Append(tray)

	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(keyval, _ uint, state gdk.ModifierType) bool {
		if keyval != gdk.KEY_Return && keyval != gdk.KEY_KP_Enter {
			return false
		}
		if state&gdk.ShiftMask != 0 || !v.enterSend {
			return false // newline
		}
		v.submit()
		return true
	})
	v.composer.AddController(keys)

	v.composer.Buffer().ConnectChanged(func() { v.onComposerChanged() })
	return v
}

// iconButton is a flat round header/composer button.
func iconButton(icon, tip string, onClick func()) *gtk.Button {
	b := gtk.NewButtonFromIconName(icon)
	b.AddCSSClass("icon-btn")
	b.SetTooltipText(tip)
	b.SetVAlign(gtk.AlignCenter)
	b.ConnectClicked(onClick)
	return b
}

// intro is the empty-pane placeholder shown until a chat is opened.
func intro() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	box.AddCSSClass("intro-panel")
	mid := gtk.NewBox(gtk.OrientationVertical, 8)
	mid.SetVExpand(true)
	mid.SetVAlign(gtk.AlignCenter)
	icon := gtk.NewImageFromIconName("wl-chats-symbolic")
	icon.AddCSSClass("intro-icon")
	mid.Append(icon)
	title := gtk.NewLabel("WhatsApp for Linux")
	title.AddCSSClass("intro-title")
	mid.Append(title)
	text := gtk.NewLabel("Send and receive messages without keeping your phone online.\nUse WhatsApp on up to 4 linked devices and 1 phone at the same time.")
	text.AddCSSClass("intro-text")
	text.SetJustify(gtk.JustifyCenter)
	mid.Append(text)
	box.Append(mid)
	foot := gtk.NewBox(gtk.OrientationHorizontal, 6)
	foot.SetHAlign(gtk.AlignCenter)
	foot.AddCSSClass("intro-foot")
	lock := gtk.NewImageFromIconName("wl-lock-symbolic")
	foot.Append(lock)
	foot.Append(gtk.NewLabel("Your personal messages are end-to-end encrypted"))
	box.Append(foot)
	return box
}

// OpenChat switches the pane to a chat with its recent messages,
// inserting date separators and one unread divider like the
// Windows client.
func (v *View) OpenChat(c bridge.Chat, recent []bridge.Message) {
	v.chat, v.kind, v.hasChat, v.replyTo = c.JID, c.Kind, true, ""
	v.stack.SetVisibleChildName("chat")
	v.name.SetText(c.Title)
	v.SetPresence("")
	if v.avatar != nil {
		v.avatarBox.Remove(v.avatar)
	}
	v.avatar = widgets.AvatarFor(c.Title, c.Kind == bridge.ChatGroup, 40)
	v.avatarBox.Append(v.avatar)
	v.list.RemoveAll()
	v.rows = map[string]*gtk.Box{}
	v.first = map[string]bool{}
	v.lastRun = ""
	v.msgs = v.msgs[:0]
	v.order = v.order[:0]
	v.findIdx = 0
	v.stick = true
	v.list.Append(widgets.CenterPill("Messages are end-to-end encrypted. No one outside of this chat can read or listen to them.", "e2e-note"))
	dividerAt := -1
	if c.Unread > 0 && c.Unread < len(recent) {
		dividerAt = len(recent) - c.Unread
	}
	lastDay := ""
	for i, m := range recent {
		if day := m.Ts.Format("2006-01-02"); day != lastDay {
			lastDay = day
			v.list.Append(widgets.CenterPill(dayLabel(m.Ts), "date-pill"))
			v.lastRun = ""
		}
		if i == dividerAt {
			v.list.Append(widgets.CenterPill(unreadLabel(c.Unread), "unread-divider"))
			v.lastRun = ""
		}
		v.AddMessage(m)
	}
	v.composer.Buffer().SetText(c.Draft)
	v.swapMicSend()
	v.replyRev.SetRevealChild(false)
	v.ToggleSearch(false)
	v.emojiRev.SetRevealChild(false)
}

// unreadLabel is the divider text ("1 unread message", "3 unread messages").
func unreadLabel(n int) string {
	if n == 1 {
		return "1 unread message"
	}
	return fmt.Sprintf("%d unread messages", n)
}

// dayLabel renders Today / Yesterday / date like the reference.
func dayLabel(t time.Time) string {
	now := time.Now()
	y, m, d := t.Date()
	ny, nm, nd := now.Date()
	if y == ny && m == nm && d == nd {
		return "Today"
	}
	yest := now.Add(-24 * time.Hour)
	if yy, ym, yd := yest.Date(); y == yy && m == ym && d == yd {
		return "Yesterday"
	}
	return t.Format("02/01/2006")
}

// AddMessage appends one message row; a message already shown
// (state updates, edits) is replaced at its existing position.
func (v *View) AddMessage(m bridge.Message) {
	if old, ok := v.rows[m.ID]; ok {
		pos := -1
		if r, ok := old.Parent().(*gtk.ListBoxRow); ok {
			pos = r.Index()
		}
		v.list.Remove(old)
		for i, em := range v.msgs {
			if em.ID == m.ID {
				v.msgs[i] = m
				break
			}
		}
		row := widgets.MessageRowRich(m, v.senderName(m), v.kind == bridge.ChatGroup, v.first[m.ID])
		v.rows[m.ID] = row
		v.list.Insert(row, pos)
		return
	}
	run := runKey(m)
	first := run != v.lastRun
	v.lastRun = run
	v.first[m.ID] = first
	v.order = append(v.order, m.ID)
	v.msgs = append(v.msgs, m)
	row := widgets.MessageRowRich(m, v.senderName(m), v.kind == bridge.ChatGroup, first)
	v.rows[m.ID] = row
	v.list.Append(row)
}

// runKey groups consecutive messages from one sender on one day.
func runKey(m bridge.Message) string {
	if m.IsFromMe {
		return "me|" + m.Ts.Format("2006-01-02")
	}
	return m.Sender.String() + "|" + m.Ts.Format("2006-01-02")
}

// senderName resolves a group bubble's display name via NameFor.
func (v *View) senderName(m bridge.Message) string {
	if m.IsFromMe || v.cb.NameFor == nil {
		return ""
	}
	if n := v.cb.NameFor(m.Sender); n != "" {
		return n
	}
	return m.Sender.User
}

// ToggleSearch shows (show=true) or hides the in-chat finder.
func (v *View) ToggleSearch(show bool) {
	v.findRev.SetRevealChild(show)
	if show {
		v.findIdx = 0
		v.findEntry.GrabFocus()
	} else {
		v.findEntry.SetText("")
		v.composer.GrabFocus()
	}
}

// findNext selects the next row whose body matches the query.
func (v *View) findNext() {
	q := strings.ToLower(strings.TrimSpace(v.findEntry.Buffer().Text()))
	if q == "" || len(v.msgs) == 0 {
		return
	}
	for n := 0; n < len(v.msgs); n++ {
		v.findIdx = (v.findIdx + 1) % len(v.msgs)
		if !strings.Contains(strings.ToLower(v.msgs[v.findIdx].Body), q) {
			continue
		}
		if r := v.list.RowAtIndex(v.rowIndex(v.msgs[v.findIdx].ID)); r != nil {
			v.list.SelectRow(r)
			r.GrabFocus()
		}
		return
	}
}

// rowIndex maps a message id to its position in the list.
func (v *View) rowIndex(id string) int {
	for i, oid := range v.order {
		if oid == id {
			return i
		}
	}
	return 0
}

// SetPresence shows presence/typing state in the header (WL-2.8).
func (v *View) SetPresence(s string) { v.presence.SetText(s) }

// SetBackVisible toggles the narrow-mode back button (§9).
func (v *View) SetBackVisible(show bool) { v.back.SetVisible(show) }

// ReplyTo arms the reply banner for a message id.
func (v *View) ReplyTo(id, preview string) {
	v.replyTo = id
	v.replyLbl.SetText(preview)
	v.replyRev.SetRevealChild(true)
}

// Draft returns the current composer text.
func (v *View) Draft() string {
	b := v.composer.Buffer()
	return b.Text(b.StartIter(), b.EndIter(), false)
}

// InsertText inserts s at the composer cursor (emoji picker).
func (v *View) InsertText(s string) {
	if s == "" {
		return
	}
	buf := v.composer.Buffer()
	iter := buf.IterAtMark(buf.GetInsert())
	buf.Insert(iter, s)
	v.composer.GrabFocus()
}

// swapMicSend shows the mic when the composer is empty, Send after
// the first character — like the Windows client.
func (v *View) swapMicSend() {
	empty := strings.TrimSpace(v.Draft()) == ""
	v.micBtn.SetVisible(empty && !v.recording)
	v.sendBtn.SetVisible(!empty && !v.recording)
	v.placeholder.SetVisible(v.Draft() == "")
}

// SetMicRecording flips the mic button into its recording state.
func (v *View) SetMicRecording(on bool) {
	v.recording = on
	v.micBtn.SetVisible(on)
	if on {
		v.micBtn.AddCSSClass("recording")
	} else {
		v.micBtn.RemoveCSSClass("recording")
	}
	if on {
		v.sendBtn.SetVisible(false)
		v.composer.SetEditable(false)
	} else {
		v.composer.SetEditable(true)
		v.swapMicSend()
	}
}

func (v *View) submit() {
	if !v.hasChat {
		return
	}
	text := strings.TrimSpace(v.Draft())
	if text == "" {
		return
	}
	v.cb.OnSend(v.chat, text, v.replyTo)
	v.replyTo = ""
	v.replyRev.SetRevealChild(false)
	v.composer.Buffer().SetText("")
	v.setTyping(false)
}

func (v *View) onComposerChanged() {
	v.swapMicSend()
	if !v.hasChat {
		return
	}
	empty := strings.TrimSpace(v.Draft()) == ""
	v.setTyping(!empty)
	// Draft persistence: 500 ms debounce (§9).
	glib.TimeoutAdd(500, func() bool {
		v.cb.OnDraft(v.chat, v.Draft())
		return false
	})
}

func (v *View) setTyping(on bool) {
	if v.typingOn == on || !v.hasChat {
		return
	}
	v.typingOn = on
	v.cb.OnTyping(v.chat, on)
	// Auto-clear after 3 s without edits (§9 typing debounce).
	if on {
		glib.TimeoutAdd(3000, func() bool {
			if v.typingOn {
				v.setTyping(false)
			}
			return false
		})
	}
}
