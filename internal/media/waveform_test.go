package media

import (
	"math"
	"testing"
)

func TestWaveformSine(t *testing.T) {
	const n = 16000
	samples := make([]int16, n)
	for i := range samples {
		samples[i] = int16(30000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}
	w := Waveform(samples)
	if len(w) != WaveformPeaks {
		t.Fatalf("len=%d", len(w))
	}
	for i, p := range w {
		if p < 200 {
			t.Fatalf("peak %d too low: %d", i, p)
		}
	}
}

func TestWaveformSilenceAndEmpty(t *testing.T) {
	w := Waveform(make([]int16, 1000))
	for _, p := range w {
		if p != 0 {
			t.Fatal("silence must be zero")
		}
	}
	if got := Waveform(nil); len(got) != WaveformPeaks {
		t.Fatal("empty keeps resolution")
	}
}
