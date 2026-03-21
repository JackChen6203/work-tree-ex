package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	authStateMu.Lock()
	pendingCodes = map[string]codeEntry{}
	oauthStates = map[string]oauthStateEntry{}
	activeUser = nil
	authStateMu.Unlock()
	_ = os.Setenv("FRONTEND_BASE_URL", "http://localhost:5173")

	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1.Group("/auth"))
	return r
}

func TestMagicLinkRequestVerifyAndSession(t *testing.T) {
	r := setupAuthRouter()

	requestBody := mustMarshalAuth(t, map[string]string{"email": "demo@example.com"})
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", bytes.NewBuffer(requestBody))
	requestReq.Header.Set("Content-Type", "application/json")
	requestW := httptest.NewRecorder()
	r.ServeHTTP(requestW, requestReq)

	if requestW.Code != http.StatusOK {
		t.Fatalf("expected request status 200, got %d", requestW.Code)
	}

	var requestResp struct {
		Data struct {
			PreviewCode string `json:"previewCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(requestW.Body.Bytes(), &requestResp); err != nil {
		t.Fatalf("decode request response: %v", err)
	}
	if requestResp.Data.PreviewCode == "" {
		t.Fatalf("expected preview code")
	}

	verifyBody := mustMarshalAuth(t, map[string]string{
		"email": "demo@example.com",
		"code":  requestResp.Data.PreviewCode,
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-magic-link", bytes.NewBuffer(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyW := httptest.NewRecorder()
	r.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code != http.StatusOK {
		t.Fatalf("expected verify status 200, got %d", verifyW.Code)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionW := httptest.NewRecorder()
	r.ServeHTTP(sessionW, sessionReq)

	if sessionW.Code != http.StatusOK {
		t.Fatalf("expected session status 200, got %d", sessionW.Code)
	}

	var sessionResp struct {
		Data struct {
			User *struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sessionW.Body.Bytes(), &sessionResp); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	if sessionResp.Data.User == nil || sessionResp.Data.User.Email != "demo@example.com" {
		t.Fatalf("expected active session user")
	}
}

func TestVerifyMagicLinkRejectsInvalidCode(t *testing.T) {
	r := setupAuthRouter()

	requestBody := mustMarshalAuth(t, map[string]string{"email": "demo@example.com"})
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", bytes.NewBuffer(requestBody))
	requestReq.Header.Set("Content-Type", "application/json")
	requestW := httptest.NewRecorder()
	r.ServeHTTP(requestW, requestReq)

	if requestW.Code != http.StatusOK {
		t.Fatalf("expected request status 200, got %d", requestW.Code)
	}

	verifyBody := mustMarshalAuth(t, map[string]string{
		"email": "demo@example.com",
		"code":  "000000",
	})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-magic-link", bytes.NewBuffer(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyW := httptest.NewRecorder()
	r.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code != http.StatusUnauthorized {
		t.Fatalf("expected verify status 401, got %d", verifyW.Code)
	}
}

func TestOAuthStartRedirectsAndCallbackSetsSession(t *testing.T) {
	r := setupAuthRouter()

	startReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/start", nil)
	startW := httptest.NewRecorder()
	r.ServeHTTP(startW, startReq)

	if startW.Code != http.StatusFound {
		t.Fatalf("expected oauth start status 302, got %d", startW.Code)
	}

	location := startW.Header().Get("Location")
	if location == "" {
		t.Fatalf("expected redirect location from oauth start")
	}
	if !strings.Contains(location, "/api/v1/auth/oauth/google/callback") {
		t.Fatalf("expected callback redirect in dev mode, got %s", location)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, location, nil)
	callbackW := httptest.NewRecorder()
	r.ServeHTTP(callbackW, callbackReq)

	if callbackW.Code != http.StatusFound {
		t.Fatalf("expected oauth callback status 302, got %d", callbackW.Code)
	}

	finalLocation := callbackW.Header().Get("Location")
	if !strings.Contains(finalLocation, "oauth=success") {
		t.Fatalf("expected frontend success redirect, got %s", finalLocation)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionW := httptest.NewRecorder()
	r.ServeHTTP(sessionW, sessionReq)

	if sessionW.Code != http.StatusOK {
		t.Fatalf("expected session status 200, got %d", sessionW.Code)
	}

	var sessionResp struct {
		Data struct {
			User *struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sessionW.Body.Bytes(), &sessionResp); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	if sessionResp.Data.User == nil || !strings.Contains(sessionResp.Data.User.Email, "@oauth.local") {
		t.Fatalf("expected oauth session user")
	}
}

func mustMarshalAuth(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	return data
}
