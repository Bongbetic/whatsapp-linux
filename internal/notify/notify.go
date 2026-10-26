// Package notify posts message notifications (WL-2.5): GNotification
// with Reply / Mark-read actions, unread badge accounting, and the
// tray contract. GNOME has no tray host for verification in CI, so
// the StatusNotifier tray ships as a NoopTray behind the Tray
// interface (real SNI lands with a verification host); close-to-tray
// keeps the process alive regardless.
package notify

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/rs/zerolog"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/logx"
	"github.com/Bongbetic/whatsapp-linux/internal/settings"
)

// Tray is the StatusNotifier contract (WL-2.5 follow-up).
type Tray interface {
	SetUnread(n int)
	SetConnected(online bool)
}

// NoopTray satisfies Tray where no tray host is verified.
type NoopTray struct{}

func (NoopTray) SetUnread(int)     {}
func (NoopTray) SetConnected(bool) {}

// Notifier posts GNotifications for inbound messages.
type Notifier struct {
	app      *gtk.Application
	settings *settings.Store
	log      zerolog.Logger
	tray     Tray
	unread   int
	// lastChat is the most recent notification's chat. Gio delivers
	// inline-reply text as the action parameter, so the reply target
	// must be tracked here; replies go to the newest notification.
	lastChat bridge.Chat
}

// New wires Reply / Mark-read actions onto the application.
func New(app *gtk.Application, s *settings.Store, log zerolog.Logger, tray Tray, onReply func(chatJID, text string), onMarkRead func(chatJID string), onOpen func(chatJID string)) *Notifier {
	n := &Notifier{app: app, settings: s, log: log, tray: tray}
	reply := gio.NewSimpleAction("reply", glib.NewVariantType("s"))
	reply.ConnectActivate(func(p *glib.Variant) {
		// Gio substitutes the typed reply text for the parameter,
		// dropping the button target: route to the newest chat.
		if n.lastChat.JID.String() != "@" {
			onReply(n.lastChat.JID.String(), p.String())
		}
	})
	n.app.AddAction(reply)
	mark := gio.NewSimpleAction("mark-read", glib.NewVariantType("s"))
	mark.ConnectActivate(func(p *glib.Variant) {
		onMarkRead(p.String())
	})
	n.app.AddAction(mark)
	open := gio.NewSimpleAction("open-chat", glib.NewVariantType("s"))
	open.ConnectActivate(func(p *glib.Variant) {
		onOpen(p.String())
	})
	n.app.AddAction(open)
	return n
}

// MessageArrived notifies once per inbound message (preview + sound
// per settings; D-9 scrubbed logging only).
func (n *Notifier) MessageArrived(chat bridge.Chat, m bridge.Message) {
	n.unread++
	n.lastChat = chat
	n.tray.SetUnread(n.unread)
	title := chat.Title
	if title == "" {
		title = logx.Scrub(chat.JID.String())
	}
	body := bodyPreview(m, n.settings.GetBool(settings.KeyNotifyPreview))
	note := gio.NewNotification(title)
	note.SetBody(body)
	note.SetCategory("im.received")
	jid := glib.NewVariantString(chat.JID.String())
	note.AddButtonWithTarget("Reply", "app.reply", jid)
	note.AddButtonWithTarget("Mark as read", "app.mark-read", jid)
	note.SetDefaultActionAndTarget("app.open-chat", jid)
	n.app.SendNotification(fmt.Sprintf("msg-%s-%s", chat.JID, m.ID), note)
	if n.settings.GetBool(settings.KeyNotifySound) {
		playHint()
	}
}

// Clear resets the unread count (window focused / chat opened).
func (n *Notifier) Clear() {
	n.unread = 0
	n.tray.SetUnread(0)
}

// SetConnected mirrors link state to the tray (dim when offline).
func (n *Notifier) SetConnected(online bool) { n.tray.SetConnected(online) }

// bodyPreview honors notify-preview; never leaks text when off.
func bodyPreview(m bridge.Message, preview bool) string {
	if !preview {
		return "New message"
	}
	switch m.Kind {
	case bridge.MsgText, bridge.MsgEdited:
		return m.Body
	case bridge.MsgImage:
		return "📷 " + m.Body
	case bridge.MsgVideo:
		return "🎬 " + m.Body
	case bridge.MsgVoice, bridge.MsgAudio:
		return "🎙 Voice message"
	case bridge.MsgDoc:
		return "📄 Document"
	case bridge.MsgSticker:
		return "Sticker"
	default:
		return "New message"
	}
}
