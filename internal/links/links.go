// Package links parses whatsapp://, tel: and wa.me URLs (§11)
// into dial/send intents for single-instance handoff.
package links

import (
	"fmt"
	"net/url"
	"strings"
)

// Intent is one parsed inbound link.
type Intent struct {
	Kind  string // "chat" (open/send) or "call"
	Phone string // digits only, no '+'
	Text  string // prefilled text, may be ""
}

// Parse handles whatsapp://send?phone=&text=, https://wa.me/<phone>?text=,
// tel:<phone>. Anything else errors.
func Parse(raw string) (Intent, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Intent{}, err
	}
	switch strings.ToLower(u.Scheme) {
	case "whatsapp":
		if u.Host == "send" || strings.HasPrefix(u.Path, "/send") {
			q := u.Query()
			phone := digits(q.Get("phone"))
			if phone == "" {
				return Intent{}, fmt.Errorf("whatsapp link has no phone: %q", raw)
			}
			return Intent{Kind: "chat", Phone: phone, Text: q.Get("text")}, nil
		}
	case "https", "http":
		if strings.EqualFold(u.Host, "wa.me") {
			phone := digits(strings.Trim(u.Path, "/"))
			if phone == "" {
				return Intent{}, fmt.Errorf("wa.me link has no phone: %q", raw)
			}
			return Intent{Kind: "chat", Phone: phone, Text: u.Query().Get("text")}, nil
		}
	case "tel":
		phone := digits(u.Opaque)
		if phone == "" {
			phone = digits(u.Path)
		}
		if phone == "" {
			return Intent{}, fmt.Errorf("tel link has no phone: %q", raw)
		}
		return Intent{Kind: "chat", Phone: phone}, nil
	}
	return Intent{}, fmt.Errorf("unsupported link: %q", raw)
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}
