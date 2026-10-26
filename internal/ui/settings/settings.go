// Package settings renders the in-app Settings pages (§9 ui/settings,
// §10): one row per local GSettings key plus the privacy entry points
// (which call through Engine, never local shadow state).
package settings

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/settings"
)

// row binds one boolean key to a switch.
func boolRow(s *settings.Store, key, label string) *gtk.Box {
	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.AddCSSClass("setting-row")
	lbl := gtk.NewLabel(label)
	lbl.SetHExpand(true)
	lbl.SetXAlign(0)
	lbl.SetWrap(true)
	row.Append(lbl)
	sw := gtk.NewSwitch()
	sw.SetVAlign(gtk.AlignCenter)
	sw.SetActive(s.GetBool(key))
	sw.ConnectStateSet(func(bool) bool {
		s.Set(key, sw.Active())
		return false
	})
	row.Append(sw)
	return row
}

// Section returns a left-aligned group heading.
func Section(title string) *gtk.Label {
	l := gtk.NewLabel(title)
	l.AddCSSClass("section-title")
	l.SetXAlign(0)
	return l
}

// actionButton is a full-width, left-aligned row button.
func actionButton(text string) *gtk.Button {
	l := gtk.NewLabel(text)
	l.SetXAlign(0)
	b := gtk.NewButton()
	b.SetChild(l)
	b.AddCSSClass("drawer-action")
	return b
}

// View is the settings stack widget.
type View struct {
	Box *gtk.Box
}

// New builds every §10 local-key row. onPrivacy opens the
// server-side privacy pages (WL-4.6).
func New(s *settings.Store, onPrivacy func()) *View {
	v := &View{Box: gtk.NewBox(gtk.OrientationVertical, 2)}
	v.Box.Append(Section("General"))
	v.Box.Append(boolRow(s, settings.KeyAutostart, "Start on login"))
	v.Box.Append(boolRow(s, settings.KeyCloseToTray, "Keep running in tray on close"))
	v.Box.Append(boolRow(s, settings.KeyEnterToSend, "Enter sends (Shift+Enter newline)"))
	v.Box.Append(boolRow(s, settings.KeySpellcheck, "Spell-check composer"))
	v.Box.Append(Section("Downloads"))
	v.Box.Append(boolRow(s, settings.KeyAutoPhotos, "Auto-download photos"))
	v.Box.Append(boolRow(s, settings.KeyAutoAudio, "Auto-download voice & audio"))
	v.Box.Append(boolRow(s, settings.KeyAutoVideo, "Auto-download video"))
	v.Box.Append(boolRow(s, settings.KeyAutoDocs, "Auto-download documents"))
	v.Box.Append(Section("Notifications"))
	v.Box.Append(boolRow(s, settings.KeyNotifyPreview, "Show message preview"))
	v.Box.Append(boolRow(s, settings.KeyNotifySound, "Play sound"))
	v.Box.Append(Section("Privacy"))
	priv := actionButton("Privacy & safety…")
	priv.ConnectClicked(onPrivacy)
	v.Box.Append(priv)
	return v
}
