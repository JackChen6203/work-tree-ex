package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	SetPool(nil)
	resetPushGatewayForTest()
	notificationsMu.Lock()
	items = []notification{
		{ID: "n-1", Type: "ai_plan_ready", Title: "AI draft 已完成", Body: "候選方案可比較", Link: "/dashboard"},
		{ID: "n-2", Type: "member_joined", Title: "成員接受邀請", Body: "Mina 已加入行程", Link: "/trips"},
	}
	dedupeStore = map[string]time.Time{}
	pushDeliveries = map[string]*pushDelivery{}
	fcmTokensByToken = map[string]fcmToken{}
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

func TestListNotificationsPagination(t *testing.T) {
	r := setupRouter()

	notificationsMu.Lock()
	items = []notification{
		{ID: "n-1", Type: "ai_plan_ready", Title: "AI 1", Body: "Body 1", Link: "/dashboard"},
		{ID: "n-2", Type: "member_joined", Title: "Join 2", Body: "Body 2", Link: "/trips"},
		{ID: "n-3", Type: "budget_alert", Title: "Budget 3", Body: "Body 3", Link: "/trips/t-1/budget"},
	}
	notificationsMu.Unlock()

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=1", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list limit, got %d body=%s", listW.Code, listW.Body.String())
	}

	var firstPage struct {
		Data []notification `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if len(firstPage.Data) != 1 || firstPage.Data[0].ID != "n-1" {
		t.Fatalf("expected first page to contain only n-1, got %+v", firstPage.Data)
	}

	cursorReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?cursor=n-1&limit=2", nil)
	cursorW := httptest.NewRecorder()
	r.ServeHTTP(cursorW, cursorReq)
	if cursorW.Code != http.StatusOK {
		t.Fatalf("expected 200 cursor list, got %d body=%s", cursorW.Code, cursorW.Body.String())
	}

	var secondPage struct {
		Data []notification `json:"data"`
	}
	if err := json.Unmarshal(cursorW.Body.Bytes(), &secondPage); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if len(secondPage.Data) != 2 || secondPage.Data[0].ID != "n-2" || secondPage.Data[1].ID != "n-3" {
		t.Fatalf("expected n-2 and n-3 after cursor, got %+v", secondPage.Data)
	}
}

func TestListNotificationsInvalidPagination(t *testing.T) {
	r := setupRouter()

	invalidLimitReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=0", nil)
	invalidLimitW := httptest.NewRecorder()
	r.ServeHTTP(invalidLimitW, invalidLimitReq)
	if invalidLimitW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid limit, got %d body=%s", invalidLimitW.Code, invalidLimitW.Body.String())
	}

	missingCursorReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?cursor=missing", nil)
	missingCursorW := httptest.NewRecorder()
	r.ServeHTTP(missingCursorW, missingCursorReq)
	if missingCursorW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 missing cursor, got %d body=%s", missingCursorW.Code, missingCursorW.Body.String())
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

func TestCleanupReadNotifications(t *testing.T) {
	r := setupRouter()

	readReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/n-2/read", nil)
	readW := httptest.NewRecorder()
	r.ServeHTTP(readW, readReq)
	if readW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 read, got %d", readW.Code)
	}

	cleanupReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/cleanup-read", nil)
	cleanupW := httptest.NewRecorder()
	r.ServeHTTP(cleanupW, cleanupReq)
	if cleanupW.Code != http.StatusOK {
		t.Fatalf("expected 200 cleanup, got %d body=%s", cleanupW.Code, cleanupW.Body.String())
	}

	var cleanupResp struct {
		Data struct {
			DeletedCount int `json:"deletedCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(cleanupW.Body.Bytes(), &cleanupResp); err != nil {
		t.Fatalf("decode cleanup response: %v", err)
	}
	if cleanupResp.Data.DeletedCount != 1 {
		t.Fatalf("expected deletedCount=1, got %d", cleanupResp.Data.DeletedCount)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", listW.Code)
	}

	var listResp struct {
		Data []notification `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Data) != 1 || listResp.Data[0].ID != "n-1" {
		t.Fatalf("expected only unread n-1 to remain")
	}
}

func TestUpsertAndDeactivateFCMToken(t *testing.T) {
	r := setupRouter()

	body := `{"token":"fcm-token-1","platform":"web","userId":"u-1"}`
	upsertReq := httptest.NewRequest(http.MethodPost, "/api/v1/fcm-tokens", strings.NewReader(body))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertW := httptest.NewRecorder()
	r.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("expected 200 upsert fcm token, got %d body=%s", upsertW.Code, upsertW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/fcm-tokens", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list fcm tokens, got %d", listW.Code)
	}
	if !strings.Contains(listW.Body.String(), "fcm-token-1") {
		t.Fatalf("expected token in list response, got %s", listW.Body.String())
	}

	deactivateReq := httptest.NewRequest(http.MethodDelete, "/api/v1/fcm-tokens/fcm-token-1", nil)
	deactivateW := httptest.NewRecorder()
	r.ServeHTTP(deactivateW, deactivateReq)
	if deactivateW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deactivate fcm token, got %d body=%s", deactivateW.Code, deactivateW.Body.String())
	}
}

func TestUpsertFCMTokenValidation(t *testing.T) {
	r := setupRouter()

	body := `{"token":"fcm-token-2","platform":"windows"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fcm-tokens", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid platform, got %d body=%s", w.Code, w.Body.String())
	}
}

