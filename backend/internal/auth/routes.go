package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	pjwt "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/jwt"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/mailer"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type magicLinkRequestInput struct {
	Email string `json:"email"`
}

type magicLinkVerifyInput struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type sessionUser struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

type codeEntry struct {
	CodeHash       [32]byte
	ExpiresAt      time.Time
	FailedAttempts int
}

const sessionCookieName = "tt_session"

// accessTokenTTL is the default JWT access token duration.
const accessTokenTTL = 15 * time.Minute

// webSessionTTL is cookie session duration for browser session hydration.
const webSessionTTL = 24 * time.Hour

// refreshTokenTTL is the default refresh token duration.
const refreshTokenTTL = 7 * 24 * time.Hour
const magicLinkSendCooldown = 60 * time.Second

type sessionEntry struct {
	User             *sessionUser
	RefreshTokenHash [32]byte
	FamilyID         string
	Used             bool
	ExpiresAt        time.Time
}

type oauthStateEntry struct {
	Provider  string
	ExpiresAt time.Time
}

type oauthProviderConfig struct {
	AuthorizeURL string
	Scope        string
	ClientIDEnv  string
	ClientSecret string
}

var oauthProviders = map[string]oauthProviderConfig{
	"google": {
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		Scope:        "openid email profile",
		ClientIDEnv:  "OAUTH_GOOGLE_CLIENT_ID",
		ClientSecret: "OAUTH_GOOGLE_CLIENT_SECRET",
	},
	"apple": {
		AuthorizeURL: "https://appleid.apple.com/auth/authorize",
		Scope:        "name email",
		ClientIDEnv:  "OAUTH_APPLE_CLIENT_ID",
	},
	"facebook": {
		AuthorizeURL: "https://www.facebook.com/v20.0/dialog/oauth",
		Scope:        "public_profile,email",
		ClientIDEnv:  "OAUTH_FACEBOOK_CLIENT_ID",
	},
	"x": {
		AuthorizeURL: "https://twitter.com/i/oauth2/authorize",
		Scope:        "tweet.read users.read offline.access",
		ClientIDEnv:  "OAUTH_X_CLIENT_ID",
	},
	"github": {
		AuthorizeURL: "https://github.com/login/oauth/authorize",
		Scope:        "read:user user:email",
		ClientIDEnv:  "OAUTH_GITHUB_CLIENT_ID",
	},
	"line": {
		AuthorizeURL: "https://access.line.me/oauth2/v2.1/authorize",
		Scope:        "profile openid email",
		ClientIDEnv:  "OAUTH_LINE_CLIENT_ID",
	},
	"kakao": {
		AuthorizeURL: "https://kauth.kakao.com/oauth/authorize",
		Scope:        "profile_nickname account_email",
		ClientIDEnv:  "OAUTH_KAKAO_CLIENT_ID",
	},
	"wechat": {
		AuthorizeURL: "https://open.weixin.qq.com/connect/qrconnect",
		Scope:        "snsapi_login",
		ClientIDEnv:  "OAUTH_WECHAT_CLIENT_ID",
	},
	"tripadvisor": {
		AuthorizeURL: "https://www.tripadvisor.com/oauth/authorize",
		Scope:        "profile",
		ClientIDEnv:  "OAUTH_TRIPADVISOR_CLIENT_ID",
	},
	"booking": {
		AuthorizeURL: "https://account.booking.com/oauth/authorize",
		Scope:        "profile",
		ClientIDEnv:  "OAUTH_BOOKING_CLIENT_ID",
	},
}

var (
	authStateMu     sync.RWMutex
	pendingCodes    = map[string]codeEntry{}
	magicLinkSentAt = map[string]time.Time{}
	oauthStates     = map[string]oauthStateEntry{}
	sessions        = map[string]*sessionUser{}
	refreshSessions = map[string]*sessionEntry{} // keyed by refresh token hash hex
)

const maxMagicLinkFailedAttempts = 5

var (
	googleTokenEndpoint    = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
)

func RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/request-magic-link", requestMagicLink)
	group.POST("/verify-magic-link", verifyMagicLink)
	group.GET("/oauth/:provider/start", startOAuth)
	group.GET("/oauth/:provider/callback", callbackOAuth)
	group.GET("/session", getSession)
	group.POST("/logout", logout)
	group.POST("/refresh", refreshToken)
	group.POST("/verify-invite", verifyInviteToken)
}

