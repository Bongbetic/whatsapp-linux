// Package convo renders one open conversation (WL-2.2/2.3): header
// with presence, message list, reply banner, and composer with
// Enter-to-send, typing debounce, and draft callbacks.
package convo

import (
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

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
	// NameFor resolves a sender JID to a display name (group bubbles).
	NameFor func(bridge.JID) string
}

// View is the conversation pane widget.
type View struct {
	Box       *gtk.Box
	back      *gtk.Button
	avatarBox *gtk.Box
	avatar    gtk.Widgetter
	name      *gtk.Label
	presence  *gtk.Label
	list      *gtk.ListBox
	composer  *gtk.TextView
	sendBtn   *gtk.Button
	micBtn    *gtk.Button
	emojiRev  *gtk.Revealer
	replyRev  *gtk.Revealer
	replyLbl  *gtk.Label
	findRev   *gtk.Revealer
	findEntry *gtk.Entry
	chat      bridge.JID
	kind      string
	hasChat   bool
	replyTo   string
	cb        Callbacks
	enterSend bool
	typingOn  bool
	recording bool
	rows      map[string]*gtk.Box
	msgs      []bridge.Message
	order     []string
	findIdx   int
}

// New builds the pane. enterSend follows settings enter-to-send.
func New(cb Callbacks, enterSend bool) *View {
	v := &View{cb: cb, enterSend: enterSend, rows: map[string]*gtk.Box{}}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)

	head := gtk.NewBox(gtk.OrientationHorizontal, 8)
	v.back = gtk.NewButtonFromIconName("go-previous-symbolic")
	v.back.SetTooltipText("Back to chats")
	v.back.SetVisible(false)
	v.back.ConnectClicked(func() { v.cb.OnBack() })
	head.Append(v.back)
	v.avatarBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	head.Append(v.avatarBox)
	nametag := gtk.NewBox(gtk.OrientationVertical, 0)
	nametag.SetHExpand(true)
	v.name = gtk.NewLabel("No chat selected")
	v.name.SetXAlign(0)
	nametag.Append(v.name)
	v.presence = gtk.NewLabel("")
	v.presence.AddCSSClass("dim-label")
	v.presence.SetXAlign(0)
	nametag.Append(v.presence)
	head.Append(nametag)
	audioBtn := gtk.NewButtonFromIconName("call-start-symbolic")
	audioBtn.SetTooltipText("Voice call")
	audioBtn.ConnectClicked(func() {
		if v.hasChat {
			v.cb.OnCall(v.chat, false)
		}
	})
	head.Append(audioBtn)
	videoBtn := gtk.NewButtonFromIconName("camera-video-symbolic")
	videoBtn.SetTooltipText("Video call")
	videoBtn.ConnectClicked(func() {
		if v.hasChat {
			v.cb.OnCall(v.chat, true)
		}
	})
	head.Append(videoBtn)
	v.Box.Append(head)

	v.replyRev, v.replyLbl = widgets.Banner()
	v.Box.Append(v.replyRev)

	// In-chat search (Ctrl+Shift+F): Enter jumps to the next match.
	v.findRev = gtk.NewRevealer()
	v.findEntry = gtk.NewEntry()
	v.findEntry.SetPlaceholderText("Find in conversation (Enter = next, Esc = close)")
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
	v.findRev.SetChild(v.findEntry)
	v.Box.Append(v.findRev)

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	v.list = gtk.NewListBox()
	v.list.AddCSSClass("chat-bg")
	scroll.SetChild(v.list)
	v.Box.Append(scroll)

	// Emoji picker drawer (revealer under the list).
	v.emojiRev = gtk.NewRevealer()
	emojiBox := emoji.Picker(func(e string) { v.InsertText(e) })
	v.emojiRev.SetChild(emojiBox)
	v.Box.Append(v.emojiRev)

	bar := gtk.NewBox(gtk.OrientationHorizontal, 6)
	attach := gtk.NewButtonFromIconName("mail-attachment-symbolic")
	attach.SetTooltipText("Attach")
	attach.ConnectClicked(func() {
		if v.hasChat {
			v.cb.OnAttach(v.chat)
		}
	})
	bar.Append(attach)

	emoBtn := gtk.NewButtonFromIconName("face-smile-symbolic")
	emoBtn.SetTooltipText("Emoji")
	emoBtn.ConnectClicked(func() {
		v.emojiRev.SetRevealChild(!v.emojiRev.RevealChild())
	})
	bar.Append(emoBtn)

	cscroll := gtk.NewScrolledWindow()
	cscroll.SetHExpand(true)
	v.composer = gtk.NewTextView()
	v.composer.SetWrapMode(gtk.WrapWordChar)
	v.composer.SetAcceptsTab(false)
	cscroll.SetChild(v.composer)
	bar.Append(cscroll)

	v.micBtn = gtk.NewButtonFromIconName("microphone-sensitivity-medium-symbolic")
	v.micBtn.SetTooltipText("Voice message")
	v.micBtn.AddCSSClass("round-btn")
	v.micBtn.ConnectClicked(func() { v.cb.OnMic() })
	bar.Append(v.micBtn)

	v.sendBtn = gtk.NewButtonFromIconName("mail-send-symbolic")
	v.sendBtn.SetTooltipText("Send")
	v.sendBtn.AddCSSClass("round-btn")
	v.sendBtn.AddCSSClass("send-btn")
	v.sendBtn.ConnectClicked(func() { v.submit() })
	bar.Append(v.sendBtn)
	v.swapMicSend()
	v.Box.Append(bar)

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

