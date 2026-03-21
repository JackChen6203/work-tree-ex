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
	resetMemberStoreForTests()
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

func TestAddAndListTripMembers(t *testing.T) {
	r := setupRouter()
	tripID := createTripForTest(t, r, "idem-members")

	addBody := map[string]any{
		"email":       "friend@example.com",
		"displayName": "Friend",
		"role":        "editor",
	}
	addJSON := mustMarshal(t, addBody)

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID+"/members", bytes.NewBuffer(addJSON))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Idempotency-Key", "member-add-1")
	addW := httptest.NewRecorder()
	r.ServeHTTP(addW, addReq)
	if addW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", addW.Code, addW.Body.String())
	}

	idemReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID+"/members", bytes.NewBuffer(addJSON))
	idemReq.Header.Set("Content-Type", "application/json")
	idemReq.Header.Set("Idempotency-Key", "member-add-1")
	idemW := httptest.NewRecorder()
	r.ServeHTTP(idemW, idemReq)
	if idemW.Code != http.StatusOK {
		t.Fatalf("expected 200 for idempotent replay, got %d body=%s", idemW.Code, idemW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID+"/members", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listW.Code, listW.Body.String())
	}

	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listed); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listed.Data) != 1 {
		t.Fatalf("expected 1 member, got %d", len(listed.Data))
	}
}

func TestAddTripMemberValidation(t *testing.T) {
	r := setupRouter()
	tripID := createTripForTest(t, r, "idem-members-validation")

	badRoleBody := map[string]any{
		"email": "friend@example.com",
		"role":  "admin",
	}
	badRoleJSON := mustMarshal(t, badRoleBody)

	badRoleReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID+"/members", bytes.NewBuffer(badRoleJSON))
	badRoleReq.Header.Set("Content-Type", "application/json")
	badRoleReq.Header.Set("Idempotency-Key", "member-add-invalid-role")
	badRoleW := httptest.NewRecorder()
	r.ServeHTTP(badRoleW, badRoleReq)
	if badRoleW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", badRoleW.Code, badRoleW.Body.String())
	}

	missingIdentityBody := map[string]any{
		"role": "viewer",
	}
	missingIdentityJSON := mustMarshal(t, missingIdentityBody)

	missingIdentityReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID+"/members", bytes.NewBuffer(missingIdentityJSON))
	missingIdentityReq.Header.Set("Content-Type", "application/json")
	missingIdentityReq.Header.Set("Idempotency-Key", "member-add-missing-identity")
	missingIdentityW := httptest.NewRecorder()
	r.ServeHTTP(missingIdentityW, missingIdentityReq)
	if missingIdentityW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", missingIdentityW.Code, missingIdentityW.Body.String())
	}
}

func createTripForTest(t *testing.T, r *gin.Engine, idempotencyKey string) string {
	t.Helper()

	createBody := map[string]any{
		"name":            "Members Trip",
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
	req.Header.Set("Idempotency-Key", idempotencyKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	tripID, ok := created.Data["id"].(string)
	if !ok || tripID == "" {
		t.Fatalf("expected created trip id")
	}

	return tripID
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}

	return data
}
