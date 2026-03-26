package budget

import (
	"errors"
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
	Category     string  `json:"category"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	ExpenseAt    *string `json:"expenseAt"`
	Note         string  `json:"note"`
	LinkedItemID *string `json:"linkedItemId"`
}

type expense struct {
	ID           string    `json:"id"`
	TripID       string    `json:"tripId"`
	Category     string    `json:"category"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	ExpenseAt    *string   `json:"expenseAt,omitempty"`
	Note         string    `json:"note,omitempty"`
	LinkedItemID *string   `json:"linkedItemId,omitempty"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
}

type expensePatchInput struct {
	Category  *string  `json:"category"`
	Amount    *float64 `json:"amount"`
	Currency  *string  `json:"currency"`
	ExpenseAt *string  `json:"expenseAt"`
	Note      *string  `json:"note"`
}

var (
	budgetMu           sync.RWMutex
	profilesByTrip     = map[string]budgetProfile{}
	expensesByTrip     = map[string][]expense{}
	budgetIdempotency  = map[string]string{}
	expenseIdempotency = map[string]string{}
	expenseByID        = map[string]expense{}
	expenseVersionByID = map[string]int{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips/:tripId/budget", getBudget)
	v1.PUT("/trips/:tripId/budget", upsertBudget)
	v1.GET("/trips/:tripId/expenses", listExpenses)
	v1.POST("/trips/:tripId/expenses", createExpense)
	v1.PATCH("/trips/:tripId/expenses/:expenseId", patchExpense)
	v1.DELETE("/trips/:tripId/expenses/:expenseId", deleteExpense)
	registerRateRoutes(v1)
}

func getBudget(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	if getPool() != nil {
		profile, ok, err := getBudgetProfilePostgres(c.Request.Context(), tripID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load budget profile", nil)
			return
		}
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

		actual, err := getActualSpendPostgres(c.Request.Context(), tripID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load expenses", nil)
			return
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
		return
	}

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

	if getPool() != nil {
		budgetMu.Lock()
		existingTrip, replay := budgetIdempotency[idempotencyKey]
		budgetMu.Unlock()
		if replay {
			existing, ok, err := getBudgetProfilePostgres(c.Request.Context(), existingTrip)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load budget profile", nil)
				return
			}
			if ok {
				response.JSON(c, http.StatusOK, existing)
				return
			}
		}

		profile, err := upsertBudgetPostgres(c.Request.Context(), tripID, in)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to upsert budget profile", nil)
			return
		}
		budgetMu.Lock()
		budgetIdempotency[idempotencyKey] = tripID
		budgetMu.Unlock()
		response.JSON(c, http.StatusOK, profile)
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
	if getPool() != nil {
		copyItems, err := listExpensesPostgres(c.Request.Context(), tripID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list expenses", nil)
			return
		}
		response.JSON(c, http.StatusOK, copyItems)
		return
	}

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
	if !isValidExpenseCategory(in.Category) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "category must be lodging/transit/food/attraction/shopping/misc", nil)
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

	if getPool() != nil {
		budgetMu.Lock()
		existingID, replay := expenseIdempotency[idempotencyKey]
		budgetMu.Unlock()
		if replay {
			existing, err := getExpensePostgres(c.Request.Context(), tripID, existingID)
			if err == nil {
				response.JSON(c, http.StatusCreated, existing)
				return
			}
		}

		item, err := createExpensePostgres(c.Request.Context(), tripID, in)
		if err != nil {
			if strings.Contains(err.Error(), "expenseAt") {
				response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, err.Error(), nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to create expense", nil)
			return
		}
		budgetMu.Lock()
		expenseIdempotency[idempotencyKey] = item.ID
		budgetMu.Unlock()
		response.JSON(c, http.StatusCreated, item)
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
		ID:           uuid.NewString(),
		TripID:       tripID,
		Category:     in.Category,
		Amount:       in.Amount,
		Currency:     strings.ToUpper(in.Currency),
		ExpenseAt:    in.ExpenseAt,
		Note:         in.Note,
		LinkedItemID: in.LinkedItemID,
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	expensesByTrip[tripID] = append(expensesByTrip[tripID], item)
	expenseByID[item.ID] = item
	expenseIdempotency[idempotencyKey] = item.ID
	budgetMu.Unlock()

	response.JSON(c, http.StatusCreated, item)
}

func patchExpense(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	expenseID := strings.TrimSpace(c.Param("expenseId"))
	if expenseID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "expenseId is required", nil)
		return
	}

	var in expensePatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if in.Amount != nil && *in.Amount < 0 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "amount must be non-negative", nil)
		return
	}
	if in.Currency != nil && len(strings.TrimSpace(*in.Currency)) != 3 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBudgetCurrency, "currency must be ISO-4217 code", nil)
		return
	}

	if getPool() != nil {
		item, err := patchExpensePostgres(c.Request.Context(), tripID, expenseID, in)
		if err != nil {
			switch {
			case errors.Is(err, ErrExpenseNotFound):
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "expense not found", gin.H{"expenseId": expenseID})
			case strings.Contains(err.Error(), "expenseAt"):
				response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, err.Error(), nil)
			default:
				response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to patch expense", nil)
			}
			return
		}
		response.JSON(c, http.StatusOK, item)
		return
	}

	budgetMu.Lock()
	defer budgetMu.Unlock()

	item, ok := expenseByID[expenseID]
	if !ok || item.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "expense not found", gin.H{"expenseId": expenseID})
		return
	}

	if in.Category != nil {
		item.Category = strings.TrimSpace(*in.Category)
	}
	if in.Amount != nil {
		item.Amount = *in.Amount
	}
	if in.Currency != nil {
		item.Currency = strings.ToUpper(strings.TrimSpace(*in.Currency))
	}
	if in.ExpenseAt != nil {
		item.ExpenseAt = in.ExpenseAt
	}
	if in.Note != nil {
		item.Note = strings.TrimSpace(*in.Note)
	}
	item.Version++

	expenseByID[expenseID] = item
	for i := range expensesByTrip[tripID] {
		if expensesByTrip[tripID][i].ID == expenseID {
			expensesByTrip[tripID][i] = item
			break
		}
	}

	response.JSON(c, http.StatusOK, item)
}

func deleteExpense(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	expenseID := strings.TrimSpace(c.Param("expenseId"))
	if expenseID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "expenseId is required", nil)
		return
	}

	if getPool() != nil {
		if err := deleteExpensePostgres(c.Request.Context(), tripID, expenseID); err != nil {
			if errors.Is(err, ErrExpenseNotFound) {
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "expense not found", gin.H{"expenseId": expenseID})
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to delete expense", nil)
			return
		}
		response.NoContent(c)
		return
	}

	budgetMu.Lock()
	defer budgetMu.Unlock()

	item, ok := expenseByID[expenseID]
	if !ok || item.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "expense not found", gin.H{"expenseId": expenseID})
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

func isValidExpenseCategory(category string) bool {
	switch strings.TrimSpace(category) {
	case "lodging", "transit", "food", "attraction", "shopping", "misc":
		return true
	default:
		return false
	}
}

// --- Currency conversion snapshot ---

type currencyRate struct {
	From      string     `json:"from"`
	To        string     `json:"to"`
	Rate      float64    `json:"rate"`
	Source    string     `json:"source"`
	FetchedAt time.Time  `json:"fetchedAt"`
	StaleAt   *time.Time `json:"staleAt,omitempty"`
}

var (
	rateCache = map[string]currencyRate{}
)

func init() {
	now := time.Now().UTC()
	seedRates := []currencyRate{
		{From: "USD", To: "JPY", Rate: 149.50, Source: "mock", FetchedAt: now},
		{From: "USD", To: "TWD", Rate: 32.10, Source: "mock", FetchedAt: now},
		{From: "USD", To: "EUR", Rate: 0.92, Source: "mock", FetchedAt: now},
		{From: "JPY", To: "TWD", Rate: 0.215, Source: "mock", FetchedAt: now},
		{From: "TWD", To: "JPY", Rate: 4.66, Source: "mock", FetchedAt: now},
		{From: "EUR", To: "USD", Rate: 1.09, Source: "mock", FetchedAt: now},
	}
	for _, r := range seedRates {
		rateCache[r.From+":"+r.To] = r
	}
}

func registerRateRoutes(g *gin.RouterGroup) {
	g.GET("/trips/:tripId/budget/rates", getRates)
	g.POST("/trips/:tripId/budget/rates/refresh", refreshRates)
}

func getRates(c *gin.Context) {
	from := strings.ToUpper(strings.TrimSpace(c.Query("from")))
	to := strings.ToUpper(strings.TrimSpace(c.Query("to")))

	budgetMu.RLock()
	defer budgetMu.RUnlock()

	if from != "" && to != "" {
		key := from + ":" + to
		rate, ok := rateCache[key]
		if !ok {
			response.Error(c, http.StatusNotFound, perrors.CodeBudgetRateUnavailable,
				"exchange rate not available for "+from+"/"+to, nil)
			return
		}
		response.JSON(c, http.StatusOK, rate)
		return
	}

	rates := make([]currencyRate, 0, len(rateCache))
	for _, r := range rateCache {
		rates = append(rates, r)
	}
	response.JSON(c, http.StatusOK, rates)
}

func refreshRates(c *gin.Context) {
	from := strings.ToUpper(strings.TrimSpace(c.Query("from")))
	to := strings.ToUpper(strings.TrimSpace(c.Query("to")))
	if from == "" || to == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "from and to query params required", nil)
		return
	}

	budgetMu.Lock()
	defer budgetMu.Unlock()

	key := from + ":" + to
	existing, hasExisting := rateCache[key]

	// Simulate API call (mock: always succeed with slightly updated rate)
	now := time.Now().UTC()
	newRate := currencyRate{
		From:      from,
		To:        to,
		Rate:      existing.Rate,
		Source:    "mock-api",
		FetchedAt: now,
	}

	if hasExisting {
		// Simulate slight rate change
		newRate.Rate = existing.Rate * 1.001
	} else {
		// Unknown pair → mark as stale fallback
		stale := now
		newRate.Rate = 1.0
		newRate.Source = "fallback"
		newRate.StaleAt = &stale
	}

	rateCache[key] = newRate
	response.JSON(c, http.StatusOK, newRate)
}

// ConvertAmount converts an amount from one currency to another using cached rates.
func ConvertAmount(from, to string, amount float64) (float64, bool) {
	if from == to {
		return amount, true
	}
	budgetMu.RLock()
	defer budgetMu.RUnlock()
	rate, ok := rateCache[from+":"+to]
	if !ok {
		return 0, false
	}
	return amount * rate.Rate, true
}
