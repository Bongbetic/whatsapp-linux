package notify

// playHint plays the notification sound. Real synthesized tones land
// with data/sounds at WL-3.6; until then this is intentionally silent
// (a notification with no sound beats a broken audio pipeline).
func playHint() {}
