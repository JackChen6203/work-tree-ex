package auth

import (
	"crypto/rand"
	"fmt"
	"net/mail"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
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
	Code      string
	ExpiresAt time.Time
}

type oauthStateEntry struct {
	Provider  string
	ExpiresAt time.Time
}

type oauthProviderConfig struct {
	AuthorizeURL string
	Scope        string
	ClientIDEnv  string
}

var oauthProviders = map[string]oauthProviderConfig{
	"google": {
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		Scope:        "openid email profile",
		ClientIDEnv:  "OAUTH_GOOGLE_CLIENT_ID",
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
	authStateMu  sync.Mutex
	pendingCodes = map[string]codeEntry{}
	oauthStates  = map[string]oauthStateEntry{}
	activeUser   *sessionUser
)

func RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/request-magic-link", requestMagicLink)
	group.POST("/verify-magic-link", verifyMagicLink)
	group.GET("/oauth/:provider/start", startOAuth)
	group.GET("/oauth/:provider/callback", callbackOAuth)
	group.GET("/session", getSession)
	group.POST("/logout", logout)
}

func startOAuth(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	config, ok := oauthProviders[provider]
	if !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "oauth provider is not supported", gin.H{"provider": provider})
		return
	}

	state, err := generateOpaqueToken(32)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate oauth state", nil)
		return
	}

	authStateMu.Lock()
	oauthStates[state] = oauthStateEntry{Provider: provider, ExpiresAt: time.Now().Add(10 * time.Minute)}
	authStateMu.Unlock()

	callbackURL := buildOAuthCallbackURL(c, provider)
	clientID := strings.TrimSpace(os.Getenv(config.ClientIDEnv))
	if clientID == "" {
		devRedirect := callbackURL + "?code=dev-oauth-code&state=" + url.QueryEscape(state)
		c.Redirect(http.StatusFound, devRedirect)
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
	c.Redirect(http.StatusFound, redirectURL)
}

func callbackOAuth(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if _, ok := oauthProviders[provider]; !ok {
		response.Error(c, http.StatusNotFound, perrors.CodeBadRequest, "oauth provider is not supported", gin.H{"provider": provider})
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if state == "" || code == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "oauth callback requires code and state", nil)
		return
	}

	authStateMu.Lock()
	stateEntry, ok := oauthStates[state]
	if !ok || time.Now().After(stateEntry.ExpiresAt) {
		delete(oauthStates, state)
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "oauth state is invalid or expired", nil)
		return
	}
	if stateEntry.Provider != provider {
		delete(oauthStates, state)
		authStateMu.Unlock()
		response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "oauth state provider mismatch", nil)
		return
	}
	delete(oauthStates, state)

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("%s_user_%d@oauth.local", provider, timestamp)
	user := &sessionUser{
		ID:     fmt.Sprintf("oauth_%s_%d", provider, timestamp),
		Name:   strings.ToUpper(provider[:1]) + provider[1:] + " User",
		Email:  email,
		Avatar: strings.ToUpper(provider[:1]) + "U",
	}
	activeUser = user
	authStateMu.Unlock()

	frontendURL := buildFrontendBaseURL(c)
	redirectURL := frontendURL + "/login?oauth=success&provider=" + url.QueryEscape(provider)
	c.Redirect(http.StatusFound, redirectURL)
}

func requestMagicLink(c *gin.Context) {
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

	code, err := generateCode(6)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to generate magic-link code", nil)
		return
	}

	authStateMu.Lock()
	pendingCodes[email] = codeEntry{Code: code, ExpiresAt: time.Now().Add(10 * time.Minute)}
	authStateMu.Unlock()

	response.JSON(c, http.StatusOK, gin.H{
		"sent":       true,
		"expiresIn":  600,
		"previewCode": code,
	})
}

func verifyMagicLink(c *gin.Context) {
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

	if entry.Code != strings.TrimSpace(in.Code) {
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
	activeUser = user
	authStateMu.Unlock()

	response.JSON(c, http.StatusOK, gin.H{
		"user":  user,
		"roles": []string{"owner"},
	})
}

func getSession(c *gin.Context) {
	authStateMu.Lock()
	defer authStateMu.Unlock()

	if activeUser == nil {
		response.JSON(c, http.StatusOK, gin.H{"user": nil, "roles": []string{}})
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"user": activeUser, "roles": []string{"owner"}})
}

func logout(c *gin.Context) {
	authStateMu.Lock()
	activeUser = nil
	authStateMu.Unlock()

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