type stubPushGateway struct {
	sendFn func(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error)
}

func (s stubPushGateway) Send(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error) {
	return s.sendFn(ctx, tokens, msg)
}

func TestTriggerNotificationPushRetriesToDLQ(t *testing.T) {
	r := setupRouter()

	upsertReq := httptest.NewRequest(http.MethodPost, "/api/v1/fcm-tokens", strings.NewReader(`{"token":"fcm-retry","platform":"web","userId":"u-1"}`))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertW := httptest.NewRecorder()
	r.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("expected 200 upsert token, got %d body=%s", upsertW.Code, upsertW.Body.String())
	}

	setPushGatewayForTest(stubPushGateway{
		sendFn: func(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error) {
			return pushGatewayResult{
				SuccessCount: 0,
				FailureCount: len(tokens),
				Retryable:    true,
			}, errors.New("firebase unavailable")
		},
	})

	triggerReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/trigger", strings.NewReader(`{
		"eventType":"trip_updated",
		"resourceId":"trip-1",
		"title":"Trip Updated",
		"body":"Something changed",
		"link":"/trips/trip-1",
		"userId":"u-1"
	}`))
	triggerReq.Header.Set("Content-Type", "application/json")
	triggerW := httptest.NewRecorder()
	r.ServeHTTP(triggerW, triggerReq)
	if triggerW.Code != http.StatusCreated {
		t.Fatalf("expected 201 trigger, got %d body=%s", triggerW.Code, triggerW.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/push-status", nil)
	statusW := httptest.NewRecorder()
	r.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("expected 200 push-status, got %d body=%s", statusW.Code, statusW.Body.String())
	}
	if !strings.Contains(statusW.Body.String(), `"status":"dlq"`) {
		t.Fatalf("expected dlq status, got %s", statusW.Body.String())
	}
}

func TestTriggerNotificationPushInvalidTokenDeactivated(t *testing.T) {
	r := setupRouter()

	upsertReq := httptest.NewRequest(http.MethodPost, "/api/v1/fcm-tokens", strings.NewReader(`{"token":"fcm-invalid","platform":"web","userId":"u-2"}`))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertW := httptest.NewRecorder()
	r.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("expected 200 upsert token, got %d body=%s", upsertW.Code, upsertW.Body.String())
	}

	setPushGatewayForTest(stubPushGateway{
		sendFn: func(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error) {
			return pushGatewayResult{
				SuccessCount: 0,
				FailureCount: len(tokens),
				Retryable:    false,
				InvalidTokens: []string{
					"fcm-invalid",
				},
			}, errors.New("registration token is not registered")
		},
	})

	triggerReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/trigger", strings.NewReader(`{
		"eventType":"trip_updated",
		"resourceId":"trip-2",
		"title":"Trip Updated",
		"body":"Something changed",
		"link":"/trips/trip-2",
		"userId":"u-2"
	}`))
	triggerReq.Header.Set("Content-Type", "application/json")
	triggerW := httptest.NewRecorder()
	r.ServeHTTP(triggerW, triggerReq)
	if triggerW.Code != http.StatusCreated {
		t.Fatalf("expected 201 trigger, got %d body=%s", triggerW.Code, triggerW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/fcm-tokens?userId=u-2", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 list tokens, got %d body=%s", listW.Code, listW.Body.String())
	}
	if strings.Contains(listW.Body.String(), `"isActive":true`) {
		t.Fatalf("expected token to be deactivated, got %s", listW.Body.String())
	}
}
