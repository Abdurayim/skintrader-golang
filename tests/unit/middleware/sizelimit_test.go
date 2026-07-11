package middleware_test

import (
	"bytes"
	"crypto/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"skintrader-go/internal/middleware"
)

// buildMultipartBody creates a multipart form with a "file" part of the given
// size plus a "documentType" field, mirroring the KYC upload request.
func buildMultipartBody(t *testing.T, fileSize int) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("documentType", "selfie"); err != nil {
		t.Fatalf("write field: %v", err)
	}

	part, err := writer.CreateFormFile("file", "selfie.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	payload := make([]byte, fileSize)
	if _, err := rand.Read(payload); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func newUploadRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestSizeLimit(15 * 1024 * 1024))
	r.POST("/upload", func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "File is required"})
			return
		}
		defer file.Close()
		docType := c.PostForm("documentType")
		c.JSON(http.StatusOK, gin.H{"success": true, "filename": header.Filename, "documentType": docType})
	})
	return r
}

func TestRequestSizeLimit_MultipartUploads(t *testing.T) {
	r := newUploadRouter()

	cases := []struct {
		name       string
		fileSize   int
		wantStatus int
	}{
		{"small 1MB", 1 * 1024 * 1024, http.StatusOK},
		{"typical 6MB", 6 * 1024 * 1024, http.StatusOK},
		{"frontend cap 10MB", 10 * 1024 * 1024, http.StatusOK},
		{"just under limit 14MB", 14 * 1024 * 1024, http.StatusOK},
		{"over limit 16MB", 16 * 1024 * 1024, http.StatusRequestEntityTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, contentType := buildMultipartBody(t, tc.fileSize)
			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			req.Header.Set("Content-Type", contentType)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestRequestSizeLimit_ChunkedMultipart simulates a request without
// Content-Length (as proxies sometimes forward), which the old limitedReader
// implementation did not handle.
func TestRequestSizeLimit_ChunkedMultipart(t *testing.T) {
	r := newUploadRouter()

	body, contentType := buildMultipartBody(t, 6*1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = -1 // unknown length / chunked

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}
