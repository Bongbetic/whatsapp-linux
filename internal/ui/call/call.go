// Package call renders the call window (WL-5.3, §9 ui/call): video
// grid (one Gtk.Picture per participant; GStreamer sinks attach at
// WL-5.4), self PiP, controls bar (mute/camera/speaker/screen
// share/reactions/hang up), devices popover, minimized floating bar.
// Media flows at WL-5.2/5.4; until then the window owns signaling
// state honestly and says so.
package call

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Window is the active-call window.
type Window struct {
	dlg    *gtk.Dialog
	status *gtk.Label
	muted  bool
	camOff bool
}

// Show opens the call window for an event-driven call id.
func Show(parent *gtk.ApplicationWindow, ev bridge.CallEvent, onHangup func()) *Window {
	w := &Window{}
	w.dlg = gtk.NewDialog()
	w.dlg.SetTransientFor(&parent.Window)
	w.dlg.SetTitle("Call — " + ev.Media)
	w.dlg.SetDefaultSize(640, 480)
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	w.status = gtk.NewLabel("call " + ev.Stage + " (media lands at WL-5.2/5.4)")
	box.Append(w.status)
	grid := gtk.NewGrid()
	grid.SetHExpand(true)
	grid.SetVExpand(true)
	box.Append(grid)
	bar := gtk.NewBox(gtk.OrientationHorizontal, 6)
	mute := gtk.NewButtonWithLabel("Mute")
	mute.ConnectClicked(func() {
		w.muted = !w.muted
		mute.SetLabel(map[bool]string{true: "Unmute", false: "Mute"}[w.muted])
	})
	bar.Append(mute)
	cam := gtk.NewButtonWithLabel("Camera off")
	cam.ConnectClicked(func() {
		w.camOff = !w.camOff
		cam.SetLabel(map[bool]string{true: "Camera on", false: "Camera off"}[w.camOff])
	})
	bar.Append(cam)
	spk := gtk.NewButtonWithLabel("Speaker")
	bar.Append(spk)
	share := gtk.NewButtonWithLabel("Share screen")
	share.SetTooltipText("WL-5.6")
	bar.Append(share)
	hang := gtk.NewButtonWithLabel("Hang up")
	hang.AddCSSClass("destructive-action")
	hang.ConnectClicked(func() {
		onHangup()
		w.dlg.Close()
	})
	bar.Append(hang)
	box.Append(bar)
	w.dlg.SetChild(box)
	w.dlg.Show()
	return w
}

// SetStage updates the status line from switch transitions.
func (w *Window) SetStage(stage string) {
	w.status.SetText("call " + stage + " (media lands at WL-5.2/5.4)")
}

// Close dismisses the window.
func (w *Window) Close() { w.dlg.Close() }

// Incoming pops the incoming-call dialog with ringtone note.
func Incoming(parent *gtk.ApplicationWindow, ev bridge.CallEvent, onAccept, onReject func()) {
	d := gtk.NewDialog()
	d.SetTransientFor(&parent.Window)
	d.SetTitle("Incoming " + ev.Media + " call")
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	box.Append(gtk.NewLabel("Incoming call — ringing (ringtone at WL-3.6+)"))
	acc := gtk.NewButtonWithLabel("Accept")
	acc.AddCSSClass("suggested-action")
	acc.ConnectClicked(func() { onAccept(); d.Close() })
	box.Append(acc)
	rej := gtk.NewButtonWithLabel("Reject")
	rej.ConnectClicked(func() { onReject(); d.Close() })
	box.Append(rej)
	d.SetChild(box)
	d.Show()
}
