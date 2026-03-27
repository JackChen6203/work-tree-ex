package ai

import (
	"context"
	"errors"
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
		WakePattern         string   `json:"wakePattern"`
		TransportPreference string   `json:"transportPreference"`
		PoiDensity          string   `json:"poiDensity"`
		MustVisit           []string `json:"mustVisit"`
		Avoid               []string `json:"avoid"`
	} `json:"constraints"`
}

type planDraft struct {
	ID               string    `json:"id"`
	TripID           string    `json:"tripId"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Summary          string    `json:"summary"`
	Warnings         []string  `json:"warnings"`
	TotalEstimated   float64   `json:"totalEstimated"`
	Budget           float64   `json:"budget"`
	Currency         string    `json:"currency"`
	PromptTokens     int       `json:"promptTokens,omitempty"`
	CompletionTokens int       `json:"completionTokens,omitempty"`
	EstimatedCost    float64   `json:"estimatedCost,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

var (
	plannerMu         sync.RWMutex
	plansByTrip       = map[string][]planDraft{}
	planByID          = map[string]planDraft{}
	createIdempotency = map[string]string{}
	adoptIdempotency  = map[string]gin.H{}
	// Distributed lock: prevent duplicate jobs per trip
	tripJobLocks = map[string]bool{}
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
	plannerMu.Unlock()

	releaseLock, lockAcquired := acquireTripPlanningLock(c.Request.Context(), tripID)
	if !lockAcquired {
		response.Error(c, http.StatusConflict, perrors.CodeConflict,
			"a planning job is already running for this trip", gin.H{"tripId": tripID})
		return
	}
	defer releaseLock()

	providerCfg, err := resolveProviderRuntimeConfig(c.Request.Context(), in.ProviderConfigID)
	if err != nil {
		if errors.Is(err, ErrProviderConfigNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "provider config not found", gin.H{"providerConfigId": in.ProviderConfigID})
			return
		}
		if errors.Is(err, ErrProviderUnsupported) {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "provider is not supported", gin.H{"providerConfigId": in.ProviderConfigID})
			return
		}
		if errors.Is(err, ErrProviderKeyDecrypt) {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "provider api key decryption failed", gin.H{"providerConfigId": in.ProviderConfigID})
			return
		}
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to resolve provider config", nil)
		return
	}
	provider := BuildProvider(providerCfg)

	prompt := BuildPrompt(
		"Trip Plan", tripID,
		time.Now().Format("2006-01-02"), time.Now().AddDate(0, 0, 3).Format("2006-01-02"),
		in.Constraints.TravelersCount,
		map[string]string{
			"pace":                in.Constraints.Pace,
			"wakePattern":         in.Constraints.WakePattern,
			"transportPreference": in.Constraints.TransportPreference,
			"poiDensity":          in.Constraints.PoiDensity,
		},
		map[string]any{
			"totalBudget": in.Constraints.TotalBudget,
			"currency":    in.Constraints.Currency,
			"mustVisit":   in.Constraints.MustVisit,
			"avoid":       in.Constraints.Avoid,
		},
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	rawOutput, usage, err := provider.GeneratePlan(ctx, prompt.FullPrompt())
	if err != nil {
		statusCode, mappedCode, message := mapProviderGenerateError(err)
		if getPool() != nil {
			recordFailedPlanRequestPostgres(c.Request.Context(), tripID, in, provider.Name(), mappedCode, err.Error())
		}
		response.Error(c, statusCode, mappedCode, message, gin.H{
			"provider": provider.Name(),
		})
		return
	}

	// Parse structured JSON output
	_, parseErr := ParseStructuredOutput(provider.Name(), rawOutput)
	if parseErr != nil {
		if getPool() != nil {
			recordFailedPlanRequestPostgres(c.Request.Context(), tripID, in, provider.Name(), perrors.CodeAIProviderInvalidOutput, parseErr.Error())
		}
		response.Error(c, http.StatusBadGateway, perrors.CodeAIProviderInvalidOutput,
			"AI provider returned invalid JSON", gin.H{"provider": provider.Name()})
		return
	}

	// Calculate budget status
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
		ID:               uuid.NewString(),
		TripID:           tripID,
		Title:            title,
		Status:           status,
		Summary:          fmt.Sprintf("Estimated %.0f %s against budget %.0f %s", estimated, strings.ToUpper(in.Constraints.Currency), in.Constraints.TotalBudget, strings.ToUpper(in.Constraints.Currency)),
		Warnings:         warnings,
		TotalEstimated:   estimated,
		Budget:           in.Constraints.TotalBudget,
		Currency:         strings.ToUpper(in.Constraints.Currency),
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		EstimatedCost:    usage.EstimatedCost,
		Provider:         provider.Name(),
		CreatedAt:        time.Now().UTC(),
	}

	if getPool() != nil {
		storedDraft, err := createPlanPostgres(c.Request.Context(), tripID, in, provider.Name(), usage, status, warnings, estimated)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to persist ai draft", nil)
			return
		}
		plannerMu.Lock()
		createIdempotency[idempotencyKey] = storedDraft.ID
		plannerMu.Unlock()
		response.JSON(c, http.StatusAccepted, gin.H{"jobId": storedDraft.ID, "status": "succeeded", "acceptedAt": storedDraft.CreatedAt})
		return
	}

	plannerMu.Lock()
	plansByTrip[tripID] = append(plansByTrip[tripID], draft)
	planByID[draft.ID] = draft
	createIdempotency[idempotencyKey] = draft.ID
	plannerMu.Unlock()

	response.JSON(c, http.StatusAccepted, gin.H{"jobId": draft.ID, "status": "succeeded", "acceptedAt": draft.CreatedAt})
}

