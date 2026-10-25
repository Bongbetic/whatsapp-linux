// Package emoji is the composer emoji picker (WL-3.4): grid plus
// skin-tone popover plus search, over a curated in-tree list (no
// system emoji database dependency; full coverage follows font
// support, not data).
package emoji

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Tones are the five Fitzpatrick modifiers (plus none).
const (
	ToneNone  = ""
	ToneLight = "\U0001F3FB"
	ToneMedL  = "\U0001F3FC"
	ToneMed   = "\U0001F3FD"
	ToneMedD  = "\U0001F3FE"
	ToneDark  = "\U0001F3FF"
)

// People is the curated base set; toned entries take the popover tone.
var People = []string{
	"😀", "😁", "😂", "🤣", "😊", "😍", "😘", "😎",
	"🤔", "😐", "🙄", "😴", "😷", "🤯", "🥳", "😢",
	"😭", "😡", "👍", "👎", "👏", "🙏", "💪", "👋",
	"✌️", "🤝", "👨", "👩", "👧", "👦", "👶", "🧑",
}

// Symbols is the non-people set (no tones apply).
var Symbols = []string{
	"❤️", "💔", "🔥", "🎉", "🎂", "⚽", "🚗", "✈️",
	"☀️", "🌙", "⭐", "💡", "🎵", "📷", "🎁", "💯",
}

// WithTone appends a Fitzpatrick modifier to a base emoji.
func WithTone(base, tone string) string {
	if tone == ToneNone {
		return base
	}
	return base + tone
}

// Search filters base+symbol sets by substring (headless-testable).
// Matching is intentionally simple: names are not shipped, so it
// matches the literal emoji (paste-to-find) and always returns the
// full set on empty query.
func Search(q string) []string {
	q = strings.TrimSpace(q)
	if q == "" {
		return append(append([]string{}, People...), Symbols...)
	}
	var out []string
	for _, e := range append(append([]string{}, People...), Symbols...) {
		if strings.Contains(e, q) {
			out = append(out, e)
		}
	}
	return out
}

// Picker is the popover grid with a skin-tone row. onPick inserts the
// emoji into the composer.
func Picker(onPick func(string)) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 4)
	search := gtk.NewEntry()
	search.SetPlaceholderText("Search emoji")
	box.Append(search)

	tones := gtk.NewBox(gtk.OrientationHorizontal, 2)
	tone := ToneNone
	for _, t := range []string{ToneNone, ToneLight, ToneMedL, ToneMed, ToneMedD, ToneDark} {
		t := t
		label := "∅"
		if t != ToneNone {
			label = "👋" + t
		}
		b := gtk.NewButtonWithLabel(label)
		b.ConnectClicked(func() { tone = t })
		tones.Append(b)
	}
	box.Append(tones)

	flow := gtk.NewFlowBox()
	flow.SetMaxChildrenPerLine(8)
	box.Append(flow)
	render := func() {
		for {
			child := flow.FirstChild()
			if child == nil {
				break
			}
			flow.Remove(child)
		}
		for _, e := range Search(search.Buffer().Text()) {
			e := WithTone(e, toneFor(e, tone))
			b := gtk.NewButtonWithLabel(e)
			b.ConnectClicked(func() { onPick(e) })
			flow.Append(b)
		}
	}
	buf := search.Buffer()
	buf.ConnectInsertedText(func(uint, string, uint) { render() })
	buf.ConnectDeletedText(func(uint, uint) { render() })
	render()
	return box
}

// toneFor applies tones only to the people set.
func toneFor(e, tone string) string {
	for _, p := range People {
		if p == e {
			return tone
		}
	}
	return ToneNone
}
