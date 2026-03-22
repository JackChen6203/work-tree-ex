package sync

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

var (
	syncMu            sync.RWMutex
	entityVersions    = map[string]int{}
	latestSyncVersion int
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/sync/bootstrap", bootstrap)
	v1.POST("/sync/mutations/flush", flushMutations)
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
	payload := gin.H{
		"serverTime":           time.Now().UTC(),
		"sinceVersion":         sinceVersion,
		"tripId":               tripID,
		"fullResyncRequired":   false,
		"changedTrips":         []gin.H{},
		"changedDays":          []gin.H{},
		"changedNotifications": []gin.H{},
	}

	syncMu.RLock()
	currentVersion := latestSyncVersion
	syncMu.RUnlock()

	if sinceVersion == 0 {
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
	}
	nextVersion := latestSyncVersion
	syncMu.Unlock()

	response.JSON(c, http.StatusOK, gin.H{
		"tripId":        in.TripID,
		"acceptedCount": accepted,
		"conflictCount": len(conflicts),
		"conflicts":     conflicts,
		"nextVersion":   nextVersion,
		"serverTime":    time.Now().UTC(),
	})
}
