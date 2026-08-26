package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type generatedBodyReader struct {
	remaining int64
}

func (r *generatedBodyReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > r.remaining {
		n = r.remaining
	}
	for i := 0; i < int(n); i++ {
		p[i] = 'x'
	}
	r.remaining -= n
	return int(n), nil
}

func TestImageBodyLimitReturnsOpenAIErrorForOversizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ImageBodyLimit())
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	req.ContentLength = maxImageRequestBody + 1
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "image request body is too large") {
		t.Fatalf("unexpected error body: %s", recorder.Body.String())
	}
}

func TestImageBodyLimitRejectsCompressedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ImageBodyLimit())
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("compressed"))
	req.Header.Set("Content-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "compressed image request bodies are not supported") {
		t.Fatalf("unexpected error body: %s", recorder.Body.String())
	}
}

func TestImageBodyLimitCapsUnknownLengthBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ImageBodyLimit())
	router.POST("/", func(c *gin.Context) {
		_, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", &generatedBodyReader{remaining: maxImageRequestBody + 1})
	if req.ContentLength >= 0 {
		t.Fatalf("test request unexpectedly has a known content length: %d", req.ContentLength)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}
