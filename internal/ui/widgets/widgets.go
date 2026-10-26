// Package widgets holds small shared GTK widgets (§9 ui/widgets).
// Note: chat/conversation lists use Gtk.ListBox, not Gtk.ListView:
// ListView needs GObject models for every row, which buys nothing
// at v1 row counts; revisit if profiling demands it.
package widgets

import (
	"math"
	"strings"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Initials returns up to 2 display initials for a chat title.
func Initials(title string) string {
	var sb strings.Builder
	for _, w := range strings.Fields(title) {
		for _, r := range w {
			sb.WriteRune(r)
			break
		}
		if sb.Len() >= 4 {
			break
		}
	}
	s := sb.String()
	if s == "" {
		return "?"
	}
	return s
}

// Avatar is a 40px placeholder avatar (photo support at WL-2.8).
func Avatar(title string) *gtk.DrawingArea { return AvatarSized(title, 40) }

// AvatarSized is a person placeholder avatar at an explicit diameter.
func AvatarSized(title string, px int) *gtk.DrawingArea { return AvatarFor(title, false, px) }

// AvatarFor draws the reference client's default avatar: a grey disc
// with a person (or, for groups, two people) silhouette. Colors come
// from the .avatar CSS class (background-color, color), so light and
// dark themes both apply.
func AvatarFor(title string, group bool, px int) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetContentWidth(px)
	area.SetContentHeight(px)
	area.SetHAlign(gtk.AlignCenter)
	area.SetVAlign(gtk.AlignCenter)
	area.AddCSSClass("avatar")
	area.SetTooltipText(title)
	area.SetDrawFunc(func(da *gtk.DrawingArea, cr *cairo.Context, w, h int) {
		fw, fh := float64(w), float64(h)
		cr.Arc(fw/2, fh/2, fw/2, 0, 2*math.Pi)
		cr.Clip()
		c := da.Color()
		cr.SetSourceRGBA(float64(c.Red()), float64(c.Green()), float64(c.Blue()), float64(c.Alpha()))
		person := func(cx, headY, s float64) {
			cr.Arc(cx, headY, s*0.19, 0, 2*math.Pi)
			cr.Fill()
			cr.Arc(cx, headY+s*0.62, s*0.38, 0, 2*math.Pi)
			cr.Fill()
		}
		if group {
			person(fw*0.64, fh*0.36, fw*0.8)
			person(fw*0.38, fh*0.42, fw*0.88)
			return
		}
		person(fw/2, fh*0.38, fw)
	})
	return area
}

// Banner is a revealer row for offline/ephemeral notices (§9).
func Banner() (*gtk.Revealer, *gtk.Label) {
	rev := gtk.NewRevealer()
	label := gtk.NewLabel("")
	label.SetXAlign(0)
	label.SetWrap(true)
	label.AddCSSClass("banner-bar")
	rev.SetChild(label)
	return rev, label
}

// SetBanner shows or hides the banner with text.
func SetBanner(rev *gtk.Revealer, label *gtk.Label, text string) {
	label.SetText(text)
	rev.SetRevealChild(text != "")
}

// MessageRow builds one conversation row bubble (WL-2.2 read path).
func MessageRow(m bridge.Message) *gtk.Box { return MessageRowRich(m, "", false, true) }

// MessageRowRich builds one bubble: optional colored group sender
// name, wrapped body, and a time + ticks footer. first marks the
// first message of a run from one sender; it gets the bubble tail
// corner and the larger gap above, like the reference client.
func MessageRowRich(m bridge.Message, senderName string, showSender, first bool) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("msg-row")
	bubble := gtk.NewBox(gtk.OrientationVertical, 1)
	bubble.AddCSSClass("message-bubble")
	if first {
		box.AddCSSClass("first")
		bubble.AddCSSClass("tail")
	}
	if m.IsFromMe {
		bubble.AddCSSClass("outgoing")
		box.SetHAlign(gtk.AlignEnd)
	} else {
		bubble.AddCSSClass("incoming")
		box.SetHAlign(gtk.AlignStart)
	}
	if showSender && first && senderName != "" && !m.IsFromMe {
		nm := gtk.NewLabel("")
		nm.AddCSSClass("bubble-sender")
		nm.SetMarkup(`<span foreground="` + SenderColor(m.Sender.String()) + `">` +
			glibEscape(senderName) + `</span>`)
		nm.SetXAlign(0)
		bubble.Append(nm)
	}
	text := gtk.NewLabel(RowText(m))
	text.AddCSSClass("bubble-text")
	text.SetWrap(true)
	text.SetWrapMode(pango.WrapWordChar)
	text.SetMaxWidthChars(64)
	text.SetXAlign(0)
	text.SetSelectable(true)
	text.SetCanFocus(false)
	bubble.Append(text)
	meta := gtk.NewBox(gtk.OrientationHorizontal, 3)
	meta.SetHAlign(gtk.AlignEnd)
	ts := gtk.NewLabel(m.Ts.Format("15:04"))
	ts.AddCSSClass("bubble-meta")
	meta.Append(ts)
	if ticks, read := RowTicks(m); ticks != "" {
		tk := gtk.NewLabel(ticks)
		tk.AddCSSClass("tick")
		if read {
			tk.AddCSSClass("read")
		}
		meta.Append(tk)
	}
	bubble.Append(meta)
	box.Append(bubble)
	return box
}

