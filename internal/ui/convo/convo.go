// Package convo renders one open conversation (WL-2.2/2.3): header
// with presence, message list, reply banner, and composer with
// Enter-to-send, typing debounce, and draft callbacks.
package convo

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// Callbacks wires the view to the app shell.
type Callbacks struct {
	OnSend   func(chat bridge.JID, text, quotedID string)
	OnTyping func(chat bridge.JID, typing bool)
	OnDraft  func(chat bridge.JID, text string)
	OnAttach func(chat bridge.JID)
	OnCall   func(chat bridge.JID, video bool)
}

// View is the conversation pane widget.
type View struct {
	Box       *gtk.Box
	name      *gtk.Label
	presence  *gtk.Label
	list      *gtk.ListBox
	composer  *gtk.TextView
	replyRev  *gtk.Revealer
	replyLbl  *gtk.Label
	chat      bridge.JID
	hasChat   bool
	replyTo   string
	cb        Callbacks
	enterSend bool
	typingOn  bool
	rows      map[string]*gtk.Box
}

// New builds the pane. enterSend follows settings enter-to-send.
func New(cb Callbacks, enterSend bool) *View {
	v := &View{cb: cb, enterSend: enterSend, rows: map[string]*gtk.Box{}}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)

	head := gtk.NewBox(gtk.OrientationHorizontal, 8)
	v.name = gtk.NewLabel("No chat selected")
	v.name.SetHExpand(true)
	v.name.SetXAlign(0)
	head.Append(v.name)
	v.presence = gtk.NewLabel("")
	v.presence.AddCSSClass("dim-label")
	head.Append(v.presence)
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

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	v.list = gtk.NewListBox()
	scroll.SetChild(v.list)
	v.Box.Append(scroll)

	bar := gtk.NewBox(gtk.OrientationHorizontal, 6)
	attach := gtk.NewButtonFromIconName("mail-attachment-symbolic")
	attach.SetTooltipText("Attach")
	attach.ConnectClicked(func() {
		if v.hasChat {
			v.cb.OnAttach(v.chat)
		}
	})
	bar.Append(attach)

	cscroll := gtk.NewScrolledWindow()
	cscroll.SetHExpand(true)
	v.composer = gtk.NewTextView()
	v.composer.SetWrapMode(gtk.WrapWordChar)
	v.composer.SetAcceptsTab(false)
	cscroll.SetChild(v.composer)
	bar.Append(cscroll)

	send := gtk.NewButtonFromIconName("mail-send-symbolic")
	send.SetTooltipText("Send")
	send.ConnectClicked(func() { v.submit() })
	bar.Append(send)
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

// OpenChat switches the pane to a chat with its recent messages.
func (v *View) OpenChat(c bridge.Chat, recent []bridge.Message) {
	v.chat, v.hasChat, v.replyTo = c.JID, true, ""
	v.name.SetText(c.Title)
	v.SetPresence("")
	v.list.RemoveAll()
	v.rows = map[string]*gtk.Box{}
	for _, m := range recent {
		v.AddMessage(m)
	}
	v.composer.Buffer().SetText(c.Draft)
	widgets.SetBanner(v.replyRev, v.replyLbl, "")
}

// AddMessage appends one message row, replacing any earlier row for
// the same id (state updates, edits).
func (v *View) AddMessage(m bridge.Message) {
	if old, ok := v.rows[m.ID]; ok {
		v.list.Remove(old)
	}
	row := widgets.MessageRow(m)
	v.rows[m.ID] = row
	v.list.Append(row)
}

// SetPresence shows presence/typing state in the header (WL-2.8).
func (v *View) SetPresence(s string) { v.presence.SetText(s) }

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
