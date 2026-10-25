package media

// Waveform computes WaveformPeaks normalized peaks (0..255) over PCM
// samples during recording; stored in messages.waveform (§12).
func Waveform(samples []int16) []byte {
	out := make([]byte, WaveformPeaks)
	if len(samples) == 0 {
		return out
	}
	per := len(samples) / WaveformPeaks
	if per == 0 {
		per = 1
	}
	for i := range out {
		lo := i * per
		hi := lo + per
		if hi > len(samples) {
			hi = len(samples)
		}
		if lo >= len(samples) {
			break
		}
		var peak int32
		for _, s := range samples[lo:hi] {
			v := int32(s)
			if v < 0 {
				v = -v
			}
			if v > peak {
				peak = v
			}
		}
		out[i] = byte(peak * 255 / 32768)
	}
	return out
}
