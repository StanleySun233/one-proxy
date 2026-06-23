package probe

import (
	"testing"
	"time"
)

func TestDurationMsRoundsSubMillisecondUp(t *testing.T) {
	if got := durationMs(time.Nanosecond); got != 1 {
		t.Fatalf("durationMs(time.Nanosecond) = %d, want 1", got)
	}
}

func TestDurationMsKeepsZeroForInvalidDuration(t *testing.T) {
	if got := durationMs(0); got != 0 {
		t.Fatalf("durationMs(0) = %d, want 0", got)
	}
}
