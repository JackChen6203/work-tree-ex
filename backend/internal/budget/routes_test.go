package budget

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
	budgetMu.Lock()
	profilesByTrip = map[string]budgetProfile{}
	expensesByTrip = map[string][]expense{}
	budgetIdempotency = map[string]string{}
	expenseIdempotency = map[string]string{}
	expenseByID = map[string]expense{}
	budgetMu.Unlock()

	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1)
	return r
}

func TestUpsertBudgetAndExpenseFlow(t *testing.T) {
	r := setupRouter()

	budgetBody := mustJSON(t, map[string]any{
		"totalBudget": 50000,
		"currency":    "JPY",
		"categories":  []map[string]any{{"category": "food", "plannedAmount": 15000}},
	})

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/trips/t-1/budget", bytes.NewBuffer(budgetBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Idempotency-Key", "b-1")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", putW.Code)
	}

	expenseBody := mustJSON(t, map[string]any{
		"category": "food",
		"amount":   60000,
		"currency": "JPY",
	})

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-1/expenses", bytes.NewBuffer(expenseBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Idempotency-Key", "e-1")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", postW.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/t-1/budget", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var resp struct {
		Data struct {
			OverBudget  bool    `json:"overBudget"`
			ActualSpend float64 `json:"actualSpend"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Data.OverBudget {
		t.Fatalf("expected overBudget=true")
	}
	if resp.Data.ActualSpend < 60000 {
		t.Fatalf("expected actual spend >= 60000")
	}
}

func TestBudgetRequiresCurrencyAndIdempotency(t *testing.T) {
	r := setupRouter()

	body := mustJSON(t, map[string]any{
		"totalBudget": 10000,
		"currency":    "JP",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/trips/t-2/budget", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "b-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid currency, got %d", w.Code)
	}
}

func TestDeleteExpense(t *testing.T) {
	r := setupRouter()

	expenseBody := mustJSON(t, map[string]any{
		"category": "food",
		"amount":   1200,
		"currency": "JPY",
	})

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-3/expenses", bytes.NewBuffer(expenseBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Idempotency-Key", "e-delete-1")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", postW.Code)
	}

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/trips/t-3/expenses/"+created.Data.ID, nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", deleteW.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/t-3/expenses", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}

	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Data) != 0 {
		t.Fatalf("expected no expenses after delete, got %d", len(listed.Data))
	}
}

func TestDeleteExpenseNotFound(t *testing.T) {
	r := setupRouter()

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/trips/t-4/expenses/missing", nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", deleteW.Code)
	}
}

func TestPatchExpense(t *testing.T) {
	r := setupRouter()

	expenseBody := mustJSON(t, map[string]any{
		"category": "food",
		"amount":   1200,
		"currency": "JPY",
		"note":     "Lunch",
	})

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-5/expenses", bytes.NewBuffer(expenseBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Idempotency-Key", "e-patch-1")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", postW.Code)
	}

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	patchBody := mustJSON(t, map[string]any{
		"amount": 2600,
		"note":   "Dinner",
	})
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/trips/t-5/expenses/"+created.Data.ID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", patchW.Code, patchW.Body.String())
	}

	var patched struct {
		Data struct {
			Amount  float64 `json:"amount"`
			Note    string  `json:"note"`
			Version int     `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(patchW.Body.Bytes(), &patched); err != nil {
		t.Fatalf("decode patch response: %v", err)
	}
	if patched.Data.Amount != 2600 {
		t.Fatalf("expected amount 2600, got %v", patched.Data.Amount)
	}
	if patched.Data.Note != "Dinner" {
		t.Fatalf("expected note Dinner, got %s", patched.Data.Note)
	}
	if patched.Data.Version != 2 {
		t.Fatalf("expected version 2, got %d", patched.Data.Version)
	}
}

func TestPatchExpenseValidation(t *testing.T) {
	r := setupRouter()

	patchBody := mustJSON(t, map[string]any{
		"amount": -1,
	})
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/trips/t-6/expenses/not-found", bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", patchW.Code)
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

func TestGetExchangeRates(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/t-rate/budget/rates?from=USD&to=JPY", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Rate float64 `json:"rate"`
			From string  `json:"from"`
			To   string  `json:"to"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Rate < 100 {
		t.Fatalf("expected USD/JPY rate > 100, got %f", resp.Data.Rate)
	}
}

func TestGetExchangeRateNotFound(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/t-rate/budget/rates?from=ABC&to=XYZ", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRefreshExchangeRate(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-rate/budget/rates/refresh?from=USD&to=JPY", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Source != "mock-api" {
		t.Fatalf("expected source mock-api, got %s", resp.Data.Source)
	}
}
