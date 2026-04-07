package httpserver

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

const (
	csrfHeaderName           = "X-CSRF-Token"
	csrfCookieName           = "tt_csrf"
	sessionCookieName        = "tt_session"
	csrfCookieMaxAgeSeconds  = 24 * 60 * 60
	csrfInvalidTokenResponse = "CSRF token is invalid"
)

// csrfMiddleware enforces CSRF token validation for state-changing requests
// that use cookie-based session authentication. Token-based (Bearer) auth
// is inherently CSRF-immune, so those requests are skipped.
func csrfMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		// Bearer token auth is CSRF-immune — skip.
		if strings.HasPrefix(strings.TrimSpace(c.GetHeader("Authorization")), "Bearer ") {
			c.Next()
			return
		}

		// No cookie session — skip (probably a public endpoint).
		if _, err := c.Cookie(sessionCookieName); err != nil {
			c.Next()
			return
		}

		// Cookie-based session + state-changing method → require CSRF token.
		csrfToken := strings.TrimSpace(c.GetHeader(csrfHeaderName))
		if csrfToken == "" {
			response.Error(c, http.StatusForbidden, perrors.CodeCSRFInvalid, "CSRF token is required for session-based requests", nil)
			c.Abort()
			return
		}

		csrfCookie, err := c.Cookie(csrfCookieName)
		if err != nil || strings.TrimSpace(csrfCookie) == "" {
			// Bootstrap existing sessions that predate the CSRF cookie. The
			// header is still required, so cross-site form posts cannot pass.
			setCSRFTokenCookie(c, csrfToken)
			c.Next()
			return
		}

		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(csrfCookie)), []byte(csrfToken)) != 1 {
			response.Error(c, http.StatusForbidden, perrors.CodeCSRFInvalid, csrfInvalidTokenResponse, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func setCSRFTokenCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   csrfCookieMaxAgeSeconds,
	})
}

func isSecureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}
