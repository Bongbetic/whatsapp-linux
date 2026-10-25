package widgets

import (
	"fmt"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// VoicePlayer plays a voice note / audio file with play/pause, seek,
// and elapsed/totaltime. Speed selection is intentionally 1× in v1:
// GtkMediaStream exposes no playback-rate API, so 1.5×/2× waits for
// the GStreamer-level player shared with calls (Phase 5).
type VoicePlayer struct {
	Box   *gtk.Box
	video *gtk.Video
	play  *gtk.Button
	pos   *gtk.Label
	seek  *gtk.Scale
}

// NewVoicePlayer builds the player for a local audio path.
func NewVoicePlayer(path string) *VoicePlayer {
	v := &VoicePlayer{}
	v.Box = gtk.NewBox(gtk.OrientationHorizontal, 6)
	v.video = gtk.NewVideoForFilename(path)
	v.play = gtk.NewButtonFromIconName("media-playback-start-symbolic")
	v.play.SetTooltipText("Play voice message")
	v.play.ConnectClicked(func() { v.toggle() })
	v.Box.Append(v.play)
	v.seek = gtk.NewScaleWithRange(gtk.OrientationHorizontal, 0, 100, 1)
	v.seek.SetHExpand(true)
	v.seek.ConnectValueChanged(func() {
		ms := v.stream()
		if ms == nil || !ms.IsSeekable() || ms.Duration() <= 0 {
			return
		}
		ms.Seek(int64(v.seek.Value() / 100 * float64(ms.Duration())))
	})
	v.Box.Append(v.seek)
	v.pos = gtk.NewLabel("0:00 / 0:00")
	v.Box.Append(v.pos)
	glib.TimeoutAdd(250, func() bool {
		v.poll()
		return true
	})
	return v
}

func (v *VoicePlayer) stream() *gtk.MediaStream {
	ms, ok := v.video.MediaStream().(*gtk.MediaStream)
	if !ok {
		return nil
	}
	return ms
}

func (v *VoicePlayer) toggle() {
	ms := v.stream()
	if ms == nil {
		return
	}
	if ms.Playing() {
		ms.Pause()
		v.play.SetIconName("media-playback-start-symbolic")
	} else {
		ms.Play()
		v.play.SetIconName("media-playback-pause-symbolic")
	}
}

func (v *VoicePlayer) poll() {
	ms := v.stream()
	if ms == nil || ms.Duration() <= 0 {
		return
	}
	v.seek.SetValue(float64(ms.Timestamp()) / float64(ms.Duration()) * 100)
	v.pos.SetText(fmt.Sprintf("%s / %s", stamp(ms.Timestamp()), stamp(ms.Duration())))
}

// stamp formats microseconds (GtkMediaStream units) as m:ss.
func stamp(us int64) string {
	d := time.Duration(us) * time.Microsecond
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