func startOAuth(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	config, ok := oauthProviders[provider]
	if !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "oauth provider is not supported", gin.H{"provider": provider})
		return
	}

	state, err := generateOpaqueToken(32)
	if err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate oauth state", nil)
		return
	}

	authStateMu.Lock()
	oauthStates[state] = oauthStateEntry{Provider: provider, ExpiresAt: time.Now().Add(10 * time.Minute)}
	authStateMu.Unlock()

	callbackURL := buildOAuthCallbackURL(c, provider)
	clientID := strings.TrimSpace(os.Getenv(config.ClientIDEnv))
	if clientID == "" {
		if isProductionAuthMode() {
			response.Error(c, http.StatusServiceUnavailable, perrors.CodeNotImplemented, "oauth provider is not configured", gin.H{"provider": provider})
			return
		}
		devRedirect := callbackURL + "?code=dev-oauth-code&state=" + url.QueryEscape(state)
		c.Redirect(http.StatusFound, devRedirect)
		return
	}
	if provider == "google" && strings.TrimSpace(os.Getenv(config.ClientSecret)) == "" && isProductionAuthMode() {
		response.Error(c, http.StatusServiceUnavailable, perrors.CodeNotImplemented, "google oauth is not fully configured", nil)
		return
	}
	if provider != "google" && isProductionAuthMode() {
		response.Error(c, http.StatusServiceUnavailable, perrors.CodeNotImplemented, "oauth provider is not enabled in production", gin.H{"provider": provider})
		return
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", callbackURL)
	query.Set("scope", config.Scope)
	query.Set("state", state)
	query.Set("prompt", "consent")

	redirectURL := config.AuthorizeURL + "?" + query.Encode()
	if wantsJSONRedirect(c) {
		response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func callbackOAuth(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if _, ok := oauthProviders[provider]; !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "oauth provider is not supported", gin.H{"provider": provider})
		return
	}

	if reason := strings.TrimSpace(c.Query("error")); reason != "" {
		frontendURL := buildFrontendBaseURL(c)
		redirectURL := frontendURL + "/login?oauth=error&provider=" + url.QueryEscape(provider) + "&reason=" + url.QueryEscape(reason)
		if wantsJSONRedirect(c) {
			response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if state == "" || code == "" {
		frontendURL := buildFrontendBaseURL(c)
		redirectURL := frontendURL + "/login?oauth=error&provider=" + url.QueryEscape(provider) + "&reason=missing_code_or_state"
		if wantsJSONRedirect(c) {
			response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	authStateMu.Lock()
	stateEntry, ok := oauthStates[state]
	if !ok || time.Now().After(stateEntry.ExpiresAt) {
		delete(oauthStates, state)
		authStateMu.Unlock()
		frontendURL := buildFrontendBaseURL(c)
		redirectURL := frontendURL + "/login?oauth=error&provider=" + url.QueryEscape(provider) + "&reason=invalid_state"
		if wantsJSONRedirect(c) {
			response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	if stateEntry.Provider != provider {
		delete(oauthStates, state)
		authStateMu.Unlock()
		frontendURL := buildFrontendBaseURL(c)
		redirectURL := frontendURL + "/login?oauth=error&provider=" + url.QueryEscape(provider) + "&reason=provider_mismatch"
		if wantsJSONRedirect(c) {
			response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	delete(oauthStates, state)

	user, err := resolveOAuthUser(c, provider, code)
	if err != nil {
		authStateMu.Unlock()
		frontendURL := buildFrontendBaseURL(c)
		redirectURL := frontendURL + "/login?oauth=error&provider=" + url.QueryEscape(provider) + "&reason=" + url.QueryEscape("oauth_exchange_failed")
		if wantsJSONRedirect(c) {
			response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	sessionID, err := upsertSessionLocked(c, user)
	if err != nil {
		authStateMu.Unlock()
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to create session", nil)
		return
	}
	authStateMu.Unlock()
	setCachedWebSession(c.Request.Context(), sessionID, user, webSessionTTL)

	frontendURL := buildFrontendBaseURL(c)
	redirectURL := frontendURL + "/login?oauth=success&provider=" + url.QueryEscape(provider)
	if wantsJSONRedirect(c) {
		response.JSON(c, http.StatusOK, gin.H{"redirectTo": redirectURL})
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func requestMagicLink(c *gin.Context) {
	if !magicLinkAuthEnabled() {
		response.Error(c, http.StatusServiceUnavailable, perrors.CodeNotImplemented, "magic-link auth is disabled", nil)
		return
	}

	var in magicLinkRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	email := normalizeEmail(in.Email)
	if email == "" || !isValidEmail(email) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "valid email is required", nil)
		return
	}

	authStateMu.RLock()
	lastSentAt, seen := magicLinkSentAt[email]
	authStateMu.RUnlock()
	if seen {
		retryAfter := time.Until(lastSentAt.Add(magicLinkSendCooldown))
		if retryAfter > 0 {
			retryAfterSeconds := int(retryAfter.Seconds())
			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			response.Error(c, http.StatusTooManyRequests, perrors.CodeRateLimitExceeded, "magic-link was recently sent; please retry later", gin.H{
				"retryAfterSec": retryAfterSeconds,
			})
			return
		}
	}

	code, err := generateCode(6)
	if err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate magic-link code", nil)
		return
	}

	authStateMu.Lock()
	pendingCodes[email] = codeEntry{CodeHash: sha256.Sum256([]byte(code)), ExpiresAt: time.Now().Add(10 * time.Minute)}
	magicLinkSentAt[email] = time.Now().UTC()
	authStateMu.Unlock()

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	verifyURL := buildFrontendBaseURL(c) + "/login?method=magic-link&email=" + url.QueryEscape(email) + "&code=" + url.QueryEscape(code)
	message := mailer.BuildMagicLinkMessage(email, requestLocale(c), code, verifyURL, expiresAt)
	if err := mailer.Send(c.Request.Context(), message); err != nil {
		authStateMu.Lock()
		delete(pendingCodes, email)
		delete(magicLinkSentAt, email)
		authStateMu.Unlock()
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to send magic-link email", nil)
		return
	}

	payload := gin.H{
		"sent":      true,
		"expiresIn": 600,
	}
	if magicLinkPreviewEnabled() {
		payload["previewCode"] = code
	}

	response.JSON(c, http.StatusOK, payload)
}

func verifyMagicLink(c *gin.Context) {
	if !magicLinkAuthEnabled() {
		response.Error(c, http.StatusServiceUnavailable, perrors.CodeNotImplemented, "magic-link auth is disabled", nil)
		return
	}

	var in magicLinkVerifyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	email := normalizeEmail(in.Email)
	if email == "" || !isValidEmail(email) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "valid email is required", nil)
		return
	}
	if strings.TrimSpace(in.Code) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "code is required", nil)
		return
	}

	authStateMu.Lock()
	entry, ok := pendingCodes[email]
	if !ok || time.Now().After(entry.ExpiresAt) {
		delete(pendingCodes, email)
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "magic-link code is invalid or expired", nil)
		return
	}

	codeHash := sha256.Sum256([]byte(strings.TrimSpace(in.Code)))
	if subtle.ConstantTimeCompare(entry.CodeHash[:], codeHash[:]) != 1 {
		entry.FailedAttempts++
		if entry.FailedAttempts >= maxMagicLinkFailedAttempts {
			delete(pendingCodes, email)
		} else {
			pendingCodes[email] = entry
		}
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "magic-link code is invalid or expired", nil)
		return
	}

	delete(pendingCodes, email)
	user := &sessionUser{
		ID:     fmt.Sprintf("u_%d", time.Now().UnixNano()),
		Name:   displayNameFromEmail(email),
		Email:  email,
		Avatar: avatarFromEmail(email),
	}
	sessionID, err := upsertSessionLocked(c, user)
	if err != nil {
		authStateMu.Unlock()
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to create session", nil)
		return
	}
	authStateMu.Unlock()
	setCachedWebSession(c.Request.Context(), sessionID, user, webSessionTTL)

	response.JSON(c, http.StatusOK, gin.H{
		"user":  user,
		"roles": []string{"owner"},
	})
}

func getSession(c *gin.Context) {
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(sessionID) == "" {
		response.JSON(c, http.StatusOK, gin.H{"user": nil, "roles": []string{}})
		return
	}

	authStateMu.RLock()
	user, ok := sessions[sessionID]
	authStateMu.RUnlock()
	if !ok {
		if cachedUser, cacheOK := getCachedWebSession(c.Request.Context(), sessionID); cacheOK {
			user = cachedUser
			ok = true
			authStateMu.Lock()
			sessions[sessionID] = cachedUser
			authStateMu.Unlock()
		}
	}
	if !ok {
		response.JSON(c, http.StatusOK, gin.H{"user": nil, "roles": []string{}})
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"user": user, "roles": []string{"owner"}})
}

func logout(c *gin.Context) {
	sessionID := ""
	if rawSessionID, err := c.Cookie(sessionCookieName); err == nil {
		sessionID = strings.TrimSpace(rawSessionID)
	}
	if sessionID != "" {
		authStateMu.Lock()
		delete(sessions, sessionID)
		authStateMu.Unlock()
		deleteCachedWebSession(c.Request.Context(), sessionID)
	}

	clearSessionCookie(c)

	response.NoContent(c)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func generateCode(length int) (string, error) {
	if length < 4 {
		length = 4
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	for i := range buf {
		buf[i] = '0' + (buf[i] % 10)
	}

	return string(buf), nil
}

func generateOpaqueToken(length int) (string, error) {
	if length < 16 {
		length = 16
	}

	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	randBytes := make([]byte, length)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(randBytes[i])%len(alphabet)]
	}

	return string(buf), nil
}

func buildOAuthCallbackURL(c *gin.Context, provider string) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/api/v1/auth/oauth/%s/callback", scheme, c.Request.Host, provider)
}

func buildFrontendBaseURL(c *gin.Context) string {
	if v := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}

	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin != "" {
		return strings.TrimRight(origin, "/")
	}

	return "http://localhost:5173"
}

