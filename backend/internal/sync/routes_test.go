package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	syncMu.Lock()
	entityVersions = map[string]int{}
	latestSyncVersion = 0
	syncMu.Unlock()
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

func TestFlushMutationsSuccessAndConflict(t *testing.T) {
	r := setupRouter()

	successReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(`{"tripId":"trip-123","mutations":[{"id":"m-1","entityType":"itinerary_item","entityId":"i-1","baseVersion":0}]}`))
	successReq.Header.Set("Content-Type", "application/json")
	successW := httptest.NewRecorder()
	r.ServeHTTP(successW, successReq)
	if successW.Code != http.StatusOK {
		t.Fatalf("expected 200 flush success, got %d body=%s", successW.Code, successW.Body.String())
	}

	var successResp struct {
		Data struct {
			AcceptedCount int `json:"acceptedCount"`
			ConflictCount int `json:"conflictCount"`
			NextVersion   int `json:"nextVersion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(successW.Body.Bytes(), &successResp); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if successResp.Data.AcceptedCount != 1 || successResp.Data.ConflictCount != 0 {
		t.Fatalf("unexpected success counters: accepted=%d conflict=%d", successResp.Data.AcceptedCount, successResp.Data.ConflictCount)
	}

	conflictReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(`{"tripId":"trip-123","mutations":[{"id":"m-2","entityType":"itinerary_item","entityId":"i-1","baseVersion":0}]}`))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictW := httptest.NewRecorder()
	r.ServeHTTP(conflictW, conflictReq)
	if conflictW.Code != http.StatusOK {
		t.Fatalf("expected 200 flush conflict, got %d body=%s", conflictW.Code, conflictW.Body.String())
	}

	var conflictResp struct {
		Data struct {
			AcceptedCount int `json:"acceptedCount"`
			ConflictCount int `json:"conflictCount"`
			Conflicts     []struct {
				ID     string `json:"id"`
				Reason string `json:"reason"`
			} `json:"conflicts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(conflictW.Body.Bytes(), &conflictResp); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if conflictResp.Data.AcceptedCount != 0 || conflictResp.Data.ConflictCount != 1 {
		t.Fatalf("unexpected conflict counters: accepted=%d conflict=%d", conflictResp.Data.AcceptedCount, conflictResp.Data.ConflictCount)
	}
	if len(conflictResp.Data.Conflicts) != 1 || conflictResp.Data.Conflicts[0].Reason != "version_conflict" {
		t.Fatalf("expected version_conflict")
	}
}

func TestFlushMutationsValidation(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(`{"tripId":"","mutations":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
