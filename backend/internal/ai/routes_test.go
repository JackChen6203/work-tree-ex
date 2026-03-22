package ai

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
	plannerMu.Lock()
	plansByTrip = map[string][]planDraft{}
	planByID = map[string]planDraft{}
	createIdempotency = map[string]string{}
	adoptIdempotency = map[string]gin.H{}
	plannerMu.Unlock()

	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1)
	return r
}

func TestCreatePlanValidationAndAdoptFlow(t *testing.T) {
	r := setupRouter()

	body := mustJSON(t, map[string]any{
		"providerConfigId": "cfg_1",
		"title":            "Packed test",
		"constraints": map[string]any{
			"totalBudget": 10000,
			"currency":    "JPY",
			"pace":        "packed",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/ai/plans", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "ai-create-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	var created struct {
		Data struct {
			JobID string `json:"jobId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/trip-1/ai/plans", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", listW.Code)
	}

	var listResp struct {
		Data []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Data) != 1 || listResp.Data[0].Status != "invalid" {
		t.Fatalf("expected one invalid draft")
	}

	adoptReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/ai/plans/"+created.Data.JobID+"/adopt", nil)
	adoptReq.Header.Set("Idempotency-Key", "adopt-1")
	adoptW := httptest.NewRecorder()
	r.ServeHTTP(adoptW, adoptReq)

	if adoptW.Code != http.StatusConflict {
		t.Fatalf("expected 409 for invalid draft adopt, got %d", adoptW.Code)
	}
}

func TestCreatePlanIdempotencyAndValidAdopt(t *testing.T) {
	r := setupRouter()

	body := mustJSON(t, map[string]any{
		"providerConfigId": "cfg_2",
		"title":            "Relaxed test",
		"constraints": map[string]any{
			"totalBudget": 20000,
			"currency":    "JPY",
			"pace":        "relaxed",
		},
	})

	create := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-2/ai/plans", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "ai-create-2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", w.Code)
		}
		var resp struct {
			Data struct {
				JobID string `json:"jobId"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		return resp.Data.JobID
	}

	first := create()
	second := create()
	if first != second {
		t.Fatalf("expected same job id for idempotent create")
	}

	adoptReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-2/ai/plans/"+first+"/adopt", nil)
	adoptReq.Header.Set("Idempotency-Key", "adopt-2")
	adoptW := httptest.NewRecorder()
	r.ServeHTTP(adoptW, adoptReq)
	if adoptW.Code != http.StatusOK {
		t.Fatalf("expected 200 adopt, got %d", adoptW.Code)
	}
}

func TestGetPlanSuccessAndNotFound(t *testing.T) {
	r := setupRouter()

	body := mustJSON(t, map[string]any{
		"providerConfigId": "cfg_3",
		"title":            "Inspect test",
		"constraints": map[string]any{
			"totalBudget": 18000,
			"currency":    "JPY",
			"pace":        "balanced",
		},
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-3/ai/plans", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Idempotency-Key", "ai-create-3")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusAccepted {
		t.Fatalf("expected 202 create, got %d", createW.Code)
	}

	var created struct {
		Data struct {
			JobID string `json:"jobId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/trip-3/ai/plans/"+created.Data.JobID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 get, got %d", getW.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/trip-3/ai/plans/not-found", nil)
	missingW := httptest.NewRecorder()
	r.ServeHTTP(missingW, missingReq)
	if missingW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 get missing, got %d", missingW.Code)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	return data
}
