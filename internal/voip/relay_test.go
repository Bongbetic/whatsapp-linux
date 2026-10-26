package voip

import (
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
)

func transportNode() *waBinary.Node {
	return &waBinary.Node{
		Tag: "transport",
		Content: []waBinary.Node{
			{Tag: "relay", Attrs: waBinary.Attrs{"ip": "10.0.0.9", "port": "3478"}},
			{Tag: "key", Attrs: waBinary.Attrs{"key": "c2VjcmV0"}},
		},
	}
}

func TestFromNodeTriple(t *testing.T) {
	relays := FromNode(transportNode())
	if len(relays) != 1 {
		t.Fatalf("relays=%+v", relays)
	}
	r := relays[0]
	if r.Host != "10.0.0.9" || r.Port != 3478 || r.Key != "c2VjcmV0" {
		t.Fatalf("relay=%+v", r)
	}
}

func TestFromNodeAmbiguousOrEmpty(t *testing.T) {
	if FromNode(nil) != nil {
		t.Fatal("nil node must yield nil")
	}
	// Two keys: ambiguous, refuse to guess.
	n := transportNode()
	kids := n.GetChildren()
	kids = append(kids, waBinary.Node{Tag: "key", Attrs: waBinary.Attrs{"key": "b3RoZXI="}})
	n.Content = kids
	if FromNode(n) != nil {
		t.Fatal("multi-key stanza must yield nil")
	}
	// No endpoint at all.
	if got := FromNode(&waBinary.Node{Tag: "transport"}); got != nil {
		t.Fatalf("endpoint-less stanza must yield nil, got %+v", got)
	}
}

func TestDialAudioPipelines(t *testing.T) {
	leg := DialAudio("10.0.0.9", 3478, "c2VjcmV0")
	if leg == nil || len(leg.send) == 0 || len(leg.recv) == 0 {
		t.Fatal("pipelines must be built")
	}
	joined := ""
	for _, a := range leg.send {
		joined += a + " "
	}
	for _, want := range []string{"srtpenc", "udpsink", "host=10.0.0.9", "port=3478", "key=c2VjcmV0"} {
		found := false
		for _, a := range leg.send {
			if a == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("send pipeline missing %q in %q", want, joined)
		}
	}
}
