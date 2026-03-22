package maps

import (
	"bytes"
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

func TestSearchPlaces(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/maps/search?q=kyoto", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestEstimateRoute(t *testing.T) {
	r := setupRouter()
	body := []byte(`{"origin":{"lat":35.0,"lng":135.0},"destination":{"lat":35.01,"lng":135.02},"mode":"transit"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/maps/routes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSearchPlacesLimitValidation(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/maps/search?q=kyoto&limit=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSearchPlacesLimitApplied(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/maps/search?q=kyoto&limit=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !(len(body) > 0) || !(bytes.Count([]byte(body), []byte("providerPlaceId")) == 1) {
		t.Fatalf("expected exactly one place in payload, got %s", body)
	}
}