func requestLocale(c *gin.Context) string {
	value := strings.TrimSpace(c.GetHeader("Accept-Language"))
	if value == "" {
		return "zh-TW"
	}
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return "zh-TW"
	}
	return strings.TrimSpace(parts[0])
}

func upsertSessionLocked(c *gin.Context, user *sessionUser) (string, error) {
	sessionID, err := generateOpaqueToken(48)
	if err != nil {
		return "", err
	}

	sessions[sessionID] = user
	setSessionCookie(c, sessionID)
	return sessionID, nil
}

func setSessionCookie(c *gin.Context, sessionID string) {
	isSecure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(webSessionTTL.Seconds()),
	})
}

func clearSessionCookie(c *gin.Context) {
	isSecure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func displayNameFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	if local == "" {
		return "Traveler"
	}

	parts := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(parts) == 0 {
		parts = []string{local}
	}

	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])
	}

	return strings.Join(parts, " ")
}

func isProductionAuthMode() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "prod")
}

func magicLinkPreviewEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_ALLOW_MAGIC_LINK_PREVIEW")), "true")
}

func magicLinkAuthEnabled() bool {
	if isProductionAuthMode() {
		return false
	}
	return magicLinkPreviewEnabled()
}

func wantsJSONRedirect(c *gin.Context) bool {
	return strings.EqualFold(strings.TrimSpace(c.Query("transport")), "json")
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type googleUserInfoResponse struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func resolveOAuthUser(c *gin.Context, provider, code string) (*sessionUser, error) {
	if provider == "google" && strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_ID")) != "" && strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")) != "" {
		return exchangeGoogleOAuthCode(c, code)
	}

	if isProductionAuthMode() {
		return nil, fmt.Errorf("oauth provider %s is not available in production", provider)
	}

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("%s_user_%d@oauth.local", provider, timestamp)
	return &sessionUser{
		ID:     fmt.Sprintf("oauth_%s_%d", provider, timestamp),
		Name:   strings.ToUpper(provider[:1]) + provider[1:] + " User",
		Email:  email,
		Avatar: strings.ToUpper(provider[:1]) + "U",
	}, nil
}

