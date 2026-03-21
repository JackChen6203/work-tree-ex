package users

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

type profile struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
	Currency    string `json:"currency"`
}

type profilePatchInput struct {
	DisplayName *string `json:"displayName"`
	Locale      *string `json:"locale"`
	Timezone    *string `json:"timezone"`
	Currency    *string `json:"currency"`
}

type preference struct {
	TripPace            string   `json:"tripPace"`
	WakePattern         string   `json:"wakePattern"`
	TransportPreference string   `json:"transportPreference"`
	FoodPreference      []string `json:"foodPreference"`
	AvoidTags           []string `json:"avoidTags"`
	Version             int      `json:"version"`
}

type llmProvider struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Label     string    `json:"label"`
	Model     string    `json:"model"`
	MaskedKey string    `json:"maskedKey"`
	CreatedAt time.Time `json:"createdAt"`
}

type llmProviderInput struct {
	Provider                string `json:"provider"`
	Label                   string `json:"label"`
	EncryptedAPIKeyEnvelope string `json:"encryptedApiKeyEnvelope"`
	Model                   string `json:"model"`
}

var (
	usersMu sync.RWMutex
	me      = profile{
		ID:          "u_01",
		Email:       "ariel@example.com",
		DisplayName: "Ariel Chen",
		Locale:      "zh-TW",
		Timezone:    "Asia/Tokyo",
		Currency:    "JPY",
	}
	myPreference = preference{
		TripPace:            "balanced",
		WakePattern:         "normal",
		TransportPreference: "transit",
		FoodPreference:      []string{"coffee", "local"},
		AvoidTags:           []string{"too-many-transfers"},
		Version:             1,
	}
	providerList = []llmProvider{}
)

func RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/me", getMe)
	group.PATCH("/me", patchMe)
	group.GET("/me/preferences", getMyPreferences)
	group.PUT("/me/preferences", putMyPreferences)
	group.GET("/me/llm-providers", listMyProviders)
	group.POST("/me/llm-providers", createMyProvider)
	group.DELETE("/me/llm-providers/:providerId", deleteMyProvider)
}

func getMe(c *gin.Context) {
	usersMu.RLock()
	item := me
	usersMu.RUnlock()
	response.JSON(c, http.StatusOK, item)
}

func patchMe(c *gin.Context) {
	var in profilePatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	usersMu.Lock()
	if in.DisplayName != nil {
		me.DisplayName = strings.TrimSpace(*in.DisplayName)
	}
	if in.Locale != nil {
		me.Locale = strings.TrimSpace(*in.Locale)
	}
	if in.Timezone != nil {
		me.Timezone = strings.TrimSpace(*in.Timezone)
	}
	if in.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*in.Currency))
		if len(currency) != 3 {
			usersMu.Unlock()
			response.Error(c, http.StatusBadRequest, perrors.CodeBudgetCurrency, "currency must be ISO-4217 code", nil)
			return
		}
		me.Currency = currency
	}
	updated := me
	usersMu.Unlock()

	response.JSON(c, http.StatusOK, updated)
}

func getMyPreferences(c *gin.Context) {
	usersMu.RLock()
	item := myPreference
	usersMu.RUnlock()
	response.JSON(c, http.StatusOK, item)
}

func putMyPreferences(c *gin.Context) {
	var in preference
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.TripPace) == "" || strings.TrimSpace(in.WakePattern) == "" || strings.TrimSpace(in.TransportPreference) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "tripPace, wakePattern, and transportPreference are required", nil)
		return
	}

	usersMu.Lock()
	in.Version = myPreference.Version + 1
	myPreference = in
	updated := myPreference
	usersMu.Unlock()

	response.JSON(c, http.StatusOK, updated)
}

func listMyProviders(c *gin.Context) {
	usersMu.RLock()
	items := append([]llmProvider{}, providerList...)
	usersMu.RUnlock()
	response.JSON(c, http.StatusOK, items)
}

func createMyProvider(c *gin.Context) {
	var in llmProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.Provider) == "" || strings.TrimSpace(in.Model) == "" || strings.TrimSpace(in.EncryptedAPIKeyEnvelope) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "provider, model, and encryptedApiKeyEnvelope are required", nil)
		return
	}

	masked := "****"
	if len(in.EncryptedAPIKeyEnvelope) > 4 {
		masked = "****" + in.EncryptedAPIKeyEnvelope[len(in.EncryptedAPIKeyEnvelope)-4:]
	}

	item := llmProvider{
		ID:        uuid.NewString(),
		Provider:  strings.TrimSpace(in.Provider),
		Label:     strings.TrimSpace(in.Label),
		Model:     strings.TrimSpace(in.Model),
		MaskedKey: masked,
		CreatedAt: time.Now().UTC(),
	}

	usersMu.Lock()
	providerList = append(providerList, item)
	usersMu.Unlock()

	response.JSON(c, http.StatusCreated, item)
}

func deleteMyProvider(c *gin.Context) {
	providerID := strings.TrimSpace(c.Param("providerId"))
	if providerID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "providerId is required", nil)
		return
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	for i := range providerList {
		if providerList[i].ID != providerID {
			continue
		}
		providerList = append(providerList[:i], providerList[i+1:]...)
		response.NoContent(c)
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "provider not found", gin.H{"providerId": providerID})
}