// glibEscape escapes Pango markup in user content.
func glibEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// RowText renders the primary line per message kind (WL-2.2).
func RowText(m bridge.Message) string {
	switch m.Kind {
	case bridge.MsgText, bridge.MsgEdited:
		if m.Edited {
			return m.Body + " (edited)"
		}
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
	case bridge.MsgLocation, bridge.MsgLiveLocation:
		return "📍 " + m.Body
	case bridge.MsgContact:
		return "👤 " + m.Body
	case bridge.MsgPoll:
		return "📊 " + m.Body
	case bridge.MsgReaction:
		return m.Body
	case bridge.MsgRevoked:
		return "🚫 This message was deleted"
	case bridge.MsgCallLog:
		return "📞 Call"
	case bridge.MsgSystem, bridge.MsgEvent:
		return m.Body
	default:
		return m.Body
	}
}

// RowMeta renders time + state ticks (WL-2.8).
func RowMeta(m bridge.Message) string {
	ts := m.Ts.Format("15:04")
	if !m.IsFromMe {
		return ts
	}
	ticks := ""
	switch m.State {
	case bridge.StateSending:
		ticks = " ◷"
	case bridge.StateSent:
		ticks = " ✓"
	case bridge.StateDelivered:
		ticks = " ✓✓"
	case bridge.StateRead, bridge.StatePlayed:
		ticks = " ✓✓"
	}
	return ts + ticks
}

// RowTicks returns the delivery ticks for an outgoing message and
// whether they show the read color (styled by CSS .tick.read).
func RowTicks(m bridge.Message) (ticks string, read bool) {
	if !m.IsFromMe {
		return "", false
	}
	switch m.State {
	case bridge.StateSending:
		return "◷", false
	case bridge.StateSent:
		return "✓", false
	case bridge.StateDelivered:
		return "✓✓", false
	case bridge.StateRead, bridge.StatePlayed:
		return "✓✓", true
	}
	return "", false
}

// senderPalette mirrors the Windows client's per-sender hues.
var senderPalette = []string{
	"#E17076", "#E8833A", "#D19A00", "#35CD96",
	"#00A0F4", "#7F66FF", "#E14E9A", "#00ACC1",
}

// SenderColor picks a stable hue for a sender JID string.
func SenderColor(jid string) string {
	h := 0
	for i := 0; i < len(jid); i++ {
		h = h*31 + int(jid[i])
	}
	if h < 0 {
		h = -h
	}
	return senderPalette[h%len(senderPalette)]
}

// CenterPill builds a centered pill label (date separators, unread
// dividers) with the given CSS class.
func CenterPill(text, class string) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass(class + "-row")
	lbl := gtk.NewLabel(text)
	lbl.SetHExpand(true)
	lbl.SetHAlign(gtk.AlignCenter)
	lbl.SetWrap(true)
	lbl.SetMaxWidthChars(64)
	lbl.SetJustify(gtk.JustifyCenter)
	lbl.AddCSSClass(class)
	box.Append(lbl)
	return box
}

// ChatPreview renders a one-line list preview: "You: …", "Name: …"
// or plain text, truncated, with a media glyph for non-text.
func ChatPreview(m bridge.Message, senderName string) string {
	prefix := ""
	if m.IsFromMe {
		prefix = "You: "
	} else if senderName != "" {
		prefix = senderName + ": "
	}
	body := RowText(m)
	const max = 48
	runes := []rune(body)
	if len(runes) > max {
		body = string(runes[:max-1]) + "…"
	}
	return prefix + body
}
