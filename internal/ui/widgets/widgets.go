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
func Avatar(title string) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetContentWidth(40)
	area.SetContentHeight(40)
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
func MessageRow(m bridge.Message) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 6)
	bubble := gtk.NewBox(gtk.OrientationVertical, 2)
	bubble.AddCSSClass("message-bubble")
	if m.IsFromMe {
		bubble.AddCSSClass("outgoing")
		box.SetHAlign(gtk.AlignEnd)
	} else {
		box.SetHAlign(gtk.AlignStart)
	}
	text := gtk.NewLabel(RowText(m))
	text.SetWrap(true)
	text.SetXAlign(0)
	bubble.Append(text)
	meta := gtk.NewLabel(RowMeta(m))
	meta.AddCSSClass("dim-label")
	meta.SetXAlign(1)
	bubble.Append(meta)
	box.Append(bubble)
	return box
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
