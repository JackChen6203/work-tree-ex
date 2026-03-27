package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/jwt"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

// publicPrefixes defines route prefixes that skip JWT enforcement.
var publicPrefixes = []string{
	"/healthz",
	"/readyz",
	"/api/v1/auth/",
}

func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(path, prefix) || path == strings.TrimSuffix(prefix, "/") {
				c.Next()
				return
			}
		}

		// Also allow cookie-based sessions to pass through (backward compat).
		if _, err := c.Cookie("tt_session"); err == nil {
			c.Next()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "authentication required", nil)
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "invalid authorization format", nil)
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		claims, err := jwt.Validate(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "invalid or expired token", nil)
			c.Abort()
			return
		}

		c.Set("userID", claims.Sub)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}
