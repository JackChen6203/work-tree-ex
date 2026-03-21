package sync

import (
  "net/http"
  "net/http/httptest"
  "testing"

  "github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
  gin.SetMode(gin.TestMode)
  r := gin.New()
  v1 := r.Group("/api/v1")
  RegisterRoutes(v1)
  return r
}

func TestBootstrapSuccess(t *testing.T) {
  r := setupRouter()

  req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/bootstrap?tripId=kyoto-2026&sinceVersion=0", nil)
  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)

  if w.Code != http.StatusOK {
    t.Fatalf("expected 200, got %d", w.Code)
  }
}

func TestBootstrapRejectsInvalidSinceVersion(t *testing.T) {
  r := setupRouter()

  req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/bootstrap?sinceVersion=abc", nil)
  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)

  if w.Code != http.StatusBadRequest {
    t.Fatalf("expected 400, got %d", w.Code)
  }
}
