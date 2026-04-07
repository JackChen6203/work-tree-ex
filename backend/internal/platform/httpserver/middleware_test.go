package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCorsAllowsCsrfHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware([]string{"https://app.example.com"}))
	r.GET("/api/v1/trips", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/trips", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowedHeaders, "X-CSRF-Token") {
		t.Fatalf("expected CORS allow headers to include X-CSRF-Token, got %q", allowedHeaders)
	}
}
