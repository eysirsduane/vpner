package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
)

func TestClientInfoMiddlewareReadsMappedClientTime(t *testing.T) {
	expected, err := time.Parse(time.RFC3339Nano, "2026-07-11T20:30:15.123+08:00")
	if err != nil {
		t.Fatalf("parse expected client time: %v", err)
	}

	router := gin.New()
	router.Use(ClientInfoMiddleware())
	router.GET("/", func(c *gin.Context) {
		info := CurrentClientInfo(c)
		if !info.ClientTime.Equal(expected) {
			t.Fatalf("client time = %s, want %s", info.ClientTime, expected)
		}
		_, offset := info.ClientTime.Zone()
		if offset != 8*60*60 {
			t.Fatalf("client time offset = %d, want %d", offset, 8*60*60)
		}
		if info.TimeZone != "Asia/Shanghai" {
			t.Fatalf("time zone = %q, want %q", info.TimeZone, "Asia/Shanghai")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mapping.HeaderField("client_time"), "2026-07-11T20:30:15.123+08:00")
	req.Header.Set(mapping.HeaderField("time_zone"), "Asia/Shanghai")
	router.ServeHTTP(httptest.NewRecorder(), req)
}
