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

func TestPopupCanShowToIPRegion(t *testing.T) {
	testCases := []struct {
		name   string
		popup  Popup
		region string
		want   bool
	}{
		{name: "地区限制关闭", popup: Popup{OnlyMainland: 0}, region: "995|美国|0|加利福尼亚|洛杉矶|谷歌", want: true},
		{name: "中国大陆", popup: Popup{OnlyMainland: 1}, region: "995|中国|0|上海|上海市|电信", want: true},
		{name: "香港", popup: Popup{OnlyMainland: 1}, region: "2163|中国|0|香港|0|电讯盈科", want: false},
		{name: "澳门", popup: Popup{OnlyMainland: 1}, region: "2164|中国|0|澳门|0|电讯", want: false},
		{name: "台湾", popup: Popup{OnlyMainland: 1}, region: "2165|中国|0|台湾|台北市|中华电信", want: false},
		{name: "海外", popup: Popup{OnlyMainland: 1}, region: "995|美国|0|加利福尼亚|洛杉矶|谷歌", want: false},
		{name: "未知", popup: Popup{OnlyMainland: 1}, region: "995|0|0|0|0|0", want: false},
		{name: "空值", popup: Popup{OnlyMainland: 1}, region: "", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.popup.CanShowToIPRegion(testCase.region); got != testCase.want {
				t.Fatalf("CanShowToIPRegion(%q) = %v, want %v", testCase.region, got, testCase.want)
			}
		})
	}
}
