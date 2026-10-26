package app

import (
	"context"
	"os"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ShotConfig turns the app into a one-shot screenshot renderer (dev
// only, driven by cmd/wlshot): show Page, render the window to Out as
// PNG, then quit.
type ShotConfig struct {
	Out  string // PNG path
	Page string // chat|list|login|settings|status|channels|calls
	Dark bool   // force dark color scheme
}

// appFlags makes shot runs non-unique so a running real instance on
// the same session bus never swallows the activation.
func appFlags(cfg Config) gio.ApplicationFlags {
	if cfg.Shot != nil {
		return gio.ApplicationHandlesOpen | gio.ApplicationNonUnique
	}
	return gio.ApplicationHandlesOpen
}

// shotDark pins the color scheme (dark or light). Must run before widgets exist.
// gtk-interface-color-scheme (GTK 4.20+) drives the CSS
// prefers-color-scheme media feature: 1 default, 2 dark, 3 light.
func (a *App) shotDark() {
	if a.shot == nil {
		return
	}
	s := gtk.SettingsGetDefault()
	if a.shot.Dark {
		s.SetObjectProperty("gtk-application-prefer-dark-theme", true)
		s.SetObjectProperty("gtk-interface-color-scheme", 2)
		return
	}
	// Pin light so the host's own dark preference never leaks in.
	s.SetObjectProperty("gtk-application-prefer-dark-theme", false)
	s.SetObjectProperty("gtk-interface-color-scheme", 3)
}

// shotSchedule runs at the end of activate(): wait for the fake engine
// to settle, switch to the requested page, then render.
func (a *App) shotSchedule() {
	if a.shot == nil {
		return
	}
	// Without a window manager (xvfb) GTK ignores the default size and
	// grows the surface to the screen; a fixed-size window keeps WxH.
	a.win.SetResizable(false)
	glib.TimeoutAdd(900, func() bool {
		a.shotPage()
		glib.TimeoutAdd(1200, func() bool {
			a.shotRender()
			return false
		})
		return false
	})
}

func (a *App) shotPage() {
	switch a.shot.Page {
	case "login":
		return
	case "list":
		return
	case "settings", "status", "channels", "calls":
		a.showPage(a.shot.Page)
	default: // chat
		chats, err := a.db.ListChats(context.Background(), 1, 0, "", false)
		if err == nil && len(chats) > 0 {
			a.openChat(chats[0])
		}
	}
}

func (a *App) shotRender() {
	defer a.gtkApp.Quit()
	_ = os.Remove(a.shot.Out)
	if !savePNG(a.win, a.shot.Out, a.win.Width(), a.win.Height()) {
		a.log.Error().Msg("shot: render/save png failed")
	}
}
