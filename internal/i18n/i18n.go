// Package i18n holds UI string catalogs (WL-6.3). English is the
// source language; T() falls back to the key (which is the English
// text) when a translation is missing, so untranslated builds stay
// readable. .po import lands when the first non-en catalog does.
package i18n

import (
	"fmt"
	"sync"
)

var (
	mu   sync.RWMutex
	lang = "en"
	// catalog maps language → key → translation. English entries
	// mirror their keys (source of truth for translators).
	catalog = map[string]map[string]string{
		"en": {
			"login.title":  "Link this device",
			"login.scan":   "Scan the QR code with WhatsApp → Linked devices",
			"login.cancel": "Cancel",
			"login.code":   "Get code",
		},
	}
)

// SetLanguage selects the UI language (unsupported → English).
func SetLanguage(l string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := catalog[l]; ok {
		lang = l
		return
	}
	lang = "en"
}

// T translates key with optional %v args, falling back to the key.
func T(key string, args ...any) string {
	mu.RLock()
	s, ok := catalog[lang][key]
	mu.RUnlock()
	if !ok {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// Add merges one language catalog (used by .po import later).
func Add(language string, entries map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if catalog[language] == nil {
		catalog[language] = map[string]string{}
	}
	for k, v := range entries {
		catalog[language][k] = v
	}
}
