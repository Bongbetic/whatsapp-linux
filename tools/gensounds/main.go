// Command gensounds synthesizes the original alert tones for
// data/sounds (WL-3.6, clean-room: no Meta audio anywhere).
// Run: go run ./tools/gensounds [outdir]
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const rate = 22050

func main() {
	dir := "data/sounds"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	tones := map[string][]seg{
		"notice.wav":   {{880, 0, 0.12}, {660, 0.12, 0.24}},
		"sent.wav":     {{1200, 0, 0.08}},
		"receive.wav":  {{660, 0, 0.12}},
		"ringtone.wav": {{440, 0, 0.4}, {480, 0.5, 0.9}, {440, 1.0, 1.4}, {480, 1.5, 1.9}},
	}
	for name, segs := range tones {
		total := 0.0
		for _, s := range segs {
			if s.end > total {
				total = s.end
			}
		}
		n := int(total * rate)
		pcm := make([]int16, n)
		for _, s := range segs {
			for i := int(s.start * rate); i < int(s.end*rate) && i < n; i++ {
				t := float64(i) / rate
				env := envelope(float64(i-int(s.start*rate)) / (rate * (s.end - s.start)))
				pcm[i] += int16(12000 * env * math.Sin(2*math.Pi*s.freq*t))
			}
		}
		if err := writeWAV(filepath.Join(dir, name), pcm); err != nil {
			panic(err)
		}
		fmt.Println("wrote", name)
	}
}

type seg struct {
	freq       float64
	start, end float64
}

// envelope fades 5 ms in/out to avoid clicks.
func envelope(x float64) float64 {
	if x < 0.05 {
		return x / 0.05
	}
	if x > 0.95 {
		return (1 - x) / 0.05
	}
	return 1
}

func writeWAV(path string, pcm []int16) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data := 2 * len(pcm)
	hdr := []any{
		[4]byte{'R', 'I', 'F', 'F'}, uint32(36 + data),
		[4]byte{'W', 'A', 'V', 'E'}, [4]byte{'f', 'm', 't', ' '},
		uint32(16), uint16(1), uint16(1), uint32(rate),
		uint32(rate * 2), uint16(2), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, uint32(data),
	}
	for _, h := range hdr {
		if err := binary.Write(f, binary.LittleEndian, h); err != nil {
			return err
		}
	}
	return binary.Write(f, binary.LittleEndian, pcm)
}
