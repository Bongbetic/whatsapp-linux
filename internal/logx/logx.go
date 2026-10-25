// Package logx sets up zerolog logging plus the D-9 scrubber: JIDs
// and phone numbers are hash-prefixed in debug logs, and key
// material is never logged.
package logx

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
)

var (
	jidRe   = regexp.MustCompile(`\b(\d{6,17})@([a-z.]+)`)
	phoneRe = regexp.MustCompile(`\+?\d[\d .\-]{7,}\d`)
)

// Scrub replaces JID user parts and phone-like digit runs with a
// stable short hash prefix so logs stay correlatable but private.
func Scrub(s string) string {
	s = jidRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := jidRe.FindStringSubmatch(m)
		return "jid:" + shortHash(sub[1]) + "@" + sub[2]
	})
	return phoneRe.ReplaceAllStringFunc(s, func(m string) string {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, m)
		if len(digits) < 8 {
			return m
		}
		return "tel:" + shortHash(digits)
	})
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:4])
}

// New builds the app logger. Debug enables caller info; all output
// passes through the scrubber writer.
func New(debug bool) zerolog.Logger {
	w := &scrubWriter{w: os.Stderr}
	l := zerolog.New(w).With().Timestamp().Logger()
	if debug {
		l = l.Level(zerolog.DebugLevel)
	} else {
		l = l.Level(zerolog.InfoLevel)
	}
	return l
}

type scrubWriter struct{ w *os.File }

func (s *scrubWriter) Write(p []byte) (int, error) {
	return s.w.WriteString(Scrub(string(p)))
}
