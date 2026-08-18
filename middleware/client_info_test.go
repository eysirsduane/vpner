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
		if info.Language != "zh-Hans" {
			t.Fatalf("language = %q, want %q", info.Language, "zh-Hans")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mapping.HeaderField("client_time"), "2026-07-11T20:30:15.123+08:00")
	req.Header.Set(mapping.HeaderField("time_zone"), "Asia/Shanghai")
	req.Header.Set(mapping.HeaderField("app_store_region"), " CHN ")
	req.Header.Set(mapping.HeaderField("language"), " ZH-HANS ")
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

func TestNormalizeLanguage(t *testing.T) {
	tests := map[string]string{
		" zh-Hans ": "zh-Hans",
		"ZH-HANS":   "zh-Hans",
		"EN":        "en",
		"zh":        "",
		"zh-CN":     "",
		"":          "",
	}
	for input, want := range tests {
		if got := normalizeLanguage(input); got != want {
			t.Fatalf("normalizeLanguage(%q) = %q, want %q", input, got, want)
		}
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
	for _, field := range []string{"app_store_region", "language"} {
		want := mapping.HeaderField(field)
		if !strings.Contains(allowedHeaders, want) {
			t.Fatalf("allowed headers = %q, want mapped header %q", allowedHeaders, want)
		}
	}
}
