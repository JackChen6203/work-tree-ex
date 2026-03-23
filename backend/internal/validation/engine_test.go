package validation

import (
	"testing"
)

func TestValidDraft(t *testing.T) {
	oh := "09:00-18:00"
	draft := AiDraftPayload{
		Title:   "Tokyo Trip",
		Summary: "A great trip",
		Days: []AiDraftDay{
			{
				Date:  "2026-04-01",
				Theme: "Exploring Shibuya",
				Items: []AiDraftItem{
					{
						Title:    "Visit Meiji Shrine",
						ItemType: "place_visit",
						Place:    &AiDraftPlace{Name: "Meiji Shrine", ProviderPlaceID: "p1", Lat: 35.6764, Lng: 139.6993, OpeningHours: &oh},
					},
				},
			},
		},
		BudgetSummary: AiBudgetSummary{TotalEstimated: AmountWithCurrency{Amount: 5000, Currency: "JPY"}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-03", TotalBudget: 10000, Currency: "JPY"}

	result := Validate(draft, ctx)
	if result.Status != "valid" {
		t.Errorf("expected valid, got %s with issues: %+v", result.Status, result.Results)
	}
}

func TestSchemaInvalidMissingTitle(t *testing.T) {
	draft := AiDraftPayload{
		Days: []AiDraftDay{{Date: "2026-04-01", Items: []AiDraftItem{{Title: "x", ItemType: "meal"}}}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	if result.Status != "invalid" {
		t.Errorf("expected invalid, got %s", result.Status)
	}
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleSchemaInvalid && i.Message == "draft title is required" {
			found = true
		}
	}
	if !found {
		t.Error("expected SCHEMA_INVALID for missing title")
	}
}

func TestSchemaInvalidBadItemType(t *testing.T) {
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{{
			Date:  "2026-04-01",
			Items: []AiDraftItem{{Title: "x", ItemType: "invalid_type"}},
		}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	if result.Status != "invalid" {
		t.Errorf("expected invalid, got %s", result.Status)
	}
}

func TestBudgetWarning(t *testing.T) {
	draft := AiDraftPayload{
		Title:         "Test",
		Days:          []AiDraftDay{{Date: "2026-04-01", Items: []AiDraftItem{{Title: "x", ItemType: "meal"}}}},
		BudgetSummary: AiBudgetSummary{TotalEstimated: AmountWithCurrency{Amount: 11500}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01", TotalBudget: 10000}

	result := Validate(draft, ctx)
	if result.Status != "warning" {
		t.Errorf("expected warning, got %s", result.Status)
	}
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleBudgetWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected BUDGET_OVER_10_PCT warning")
	}
}

func TestBudgetInvalid(t *testing.T) {
	draft := AiDraftPayload{
		Title:         "Test",
		Days:          []AiDraftDay{{Date: "2026-04-01", Items: []AiDraftItem{{Title: "x", ItemType: "meal"}}}},
		BudgetSummary: AiBudgetSummary{TotalEstimated: AmountWithCurrency{Amount: 12500}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01", TotalBudget: 10000}

	result := Validate(draft, ctx)
	if result.Status != "invalid" {
		t.Errorf("expected invalid, got %s", result.Status)
	}
}

func TestDayOutOfRange(t *testing.T) {
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{
			{Date: "2026-03-31", Items: []AiDraftItem{{Title: "x", ItemType: "meal"}}},
		},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-03"}

	result := Validate(draft, ctx)
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleItemOutOfRange {
			found = true
		}
	}
	if !found {
		t.Error("expected ITEM_OUT_OF_DATE_RANGE")
	}
}

func TestGeoImpossibleTravel(t *testing.T) {
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{{
			Date: "2026-04-01",
			Items: []AiDraftItem{
				{Title: "Tokyo", ItemType: "place_visit", Place: &AiDraftPlace{Name: "Tokyo", ProviderPlaceID: "t1", Lat: 35.6762, Lng: 139.6503}},
				{Title: "Taipei", ItemType: "place_visit", Place: &AiDraftPlace{Name: "Taipei", ProviderPlaceID: "tp1", Lat: 25.0330, Lng: 121.5654}},
			},
		}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleGeoImpossibleTravel {
			found = true
		}
	}
	if !found {
		t.Error("expected GEO_IMPOSSIBLE_TRAVEL warning for Tokyo-Taipei (~2100km)")
	}
}

func TestSafetyScriptInjection(t *testing.T) {
	draft := AiDraftPayload{
		Title: "My Trip <script>alert('xss')</script>",
		Days:  []AiDraftDay{{Date: "2026-04-01", Items: []AiDraftItem{{Title: "x", ItemType: "meal"}}}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	if result.Status != "invalid" {
		t.Errorf("expected invalid, got %s", result.Status)
	}
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleSafetyPromptInjection {
			found = true
		}
	}
	if !found {
		t.Error("expected SAFETY_PROMPT_INJECTION")
	}
}

func TestDuplicatePOI(t *testing.T) {
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{
			{Date: "2026-04-01", Items: []AiDraftItem{
				{Title: "A", ItemType: "place_visit", Place: &AiDraftPlace{Name: "X", ProviderPlaceID: "same", Lat: 35, Lng: 139}},
			}},
			{Date: "2026-04-02", Items: []AiDraftItem{
				{Title: "B", ItemType: "place_visit", Place: &AiDraftPlace{Name: "X", ProviderPlaceID: "same", Lat: 35, Lng: 139}},
			}},
		},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-02"}

	result := Validate(draft, ctx)
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleDuplicatePOI {
			found = true
		}
	}
	if !found {
		t.Error("expected DUPLICATE_POI")
	}
}

func TestMissingOpeningHours(t *testing.T) {
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{{
			Date: "2026-04-01",
			Items: []AiDraftItem{
				{Title: "A", ItemType: "place_visit", Place: &AiDraftPlace{Name: "X", ProviderPlaceID: "p1", Lat: 35, Lng: 139}},
			},
		}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleMissingOpeningHours {
			found = true
		}
	}
	if !found {
		t.Error("expected MISSING_OPENING_HOURS info")
	}
}

func TestTimeOverlapInDraft(t *testing.T) {
	s1, e1 := "2026-04-01T09:00:00Z", "2026-04-01T11:00:00Z"
	s2, e2 := "2026-04-01T10:00:00Z", "2026-04-01T12:00:00Z"
	draft := AiDraftPayload{
		Title: "Test",
		Days: []AiDraftDay{{
			Date: "2026-04-01",
			Items: []AiDraftItem{
				{Title: "A", ItemType: "place_visit", StartAt: &s1, EndAt: &e1},
				{Title: "B", ItemType: "place_visit", StartAt: &s2, EndAt: &e2},
			},
		}},
	}
	ctx := TripContext{StartDate: "2026-04-01", EndDate: "2026-04-01"}

	result := Validate(draft, ctx)
	found := false
	for _, i := range result.Results {
		if i.RuleCode == RuleTimeOverlap {
			found = true
		}
	}
	if !found {
		t.Error("expected TIME_OVERLAP")
	}
}
