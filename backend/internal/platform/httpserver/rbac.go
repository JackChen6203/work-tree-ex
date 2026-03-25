package httpserver

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

// MembershipEntry holds a user's role for a specific trip.
type MembershipEntry struct {
	UserID string
	Role   string
}

var (
	membershipMu sync.RWMutex
	// MembershipStore maps tripID to a list of membership entries.
	// In production this would query the DB; here we use an in-memory store.
	MembershipStore = map[string][]MembershipEntry{}
)

// ResetMembershipStoreForTests clears the RBAC membership store.
func ResetMembershipStoreForTests() {
	membershipMu.Lock()
	defer membershipMu.Unlock()
	MembershipStore = map[string][]MembershipEntry{}
}

// AddMembership registers a user's role for a trip (for dev/test use).
func AddMembership(tripID, userID, role string) {
	membershipMu.Lock()
	defer membershipMu.Unlock()
	MembershipStore[tripID] = append(MembershipStore[tripID], MembershipEntry{
		UserID: userID,
		Role:   role,
	})
}

// RequireRole returns a Gin middleware that checks the authenticated user's
// trip membership role against the allowed roles. The tripId must be a URL
// param named "tripId". The user ID must be set in context by the JWT middleware
// (key: "userID").
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		tripID := strings.TrimSpace(c.Param("tripId"))
		if tripID == "" {
			// No tripId in route — skip RBAC (not a trip-scoped route).
			c.Next()
			return
		}

		userIDVal, exists := c.Get("userID")
		if !exists {
			response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "authentication required", nil)
			c.Abort()
			return
		}
		userID, _ := userIDVal.(string)
		if userID == "" {
			response.Error(c, http.StatusUnauthorized, perrors.CodeUnauthorized, "authentication required", nil)
			c.Abort()
			return
		}

		membershipMu.RLock()
		members := MembershipStore[tripID]
		var userRole string
		for _, m := range members {
			if m.UserID == userID {
				userRole = m.Role
				break
			}
		}
		membershipMu.RUnlock()

		if userRole == "" || !allowed[userRole] {
			response.Error(c, http.StatusForbidden, perrors.CodeForbidden, "insufficient permissions for this trip", gin.H{
				"requiredRoles": allowedRoles,
			})
			c.Abort()
			return
		}

		c.Set("tripRole", userRole)
		c.Next()
	}
}
