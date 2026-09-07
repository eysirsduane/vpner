package middleware

import (
	"bytes"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"just-vpn/model"
	"just-vpn/pkg/jwt"

	"github.com/gin-gonic/gin"
)

type debugResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *debugResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *debugResponseWriter) WriteString(data string) (int, error) {
	w.body.WriteString(data)
	return w.ResponseWriter.WriteString(data)
}

func RequestResponseDebugMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requestResponseDebugEnabled(model.ConfigValue(model.ConfigRequestResponseDebug, "off")) {
			c.Next()
			return
		}

		debugUsers := model.ConfigValue(model.ConfigRequestResponseDebugUsers, "")
		if strings.TrimSpace(debugUsers) == "" {
			c.Next()
			return
		}
		token := bearerToken(c)
		if token == "" {
			c.Next()
			return
		}
		claims, err := jwt.Parse(token)
		if err != nil || !debugUserSelected(debugUsers, claims.UserId) {
			c.Next()
			return
		}

		captureRequestResponse(c, claims.UserId)
	}
}

func captureRequestResponse(c *gin.Context, userID int) {
	requestBody, err := readAndRestoreRequestBody(c)
	if err != nil {
		log.Printf("[request_debug] user_id=%d method=%s path=%s read_body_error=%v", userID, c.Request.Method, c.Request.URL.Path, err)
	}
	log.Printf(
		"[request_debug] user_id=%d client_ip=%s method=%s path=%s query=%q body=%q",
		userID,
		c.ClientIP(),
		c.Request.Method,
		c.Request.URL.Path,
		c.Request.URL.RawQuery,
		requestBody,
	)

	writer := &debugResponseWriter{ResponseWriter: c.Writer}
	c.Writer = writer
	startedAt := time.Now()
	c.Next()

	log.Printf(
		"[response_debug] user_id=%d method=%s path=%s http_status=%d duration=%s body=%q",
		userID,
		c.Request.Method,
		c.Request.URL.Path,
		writer.Status(),
		time.Since(startedAt),
		writer.body.String(),
	)
}

func requestResponseDebugEnabled(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "on")
}

func debugUserSelected(value string, userID int) bool {
	target := strconv.Itoa(userID)
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func readAndRestoreRequestBody(c *gin.Context) (string, error) {
	if c.Request.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return string(body), nil
}
