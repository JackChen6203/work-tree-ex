package validation

import (
	"math"
	"regexp"
	"strings"
	"time"
)

// ValidationResult holds the overall status and individual issues found.
type ValidationResult struct {
	Status  string            `json:"status"` // valid | warning | invalid
	Results []ValidationIssue `json:"results"`
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
	Severity string         `json:"severity"` // error | warning | info
	RuleCode string         `json:"ruleCode"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details,omitempty"`
}

// Rule codes
const (
	RuleSchemaInvalid         = "SCHEMA_INVALID"
	RuleItemOutOfRange        = "ITEM_OUT_OF_DATE_RANGE"
	RuleTimeOverlap           = "TIME_OVERLAP"
	RuleBudgetWarning         = "BUDGET_OVER_10_PCT"
	RuleBudgetInvalid         = "BUDGET_OVER_20_PCT"
	RuleDuplicatePOI          = "DUPLICATE_POI"
	RuleGeoImpossibleTravel   = "GEO_IMPOSSIBLE_TRAVEL"
	RuleUnverifiedPlace       = "UNVERIFIED_PLACE"
	RuleMissingOpeningHours   = "MISSING_OPENING_HOURS"
	RuleSafetyPromptInjection = "SAFETY_PROMPT_INJECTION"
)

// AiDraftPayload is the structured output from the AI planner.
type AiDraftPayload struct {
	Title          string             `json:"title"`
	Summary        string             `json:"summary"`
	Days           []AiDraftDay       `json:"days"`
	BudgetSummary  AiBudgetSummary    `json:"budgetSummary"`
	GlobalWarnings []string           `json:"globalWarnings"`
}

// AiDraftDay represents one day in an AI draft.
type AiDraftDay struct {
	Date                string             `json:"date"`
	Theme               string             `json:"theme"`
	Items               []AiDraftItem      `json:"items"`
	DailyBudgetEstimate AmountWithCurrency `json:"dailyBudgetEstimate"`
}

// AiDraftItem represents one itinerary item in an AI draft.
type AiDraftItem struct {
	Title         string             `json:"title"`
	ItemType      string             `json:"itemType"`
	StartAt       *string            `json:"startAt"`
	EndAt         *string            `json:"endAt"`
	Place         *AiDraftPlace      `json:"place"`
	EstimatedCost AmountWithCurrency `json:"estimatedCost"`
	Confidence    string             `json:"confidence"` // high | medium | low
	Warnings      []string           `json:"warnings"`
}

