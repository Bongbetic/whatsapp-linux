// Package widgets holds small shared GTK widgets (§9 ui/widgets).
// Note: chat/conversation lists use Gtk.ListBox, not Gtk.ListView:
// ListView needs GObject models for every row, which buys nothing
// at v1 row counts; revisit if profiling demands it.
package widgets

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

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

// Avatar is a 40px circle with initials (photo support at WL-2.8).
func Avatar(title string) *gtk.DrawingArea { return AvatarSized(title, 40) }

// AvatarSized is Avatar at an explicit diameter.
func AvatarSized(title string, px int) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetContentWidth(px)
	area.SetContentHeight(px)
	initials := Initials(title)
	area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
		cr.SetSourceRGB(0.14, 0.83, 0.4) // WhatsApp green; full theming via CSS at WL-2.2
		cr.Arc(float64(w)/2, float64(h)/2, float64(w)/2-1, 0, 6.2832)
		cr.Fill()
		cr.SetSourceRGB(1, 1, 1)
		cr.SelectFontFace("sans", cairo.FontSlantNormal, cairo.FontWeightBold)
		cr.SetFontSize(14)
		ext := cr.TextExtents(initials)
		cr.MoveTo(float64(w)/2-ext.Width/2-ext.XBearing, float64(h)/2-ext.Height/2-ext.YBearing)
		cr.ShowText(initials)
	})
	return area
}

// Banner is a revealer row for offline/ephemeral notices (§9).
func Banner() (*gtk.Revealer, *gtk.Label) {
	rev := gtk.NewRevealer()
	label := gtk.NewLabel("")
	rev.SetChild(label)
	return rev, label
}

// SetBanner shows or hides the banner with text.
func SetBanner(rev *gtk.Revealer, label *gtk.Label, text string) {
	label.SetText(text)
	rev.SetRevealChild(text != "")
}

// MessageRow builds one conversation row bubble (WL-2.2 read path).
func MessageRow(m bridge.Message) *gtk.Box { return MessageRowRich(m, "", false) }

// MessageRowRich adds an optional colored group sender name above
// the text and renders ticks in Windows colors.
func MessageRowRich(m bridge.Message, senderName string, showSender bool) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 6)
	bubble := gtk.NewBox(gtk.OrientationVertical, 2)
	bubble.AddCSSClass("message-bubble")
	if m.IsFromMe {
		bubble.AddCSSClass("outgoing")
		box.SetHAlign(gtk.AlignEnd)
	} else {
		box.SetHAlign(gtk.AlignStart)
	}
	if showSender && senderName != "" && !m.IsFromMe {
		nm := gtk.NewLabel("")
		nm.SetMarkup(`<span foreground="` + SenderColor(m.Sender.String()) + `" weight="bold">` +
			glibEscape(senderName) + `</span>`)
		nm.SetXAlign(0)
		bubble.Append(nm)
	}
	text := gtk.NewLabel(RowText(m))
	text.SetWrap(true)
	text.SetXAlign(0)
	bubble.Append(text)
	meta := gtk.NewLabel("")
	meta.SetMarkup(RowMetaMarkup(m))
	meta.SetXAlign(1)
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

// Tick colors (Windows light theme): grey until read, blue after.
const tickGrey = "#8696A0"
const tickBlue = "#53BDEB"

// RowMetaMarkup renders time + colored ticks as Pango markup.
func RowMetaMarkup(m bridge.Message) string {
	ts := m.Ts.Format("15:04")
	if !m.IsFromMe {
		return ts
	}
	ticks, color := "", tickGrey
	switch m.State {
	case bridge.StateSending:
		ticks = " ◷"
	case bridge.StateSent:
		ticks = " ✓"
	case bridge.StateDelivered:
		ticks = " ✓✓"
	case bridge.StateRead, bridge.StatePlayed:
		ticks, color = " ✓✓", tickBlue
	}
	if ticks == "" {
		return ts
	}
	return ts + `<span foreground="` + color + `">` + ticks + `</span>`
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
	box.SetHAlign(gtk.AlignCenter)
	lbl := gtk.NewLabel(text)
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
