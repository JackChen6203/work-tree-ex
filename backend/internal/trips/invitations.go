package trips

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type invitation struct {
	ID         string     `json:"id"`
	TripID     string     `json:"tripId"`
	InvitedBy  string     `json:"invitedByUserId"`
	Email      string     `json:"inviteeEmail"`
	Role       string     `json:"role"`
	TokenHash  string     `json:"-"`
	Token      string     `json:"token,omitempty"`
	Status     string     `json:"status"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	AcceptedAt *time.Time `json:"acceptedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type createInvitationInput struct {
	Email string `json:"inviteeEmail"`
	Role  string `json:"role"`
}

var (
	invitationMu          sync.RWMutex
	invitationsByTrip     = map[string][]invitation{}
	invitationByID        = map[string]*invitation{}
	invitationIdempotency = map[string]string{}
)

func resetInvitationStoreForTests() {
	invitationMu.Lock()
	defer invitationMu.Unlock()
	invitationsByTrip = map[string][]invitation{}
	invitationByID = map[string]*invitation{}
	invitationIdempotency = map[string]string{}
}

func registerInvitationRoutes(v1 *gin.RouterGroup) {
	v1.POST("/trips/:tripId/invitations", createInvitation)
	v1.GET("/trips/:tripId/invitations", listInvitations)
	v1.POST("/trips/:tripId/invitations/:invitationId/revoke", revokeInvitation)
	v1.POST("/trips/:tripId/invitations/:invitationId/accept", acceptInvitation)
}

func createInvitation(c *gin.Context) {
	tripID := c.Param("tripId")
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
		return
	}

	var in createInvitationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	email := strings.TrimSpace(in.Email)
	if email == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "inviteeEmail is required", nil)
		return
	}

	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "viewer"
	}
	if !isValidInvitationRole(role) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "role must be editor/commenter/viewer", nil)
		return
	}

	key := tripID + ":inv:" + idempotencyKey
	invitationMu.Lock()
	defer invitationMu.Unlock()

	if existingID, ok := invitationIdempotency[key]; ok {
		if inv, exists := invitationByID[existingID]; exists {
			response.JSON(c, http.StatusOK, inv)
			return
		}
	}

	// Check for existing pending invitation to same email
	for _, inv := range invitationsByTrip[tripID] {
		if strings.EqualFold(inv.Email, email) && inv.Status == "pending" {
			response.JSON(c, http.StatusOK, inv)
			return
		}
	}

	now := time.Now().UTC()
	rawToken := uuid.NewString()
	hash := sha256.Sum256([]byte(rawToken))

	inv := invitation{
		ID:        uuid.NewString(),
		TripID:    tripID,
		InvitedBy: "system", // placeholder until auth context is wired
		Email:     email,
		Role:      role,
		TokenHash: hex.EncodeToString(hash[:]),
		Token:     rawToken,
		Status:    "pending",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	invitationsByTrip[tripID] = append(invitationsByTrip[tripID], inv)
	invitationByID[inv.ID] = &invitationsByTrip[tripID][len(invitationsByTrip[tripID])-1]
	invitationIdempotency[key] = inv.ID

	response.JSON(c, http.StatusCreated, inv)
}

func listInvitations(c *gin.Context) {
	tripID := c.Param("tripId")

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
		return
	}

	invitationMu.RLock()
	items := invitationsByTrip[tripID]
	invitationMu.RUnlock()

	if items == nil {
		items = []invitation{}
	}

	// Strip raw tokens from list view
	safe := make([]invitation, len(items))
	copy(safe, items)
	for i := range safe {
		safe[i].Token = ""
	}

	response.JSON(c, http.StatusOK, safe)
}

func revokeInvitation(c *gin.Context) {
	tripID := c.Param("tripId")
	invitationID := c.Param("invitationId")

	invitationMu.Lock()
	defer invitationMu.Unlock()

	inv, ok := invitationByID[invitationID]
	if !ok || inv.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "invitation not found", gin.H{"invitationId": invitationID})
		return
	}

	if inv.Status != "pending" {
		response.Error(c, http.StatusConflict, perrors.CodeConflict, "invitation is not pending", gin.H{"status": inv.Status})
		return
	}

	inv.Status = "revoked"
	response.JSON(c, http.StatusOK, inv)
}

func acceptInvitation(c *gin.Context) {
	tripID := c.Param("tripId")
	invitationID := c.Param("invitationId")

	invitationMu.Lock()

	inv, ok := invitationByID[invitationID]
	if !ok || inv.TripID != tripID {
		invitationMu.Unlock()
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "invitation not found", gin.H{"invitationId": invitationID})
		return
	}

	if inv.Status == "expired" || inv.ExpiresAt.Before(time.Now().UTC()) {
		inv.Status = "expired"
		invitationMu.Unlock()
		response.Error(c, http.StatusGone, perrors.CodeBadRequest, "invitation has expired", nil)
		return
	}

	if inv.Status != "pending" {
		invitationMu.Unlock()
		response.Error(c, http.StatusConflict, perrors.CodeConflict, "invitation is not pending", gin.H{"status": inv.Status})
		return
	}

	now := time.Now().UTC()
	inv.Status = "accepted"
	inv.AcceptedAt = &now
	invitationMu.Unlock()

	// Add invitee as member
	membersMu.Lock()
	member := tripMember{
		ID:          uuid.NewString(),
		Email:       inv.Email,
		DisplayName: inv.Email,
		Role:        inv.Role,
		Status:      "active",
		JoinedAt:    now,
		CreatedAt:   now,
	}
	tripMembers[tripID] = append(tripMembers[tripID], member)
	membersMu.Unlock()

	response.JSON(c, http.StatusOK, gin.H{
		"invitation": inv,
		"member":     member,
	})
}

func isValidInvitationRole(role string) bool {
	switch role {
	case "editor", "commenter", "viewer":
		return true
	default:
		return false
	}
}