// AiDraftPlace represents place info in an AI draft item.
type AiDraftPlace struct {
	Name            string  `json:"name"`
	ProviderPlaceID string  `json:"providerPlaceId"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	OpeningHours    *string `json:"openingHours"`
}

// AiBudgetSummary contains overall budget info for a draft.
type AiBudgetSummary struct {
	TotalEstimated AmountWithCurrency `json:"totalEstimated"`
	Currency       string             `json:"currency"`
}

// AmountWithCurrency pairs an amount with its currency.
type AmountWithCurrency struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TripContext provides trip metadata needed for validation.
type TripContext struct {
	StartDate      string  `json:"startDate"`
	EndDate        string  `json:"endDate"`
	Timezone       string  `json:"timezone"`
	TotalBudget    float64 `json:"totalBudget"`
	Currency       string  `json:"currency"`
	TravelersCount int     `json:"travelersCount"`
}

// Validate runs the full 5-layer validation pipeline.
func Validate(draft AiDraftPayload, ctx TripContext) ValidationResult {
	var issues []ValidationIssue

	issues = append(issues, validateSchema(draft)...)
	issues = append(issues, validateBusiness(draft, ctx)...)
	issues = append(issues, validateGeographic(draft)...)
	issues = append(issues, validateTrust(draft)...)
	issues = append(issues, validateSafety(draft)...)

	status := computeStatus(issues)
	return ValidationResult{Status: status, Results: issues}
}

// --- Layer 1: Schema Validation ---

func validateSchema(draft AiDraftPayload) []ValidationIssue {
	var issues []ValidationIssue

	if strings.TrimSpace(draft.Title) == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error", RuleCode: RuleSchemaInvalid,
			Message: "draft title is required",
		})
	}

	if len(draft.Days) == 0 {
		issues = append(issues, ValidationIssue{
			Severity: "error", RuleCode: RuleSchemaInvalid,
			Message: "draft must contain at least one day",
		})
	}

	validItemTypes := map[string]bool{
		"place_visit": true, "meal": true, "transit": true,
		"hotel": true, "free_time": true, "custom": true,
	}

	for di, day := range draft.Days {
		if day.Date == "" {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleSchemaInvalid,
				Message: "day date is required",
				Details: map[string]any{"dayIndex": di},
			})
		} else if _, err := time.Parse("2006-01-02", day.Date); err != nil {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleSchemaInvalid,
				Message: "day date must be YYYY-MM-DD format",
				Details: map[string]any{"dayIndex": di, "date": day.Date},
			})
		}

		for ii, item := range day.Items {
			if strings.TrimSpace(item.Title) == "" {
				issues = append(issues, ValidationIssue{
					Severity: "error", RuleCode: RuleSchemaInvalid,
					Message: "item title is required",
					Details: map[string]any{"dayIndex": di, "itemIndex": ii},
				})
			}
			if !validItemTypes[item.ItemType] {
				issues = append(issues, ValidationIssue{
					Severity: "error", RuleCode: RuleSchemaInvalid,
					Message:  "invalid itemType",
					Details:  map[string]any{"dayIndex": di, "itemIndex": ii, "itemType": item.ItemType},
				})
			}
			if item.StartAt != nil && item.EndAt != nil {
				s, e := *item.StartAt, *item.EndAt
				if s > e {
					issues = append(issues, ValidationIssue{
						Severity: "error", RuleCode: RuleSchemaInvalid,
						Message: "endAt must be >= startAt",
						Details: map[string]any{"dayIndex": di, "itemIndex": ii},
					})
				}
			}
		}
	}

	return issues
}

// --- Layer 2: Business Validation ---

func validateBusiness(draft AiDraftPayload, ctx TripContext) []ValidationIssue {
	var issues []ValidationIssue

	tripStart, _ := time.Parse("2006-01-02", ctx.StartDate)
	tripEnd, _ := time.Parse("2006-01-02", ctx.EndDate)

	// Days out of trip range
	for di, day := range draft.Days {
		d, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			continue
		}
		if d.Before(tripStart) || d.After(tripEnd) {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleItemOutOfRange,
				Message:  "day date is outside trip date range",
				Details:  map[string]any{"dayIndex": di, "date": day.Date},
			})
		}
	}

	// Budget check
	if ctx.TotalBudget > 0 {
		totalEstimated := draft.BudgetSummary.TotalEstimated.Amount
		ratio := totalEstimated / ctx.TotalBudget
		if ratio > 1.2 {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleBudgetInvalid,
				Message:  "estimated budget exceeds total budget by more than 20%",
				Details:  map[string]any{"estimated": totalEstimated, "budget": ctx.TotalBudget, "ratio": math.Round(ratio*100) / 100},
			})
		} else if ratio > 1.1 {
			issues = append(issues, ValidationIssue{
				Severity: "warning", RuleCode: RuleBudgetWarning,
				Message:  "estimated budget exceeds total budget by more than 10%",
				Details:  map[string]any{"estimated": totalEstimated, "budget": ctx.TotalBudget, "ratio": math.Round(ratio*100) / 100},
			})
		}
	}

	// Duplicate POI
	seen := map[string]int{}
	for di, day := range draft.Days {
		for ii, item := range day.Items {
			if item.Place != nil && item.Place.ProviderPlaceID != "" {
				if prevDay, exists := seen[item.Place.ProviderPlaceID]; exists {
					issues = append(issues, ValidationIssue{
						Severity: "warning", RuleCode: RuleDuplicatePOI,
						Message:  "duplicate place across days",
						Details:  map[string]any{"dayIndex": di, "itemIndex": ii, "firstSeenDay": prevDay, "placeId": item.Place.ProviderPlaceID},
					})
				} else {
					seen[item.Place.ProviderPlaceID] = di
				}
			}
		}
	}

	// Time overlap within each day
	for di, day := range draft.Days {
		type timed struct {
			title string
			start string
			end   string
			idx   int
		}
		var timedItems []timed
		for ii, item := range day.Items {
			if item.StartAt != nil && item.EndAt != nil {
				timedItems = append(timedItems, timed{item.Title, *item.StartAt, *item.EndAt, ii})
			}
		}
		for i := 0; i < len(timedItems); i++ {
			for j := i + 1; j < len(timedItems); j++ {
				a, b := timedItems[i], timedItems[j]
				if a.start < b.end && b.start < a.end {
					issues = append(issues, ValidationIssue{
						Severity: "warning", RuleCode: RuleTimeOverlap,
						Message:  "time overlap detected",
						Details:  map[string]any{"dayIndex": di, "itemA": a.idx, "itemB": b.idx, "titleA": a.title, "titleB": b.title},
					})
				}
			}
		}
	}

	return issues
}

// --- Layer 3: Geographic Validation ---

func validateGeographic(draft AiDraftPayload) []ValidationIssue {
	var issues []ValidationIssue

	for di, day := range draft.Days {
		var prevPlace *AiDraftPlace
		var prevIdx int
		for ii, item := range day.Items {
			if item.Place == nil {
				continue
			}
			p := item.Place

			// Coordinate sanity
			if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
				issues = append(issues, ValidationIssue{
					Severity: "error", RuleCode: RuleSchemaInvalid,
					Message:  "invalid coordinates",
					Details:  map[string]any{"dayIndex": di, "itemIndex": ii, "lat": p.Lat, "lng": p.Lng},
				})
				continue
			}

			// Impossible travel detection (>500km between consecutive items in same day)
			if prevPlace != nil {
				dist := haversineKm(prevPlace.Lat, prevPlace.Lng, p.Lat, p.Lng)
				if dist > 500 {
					issues = append(issues, ValidationIssue{
						Severity: "warning", RuleCode: RuleGeoImpossibleTravel,
						Message:  "large distance between consecutive items",
						Details:  map[string]any{"dayIndex": di, "fromItem": prevIdx, "toItem": ii, "distanceKm": math.Round(dist)},
					})
				}
			}
			prevPlace = p
			prevIdx = ii
		}
	}

	return issues
}

// haversineKm calculates distance between two lat/lng points in kilometers.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth radius km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// --- Layer 4: Trust Validation ---

func validateTrust(draft AiDraftPayload) []ValidationIssue {
	var issues []ValidationIssue

	for di, day := range draft.Days {
		for ii, item := range day.Items {
			if item.Place == nil {
				continue
			}
			if item.Place.ProviderPlaceID == "" {
				issues = append(issues, ValidationIssue{
					Severity: "warning", RuleCode: RuleUnverifiedPlace,
					Message:  "place has no provider ID, cannot verify",
					Details:  map[string]any{"dayIndex": di, "itemIndex": ii, "placeName": item.Place.Name},
				})
			}
			if item.Place.OpeningHours == nil {
				issues = append(issues, ValidationIssue{
					Severity: "info", RuleCode: RuleMissingOpeningHours,
					Message:  "opening hours not available",
					Details:  map[string]any{"dayIndex": di, "itemIndex": ii, "placeName": item.Place.Name},
				})
			}
		}
	}

	return issues
}

// --- Layer 5: Safety Validation ---

var (
	scriptPattern = regexp.MustCompile(`(?i)<\s*script|javascript:|on\w+\s*=`)
	secretPattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[:=]\s*\S+`)
)

func validateSafety(draft AiDraftPayload) []ValidationIssue {
	var issues []ValidationIssue

	texts := []string{draft.Title, draft.Summary}
	for _, day := range draft.Days {
		texts = append(texts, day.Theme)
		for _, item := range day.Items {
			texts = append(texts, item.Title)
			if item.Place != nil {
				texts = append(texts, item.Place.Name)
			}
		}
	}

	for _, text := range texts {
		if scriptPattern.MatchString(text) {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleSafetyPromptInjection,
				Message:  "potential script injection detected",
				Details:  map[string]any{"snippet": truncate(text, 100)},
			})
			break // one is enough
		}
		if secretPattern.MatchString(text) {
			issues = append(issues, ValidationIssue{
				Severity: "error", RuleCode: RuleSafetyPromptInjection,
				Message:  "potential secret/key leak detected in output",
				Details:  map[string]any{"snippet": truncate(text, 100)},
			})
			break
		}
	}

	return issues
}

// --- Helpers ---

func computeStatus(issues []ValidationIssue) string {
	hasError := false
	hasWarning := false
	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			hasError = true
		case "warning":
			hasWarning = true
		}
	}
	if hasError {
		return "invalid"
	}
	if hasWarning {
		return "warning"
	}
	return "valid"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