// OpenChat switches the pane to a chat with its recent messages,
// inserting date separators and one unread divider like the
// Windows client.
func (v *View) OpenChat(c bridge.Chat, recent []bridge.Message) {
	v.chat, v.kind, v.hasChat, v.replyTo = c.JID, c.Kind, true, ""
	v.name.SetText(c.Title)
	v.SetPresence("")
	if v.avatar != nil {
		v.avatarBox.Remove(v.avatar)
	}
	v.avatar = widgets.AvatarSized(c.Title, 40)
	v.avatarBox.Append(v.avatar)
	v.list.RemoveAll()
	v.rows = map[string]*gtk.Box{}
	v.msgs = v.msgs[:0]
	v.order = v.order[:0]
	v.findIdx = 0
	dividerAt := -1
	if c.Unread > 0 && c.Unread < len(recent) {
		dividerAt = len(recent) - c.Unread
	}
	lastDay := ""
	for i, m := range recent {
		if day := m.Ts.Format("2006-01-02"); day != lastDay {
			lastDay = day
			v.list.Append(widgets.CenterPill(dayLabel(m.Ts), "date-pill"))
		}
		if i == dividerAt {
			v.list.Append(widgets.CenterPill("Unread messages", "unread-divider"))
		}
		v.AddMessage(m)
	}
	v.composer.Buffer().SetText(c.Draft)
	v.swapMicSend()
	widgets.SetBanner(v.replyRev, v.replyLbl, "")
	v.ToggleSearch(false)
	v.emojiRev.SetRevealChild(false)
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

// AddMessage appends one message row, replacing any earlier row for
// the same id (state updates, edits).
func (v *View) AddMessage(m bridge.Message) {
	if old, ok := v.rows[m.ID]; ok {
		v.list.Remove(old)
	}
	if old, ok := v.rows[m.ID]; ok {
		v.list.Remove(old) // replaced in place below; order kept
		for i, em := range v.msgs {
			if em.ID == m.ID {
				v.msgs[i] = m
				break
			}
		}
	} else {
		v.order = append(v.order, m.ID)
		v.msgs = append(v.msgs, m)
	}
	row := widgets.MessageRowRich(m, v.senderName(m), v.kind == bridge.ChatGroup)
	v.rows[m.ID] = row
	v.list.Append(row)
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
	widgets.SetBanner(v.replyRev, v.replyLbl, "Replying: "+preview)
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
}

// SetMicRecording flips the mic button into its recording state.
func (v *View) SetMicRecording(on bool) {
	v.recording = on
	v.micBtn.SetVisible(on)
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
	widgets.SetBanner(v.replyRev, v.replyLbl, "")
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
