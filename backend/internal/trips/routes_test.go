package trips

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	SetRepository(newMemoryRepository())
	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1)
	return r
}

func TestCreateAndGetTrip(t *testing.T) {
	r := setupRouter()

	createBody := map[string]any{
		"name":            "Tokyo Trip",
		"destinationText": "Tokyo",
		"startDate":       "2026-04-10",
		"endDate":         "2026-04-15",
		"timezone":        "Asia/Tokyo",
		"currency":        "JPY",
		"travelersCount":  2,
	}
	body := mustMarshal(t, createBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	tripID, ok := created.Data["id"].(string)
	if !ok {
		t.Fatalf("expected trip id to be a string")
	}
	if tripID == "" {
		t.Fatalf("expected trip id")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}
}

func TestCreateTripRequiresIdempotencyKey(t *testing.T) {
	r := setupRouter()

	createBody := map[string]any{
		"name":           "No Key Trip",
		"startDate":      "2026-04-10",
		"endDate":        "2026-04-15",
		"timezone":       "Asia/Taipei",
		"currency":       "TWD",
		"travelersCount": 1,
	}
	body := mustMarshal(t, createBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPatchTripVersionConflict(t *testing.T) {
	r := setupRouter()

	createBody := map[string]any{
		"name":           "Version Trip",
		"startDate":      "2026-04-10",
		"endDate":        "2026-04-15",
		"timezone":       "Asia/Tokyo",
		"currency":       "JPY",
		"travelersCount": 2,
	}
	body := mustMarshal(t, createBody)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Idempotency-Key", "idem-2")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	tripID, ok := created.Data["id"].(string)
	if !ok {
		t.Fatalf("expected trip id to be a string")
	}

	patchBody := map[string]any{"name": "Updated Name"}
	patchJSON := mustMarshal(t, patchBody)

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/trips/"+tripID, bytes.NewBuffer(patchJSON))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("If-Match-Version", "99")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)

	if patchW.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", patchW.Code)
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}

	return data
}
