package voip

import "testing"

func TestOutgoingLifecycle(t *testing.T) {
	var stages []string
	sw := NewSwitch(func(c Call) { stages = append(stages, c.Stage) })
	if _, err := sw.Offer("c1", "chat", "audio"); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func() error{
		func() error { return sw.PreAccept("c1") },
		func() error { return sw.Transport("c1") },
		func() error { return sw.Connected("c1") },
		func() error { return sw.Hangup("c1") },
		func() error { return sw.End("c1", "answered") },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	c, _ := sw.Get("c1")
	if c.Stage != StageEnded || c.Outcome != "answered" {
		t.Fatalf("%+v", c)
	}
}

func TestIncomingReject(t *testing.T) {
	sw := NewSwitch(nil)
	if _, err := sw.Ring("c2", "chat", "video"); err != nil {
		t.Fatal(err)
	}
	if err := sw.Hangup("c2"); err != nil { // local reject
		t.Fatal(err)
	}
	if err := sw.End("c2", "rejected"); err != nil {
		t.Fatal(err)
	}
}

func TestIllegalTransitions(t *testing.T) {
	sw := NewSwitch(nil)
	if _, err := sw.Offer("c3", "chat", "audio"); err != nil {
		t.Fatal(err)
	}
	if err := sw.Connected("c3"); err == nil {
		t.Fatal("offering → in-call must be illegal")
	}
	if err := sw.End("nope", "failed"); err == nil {
		t.Fatal("unknown call must error")
	}
	if err := sw.End("c3", "missed"); err != nil {
		t.Fatal(err)
	}
	if err := sw.End("c3", "missed"); err != nil {
		t.Fatal("double end must be idempotent")
	}
}
