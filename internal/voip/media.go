package voip

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Call audio over GStreamer (plan §13, plan A): raw RTP/SRTP to the
// relay endpoints the peer's transport stanza names. Pipelines run
// as gst-launch-1.0 children like the voice-note path; mute stops
// the send leg only (the peer hears silence, the speaker stays up).

// AudioLeg is one direction-pair of a live call. Build with
// DialAudio once the relay triple is known; Stop is idempotent.
type AudioLeg struct {
	mu      sync.Mutex
	send    []string // mic → relay pipeline argv
	recv    []string // relay → speaker pipeline argv
	sendCmd *exec.Cmd
	recvCmd *exec.Cmd
	running bool
}

// DialAudio formats the two pipelines for host/port/srtpKey.
func DialAudio(host string, port int, srtpKey string) *AudioLeg {
	return &AudioLeg{
		send: []string{"autoaudiosrc", "!", "audioconvert", "!", "audioresample", "!",
			"opusenc", "!", "rtpopuspay", "!", "srtpenc", "key=" + srtpKey, "!",
			"udpsink", "host=" + host, fmt.Sprintf("port=%d", port)},
		recv: []string{"udpsrc", fmt.Sprintf("port=%d", port), "!", "srtpdec", "key=" + srtpKey, "!",
			"rtpopusdepay", "!", "opusdec", "!", "audioconvert", "!", "autoaudiosink"},
	}
}

// Start launches both directions. Missing gst-launch-1.0 errors.
func (l *AudioLeg) Start() error {
	launch, err := exec.LookPath("gst-launch-1.0")
	if err != nil {
		return fmt.Errorf("gst-launch-1.0 missing: %w", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return nil
	}
	l.sendCmd = spawnPipeline(launch, l.send)
	l.recvCmd = spawnPipeline(launch, l.recv)
	l.running = l.sendCmd != nil && l.recvCmd != nil
	if !l.running {
		stopCmd(l.sendCmd)
		stopCmd(l.recvCmd)
		l.sendCmd, l.recvCmd = nil, nil
		return fmt.Errorf("could not start call audio pipelines")
	}
	return nil
}

// SetMuted stops (true) or restarts (false) the microphone leg.
func (l *AudioLeg) SetMuted(muted bool) {
	launch, err := exec.LookPath("gst-launch-1.0")
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.running {
		return
	}
	if muted {
		stopCmd(l.sendCmd)
		l.sendCmd = nil
		return
	}
	if l.sendCmd == nil {
		l.sendCmd = spawnPipeline(launch, l.send)
	}
}

// Stop ends both directions (SIGINT for clean EOS, then kill).
func (l *AudioLeg) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	stopCmd(l.sendCmd)
	stopCmd(l.recvCmd)
	l.sendCmd, l.recvCmd = nil, nil
	l.running = false
}

func spawnPipeline(launch string, argv []string) *exec.Cmd {
	cmd := exec.Command(launch, append([]string{"-e"}, argv...)...)
	if err := cmd.Start(); err != nil {
		return nil
	}
	return cmd
}

// stopCmd mirrors media.Recorder.Stop: SIGINT for clean EOS, SIGKILL
// after 3 s. Nil-safe.
func stopCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}
