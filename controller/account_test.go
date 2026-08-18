package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
)

func TestApplyClientInfoToAutoLoginParamsIncludesAppStoreRegion(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(middleware.ContextClientInfoKey, middleware.ClientInfo{
		Version:        "1.2.3",
		AppStoreRegion: "CHN",
	})

	params := applyClientInfoToAutoLoginParams(context, autoLoginParams{})
	if params.Version != "1.2.3" {
		t.Fatalf("Version = %q, want %q", params.Version, "1.2.3")
	}
	if params.AppStoreRegion != "CHN" {
		t.Fatalf("AppStoreRegion = %q, want %q", params.AppStoreRegion, "CHN")
	}
}

func TestGetChangePasswordParamsOnlyUsesNewPassword(t *testing.T) {
	const originalPath = "/api/v1/change_password"
	fields := mapping.RequestFields(originalPath)
	if _, exists := fields["old_password"]; exists {
		t.Fatal("change password mapping must not expose old_password")
	}

	newPasswordField := mapping.RequestField(originalPath, "new_password")
	params := getChangePasswordParams(map[string]interface{}{
		newPasswordField: "333333",
	})
	if params.NewPassword != "333333" {
		t.Fatalf("NewPassword = %q, want %q", params.NewPassword, "333333")
	}
}

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
