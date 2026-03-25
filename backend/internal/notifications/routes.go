package notifications

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type pushDelivery struct {
	NotificationID string    `json:"notificationId"`
	Status         string    `json:"status"` // pending | sent | failed
	RetryCount     int       `json:"retryCount"`
	LastAttemptAt  time.Time `json:"lastAttemptAt"`
}

// DeliveryPrefs per-user delivery preferences
type DeliveryPrefs struct {
	InApp bool `json:"inApp"`
	Push  bool `json:"push"`
	Email bool `json:"email"`
}

var (
	notificationsMu sync.RWMutex
	items           = []notification{
		{ID: "n-1", Type: "ai_plan_ready", Title: "AI draft 已完成", Body: "候選方案可比較並採用到行程", Link: "/dashboard", CreatedAt: time.Now().Add(-3 * time.Minute).UTC()},
		{ID: "n-2", Type: "member_joined", Title: "成員接受邀請", Body: "新成員已加入行程並取得編輯權限", Link: "/trips", CreatedAt: time.Now().Add(-1 * time.Hour).UTC()},
	}

	// Dedupe: eventType:resourceID → last trigger time
	dedupeStore  = map[string]time.Time{}
	dedupeWindow = 5 * time.Minute

	// Push delivery tracking
	pushDeliveries = map[string]*pushDelivery{}

	// Default delivery preferences (per-user, simplified to global for mock)
	defaultDeliveryPrefs = DeliveryPrefs{InApp: true, Push: true, Email: false}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/notifications", listNotifications)
	v1.POST("/notifications/read-all", markAllRead)
	v1.POST("/notifications/cleanup-read", cleanupRead)
	v1.POST("/notifications/:notificationId/read", markRead)
	v1.POST("/notifications/:notificationId/unread", markUnread)
	v1.DELETE("/notifications/:notificationId", deleteNotification)
	v1.POST("/notifications/trigger", triggerNotification)
	v1.GET("/notifications/push-status", listPushDeliveries)
}

func listNotifications(c *gin.Context) {
	unreadOnly := strings.EqualFold(strings.TrimSpace(c.Query("unreadOnly")), "true")
	cursor := strings.TrimSpace(c.Query("cursor"))
	limit := 20
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "limit must be an integer between 1 and 100", nil)
			return
		}
		limit = parsed
	}

	notificationsMu.RLock()
	defer notificationsMu.RUnlock()

	start := 0
	if cursor != "" {
		start = -1
		for i, item := range items {
			if item.ID == cursor {
				start = i + 1
				break
			}
		}
		if start == -1 {
			response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "cursor not found", gin.H{"cursor": cursor})
			return
		}
	}

	copyItems := make([]notification, 0, limit)
	for i := start; i < len(items); i++ {
		item := items[i]
		if unreadOnly && item.ReadAt != nil {
			continue
		}
		copyItems = append(copyItems, item)
		if len(copyItems) == limit {
			break
		}
	}

	response.JSON(c, http.StatusOK, copyItems)
}

func markRead(c *gin.Context) {
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if notificationID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "notificationId is required", nil)
		return
	}

	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	for i := range items {
		if items[i].ID != notificationID {
			continue
		}
		now := time.Now().UTC()
		items[i].ReadAt = &now
		response.NoContent(c)
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "notification not found", gin.H{"notificationId": notificationID})
}

func markUnread(c *gin.Context) {
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if notificationID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "notificationId is required", nil)
		return
	}

	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	for i := range items {
		if items[i].ID != notificationID {
			continue
		}
		items[i].ReadAt = nil
		response.NoContent(c)
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "notification not found", gin.H{"notificationId": notificationID})
}

func markAllRead(c *gin.Context) {
	now := time.Now().UTC()

	notificationsMu.Lock()
	for i := range items {
		items[i].ReadAt = &now
	}
	notificationsMu.Unlock()

	response.NoContent(c)
}

func cleanupRead(c *gin.Context) {
	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	filtered := make([]notification, 0, len(items))
	deletedCount := 0
	for _, item := range items {
		if item.ReadAt != nil {
			deletedCount++
			continue
		}
		filtered = append(filtered, item)
	}
	items = filtered

	response.JSON(c, http.StatusOK, gin.H{"deletedCount": deletedCount})
}

func deleteNotification(c *gin.Context) {
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if notificationID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "notificationId is required", nil)
		return
	}

	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	for i := range items {
		if items[i].ID != notificationID {
			continue
		}
		items = append(items[:i], items[i+1:]...)
		response.NoContent(c)
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "notification not found", gin.H{"notificationId": notificationID})
}

// --- Event-driven notification trigger ---

type triggerInput struct {
	EventType  string `json:"eventType"`
	ResourceID string `json:"resourceId"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Link       string `json:"link"`
	UserID     string `json:"userId"`
}

func triggerNotification(c *gin.Context) {
	var in triggerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.EventType) == "" || strings.TrimSpace(in.Title) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "eventType and title are required", nil)
		return
	}

	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	// Dedupe check: same eventType + resourceID within window → skip
	dedupeKey := in.EventType + ":" + in.ResourceID
	now := time.Now().UTC()
	if lastTime, exists := dedupeStore[dedupeKey]; exists {
		if now.Sub(lastTime) < dedupeWindow {
			response.JSON(c, http.StatusOK, gin.H{
				"skipped": true,
				"reason":  "dedupe: same event triggered within " + dedupeWindow.String(),
			})
			return
		}
	}
	dedupeStore[dedupeKey] = now

	// Check delivery preferences
	prefs := defaultDeliveryPrefs
	channels := []string{}

	notifID := "n-" + strconv.Itoa(len(items)+1)
	notif := notification{
		ID:        notifID,
		Type:      in.EventType,
		Title:     in.Title,
		Body:      in.Body,
		Link:      in.Link,
		CreatedAt: now,
	}

	// In-app delivery
	if prefs.InApp {
		items = append(items, notif)
		channels = append(channels, "in_app")
	}

	// Push delivery (simulated)
	if prefs.Push {
		pd := &pushDelivery{
			NotificationID: notifID,
			Status:         "sent",
			RetryCount:     0,
			LastAttemptAt:  now,
		}
		pushDeliveries[notifID] = pd
		channels = append(channels, "push")
	}

	// Email delivery (skipped if not enabled)
	if prefs.Email {
		channels = append(channels, "email")
	}

	response.JSON(c, http.StatusCreated, gin.H{
		"notificationId": notifID,
		"channels":       channels,
		"skipped":        false,
	})
}

func listPushDeliveries(c *gin.Context) {
	notificationsMu.RLock()
	defer notificationsMu.RUnlock()

	deliveries := make([]*pushDelivery, 0, len(pushDeliveries))
	for _, pd := range pushDeliveries {
		deliveries = append(deliveries, pd)
	}

	response.JSON(c, http.StatusOK, deliveries)
}
