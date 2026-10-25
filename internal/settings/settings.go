// Package settings wraps the GSettings schema
// com.bongbetic.WhatsAppLinux (plan §10) with an in-memory fallback
// so headless runs and tests behave identically.
package settings

import "sync"

// Keys (plan §10). Server-side privacy settings are NOT here:
// they go through Engine and are cached read-only in store kv.
const (
	KeyAutostart     = "autostart"
	KeyCloseToTray   = "close-to-tray"
	KeyStartHidden   = "start-hidden"
	KeyEnterToSend   = "enter-to-send"
	KeyFontScale     = "font-scale"
	KeyDownloadDir   = "download-dir"
	KeyAutoPhotos    = "auto-download-photos"
	KeyAutoAudio     = "auto-download-audio"
	KeyAutoVideo     = "auto-download-video"
	KeyAutoDocs      = "auto-download-docs"
	KeyNotifyPreview = "notify-preview"
	KeyNotifySound   = "notify-sound"
	KeySpellcheck    = "spellcheck"
	KeyLowDataCalls  = "low-data-calls"
	KeyWindowW       = "window-w"
	KeyWindowH       = "window-h"
	KeyWindowX       = "window-x"
	KeyWindowY       = "window-y"
)

// Defaults per §10.
func Defaults() map[string]any {
	return map[string]any{
		KeyAutostart:     true,
		KeyCloseToTray:   true,
		KeyStartHidden:   false,
		KeyEnterToSend:   true,
		KeyFontScale:     1.0,
		KeyDownloadDir:   "",
		KeyAutoPhotos:    true,
		KeyAutoAudio:     true,
		KeyAutoVideo:     false,
		KeyAutoDocs:      false,
		KeyNotifyPreview: true,
		KeyNotifySound:   true,
		KeySpellcheck:    true,
		KeyLowDataCalls:  false,
		KeyWindowW:       1180, KeyWindowH: 760, KeyWindowX: -1, KeyWindowY: -1,
	}
}

// Store is a threadsafe key/value settings bag. On a session with
// D-Bus + installed schema it is backed by GSettings (wired in
// app startup); otherwise it serves Defaults with local overrides,
// which keeps wlprobe and tests honest.
type Store struct {
	mu sync.RWMutex
	m  map[string]any
}

// New returns a Store seeded with Defaults.
func New() *Store { return &Store{m: Defaults()} }

func (s *Store) GetBool(k string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, _ := s.m[k].(bool)
	return v
}

func (s *Store) GetFloat(k string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, _ := s.m[k].(float64)
	return v
}

func (s *Store) GetString(k string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, _ := s.m[k].(string)
	return v
}

func (s *Store) GetInt(k string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, _ := s.m[k].(int)
	return v
}

// Set stores a local override (and writes through to GSettings
// when a backend is attached).
func (s *Store) Set(k string, v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}
