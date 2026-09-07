package task

import (
	"testing"
	"time"
)

func TestParseNodePullInterval(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "configured seconds", value: "60", want: time.Minute},
		{name: "trim spaces", value: " 10 ", want: 10 * time.Second},
		{name: "empty", value: "", want: defaultNodePullInterval},
		{name: "invalid", value: "invalid", want: defaultNodePullInterval},
		{name: "zero", value: "0", want: defaultNodePullInterval},
		{name: "negative", value: "-1", want: defaultNodePullInterval},
		{name: "overflow", value: "18446744073709551615", want: defaultNodePullInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNodePullInterval(tt.value); got != tt.want {
				t.Fatalf("parseNodePullInterval(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}
