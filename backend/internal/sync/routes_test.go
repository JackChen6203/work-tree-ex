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
	SetPool(nil)
	syncMu.Lock()
	entityVersions = map[string]int{}
	flushIdempotencyStore = map[string]gin.H{}
	latestSyncVersion = 0
	outboxEvents = []OutboxEvent{}
	outboxByID = map[string]*OutboxEvent{}
	outboxDedupeKeys = map[string]bool{}
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
	successReq.Header.Set("Idempotency-Key", "flush-1")
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
	conflictReq.Header.Set("Idempotency-Key", "flush-2")
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

func TestFlushMutationsIdempotencyReplay(t *testing.T) {
	r := setupRouter()

	body := `{"tripId":"trip-123","mutations":[{"id":"m-1","entityType":"itinerary_item","entityId":"i-1","baseVersion":0}]}`
	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(body))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "flush-replay")
	firstW := httptest.NewRecorder()
	r.ServeHTTP(firstW, firstReq)
	if firstW.Code != http.StatusOK {
		t.Fatalf("expected 200 first flush, got %d body=%s", firstW.Code, firstW.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(body))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "flush-replay")
	secondW := httptest.NewRecorder()
	r.ServeHTTP(secondW, secondReq)
	if secondW.Code != http.StatusOK {
		t.Fatalf("expected 200 second flush replay, got %d body=%s", secondW.Code, secondW.Body.String())
	}

	var firstResp struct {
		Data struct {
			AcceptedCount int `json:"acceptedCount"`
			ConflictCount int `json:"conflictCount"`
			NextVersion   int `json:"nextVersion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(firstW.Body.Bytes(), &firstResp); err != nil {
		t.Fatalf("decode first response: %v", err)
	}

	var secondResp struct {
		Data struct {
			AcceptedCount int `json:"acceptedCount"`
			ConflictCount int `json:"conflictCount"`
			NextVersion   int `json:"nextVersion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(secondW.Body.Bytes(), &secondResp); err != nil {
		t.Fatalf("decode second response: %v", err)
	}

	if firstResp.Data != secondResp.Data {
		t.Fatalf("expected idempotent replay to return same payload: first=%+v second=%+v", firstResp.Data, secondResp.Data)
	}
}

func TestFlushMutationsRequiresIdempotencyKey(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(`{"tripId":"trip-123","mutations":[{"id":"m-1","entityType":"trip","entityId":"trip-123","baseVersion":0}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing idempotency key, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestFlushMutationsValidation(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(`{"tripId":"","mutations":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "flush-validation")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestFlushCreatesOutboxEvents(t *testing.T) {
	r := setupRouter()

	body := `{"tripId":"trip-outbox","mutations":[{"id":"m-1","entityType":"itinerary_item","entityId":"i-1","baseVersion":0}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/mutations/flush", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "flush-outbox-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Check outbox events were created
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sync/outbox/events?status=pending", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}

	var resp struct {
		Data []struct {
			ID        string `json:"id"`
			EventType string `json:"eventType"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatalf("expected outbox events from flush")
	}
	if resp.Data[0].Status != "pending" {
		t.Fatalf("expected status pending, got %s", resp.Data[0].Status)
	}
}

func TestBootstrapFullResync(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/bootstrap?tripId=trip-old&sinceVersion=0", nil)
	req.Header.Set("X-Client-Version", "5") // Too old, triggers full resync
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			FullResyncRequired bool `json:"fullResyncRequired"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Data.FullResyncRequired {
		t.Fatalf("expected fullResyncRequired=true for old client")
	}
}