func exchangeGoogleOAuthCode(c *gin.Context, code string) (*sessionUser, error) {
	form := url.Values{}
	form.Set("client_id", strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_ID")))
	form.Set("client_secret", strings.TrimSpace(os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")))
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", buildOAuthCallbackURL(c, "google"))

	req, err := http.NewRequest(http.MethodPost, googleTokenEndpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(tokenResp.Body, 1024))
		return nil, fmt.Errorf("google token exchange failed: %s", strings.TrimSpace(string(body)))
	}

	var tokenPayload googleTokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tokenPayload.AccessToken) == "" {
		return nil, fmt.Errorf("google token exchange returned empty access token")
	}

	userReq, err := http.NewRequest(http.MethodGet, googleUserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenPayload.AccessToken)

	userResp, err := client.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(userResp.Body, 1024))
		return nil, fmt.Errorf("google userinfo request failed: %s", strings.TrimSpace(string(body)))
	}

	var userPayload googleUserInfoResponse
	if err := json.NewDecoder(userResp.Body).Decode(&userPayload); err != nil {
		return nil, err
	}
	if !userPayload.EmailVerified || strings.TrimSpace(userPayload.Email) == "" || strings.TrimSpace(userPayload.Subject) == "" {
		return nil, fmt.Errorf("google user info is incomplete")
	}

	name := strings.TrimSpace(userPayload.Name)
	if name == "" {
		name = displayNameFromEmail(userPayload.Email)
	}

	return &sessionUser{
		ID:     "google_" + userPayload.Subject,
		Name:   name,
		Email:  userPayload.Email,
		Avatar: avatarFromEmail(userPayload.Email),
	}, nil
}

func avatarFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	letters := make([]rune, 0, 2)
	for _, r := range local {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letters = append(letters, r)
			if len(letters) == 2 {
				break
			}
		}
	}
	if len(letters) == 0 {
		return "TT"
	}
	return strings.ToUpper(string(letters))
}

// ── Refresh Token Rotation ───────────────────────────────────────────

type refreshTokenInput struct {
	RefreshToken string `json:"refreshToken"`
}

func refreshToken(c *gin.Context) {
	var in refreshTokenInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", nil)
		return
	}

	token := strings.TrimSpace(in.RefreshToken)
	if token == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "refreshToken is required", nil)
		return
	}

	if getPool() != nil {
		newRefreshRaw, err := generateOpaqueToken(48)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate refresh token", nil)
			return
		}

		user, err := rotateRefreshTokenPostgres(c.Request.Context(), token, newRefreshRaw, refreshTokenTTL)
		if err != nil {
			if errors.Is(err, ErrRefreshSessionReuse) {
				response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "refresh token reuse detected, session family revoked", nil)
				return
			}
			if errors.Is(err, ErrRefreshSessionExpired) {
				response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "refresh token has expired", nil)
				return
			}
			if errors.Is(err, ErrRefreshSessionInvalid) {
				response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "invalid or expired refresh token", nil)
				return
			}
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to rotate refresh token", nil)
			return
		}

		accessToken, err := generateJWTForUser(user)
		if err != nil {
			if response.DatabaseUnavailable(c, err) {
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate access token", nil)
			return
		}

		if sessionID, cookieErr := c.Cookie(sessionCookieName); cookieErr == nil && strings.TrimSpace(sessionID) != "" {
			authStateMu.Lock()
			sessions[sessionID] = user
			authStateMu.Unlock()
			setCachedWebSession(c.Request.Context(), sessionID, user, webSessionTTL)
		}

		response.JSON(c, http.StatusOK, gin.H{
			"accessToken":  accessToken,
			"refreshToken": newRefreshRaw,
			"expiresIn":    int(accessTokenTTL.Seconds()),
		})
		return
	}

	tokenHash := sha256.Sum256([]byte(token))
	hashHex := fmt.Sprintf("%x", tokenHash)

	authStateMu.Lock()

	entry, ok := refreshSessions[hashHex]
	if !ok {
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "invalid or expired refresh token", nil)
		return
	}

	// Reuse detection: if token was already used, revoke entire family.
	if entry.Used {
		familyID := entry.FamilyID
		for k, s := range refreshSessions {
			if s.FamilyID == familyID {
				delete(refreshSessions, k)
			}
		}
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "refresh token reuse detected, session family revoked", nil)
		return
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(refreshSessions, hashHex)
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "refresh token has expired", nil)
		return
	}

	// Mark old token as used (for reuse detection).
	entry.Used = true

	// Issue new refresh token with same family.
	newRefreshRaw, err := generateOpaqueToken(48)
	if err != nil {
		authStateMu.Unlock()
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate refresh token", nil)
		return
	}

	newHash := sha256.Sum256([]byte(newRefreshRaw))
	newHashHex := fmt.Sprintf("%x", newHash)
	refreshSessions[newHashHex] = &sessionEntry{
		User:             entry.User,
		RefreshTokenHash: newHash,
		FamilyID:         entry.FamilyID,
		Used:             false,
		ExpiresAt:        time.Now().Add(refreshTokenTTL),
	}
	user := entry.User
	authStateMu.Unlock()
	if sessionID, cookieErr := c.Cookie(sessionCookieName); cookieErr == nil && strings.TrimSpace(sessionID) != "" {
		authStateMu.Lock()
		sessions[sessionID] = user
		authStateMu.Unlock()
		setCachedWebSession(c.Request.Context(), sessionID, user, webSessionTTL)
	}

	accessToken, err := generateJWTForUser(user)
	if err != nil {
		if response.DatabaseUnavailable(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate access token", nil)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"accessToken":  accessToken,
		"refreshToken": newRefreshRaw,
		"expiresIn":    int(accessTokenTTL.Seconds()),
	})
}

