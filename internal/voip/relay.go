package voip

import (
	"strconv"
	"strings"

	waBinary "go.mau.fi/whatsmeow/binary"
)

// Relay is one parsed transport endpoint: UDP host/port plus the
// SRTP key, all from the peer's transport stanza.
type Relay struct {
	Host string
	Port int
	Key  string
}

// FromNode walks a transport stanza for relay triples. Keys often
// ride on a sibling of the endpoint node, so the walk collects
// host/port pairs per node and key-like values per subtree: a relay
// is returned only when exactly one key covers the endpoint
// (ambiguous multi-key stanzas return nil for the owner to capture;
// see tools/interop/phase5.md).
func FromNode(n *waBinary.Node) []Relay {
	if n == nil {
		return nil
	}
	type endpoint struct {
		host, port string
	}
	var eps []endpoint
	var keys []string
	var walk func(*waBinary.Node)
	walk = func(m *waBinary.Node) {
		if m == nil {
			return
		}
		var host, port string
		for k, v := range m.Attrs {
			s := attrString(v)
			switch strings.ToLower(k) {
			case "ip", "host", "address", "relay-ip", "relay-host":
				if host == "" {
					host = s
				}
			case "port", "relay-port":
				if port == "" {
					port = s
				}
			case "key", "srtp-key", "srtpkey", "enc-key", "enckey":
				if s != "" {
					keys = append(keys, s)
				}
			}
		}
		if host != "" && port != "" {
			eps = append(eps, endpoint{host, port})
		}
		for _, c := range m.GetChildren() {
			c := c
			walk(&c)
		}
	}
	walk(n)
	if len(eps) == 0 || len(keys) != 1 {
		return nil
	}
	var out []Relay
	for _, e := range eps {
		p, err := strconv.Atoi(e.port)
		if err != nil || p <= 0 || p > 65535 {
			continue
		}
		out = append(out, Relay{Host: e.host, Port: p, Key: keys[0]})
	}
	return out
}

func attrString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	default:
		return ""
	}
}
