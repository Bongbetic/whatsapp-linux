package media

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Recorder captures voice notes through the §12 GStreamer chain
// (PipeWire source → Opus 16 kHz mono 32 kbps → OGG).
type Recorder struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	dst  string
	stop chan struct{}
	done chan error
}

// Start begins recording to dst. Only one recording at a time.
func (r *Recorder) Start(dst string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil {
		return fmt.Errorf("already recording")
	}
	launch, err := exec.LookPath("gst-launch-1.0")
	if err != nil {
		return fmt.Errorf("gst-launch-1.0 missing: %w", err)
	}
	r.cmd = exec.Command(launch, "autoaudiosrc", "!", "audioconvert",
		"!", "audioresample", "!", "audio/x-raw,rate=16000,channels=1",
		"!", "opusenc", "bitrate=32000", "!", "oggmux",
		"!", "filesink", "location="+dst)
	r.dst, r.stop, r.done = dst, make(chan struct{}), make(chan error, 1)
	if err := r.cmd.Start(); err != nil {
		r.cmd = nil
		return err
	}
	go func(cmd *exec.Cmd, done chan error) { done <- cmd.Wait() }(r.cmd, r.done)
	return nil
}

// Stop ends the recording and returns the destination path. SIGINT
// lets gst-launch post EOS so the OGG finalizes cleanly; SIGKILL is
// only the fallback when EOS hangs.
func (r *Recorder) Stop() (string, error) {
	r.mu.Lock()
	cmd, dst := r.cmd, r.dst
	r.cmd = nil
	r.mu.Unlock()
	if cmd == nil {
		return "", fmt.Errorf("not recording")
	}
	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	select {
	case err := <-r.done:
		return dst, err
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return dst, <-r.done
	}
}

// StopAfter stops automatically after d (claims MaxVoiceSecs).
func (r *Recorder) StopAfter(d time.Duration) (string, error) {
	time.Sleep(d)
	return r.Stop()
}
