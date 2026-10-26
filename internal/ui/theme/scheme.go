package theme

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// GtkInterfaceColorScheme values (GTK 4.20+).
const (
	schemeDefault = 1
	schemeDark    = 2
	schemeLight   = 3
)

// followScheme keeps the stylesheet's prefers-color-scheme media
// feature in step with the desktop. GTK only feeds the system scheme
// to its own theme provider; an application provider evaluates
// @media (prefers-color-scheme) against its own property, which
// stays "default" (light) unless set. The CSS is re-parsed after each
// change so the media blocks are evaluated again. It loads the CSS
// itself and reports false on GTK older than 4.20, where the caller
// loads it once instead.
func followScheme(display *gdk.Display, prov *gtk.CSSProvider) bool {
	if gtk.GetMinorVersion() < 20 {
		return false
	}
	s := gtk.SettingsGetForDisplay(display)
	last := -1
	sync := func() {
		scheme := asInt(s.ObjectProperty("gtk-interface-color-scheme"))
		if scheme != schemeDark && scheme != schemeLight {
			scheme = schemeLight
			if dark, _ := s.ObjectProperty("gtk-application-prefer-dark-theme").(bool); dark {
				scheme = schemeDark
			}
		}
		if scheme == last {
			return
		}
		last = scheme
		prov.SetObjectProperty("prefers-color-scheme", scheme)
		prov.LoadFromString(css)
	}
	sync()
	s.NotifyProperty("gtk-interface-color-scheme", sync)
	s.NotifyProperty("gtk-application-prefer-dark-theme", sync)
	return true
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint32:
		return int(n)
	}
	return schemeDefault
}
