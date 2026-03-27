package sync

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	gosync "sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type mutationFlushInput struct {
	TripID    string              `json:"tripId"`
	Mutations []mutationFlushItem `json:"mutations"`
}

type mutationFlushItem struct {
	ID          string `json:"id"`
	EntityType  string `json:"entityType"`
	EntityID    string `json:"entityId"`
	BaseVersion int    `json:"baseVersion"`
}

// OutboxEvent represents a transactional outbox event.
type OutboxEvent struct {
	ID            string     `json:"id"`
	TripID        string     `json:"tripId,omitempty"`
	AggregateType string     `json:"aggregateType"`
	AggregateID   string     `json:"aggregateId"`
	EventType     string     `json:"eventType"`
	Payload       gin.H      `json:"payload"`
	DedupeKey     string     `json:"dedupeKey"`
	Status        string     `json:"status"` // pending | processed | dlq
	RetryCount    int        `json:"retryCount"`
	AvailableAt   time.Time  `json:"availableAt"`
	ProcessedAt   *time.Time `json:"processedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

const maxRetries = 3

var (
	syncMu                gosync.RWMutex
	entityVersions        = map[string]int{}
	flushIdempotencyStore = map[string]gin.H{}
	latestSyncVersion     int

	// Outbox store
	outboxEvents     = []OutboxEvent{}
	outboxByID       = map[string]*OutboxEvent{}
	outboxDedupeKeys = map[string]bool{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/sync/bootstrap", bootstrap)
	v1.POST("/sync/mutations/flush", flushMutations)
	v1.GET("/sync/outbox/events", listOutboxEvents)
	v1.POST("/sync/outbox/events/:eventId/ack", ackOutboxEvent)
}

func bootstrap(c *gin.Context) {
	sinceVersionRaw := strings.TrimSpace(c.Query("sinceVersion"))
	sinceVersion := 0
	if sinceVersionRaw != "" {
		value, err := strconv.Atoi(sinceVersionRaw)
		if err != nil || value < 0 {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "sinceVersion must be a non-negative integer", nil)
			return
		}
		sinceVersion = value
	}

	tripID := strings.TrimSpace(c.Query("tripId"))

	// Client version check for full re-sync
	clientVersion := strings.TrimSpace(c.GetHeader("X-Client-Version"))
	fullResync := false
	if clientVersion != "" {
		cv, err := strconv.Atoi(clientVersion)
		if err == nil && cv < 10 {
			fullResync = true
		}
	}

	payload := gin.H{
		"serverTime":           time.Now().UTC(),
		"sinceVersion":         sinceVersion,
		"tripId":               tripID,
		"fullResyncRequired":   fullResync,
		"changedTrips":         []gin.H{},
		"changedDays":          []gin.H{},
		"changedNotifications": []gin.H{},
	}

	syncMu.RLock()
	currentVersion := latestSyncVersion
	syncMu.RUnlock()

	if fullResync {
		// Full re-sync: return all data
		if tripID != "" {
			payload["changedTrips"] = []gin.H{{"id": tripID, "version": currentVersion}}
			payload["changedDays"] = []gin.H{{"tripId": tripID, "dayId": "day-1", "version": currentVersion}}
		}
	} else if sinceVersion == 0 {
		if tripID != "" {
			payload["changedTrips"] = []gin.H{{"id": tripID, "version": 1}}
			payload["changedDays"] = []gin.H{{"tripId": tripID, "dayId": "day-1", "version": 1}}
		}
		payload["changedNotifications"] = []gin.H{{"id": "n-1", "version": 1}}
	} else if tripID != "" && sinceVersion < currentVersion {
		payload["changedTrips"] = []gin.H{{"id": tripID, "version": currentVersion}}
		payload["changedDays"] = []gin.H{{"tripId": tripID, "dayId": "day-1", "version": currentVersion}}
	}

	response.JSON(c, http.StatusOK, payload)
}

func flushMutations(c *gin.Context) {
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	syncMu.RLock()
	if payload, ok := flushIdempotencyStore[idempotencyKey]; ok {
		syncMu.RUnlock()
		response.JSON(c, http.StatusOK, payload)
		return
	}
	syncMu.RUnlock()

	var in mutationFlushInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	in.TripID = strings.TrimSpace(in.TripID)
	if in.TripID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "tripId is required", nil)
		return
	}
	if len(in.Mutations) == 0 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "mutations is required", nil)
		return
	}

	accepted := 0
	conflicts := make([]gin.H, 0)

	syncMu.Lock()
	for _, mutation := range in.Mutations {
		mutationID := strings.TrimSpace(mutation.ID)
		entityType := strings.TrimSpace(mutation.EntityType)
		entityID := strings.TrimSpace(mutation.EntityID)
		if mutationID == "" || entityType == "" || entityID == "" || mutation.BaseVersion < 0 {
			conflicts = append(conflicts, gin.H{
				"id":       mutationID,
				"reason":   "invalid_mutation",
				"entityId": entityID,
			})
			continue
		}

		entityKey := entityType + ":" + entityID
		currentVersion := entityVersions[entityKey]
		if mutation.BaseVersion < currentVersion {
			conflicts = append(conflicts, gin.H{
				"id":              mutationID,
				"reason":          "version_conflict",
				"entityId":        entityID,
				"expectedVersion": currentVersion,
			})
			continue
		}

		latestSyncVersion++
		entityVersions[entityKey] = latestSyncVersion
		accepted++

		// Write outbox event (same "transaction")
		dedupeKey := idempotencyKey + ":" + mutationID
		if !outboxDedupeKeys[dedupeKey] {
			now := time.Now().UTC()
			evt := OutboxEvent{
				ID:            uuid.NewString(),
				TripID:        in.TripID,
				AggregateType: entityType,
				AggregateID:   entityID,
				EventType:     entityType + ".updated",
				Payload:       gin.H{"mutationId": mutationID, "version": latestSyncVersion},
				DedupeKey:     dedupeKey,
				Status:        "pending",
				RetryCount:    0,
				AvailableAt:   now,
				CreatedAt:     now,
			}
			if getPool() != nil {
				if err := createOutboxEventPostgres(c.Request.Context(), evt); err == nil {
					outboxDedupeKeys[dedupeKey] = true
				}
			} else {
				outboxEvents = append(outboxEvents, evt)
				outboxByID[evt.ID] = &outboxEvents[len(outboxEvents)-1]
				outboxDedupeKeys[dedupeKey] = true
			}
		}
	}
	nextVersion := latestSyncVersion
	payload := gin.H{
		"tripId":        in.TripID,
		"acceptedCount": accepted,
		"conflictCount": len(conflicts),
		"conflicts":     conflicts,
		"nextVersion":   nextVersion,
		"serverTime":    time.Now().UTC(),
	}
	flushIdempotencyStore[idempotencyKey] = payload
	syncMu.Unlock()

	response.JSON(c, http.StatusOK, payload)
}

func listOutboxEvents(c *gin.Context) {
	statusFilter := strings.TrimSpace(c.Query("status"))
	if statusFilter == "" {
		statusFilter = "pending"
	}

	if getPool() != nil {
		items, err := listOutboxEventsPostgres(c.Request.Context(), statusFilter)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list outbox events", nil)
			return
		}
		response.JSON(c, http.StatusOK, items)
		return
	}

	syncMu.RLock()
	defer syncMu.RUnlock()

	now := time.Now().UTC()
	items := make([]OutboxEvent, 0)
	for _, evt := range outboxEvents {
		if evt.Status == statusFilter && !evt.AvailableAt.After(now) {
			items = append(items, evt)
		}
	}

	response.JSON(c, http.StatusOK, items)
}

func ackOutboxEvent(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("eventId"))

	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if getPool() != nil {
		evt, err := ackOutboxEventPostgres(c.Request.Context(), eventID, body.Success)
		if err != nil {
			if errors.Is(err, ErrOutboxEventNotFound) {
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "outbox event not found", gin.H{"eventId": eventID})
				return
			}
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to ack outbox event", nil)
			return
		}
		response.JSON(c, http.StatusOK, evt)
		return
	}

	syncMu.Lock()
	defer syncMu.Unlock()

	evt, ok := outboxByID[eventID]
	if !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "outbox event not found", gin.H{"eventId": eventID})
		return
	}

	if body.Success {
		now := time.Now().UTC()
		evt.Status = "processed"
		evt.ProcessedAt = &now
	} else {
		evt.RetryCount++
		if evt.RetryCount > maxRetries {
			evt.Status = "dlq"
		} else {
			// Exponential backoff: 1s, 2s, 4s
			evt.AvailableAt = time.Now().UTC().Add(time.Duration(1<<evt.RetryCount) * time.Second)
		}
	}

	response.JSON(c, http.StatusOK, *evt)
}
