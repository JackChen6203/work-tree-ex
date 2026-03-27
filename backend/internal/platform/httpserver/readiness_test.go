package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadyzWithoutProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	SetReadinessProbe(nil)
	defer SetReadinessProbe(nil)

	r := gin.New()
	registerRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReadyzWithFailingProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	SetReadinessProbe(func(context.Context) error {
		return errors.New("db down")
	})
	defer SetReadinessProbe(nil)

	r := gin.New()
	registerRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
