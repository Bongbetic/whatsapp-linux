package media

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// elementsIn returns the GStreamer element names used by a pipeline
// template: the first token of every `!`-separated segment, skipping
// caps and property assignments (e.g. `x264enc tune=zerolatency` is
// the element `x264enc`).
func elementsIn(pipeline string) []string {
	var out []string
	for _, seg := range strings.Split(pipeline, "!") {
		fields := strings.Fields(seg)
		if len(fields) == 0 {
			continue
		}
		el := strings.Trim(fields[0], `"%`)
		if el == "" || strings.ContainsAny(el, "/.,:=") {
			continue
		}
		if strings.Contains(el, "%") {
			continue
		}
		out = append(out, el)
	}
	return out
}

func gstTool(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not installed", name)
	}
	return p
}

// Every element in every pipeline constant must exist headless (§12).
func TestPipelineElementsExist(t *testing.T) {
	inspect := gstTool(t, "gst-inspect-1.0")
	seen := map[string]bool{}
	for _, p := range []string{VoiceRecordPipeline, VoiceTestPipeline, VoicePlaybackPipeline, VideoTranscodePipeline} {
		for _, el := range elementsIn(strings.ReplaceAll(p, "%s", "/tmp/x")) {
			if seen[el] || el == "mux" || el == "name" {
				continue
			}
			seen[el] = true
			if out, err := exec.Command(inspect, el).CombinedOutput(); err != nil {
				t.Errorf("element %s missing: %s", el, out)
			}
		}
	}
}

// The voice chain must produce a real OGG file headless.
func TestVoiceChainProducesOGG(t *testing.T) {
	launch := gstTool(t, "gst-launch-1.0")
	dst := filepath.Join(t.TempDir(), "voice.ogg")
	cmd := exec.Command(launch, "audiotestsrc", "num-buffers=160", "!", "audioconvert",
		"!", "audioresample", "!", "audio/x-raw,rate=16000,channels=1",
		"!", "opusenc", "bitrate=32000", "!", "oggmux", "!", "filesink", "location="+dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("voice chain: %v\n%s", err, out)
	}
	fi, err := os.Stat(dst)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("no ogg produced: %v", err)
	}
	if fi.Size() < 100 {
		t.Fatalf("suspiciously small ogg: %d bytes", fi.Size())
	}
}
