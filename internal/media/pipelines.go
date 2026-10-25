// Package media implements the media specification (plan §12):
// GStreamer playback/record/transcode parameters, voice waveform,
// image preparation, sticker conversion, downloads queue, storage
// accounting. All pipeline shapes are string constants so unit tests
// can assert gst_parse_launch validity headless.
package media

// Voice recording: PipeWire/monitor source → 16 kHz mono → Opus
// 32 kbps → OGG (§12). VoiceFile receives the destination path.
const (
	VoiceRecordPipeline = `autoaudiosrc ! audioconvert ! audioresample ! ` +
		`audio/x-raw,rate=16000,channels=1 ! opusenc bitrate=32000 ! ` +
		`oggmux ! filesink location="%s"`

	// VoiceTestPipeline is the headless twin of VoiceRecordPipeline
	// (test source instead of a microphone).
	VoiceTestPipeline = `audiotestsrc num-buffers=160 ! audioconvert ! audioresample ! ` +
		`audio/x-raw,rate=16000,channels=1 ! opusenc bitrate=32000 ! ` +
		`oggmux ! filesink location="%s"`

	// VoicePlaybackPipeline plays an OGG/Opus file to the default sink.
	VoicePlaybackPipeline = `filesrc location="%s" ! oggdemux ! opusdec ! ` +
		`audioconvert ! audioresample ! autoaudiosink`

	// VideoTranscodePipeline converts to MP4 (H.264+AAC) when the
	// source is not already MP4/H.264 (§12). Encoders are openh264enc
	// + faac: both ship in gst-plugins-bad1 (the pinned dep set),
	// unlike x264enc (ugly, not installed) and voaacenc (no backend
	// lib here). Verified by TestPipelineElementsExist.
	VideoTranscodePipeline = `filesrc location="%s" ! decodebin ! videoconvert ! ` +
		`openh264enc ! mp4mux name=mux ! filesink location="%s" ` +
		`filesrc location="%s" ! decodebin ! audioconvert ! faac ! mux.`
)

// MaxVoiceSecs caps voice recordings (server limits discovered at
// WL-3.1; revisited if interop shows otherwise).
const MaxVoiceSecs = 15 * 60

// WaveformPeaks is the stored waveform resolution (§12).
const WaveformPeaks = 64
