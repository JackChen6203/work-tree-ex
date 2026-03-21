package budget

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type categoryPlan struct {
	Category      string  `json:"category"`
	PlannedAmount float64 `json:"plannedAmount"`
}

type budgetProfileInput struct {
	TotalBudget     *float64       `json:"totalBudget"`
	PerPersonBudget *float64       `json:"perPersonBudget"`
	PerDayBudget    *float64       `json:"perDayBudget"`
	Currency        string         `json:"currency"`
	Categories      []categoryPlan `json:"categories"`
}

type budgetProfile struct {
	TripID          string         `json:"tripId"`
	TotalBudget     *float64       `json:"totalBudget"`
	PerPersonBudget *float64       `json:"perPersonBudget"`
	PerDayBudget    *float64       `json:"perDayBudget"`
	Currency        string         `json:"currency"`
	Categories      []categoryPlan `json:"categories"`
	Version         int            `json:"version"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type expenseInput struct {
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	ExpenseAt *string `json:"expenseAt"`
	Note      string  `json:"note"`
}

type expense struct {
	ID        string    `json:"id"`
	TripID    string    `json:"tripId"`
	Category  string    `json:"category"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	ExpenseAt *string   `json:"expenseAt,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	budgetMu           sync.RWMutex
	profilesByTrip     = map[string]budgetProfile{}
	expensesByTrip     = map[string][]expense{}
	budgetIdempotency  = map[string]string{}
	expenseIdempotency = map[string]string{}
	expenseByID        = map[string]expense{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips/:tripId/budget", getBudget)
	v1.PUT("/trips/:tripId/budget", upsertBudget)
	v1.GET("/trips/:tripId/expenses", listExpenses)
	v1.POST("/trips/:tripId/expenses", createExpense)
	v1.DELETE("/trips/:tripId/expenses/:expenseId", deleteExpense)
}

func getBudget(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	budgetMu.RLock()
	profile, ok := profilesByTrip[tripID]
	items := expensesByTrip[tripID]
	budgetMu.RUnlock()

	if !ok {
		response.JSON(c, http.StatusOK, gin.H{
			"tripId":      tripID,
			"currency":    "JPY",
			"categories":  []categoryPlan{},
			"version":     0,
			"actualSpend": 0,
			"overBudget":  false,
		})
		return
	}

	actual := 0.0
	for _, item := range items {
		actual += item.Amount
	}
	overBudget := profile.TotalBudget != nil && actual > (*profile.TotalBudget*1.1)

	response.JSON(c, http.StatusOK, gin.H{
		"tripId":          profile.TripID,
		"totalBudget":     profile.TotalBudget,
		"perPersonBudget": profile.PerPersonBudget,
		"perDayBudget":    profile.PerDayBudget,
		"currency":        profile.Currency,
		"categories":      profile.Categories,
		"version":         profile.Version,
		"actualSpend":     actual,
		"overBudget":      overBudget,
		"createdAt":       profile.CreatedAt,
		"updatedAt":       profile.UpdatedAt,
	})
}

func upsertBudget(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in budgetProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if len(strings.TrimSpace(in.Currency)) != 3 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBudgetCurrency, "currency must be ISO-4217 code", nil)
		return
	}

	budgetMu.Lock()
	if existingTrip, ok := budgetIdempotency[idempotencyKey]; ok {
		existing := profilesByTrip[existingTrip]
		budgetMu.Unlock()
		response.JSON(c, http.StatusOK, existing)
		return
	}

	now := time.Now().UTC()
	existing, ok := profilesByTrip[tripID]
	if !ok {
		existing = budgetProfile{TripID: tripID, Version: 0, CreatedAt: now}
	}

	existing.TotalBudget = in.TotalBudget
	existing.PerPersonBudget = in.PerPersonBudget
	existing.PerDayBudget = in.PerDayBudget
	existing.Currency = strings.ToUpper(in.Currency)
	existing.Categories = in.Categories
	existing.Version++
	existing.UpdatedAt = now

	profilesByTrip[tripID] = existing
	budgetIdempotency[idempotencyKey] = tripID
	budgetMu.Unlock()

	response.JSON(c, http.StatusOK, existing)
}

func listExpenses(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	budgetMu.RLock()
	items := expensesByTrip[tripID]
	copyItems := make([]expense, len(items))
	copy(copyItems, items)
	budgetMu.RUnlock()

	response.JSON(c, http.StatusOK, copyItems)
}

func createExpense(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in expenseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.Category) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "category is required", nil)
		return
	}
	if in.Amount < 0 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "amount must be non-negative", nil)
		return
	}
	if len(strings.TrimSpace(in.Currency)) != 3 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBudgetCurrency, "currency must be ISO-4217 code", nil)
		return
	}

	budgetMu.Lock()
	if existingID, ok := expenseIdempotency[idempotencyKey]; ok {
		existing := expenseByID[existingID]
		budgetMu.Unlock()
		response.JSON(c, http.StatusCreated, existing)
		return
	}

	item := expense{
		ID:        uuid.NewString(),
		TripID:    tripID,
		Category:  in.Category,
		Amount:    in.Amount,
		Currency:  strings.ToUpper(in.Currency),
		ExpenseAt: in.ExpenseAt,
		Note:      in.Note,
		CreatedAt: time.Now().UTC(),
	}

	expensesByTrip[tripID] = append(expensesByTrip[tripID], item)
	expenseByID[item.ID] = item
	expenseIdempotency[idempotencyKey] = item.ID
	budgetMu.Unlock()

	response.JSON(c, http.StatusCreated, item)
}

func deleteExpense(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	expenseID := strings.TrimSpace(c.Param("expenseId"))
	if expenseID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "expenseId is required", nil)
		return
	}

	budgetMu.Lock()
	defer budgetMu.Unlock()

	item, ok := expenseByID[expenseID]
	if !ok || item.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "expense not found", gin.H{"expenseId": expenseID})
		return
	}

	items := expensesByTrip[tripID]
	filtered := make([]expense, 0, len(items))
	for _, candidate := range items {
		if candidate.ID != expenseID {
			filtered = append(filtered, candidate)
		}
	}
	expensesByTrip[tripID] = filtered
	delete(expenseByID, expenseID)

	response.NoContent(c)
}
