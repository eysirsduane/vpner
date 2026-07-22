package task

import (
	"testing"
	"time"
)

func TestNextMidnight(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 10, 18, 30, 0, 0, location)
	want := time.Date(2026, 7, 11, 0, 0, 0, 0, location)

	got := nextMidnight(now)
	if !got.Equal(want) {
		t.Fatalf("nextMidnight() = %v, want %v", got, want)
	}
}
