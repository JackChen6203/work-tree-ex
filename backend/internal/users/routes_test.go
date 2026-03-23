package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newUsersRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	usersMu.Lock()
	myNotificationPreference = notificationPreference{
		PushEnabled:       true,
		EmailEnabled:      false,
		DigestFrequency:   "daily",
		QuietHoursStart:   "22:00",
		QuietHoursEnd:     "07:00",
		TripUpdates:       true,
		BudgetAlerts:      true,
		AiPlanReadyAlerts: true,
		Version:           1,
	}
	providerList = []llmProvider{}
	usersMu.Unlock()
	r := gin.New()
	g := r.Group("/users")
	RegisterRoutes(g)
	return r
}

func TestGetAndPatchMe(t *testing.T) {
	r := newUsersRouter()

	getReq := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}

	body := map[string]any{"displayName": "New Name", "currency": "usd"}
	b, _ := json.Marshal(body)
	patchReq := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(b))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	r.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", patchRec.Code, patchRec.Body.String())
	}

	var resp struct {
		Data struct {
			DisplayName string `json:"displayName"`
			Currency    string `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(patchRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.DisplayName != "New Name" {
		t.Fatalf("unexpected displayName: %s", resp.Data.DisplayName)
	}
	if resp.Data.Currency != "USD" {
		t.Fatalf("unexpected currency: %s", resp.Data.Currency)
	}
}

func TestPutPreferences(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"tripPace":            "slow",
		"wakePattern":         "early",
		"transportPreference": "walk",
		"foodPreference":      []string{"vegan"},
		"avoidTags":           []string{"stairs"},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me/preferences", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			TripPace string `json:"tripPace"`
			Version  int    `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.TripPace != "slow" {
		t.Fatalf("unexpected tripPace: %s", resp.Data.TripPace)
	}
	if resp.Data.Version < 2 {
		t.Fatalf("expected version increment, got %d", resp.Data.Version)
	}
}

func TestCreateAndListProviders(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"provider":                "openai",
		"label":                   "Personal Key",
		"model":                   "gpt-4.1-mini",
		"encryptedApiKeyEnvelope": "enc_abcdefgh12345678",
	}
	b, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/users/me/llm-providers", bytes.NewReader(b))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	var created struct {
		Data struct {
			ID        string `json:"id"`
			MaskedKey string `json:"maskedKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(postRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatalf("expected id")
	}
	if created.Data.MaskedKey == "" || created.Data.MaskedKey[:4] != "****" {
		t.Fatalf("expected masked key, got %s", created.Data.MaskedKey)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/users/me/llm-providers", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listed.Data) == 0 {
		t.Fatalf("expected at least one provider")
	}
}

func TestDeleteProvider(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"provider":                "openai",
		"label":                   "Delete Me",
		"model":                   "gpt-4.1-mini",
		"encryptedApiKeyEnvelope": "enc_delete_me_123456",
	}
	b, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/users/me/llm-providers", bytes.NewReader(b))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(postRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/users/me/llm-providers/"+created.Data.ID, nil)
	deleteRec := httptest.NewRecorder()
	r.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestGetAndPutNotificationPreferences(t *testing.T) {
	r := newUsersRouter()

	getReq := httptest.NewRequest(http.MethodGet, "/users/me/notification-preferences", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}

	payload := map[string]any{
		"pushEnabled":       true,
		"emailEnabled":      true,
		"digestFrequency":   "weekly",
		"quietHoursStart":   "23:30",
		"quietHoursEnd":     "07:30",
		"tripUpdates":       true,
		"budgetAlerts":      false,
		"aiPlanReadyAlerts": true,
	}
	b, _ := json.Marshal(payload)
	putReq := httptest.NewRequest(http.MethodPut, "/users/me/notification-preferences", bytes.NewReader(b))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", putRec.Code, putRec.Body.String())
	}

	var resp struct {
		Data struct {
			DigestFrequency string `json:"digestFrequency"`
			EmailEnabled    bool   `json:"emailEnabled"`
			Version         int    `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(putRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.DigestFrequency != "weekly" {
		t.Fatalf("unexpected digestFrequency: %s", resp.Data.DigestFrequency)
	}
	if !resp.Data.EmailEnabled {
		t.Fatalf("expected emailEnabled true")
	}
	if resp.Data.Version < 2 {
		t.Fatalf("expected version increment, got %d", resp.Data.Version)
	}
}

func TestPutNotificationPreferencesValidation(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"pushEnabled":       true,
		"emailEnabled":      false,
		"digestFrequency":   "hourly",
		"quietHoursStart":   "22:00",
		"quietHoursEnd":     "07:00",
		"tripUpdates":       true,
		"budgetAlerts":      true,
		"aiPlanReadyAlerts": true,
	}
	b, _ := json.Marshal(payload)
	putReq := httptest.NewRequest(http.MethodPut, "/users/me/notification-preferences", bytes.NewReader(b))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", putRec.Code, putRec.Body.String())
	}
}

func TestListProvidersFilterByProvider(t *testing.T) {
	r := newUsersRouter()

	payloads := []map[string]any{
		{
			"provider":                "openai",
			"label":                   "OpenAI Key",
			"model":                   "gpt-4.1-mini",
			"encryptedApiKeyEnvelope": "enc_openai_12345678",
		},
		{
			"provider":                "anthropic",
			"label":                   "Anthropic Key",
			"model":                   "claude-3-5-sonnet",
			"encryptedApiKeyEnvelope": "enc_anthropic_12345678",
		},
	}

	for _, payload := range payloads {
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/users/me/llm-providers", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/users/me/llm-providers?provider=openai", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listed struct {
		Data []struct {
			Provider string `json:"provider"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Provider != "openai" {
		t.Fatalf("expected only openai provider, got %+v", listed.Data)
	}
}

func TestPutNotificationPreferencesRejectsInvalidQuietHours(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"pushEnabled":       true,
		"emailEnabled":      false,
		"digestFrequency":   "daily",
		"quietHoursStart":   "7pm",
		"quietHoursEnd":     "07:00",
		"tripUpdates":       true,
		"budgetAlerts":      true,
		"aiPlanReadyAlerts": true,
	}
	b, _ := json.Marshal(payload)
	putReq := httptest.NewRequest(http.MethodPut, "/users/me/notification-preferences", bytes.NewReader(b))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", putRec.Code, putRec.Body.String())
	}
}

func TestPutNotificationPreferencesRejectsInstantEmailDigest(t *testing.T) {
	r := newUsersRouter()

	payload := map[string]any{
		"pushEnabled":       true,
		"emailEnabled":      true,
		"digestFrequency":   "instant",
		"quietHoursStart":   "22:00",
		"quietHoursEnd":     "07:00",
		"tripUpdates":       true,
		"budgetAlerts":      true,
		"aiPlanReadyAlerts": true,
	}
	b, _ := json.Marshal(payload)
	putReq := httptest.NewRequest(http.MethodPut, "/users/me/notification-preferences", bytes.NewReader(b))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", putRec.Code, putRec.Body.String())
	}
}
