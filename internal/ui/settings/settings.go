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
	lbl := gtk.NewLabel(label)
	lbl.SetHExpand(true)
	lbl.SetXAlign(0)
	row.Append(lbl)
	sw := gtk.NewSwitch()
	sw.SetActive(s.GetBool(key))
	sw.ConnectStateSet(func(bool) bool {
		s.Set(key, sw.Active())
		return false
	})
	row.Append(sw)
	return row
}

// View is the settings stack widget.
type View struct {
	Box *gtk.Box
}

// New builds every §10 local-key row. onPrivacy opens the
// server-side privacy pages (WL-4.6).
func New(s *settings.Store, onPrivacy func()) *View {
	v := &View{Box: gtk.NewBox(gtk.OrientationVertical, 6)}
	v.Box.Append(gtk.NewLabel("General"))
	v.Box.Append(boolRow(s, settings.KeyAutostart, "Start on login"))
	v.Box.Append(boolRow(s, settings.KeyCloseToTray, "Keep running in tray on close"))
	v.Box.Append(boolRow(s, settings.KeyEnterToSend, "Enter sends (Shift+Enter newline)"))
	v.Box.Append(boolRow(s, settings.KeySpellcheck, "Spell-check composer"))
	v.Box.Append(gtk.NewLabel("Downloads"))
	v.Box.Append(boolRow(s, settings.KeyAutoPhotos, "Auto-download photos"))
	v.Box.Append(boolRow(s, settings.KeyAutoAudio, "Auto-download voice & audio"))
	v.Box.Append(boolRow(s, settings.KeyAutoVideo, "Auto-download video"))
	v.Box.Append(boolRow(s, settings.KeyAutoDocs, "Auto-download documents"))
	v.Box.Append(gtk.NewLabel("Notifications"))
	v.Box.Append(boolRow(s, settings.KeyNotifyPreview, "Show message preview"))
	v.Box.Append(boolRow(s, settings.KeyNotifySound, "Play sound"))
	priv := gtk.NewButtonWithLabel("Privacy & safety…")
	priv.ConnectClicked(onPrivacy)
	v.Box.Append(priv)
	return v
}