func issueTokenPair(user *sessionUser) (accessToken, refreshRaw, familyID string, err error) {
	familyID = uuid.NewString()
	return issueTokenPairWithFamily(user, familyID)
}

func issueTokenPairWithFamily(user *sessionUser, familyID string) (accessToken, refreshRaw, _ string, err error) {
	accessToken, err = generateJWTForUser(user)
	if err != nil {
		return "", "", familyID, err
	}

	refreshRaw, err = generateOpaqueToken(48)
	if err != nil {
		return "", "", familyID, err
	}

	if getPool() != nil {
		if err := persistRefreshSessionPostgres(context.Background(), user, familyID, refreshRaw, refreshTokenTTL); err != nil {
			return "", "", familyID, err
		}
		return accessToken, refreshRaw, familyID, nil
	}

	tokenHash := sha256.Sum256([]byte(refreshRaw))
	hashHex := fmt.Sprintf("%x", tokenHash)

	authStateMu.Lock()
	refreshSessions[hashHex] = &sessionEntry{
		User:             user,
		RefreshTokenHash: tokenHash,
		FamilyID:         familyID,
		Used:             false,
		ExpiresAt:        time.Now().Add(refreshTokenTTL),
	}
	authStateMu.Unlock()

	return accessToken, refreshRaw, familyID, nil
}

func generateJWTForUser(user *sessionUser) (string, error) {
	return pjwt.Generate(user.ID, user.Email, accessTokenTTL)
}

// ── Invite Token Verification ────────────────────────────────────────

type verifyInviteInput struct {
	Token string `json:"token"`
}

type inviteEntry struct {
	TokenHash string
	ExpiresAt time.Time
	TripID    string
	Role      string
	Used      bool
}

var pendingInvites = map[string]*inviteEntry{}

func verifyInviteToken(c *gin.Context) {
	var in verifyInviteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", nil)
		return
	}

	token := strings.TrimSpace(in.Token)
	if token == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "token is required", nil)
		return
	}

	tokenHash := sha256.Sum256([]byte(token))
	hashHex := fmt.Sprintf("%x", tokenHash)

	authStateMu.Lock()
	defer authStateMu.Unlock()

	entry, ok := pendingInvites[hashHex]
	if !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "invite token not found", nil)
		return
	}

	if time.Now().After(entry.ExpiresAt) {
		response.Error(c, http.StatusGone, perrors.CodeBadRequest, "invite token has expired", nil)
		return
	}

	if entry.Used {
		response.Error(c, http.StatusConflict, perrors.CodeConflict, "invite token has already been used", nil)
		return
	}

	entry.Used = true

	response.JSON(c, http.StatusOK, gin.H{
		"tripId": entry.TripID,
		"role":   entry.Role,
		"valid":  true,
	})
}
