package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	authStateMu.Lock()
	pendingCodes = map[string]codeEntry{}
	oauthStates = map[string]oauthStateEntry{}
	sessions = map[string]*sessionUser{}
	authStateMu.Unlock()
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:5173")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AUTH_ALLOW_MAGIC_LINK_PREVIEW", "false")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "")

	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1.Group("/auth"))
	return r
}

func TestMagicLinkRequestVerifyAndSession(t *testing.T) {
	r := setupAuthRouter(t)
	t.Setenv("AUTH_ALLOW_MAGIC_LINK_PREVIEW", "true")

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
	sessionCookie := verifyW.Result().Cookies()

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	for _, cookie := range sessionCookie {
		sessionReq.AddCookie(cookie)
	}
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
	r := setupAuthRouter(t)
	t.Setenv("AUTH_ALLOW_MAGIC_LINK_PREVIEW", "true")

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
	r := setupAuthRouter(t)

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
	sessionCookie := callbackW.Result().Cookies()

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	for _, cookie := range sessionCookie {
		sessionReq.AddCookie(cookie)
	}
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

func TestMagicLinkPreviewDisabledInProd(t *testing.T) {
	r := setupAuthRouter(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("AUTH_ALLOW_MAGIC_LINK_PREVIEW", "true")

	requestBody := mustMarshalAuth(t, map[string]string{"email": "demo@example.com"})
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-magic-link", bytes.NewBuffer(requestBody))
	requestReq.Header.Set("Content-Type", "application/json")
	requestW := httptest.NewRecorder()
	r.ServeHTTP(requestW, requestReq)

	if requestW.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected request status 503, got %d", requestW.Code)
	}
}

func TestGoogleOAuthCodeExchangeInProd(t *testing.T) {
	r := setupAuthRouter(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "google-client-secret")

	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/token":
			if err := req.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if req.Form.Get("client_id") != "google-client-id" || req.Form.Get("client_secret") != "google-client-secret" {
				t.Fatalf("unexpected google credentials")
			}
			if req.Form.Get("grant_type") != "authorization_code" {
				t.Fatalf("unexpected grant type %s", req.Form.Get("grant_type"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token-123"}`))
		case "/userinfo":
			if req.Header.Get("Authorization") != "Bearer token-123" {
				t.Fatalf("unexpected authorization header %s", req.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"google-sub","email":"demo@gmail.com","email_verified":true,"name":"Demo User"}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer oauthServer.Close()

	googleTokenEndpoint = oauthServer.URL + "/token"
	googleUserInfoEndpoint = oauthServer.URL + "/userinfo"
	defer func() {
		googleTokenEndpoint = "https://oauth2.googleapis.com/token"
		googleUserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
	}()

	startReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/start", nil)
	startReq.Host = "aitravel.dpdns.org"
	startReq.Header.Set("X-Forwarded-Proto", "https")
	startW := httptest.NewRecorder()
	r.ServeHTTP(startW, startReq)

	if startW.Code != http.StatusFound {
		t.Fatalf("expected oauth start status 302, got %d", startW.Code)
	}

	location := startW.Header().Get("Location")
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	state := parsedLocation.Query().Get("state")
	if state == "" {
		t.Fatalf("expected state in redirect location")
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?state="+url.QueryEscape(state)+"&code=real-google-code", nil)
	callbackReq.Host = "aitravel.dpdns.org"
	callbackReq.Header.Set("X-Forwarded-Proto", "https")
	callbackW := httptest.NewRecorder()
	r.ServeHTTP(callbackW, callbackReq)

	if callbackW.Code != http.StatusFound {
		t.Fatalf("expected oauth callback status 302, got %d", callbackW.Code)
	}
	if !strings.Contains(callbackW.Header().Get("Location"), "oauth=success") {
		t.Fatalf("expected success redirect, got %s", callbackW.Header().Get("Location"))
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
