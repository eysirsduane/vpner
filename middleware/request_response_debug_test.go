package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestResponseDebugEnabled(t *testing.T) {
	for _, value := range []string{"on", "ON", " on "} {
		if !requestResponseDebugEnabled(value) {
			t.Fatalf("requestResponseDebugEnabled(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", "off", "true", "1"} {
		if requestResponseDebugEnabled(value) {
			t.Fatalf("requestResponseDebugEnabled(%q) = true, want false", value)
		}
	}
}

func TestDebugUserSelected(t *testing.T) {
	if !debugUserSelected("123, 456,789", 456) {
		t.Fatal("expected user 456 to be selected")
	}
	if debugUserSelected("123,456,789", 45) {
		t.Fatal("user ID must match a complete comma-separated item")
	}
	if debugUserSelected("", 456) {
		t.Fatal("empty user list must not select a user")
	}
}

func TestReadAndRestoreRequestBody(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/debug", strings.NewReader(`{"value":1}`))

	body, err := readAndRestoreRequestBody(c)
	if err != nil {
		t.Fatalf("readAndRestoreRequestBody() error = %v", err)
	}
	if body != `{"value":1}` {
		t.Fatalf("body = %q", body)
	}
	restored, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(restored) != body {
		t.Fatalf("restored body = %q, want %q", restored, body)
	}
}

func TestDebugResponseWriterCapturesResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	writer := &debugResponseWriter{ResponseWriter: c.Writer}

	if _, err := writer.WriteString(`{"code":200}`); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if writer.body.String() != `{"code":200}` {
		t.Fatalf("captured body = %q", writer.body.String())
	}
	if recorder.Body.String() != `{"code":200}` {
		t.Fatalf("client body = %q", recorder.Body.String())
	}
}
