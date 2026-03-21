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
    "currency": "JPY",
    "categories": []map[string]any{{"category": "food", "plannedAmount": 15000}},
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
    "amount": 60000,
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
      OverBudget bool    `json:"overBudget"`
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
    "currency": "JP",
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

func mustJSON(t *testing.T, value any) []byte {
  t.Helper()

  data, err := json.Marshal(value)
  if err != nil {
    t.Fatalf("marshal json: %v", err)
  }

  return data
}
