package model

import (
	"testing"
	"time"
)

func TestPopupCanShowToRegisteredUser(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name         string
		popup        Popup
		registerTime time.Time
		want         bool
	}{
		{
			name:         "default remains visible to existing users",
			popup:        Popup{},
			registerTime: now.Add(-30 * 24 * time.Hour),
			want:         true,
		},
		{
			name:         "new user is visible",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 24},
			registerTime: now.Add(-23 * time.Hour),
			want:         true,
		},
		{
			name:         "boundary is visible",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 24},
			registerTime: now.Add(-24 * time.Hour),
			want:         true,
		},
		{
			name:         "existing user is hidden",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 24},
			registerTime: now.Add(-24*time.Hour - time.Second),
			want:         false,
		},
		{
			name:         "non-positive window is hidden",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 0},
			registerTime: now,
			want:         false,
		},
		{
			name:         "missing registration time is hidden",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 24},
			registerTime: time.Time{},
			want:         false,
		},
		{
			name:         "future registration time is hidden",
			popup:        Popup{OnlyNewUser: 1, NewUserHours: 24},
			registerTime: now.Add(time.Second),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.popup.CanShowToRegisteredUser(tt.registerTime, now); got != tt.want {
				t.Fatalf("CanShowToRegisteredUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
