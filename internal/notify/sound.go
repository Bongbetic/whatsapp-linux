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
	PlayFile(soundFile())
}

// TonePath resolves a synthesized tone (WL-3.6 data/sounds) by name:
// the installed sounds dir, then the source tree (dev runs). ""
// when not found (the caller stays silent).
func TonePath(name string) string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "..", "share", "whatsapp-linux", "sounds", name))
	}
	candidates = append(candidates, filepath.Join("data", "sounds", name))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// PlayFile plays one tone via PipeWire/Pulse (paplay), silently
// skipping when no player or tone is available.
func PlayFile(path string) {
	if path == "" {
		return
	}
	play, err := exec.LookPath("paplay")
	if err != nil {
		return
	}
	_ = exec.Command(play, path).Start()
}
