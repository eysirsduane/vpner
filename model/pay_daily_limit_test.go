package model

import "testing"

func TestComputeDailyH5AppleLimit(t *testing.T) {
	tests := []struct {
		name          string
		totalRevenue  int64
		minimumAmount int64
		percent       int
		want          int64
	}{
		{name: "percentage exceeds minimum", totalRevenue: 100000, minimumAmount: 10000, percent: 30, want: 30000},
		{name: "minimum exceeds percentage", totalRevenue: 10000, minimumAmount: 5000, percent: 30, want: 5000},
		{name: "zero percentage uses minimum", totalRevenue: 100000, minimumAmount: 8000, percent: 0, want: 8000},
		{name: "both disabled", totalRevenue: 100000, minimumAmount: 0, percent: 0, want: 0},
		{name: "percentage above range is disabled", totalRevenue: 10000, minimumAmount: 0, percent: 120, want: 0},
		{name: "negative inputs are normalized", totalRevenue: -1, minimumAmount: -1, percent: -1, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ComputeDailyH5AppleLimit(test.totalRevenue, test.minimumAmount, test.percent)
			if got != test.want {
				t.Fatalf("ComputeDailyH5AppleLimit() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestParseH5AppleAmountPercent(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{value: "7", want: 7},
		{value: "100", want: 100},
		{value: "", want: 0},
		{value: "0", want: 0},
		{value: "-1", want: 0},
		{value: "101", want: 0},
		{value: "invalid", want: 0},
	}

	for _, test := range tests {
		if got := parseH5AppleAmountPercent(test.value); got != test.want {
			t.Fatalf("parseH5AppleAmountPercent(%q) = %d, want %d", test.value, got, test.want)
		}
	}
}
