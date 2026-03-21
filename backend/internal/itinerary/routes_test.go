package itinerary

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
  itineraryMu.Lock()
  daysByTrip = map[string][]itineraryDay{}
  itemByID = map[string]itineraryItem{}
  itemTripByID = map[string]string{}
  itemCreateIdempotency = map[string]string{}
  reorderIdempotency = map[string]string{}
  itineraryMu.Unlock()

  r := gin.New()
  v1 := r.Group("/api/v1")
  RegisterRoutes(v1)
  return r
}

func TestListCreatePatchDeleteItem(t *testing.T) {
  r := setupRouter()

  listReq := httptest.NewRequest(http.MethodGet, "/api/v1/trips/t-it/days", nil)
  listW := httptest.NewRecorder()
  r.ServeHTTP(listW, listReq)
  if listW.Code != http.StatusOK {
    t.Fatalf("expected 200 list, got %d", listW.Code)
  }

  createBody := mustJSON(t, map[string]any{
    "dayId": "day-1",
    "title": "新行程",
    "itemType": "custom",
    "allDay": false,
  })
  createReq := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-it/items", bytes.NewBuffer(createBody))
  createReq.Header.Set("Content-Type", "application/json")
  createReq.Header.Set("Idempotency-Key", "it-create-1")
  createW := httptest.NewRecorder()
  r.ServeHTTP(createW, createReq)
  if createW.Code != http.StatusCreated {
    t.Fatalf("expected 201 create, got %d", createW.Code)
  }

  var created struct {
    Data struct {
      ID string `json:"id"`
      Version int `json:"version"`
    } `json:"data"`
  }
  if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
    t.Fatalf("decode create response: %v", err)
  }

  patchBody := mustJSON(t, map[string]any{"title": "更新行程"})
  patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/trips/t-it/items/"+created.Data.ID, bytes.NewBuffer(patchBody))
  patchReq.Header.Set("Content-Type", "application/json")
  patchReq.Header.Set("If-Match-Version", "1")
  patchW := httptest.NewRecorder()
  r.ServeHTTP(patchW, patchReq)
  if patchW.Code != http.StatusOK {
    t.Fatalf("expected 200 patch, got %d", patchW.Code)
  }

  deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/trips/t-it/items/"+created.Data.ID, nil)
  deleteW := httptest.NewRecorder()
  r.ServeHTTP(deleteW, deleteReq)
  if deleteW.Code != http.StatusNoContent {
    t.Fatalf("expected 204 delete, got %d", deleteW.Code)
  }
}

func TestReorderItems(t *testing.T) {
  r := setupRouter()

  body := mustJSON(t, map[string]any{
    "operations": []map[string]any{{
      "itemId": "i-1",
      "targetDayId": "day-2",
      "targetSortOrder": 1,
    }},
  })
  req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/t-r/items/reorder", bytes.NewBuffer(body))
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Idempotency-Key", "it-reorder-1")
  w := httptest.NewRecorder()
  r.ServeHTTP(w, req)
  if w.Code != http.StatusOK {
    t.Fatalf("expected 200 reorder, got %d", w.Code)
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
