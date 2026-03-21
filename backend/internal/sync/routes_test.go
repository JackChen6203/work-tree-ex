package sync

import (
	"encoding/json"
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/bootstrap?tripId=trip-123&sinceVersion=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			TripID       string `json:"tripId"`
			ChangedTrips []struct {
				ID string `json:"id"`
			} `json:"changedTrips"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.TripID != "trip-123" {
		t.Fatalf("expected tripId trip-123, got %s", resp.Data.TripID)
	}
	if len(resp.Data.ChangedTrips) != 1 || resp.Data.ChangedTrips[0].ID != "trip-123" {
		t.Fatalf("expected changedTrips to include trip-123")
	}
}

func TestBootstrapWithoutTripIDKeepsChangedTripsEmpty(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/bootstrap?sinceVersion=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			ChangedTrips []struct {
				ID string `json:"id"`
			} `json:"changedTrips"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data.ChangedTrips) != 0 {
		t.Fatalf("expected no changedTrips without tripId")
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
