package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	notificationsMu.Lock()
	items = []notification{
		{ID: "n-1", Type: "ai_plan_ready", Title: "AI draft 已完成", Body: "候選方案可比較", Link: "/dashboard"},
		{ID: "n-2", Type: "member_joined", Title: "成員接受邀請", Body: "Mina 已加入行程", Link: "/trips"},
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

func TestMarkAllNotificationsRead(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 mark-all-read, got %d", w.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", listW.Code)
	}

	notificationsMu.RLock()
	defer notificationsMu.RUnlock()
	for _, item := range items {
		if item.ReadAt == nil {
			t.Fatalf("expected notification %s to be read", item.ID)
		}
	}
}

func TestDeleteNotification(t *testing.T) {
	r := setupRouter()

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/n-1", nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete, got %d", deleteW.Code)
	}

	notificationsMu.RLock()
	defer notificationsMu.RUnlock()
	if len(items) != 1 {
		t.Fatalf("expected 1 notification left, got %d", len(items))
	}
	if items[0].ID != "n-2" {
		t.Fatalf("expected n-2 remaining, got %s", items[0].ID)
	}
}

func TestDeleteNotificationNotFound(t *testing.T) {
	r := setupRouter()

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/nope", nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 delete missing, got %d", deleteW.Code)
	}
}

func TestListNotificationsUnreadOnly(t *testing.T) {
	r := setupRouter()

	now := time.Now().UTC()
	notificationsMu.Lock()
	items[1].ReadAt = &now
	notificationsMu.Unlock()

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?unreadOnly=true", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list unread-only, got %d", listW.Code)
	}

	notificationsMu.RLock()
	defer notificationsMu.RUnlock()
	if len(items) != 2 {
		t.Fatalf("setup should keep 2 items, got %d", len(items))
	}

	if body := listW.Body.String(); body == "" || !strings.Contains(body, "n-1") || strings.Contains(body, "n-2") {
		t.Fatalf("expected only unread n-1 in payload, got %s", body)
	}
}

func TestMarkNotificationUnread(t *testing.T) {
	r := setupRouter()

	readReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/n-1/read", nil)
	readW := httptest.NewRecorder()
	r.ServeHTTP(readW, readReq)
	if readW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 read, got %d", readW.Code)
	}

	unreadReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/n-1/unread", nil)
	unreadW := httptest.NewRecorder()
	r.ServeHTTP(unreadW, unreadReq)
	if unreadW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 unread, got %d", unreadW.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", listW.Code)
	}

	var resp struct {
		Data []notification `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(resp.Data) < 1 || resp.Data[0].ID != "n-1" {
		t.Fatalf("expected n-1 in list")
	}
	if resp.Data[0].ReadAt != nil {
		t.Fatalf("expected n-1 readAt to be nil after mark unread")
	}
}
