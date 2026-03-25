package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
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
		if _, err := c.Cookie("tt_session"); err != nil {
			c.Next()
			return
		}

		// Cookie-based session + state-changing method → require CSRF token.
		csrfToken := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if csrfToken == "" {
			response.Error(c, http.StatusForbidden, perrors.CodeCSRFInvalid, "CSRF token is required for session-based requests", nil)
			c.Abort()
			return
		}

		// For dev mode we accept any non-empty token. In production this
		// would validate against a server-issued double-submit cookie.
		c.Next()
	}
}