func mapProviderGenerateError(err error) (int, string, string) {
	var timeoutErr ProviderTimeoutError
	if errors.As(err, &timeoutErr) {
		return http.StatusGatewayTimeout, perrors.CodeAIProviderTimeout, "AI provider timed out"
	}

	var circuitErr ProviderCircuitOpenError
	if errors.As(err, &circuitErr) {
		return http.StatusServiceUnavailable, perrors.CodeAIProviderCircuitOpen, "AI provider is temporarily unavailable"
	}

	var invalidOutputErr ProviderInvalidOutputError
	if errors.As(err, &invalidOutputErr) {
		return http.StatusBadGateway, perrors.CodeAIProviderInvalidOutput, "AI provider returned invalid output"
	}

	var apiErr ProviderAPIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == http.StatusTooManyRequests:
			return http.StatusTooManyRequests, perrors.CodeAIProviderQuotaExceed, "AI provider quota exceeded"
		case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
			return http.StatusBadGateway, perrors.CodeAIProviderAuthFailed, "AI provider authentication failed"
		case apiErr.StatusCode == http.StatusBadRequest || apiErr.StatusCode == http.StatusUnprocessableEntity || apiErr.StatusCode == http.StatusNotFound:
			return http.StatusBadGateway, perrors.CodeAIProviderBadRequest, "AI provider rejected request"
		default:
			return http.StatusBadGateway, perrors.CodeAIProviderUnavailable, "AI provider unavailable"
		}
	}

	return http.StatusInternalServerError, perrors.CodeInternalError, "AI provider error"
}

func listPlans(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	if getPool() != nil {
		items, err := listPlansPostgres(c.Request.Context(), tripID)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list ai plans", nil)
			return
		}
		response.JSON(c, http.StatusOK, items)
		return
	}

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

	if getPool() != nil {
		item, err := getPlanPostgres(c.Request.Context(), tripID, planID)
		if err != nil {
			if errors.Is(err, ErrAIDraftNotFound) {
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
				return
			}
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to get ai plan", nil)
			return
		}
		response.JSON(c, http.StatusOK, item)
		return
	}

	plannerMu.RLock()
	item, ok := planByID[planID]
	plannerMu.RUnlock()
	if !ok || item.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
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

	if getPool() != nil {
		plannerMu.Unlock()

		item, err := getPlanPostgres(c.Request.Context(), tripID, planID)
		if err != nil {
			if errors.Is(err, ErrAIDraftNotFound) {
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
				return
			}
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load ai plan", nil)
			return
		}

		if item.Status == "invalid" {
			response.Error(c, http.StatusConflict, perrors.CodeAIDraftInvalid, "ai draft is invalid and cannot be adopted", gin.H{"planId": planID})
			return
		}
		if item.Status == "warning" && !confirmWarnings {
			response.Error(c, http.StatusConflict, perrors.CodeAIDraftInvalid, "ai draft requires warning confirmation before adoption", gin.H{"planId": planID, "warnings": item.Warnings})
			return
		}

		if err := adoptDraftToItineraryPostgres(c.Request.Context(), tripID, planID); err != nil {
			if errors.Is(err, ErrAIDraftNotFound) {
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
				return
			}
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to adopt ai draft", nil)
			return
		}

		result := gin.H{
			"tripId":   tripID,
			"planId":   planID,
			"adopted":  true,
			"status":   item.Status,
			"warnings": item.Warnings,
		}
		if err := writeAIAuditLogPostgres(c.Request.Context(), "adopt_ai_draft", "ai_plan_drafts", planID, nil, result); err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to write audit log", nil)
			return
		}
		plannerMu.Lock()
		adoptIdempotency[idempotencyKey] = result
		plannerMu.Unlock()
		response.JSON(c, http.StatusOK, result)
		return
	}

	item, ok := planByID[planID]
	if !ok || item.TripID != tripID {
		plannerMu.Unlock()
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "ai plan not found", gin.H{"planId": planID, "tripId": tripID})
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
