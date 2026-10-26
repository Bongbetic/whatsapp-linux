package convo

import (
	"testing"
	"time"
)

func TestDayLabel(t *testing.T) {
	now := time.Now()
	if got := dayLabel(now); got != "Today" {
		t.Fatalf("got %q", got)
	}
	if got := dayLabel(now.Add(-24 * time.Hour)); got != "Yesterday" {
		t.Fatalf("got %q", got)
	}
	old := now.Add(-30 * 24 * time.Hour)
	if got := dayLabel(old); got != old.Format("02/01/2006") {
		t.Fatalf("got %q", got)
	}
}
