package emoji

import "testing"

func TestWithTone(t *testing.T) {
	if got := WithTone("👋", ToneDark); got != "👋"+ToneDark {
		t.Fatalf("got %q", got)
	}
	if got := WithTone("👋", ToneNone); got != "👋" {
		t.Fatalf("got %q", got)
	}
}

func TestSearch(t *testing.T) {
	all := Search("")
	if len(all) != len(People)+len(Symbols) {
		t.Fatalf("len=%d", len(all))
	}
	hit := Search("😀")
	if len(hit) != 1 {
		t.Fatalf("paste-to-find: %+v", hit)
	}
}

func TestToneFor(t *testing.T) {
	if toneFor("👋", ToneDark) != ToneDark {
		t.Fatal("people take tone")
	}
	if toneFor("🔥", ToneDark) != ToneNone {
		t.Fatal("symbols ignore tone")
	}
}
