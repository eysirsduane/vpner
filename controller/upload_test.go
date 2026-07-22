package controller

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUploadFileHandlerServesFileFromUploadDir(t *testing.T) {
	root := t.TempDir()
	uploadDir := filepath.Join(root, "upload")
	if err := os.Mkdir(uploadDir, 0o755); err != nil {
		t.Fatalf("mkdir upload dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	oldRoot := uploadRoot
	uploadRoot = uploadDir
	t.Cleanup(func() {
		uploadRoot = oldRoot
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/upload/*filepath", UploadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/upload/hello.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", rec.Body.String())
	}
}

func TestUploadFileHandlerReturnsNotFoundForMissingFile(t *testing.T) {
	oldRoot := uploadRoot
	uploadRoot = t.TempDir()
	t.Cleanup(func() {
		uploadRoot = oldRoot
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/upload/*filepath", UploadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/upload/missing.png", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
