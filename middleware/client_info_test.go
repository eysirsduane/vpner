package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		if info.AppStoreRegion != "CHN" {
			t.Fatalf("app store region = %q, want %q", info.AppStoreRegion, "CHN")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mapping.HeaderField("client_time"), "2026-07-11T20:30:15.123+08:00")
	req.Header.Set(mapping.HeaderField("time_zone"), "Asia/Shanghai")
	req.Header.Set(mapping.HeaderField("app_store_region"), " CHN ")
	router.ServeHTTP(httptest.NewRecorder(), req)
}

func TestNormalizeAppStoreRegionLimitsLength(t *testing.T) {
	if got := normalizeAppStoreRegion(" USA "); got != "USA" {
		t.Fatalf("normalizeAppStoreRegion() = %q, want %q", got, "USA")
	}
	if got := []rune(normalizeAppStoreRegion(strings.Repeat("界", 65))); len(got) != 64 {
		t.Fatalf("normalized app store region length = %d, want 64", len(got))
	}
}

func TestCORSAllowsMappedAppStoreRegionHeader(t *testing.T) {
	router := gin.New()
	router.Use(CORSMiddleware())
	router.OPTIONS("/", func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	allowedHeaders := response.Header().Get("Access-Control-Allow-Headers")
	want := mapping.HeaderField("app_store_region")
	if !strings.Contains(allowedHeaders, want) {
		t.Fatalf("allowed headers = %q, want mapped header %q", allowedHeaders, want)
	}
}
