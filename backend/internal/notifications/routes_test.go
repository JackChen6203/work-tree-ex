package notifications

import (
  "net/http"
  "net/http/httptest"
  "testing"

  "github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
  gin.SetMode(gin.TestMode)
  notificationsMu.Lock()
  items = []notification{
    {ID: "n-1", Type: "ai_plan_ready", Title: "AI draft 已完成", Body: "候選方案可比較", Link: "/trips/kyoto-2026/ai-planner"},
    {ID: "n-2", Type: "member_joined", Title: "成員接受邀請", Body: "Mina 已加入行程", Link: "/trips/kyoto-2026"},
  }
  notificationsMu.Unlock()

  r := gin.New()
  v1 := r.Group("/api/v1")
  RegisterRoutes(v1)
  return r
}

func TestListAndReadNotification(t *testing.T) {
  r := setupRouter()

  listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
  listW := httptest.NewRecorder()
  r.ServeHTTP(listW, listReq)

  if listW.Code != http.StatusOK {
    t.Fatalf("expected 200 list, got %d", listW.Code)
  }

  readReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/n-1/read", nil)
  readW := httptest.NewRecorder()
  r.ServeHTTP(readW, readReq)

  if readW.Code != http.StatusNoContent {
    t.Fatalf("expected 204 mark-read, got %d", readW.Code)
  }
}
