package controller

import (
	"testing"
	"time"
)

func TestIsWithinRegistrationTrial(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.Local)
	tests := []struct {
		name            string
		registerTime    time.Time
		trialSeconds    int
		wantWithinTrial bool
	}{
		{name: "within trial", registerTime: now.Add(-30 * time.Minute), trialSeconds: 3600, wantWithinTrial: true},
		{name: "at trial expiry", registerTime: now.Add(-time.Hour), trialSeconds: 3600, wantWithinTrial: false},
		{name: "after trial expiry", registerTime: now.Add(-2 * time.Hour), trialSeconds: 3600, wantWithinTrial: false},
		{name: "trial disabled", registerTime: now, trialSeconds: 0, wantWithinTrial: false},
		{name: "invalid register time", registerTime: time.Time{}, trialSeconds: 3600, wantWithinTrial: false},
		{name: "future register time", registerTime: now.Add(time.Second), trialSeconds: 3600, wantWithinTrial: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWithinRegistrationTrial(tt.registerTime, tt.trialSeconds, now)
			if got != tt.wantWithinTrial {
				t.Fatalf("isWithinRegistrationTrial() = %v, want %v", got, tt.wantWithinTrial)
			}
		})
	}
}
