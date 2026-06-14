package runtime

import (
	"testing"
	"time"
)

func TestNextTick(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 4, 500*1000*1000, time.UTC)
	got := nextTick(now, 10*time.Second)
	want := time.Date(2026, 6, 14, 12, 0, 10, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("nextTick = %s, want %s", got, want)
	}
}
