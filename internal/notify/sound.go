package notify

import (
	"os"
	"os/exec"
	"path/filepath"
)

// soundFile resolves the notification tone: per-chat override lives
// in settings at WL-3.6; until then the default notice.wav beside the
// binary, then the source-tree data dir (dev runs).
func soundFile() string {
	candidates := []string{"notice.wav"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "share", "whatsapp-linux", "sounds", "notice.wav"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// playHint plays the notification tone via PipeWire/Pulse (paplay),
// silently skipping when no player or tone is available.
func playHint() {
	f := soundFile()
	if f == "" {
		return
	}
	play, err := exec.LookPath("paplay")
	if err != nil {
		return
	}
	_ = exec.Command(play, f).Start()
}
