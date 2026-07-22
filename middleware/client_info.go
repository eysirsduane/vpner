package middleware

import (
	"strconv"
	"strings"
	"time"

	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
)

const ContextClientInfoKey = "client_info"

type ClientInfo struct {
	Platform   string
	Version    string
	Build      int
	DeviceNo   string
	ClientTime time.Time
	TimeZone   string
}

func ClientInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientTime := strings.TrimSpace(headerValue(c, "client_time"))
		timeZone := strings.TrimSpace(headerValue(c, "time_zone"))
		info := ClientInfo{
			Platform:   normalizePlatform(headerValue(c, "platform")),
			Version:    strings.TrimSpace(headerValue(c, "version")),
			Build:      headerIntValue(c, "build"),
			DeviceNo:   strings.TrimSpace(headerValue(c, "device_no")),
			ClientTime: parseClientTime(clientTime),
			TimeZone:   timeZone,
		}
		c.Set(ContextClientInfoKey, info)
		c.Next()
	}
}

func CurrentClientInfo(c *gin.Context) ClientInfo {
	value, ok := c.Get(ContextClientInfoKey)
	if !ok {
		return ClientInfo{}
	}
	info, ok := value.(ClientInfo)
	if !ok {
		return ClientInfo{}
	}
	return info
}

func headerValue(c *gin.Context, field string) string {
	return c.GetHeader(mapping.HeaderField(field))
}

func headerIntValue(c *gin.Context, field string) int {
	value := strings.TrimSpace(headerValue(c, field))
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func parseClientTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func normalizePlatform(platform string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	switch platform {
	case "ios", "iphone", "ipad":
		return "iphone"
	case "android":
		return "android"
	default:
		return platform
	}
}
