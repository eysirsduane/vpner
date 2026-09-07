package task

import (
	"testing"
	"time"
)

func TestNextDailyH5AppleLimitRun(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before daily run",
			now:  time.Date(2026, 9, 1, 18, 0, 0, 0, location),
			want: time.Date(2026, 9, 1, 23, 59, 0, 0, location),
		},
		{
			name: "at daily run",
			now:  time.Date(2026, 9, 1, 23, 59, 0, 0, location),
			want: time.Date(2026, 9, 2, 23, 59, 0, 0, location),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := nextDailyH5AppleLimitRun(test.now)
			if !got.Equal(test.want) {
				t.Fatalf("nextDailyH5AppleLimitRun() = %v, want %v", got, test.want)
			}
		})
	}
}
