package ai

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type planCreateInput struct {
	ProviderConfigID string `json:"providerConfigId"`
	Title            string `json:"title"`
	Constraints      struct {
		TotalBudget         float64  `json:"totalBudget"`
		Currency            string   `json:"currency"`
		TravelersCount      int      `json:"travelersCount"`
		Pace                string   `json:"pace"`
		TransportPreference string   `json:"transportPreference"`
		MustVisit           []string `json:"mustVisit"`
		Avoid               []string `json:"avoid"`
	} `json:"constraints"`
}

type planDraft struct {
	ID             string    `json:"id"`
	TripID         string    `json:"tripId"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	Summary        string    `json:"summary"`
	Warnings       []string  `json:"warnings"`
	TotalEstimated float64   `json:"totalEstimated"`
	Budget         float64   `json:"budget"`
	Currency       string    `json:"currency"`
	CreatedAt      time.Time `json:"createdAt"`
}

var (
	plannerMu         sync.RWMutex
	plansByTrip       = map[string][]planDraft{}
	planByID          = map[string]planDraft{}
	createIdempotency = map[string]string{}
	adoptIdempotency  = map[string]gin.H{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.POST("/trips/:tripId/ai/plans", createPlan)
	v1.GET("/trips/:tripId/ai/plans", listPlans)
	v1.GET("/trips/:tripId/ai/plans/:planId", getPlan)
	v1.POST("/trips/:tripId/ai/plans/:planId/adopt", adoptPlan)
}

func createPlan(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	if tripID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "tripId is required", nil)
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in planCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.ProviderConfigID) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "providerConfigId is required", nil)
		return
	}
	if in.Constraints.TotalBudget <= 0 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "constraints.totalBudget must be greater than 0", nil)
		return
	}
	if len(strings.TrimSpace(in.Constraints.Currency)) != 3 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBudgetCurrency, "constraints.currency must be ISO-4217 code", nil)
		return
	}

	plannerMu.Lock()
	if existingID, ok := createIdempotency[idempotencyKey]; ok {
		existing := planByID[existingID]
		plannerMu.Unlock()
		response.JSON(c, http.StatusAccepted, gin.H{"jobId": existing.ID, "status": "succeeded", "acceptedAt": existing.CreatedAt})
		return
	}

	estimated := estimateTotal(in.Constraints.TotalBudget, in.Constraints.Pace)
	ratio := estimated / in.Constraints.TotalBudget
	status := "valid"
	warnings := make([]string, 0, 2)
	if ratio > 1.2 {
		status = "invalid"
		warnings = append(warnings, fmt.Sprintf("預算超標 %.0f%%，超過 20%% 上限", (ratio-1.0)*100))
	} else if ratio > 1.1 {
		status = "warning"
		warnings = append(warnings, fmt.Sprintf("預算超標 %.0f%%，請在採用前調整", (ratio-1.0)*100))
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "AI Plan Draft"
	}

	draft := planDraft{
		ID:             uuid.NewString(),
		TripID:         tripID,
		Title:          title,
		Status:         status,
		Summary:        fmt.Sprintf("Estimated %.0f %s against budget %.0f %s", estimated, strings.ToUpper(in.Constraints.Currency), in.Constraints.TotalBudget, strings.ToUpper(in.Constraints.Currency)),
		Warnings:       warnings,
		TotalEstimated: estimated,
		Budget:         in.Constraints.TotalBudget,
		Currency:       strings.ToUpper(in.Constraints.Currency),
		CreatedAt:      time.Now().UTC(),
	}

	plansByTrip[tripID] = append(plansByTrip[tripID], draft)
	planByID[draft.ID] = draft
	createIdempotency[idempotencyKey] = draft.ID
	plannerMu.Unlock()

	response.JSON(c, http.StatusAccepted, gin.H{"jobId": draft.ID, "status": "succeeded", "acceptedAt": draft.CreatedAt})
}

func listPlans(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	plannerMu.RLock()
	items := plansByTrip[tripID]
	copyItems := make([]planDraft, len(items))
	copy(copyItems, items)
	plannerMu.RUnlock()

	response.JSON(c, http.StatusOK, copyItems)
}

func getPlan(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	planID := strings.TrimSpace(c.Param("planId"))

	plannerMu.RLock()
	item, ok := planByID[planID]
	plannerMu.RUnlock()
	if !ok || item.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
		return
	}

	response.JSON(c, http.StatusOK, item)
}

func adoptPlan(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	planID := strings.TrimSpace(c.Param("planId"))
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	confirmWarnings := strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Confirm-Warnings")), "true")
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	plannerMu.Lock()
	if existing, ok := adoptIdempotency[idempotencyKey]; ok {
		plannerMu.Unlock()
		response.JSON(c, http.StatusOK, existing)
		return
	}

	item, ok := planByID[planID]
	if !ok || item.TripID != tripID {
		plannerMu.Unlock()
		response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
		return
	}

	if item.Status == "invalid" {
		plannerMu.Unlock()
		response.Error(c, http.StatusConflict, perrors.CodeAIDraftInvalid, "ai draft is invalid and cannot be adopted", gin.H{"planId": planID})
		return
	}
	if item.Status == "warning" && !confirmWarnings {
		plannerMu.Unlock()
		response.Error(c, http.StatusConflict, perrors.CodeAIDraftInvalid, "ai draft requires warning confirmation before adoption", gin.H{"planId": planID, "warnings": item.Warnings})
		return
	}

	result := gin.H{
		"tripId":   tripID,
		"planId":   planID,
		"adopted":  true,
		"status":   item.Status,
		"warnings": item.Warnings,
	}
	adoptIdempotency[idempotencyKey] = result
	plannerMu.Unlock()

	response.JSON(c, http.StatusOK, result)
}

func estimateTotal(totalBudget float64, pace string) float64 {
	switch strings.ToLower(strings.TrimSpace(pace)) {
	case "packed":
		return totalBudget * 1.25
	case "balanced":
		return totalBudget * 1.12
	default:
		return totalBudget * 0.97
	}
}
