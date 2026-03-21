package sync

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/sync/bootstrap", bootstrap)
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

	if sinceVersion == 0 {
		if tripID != "" {
			payload["changedTrips"] = []gin.H{{"id": tripID, "version": 1}}
			payload["changedDays"] = []gin.H{{"tripId": tripID, "dayId": "day-1", "version": 1}}
		}
		payload["changedNotifications"] = []gin.H{{"id": "n-1", "version": 1}}
	}

	response.JSON(c, http.StatusOK, payload)
}
