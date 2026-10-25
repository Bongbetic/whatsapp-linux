// Package mediaview is the fullscreen media viewer (WL-3.2): image
// with caption, video with native controls (Gtk.Video), voice with
// the VoicePlayer, album prev/next, download / open-with actions.
package mediaview

import (
	"os/exec"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	"github.com/Bongbetic/whatsapp-linux/internal/ui/widgets"
)

// Item is one album entry.
type Item struct {
	Msg bridge.Message
}

// Show opens the viewer dialog over parent at index i of album.
func Show(parent *gtk.ApplicationWindow, album []Item, i int, caption string) {
	if len(album) == 0 {
		return
	}
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Media viewer")
	d.SetDefaultSize(800, 600)

	box := gtk.NewBox(gtk.OrientationVertical, 6)
	body := gtk.NewBox(gtk.OrientationVertical, 0)
	body.SetVExpand(true)
	box.Append(body)

	var render func(idx int)
	render = func(idx int) {
		if idx < 0 {
			idx = 0
		}
		if idx >= len(album) {
			idx = len(album) - 1
		}
		m := album[idx].Msg
		for {
			child := body.FirstChild()
			if child == nil {
				break
			}
			body.Remove(child)
		}
		switch m.Kind {
		case bridge.MsgVideo:
			v := gtk.NewVideoForFilename(m.MediaPath)
			v.SetVExpand(true)
			v.SetAutoplay(true)
			body.Append(v)
		case bridge.MsgVoice, bridge.MsgAudio:
			body.Append(widgets.NewVoicePlayer(m.MediaPath).Box)
		default:
			p := gtk.NewPictureForFilename(m.MediaPath)
			p.SetHExpand(true)
			p.SetVExpand(true)
			body.Append(p)
		}
		cap := gtk.NewLabel(caption)
		cap.AddCSSClass("dim-label")
		body.Append(cap)
	}

	nav := gtk.NewBox(gtk.OrientationHorizontal, 6)
	idx := i
	prev := gtk.NewButtonWithLabel("‹ Prev")
	prev.ConnectClicked(func() {
		idx--
		render(idx)
	})
	nav.Append(prev)
	next := gtk.NewButtonWithLabel("Next ›")
	next.ConnectClicked(func() {
		idx++
		render(idx)
	})
	nav.Append(next)
	open := gtk.NewButtonWithLabel("Open with…")
	open.ConnectClicked(func() {
		_ = exec.Command("xdg-open", album[idx].Msg.MediaPath).Start()
	})
	nav.Append(open)
	box.Append(nav)

	d.SetChild(box)
	render(idx)
	d.Show()
}
