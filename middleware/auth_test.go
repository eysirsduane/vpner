package middleware

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	jwtpkg "just-vpn/pkg/jwt"
	"just-vpn/pkg/setting"

	"github.com/gin-gonic/gin"
)

func TestAuthFailureCode(t *testing.T) {
	unauthorizedCause := errors.New("token is expired")
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "unauthorized", err: newAuthFailure(CodeUnauthorized, unauthorizedCause), want: CodeUnauthorized},
		{name: "wrapped unauthorized", err: fmt.Errorf("auth failed: %w", newAuthFailure(CodeUnauthorized, unauthorizedCause)), want: CodeUnauthorized},
		{name: "internal error", err: errors.New("database unavailable"), want: CodeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authFailureCode(tt.err); got != tt.want {
				t.Fatalf("authFailureCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAuthFailurePreservesCause(t *testing.T) {
	cause := errors.New("token is expired")
	err := newAuthFailure(CodeUnauthorized, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("auth failure does not preserve cause: %v", err)
	}
	if err.Error() != cause.Error() {
		t.Fatalf("auth failure message = %q, want %q", err.Error(), cause.Error())
	}
}

func TestAuthUserReturnsUnauthorizedForExpiredToken(t *testing.T) {
	oldSecret := setting.AppConfig.JwtSecret
	oldExpire := setting.AppConfig.JwtExpire
	setting.AppConfig.JwtSecret = "expired-auth-test-secret"
	setting.AppConfig.JwtExpire = -time.Hour
	t.Cleanup(func() {
		setting.AppConfig.JwtSecret = oldSecret
		setting.AppConfig.JwtExpire = oldExpire
	})

	token, err := jwtpkg.Generate(123, "device-001")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	if _, err := authUser(c); authFailureCode(err) != CodeUnauthorized {
		t.Fatalf("authUser() error = %v, code = %d, want %d", err, authFailureCode(err), CodeUnauthorized)
	}
}
