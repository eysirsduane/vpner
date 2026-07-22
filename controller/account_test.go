package controller

import (
	"testing"
	"time"

	"just-vpn/model"
)

func TestCSVContainsExact(t *testing.T) {
	tests := []struct {
		name  string
		rule  string
		value string
		want  bool
	}{
		{name: "empty rule is disabled", rule: "", value: "1.0.0", want: false},
		{name: "empty value is disabled", rule: "1.0.0", value: "", want: false},
		{name: "single version matched", rule: "1.0.0", value: "1.0.0", want: true},
		{name: "comma version matched", rule: "1.0.0,1.0.1", value: "1.0.1", want: true},
		{name: "spaces are ignored", rule: " 1.0.0, 1.0.1 ", value: "1.0.1", want: true},
		{name: "case sensitive version", rule: "1.0.0", value: "1.0.0-beta", want: false},
		{name: "all is not special", rule: "all", value: "1.0.0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := csvContainsExact(tt.rule, tt.value)
			if got != tt.want {
				t.Fatalf("csvContainsExact(%q, %q) = %v, want %v", tt.rule, tt.value, got, tt.want)
			}
		})
	}
}

func TestConfiguredMaxLoginDevices(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "default", value: "", want: 2},
		{name: "custom", value: "3", want: 3},
		{name: "invalid", value: "abc", want: 2},
		{name: "unlimited", value: "0", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configuredMaxLoginDevices(tt.value)
			if got != tt.want {
				t.Fatalf("configuredMaxLoginDevices(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsFirstMigrationActivation(t *testing.T) {
	source := "快喵-7"
	loginTime := time.Now()
	tests := []struct {
		name string
		user model.User
		want bool
	}{
		{name: "migration user never logged in", user: model.User{MigrationSource: &source}, want: true},
		{name: "migration user already logged in", user: model.User{MigrationSource: &source, LastLoginTime: &loginTime}, want: false},
		{name: "normal user never logged in", user: model.User{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFirstMigrationActivation(tt.user)
			if got != tt.want {
				t.Fatalf("isFirstMigrationActivation() = %v, want %v", got, tt.want)
			}
		})
	}
}
