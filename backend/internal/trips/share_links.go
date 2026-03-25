package trips

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type shareLink struct {
	ID          string     `json:"id"`
	TripID      string     `json:"tripId"`
	TokenHash   string     `json:"-"`
	Token       string     `json:"token,omitempty"`
	AccessScope string     `json:"accessScope"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

var (
	shareLinkMu          sync.RWMutex
	shareLinksByTrip     = map[string][]shareLink{}
	shareLinkByID        = map[string]*shareLink{}
	shareLinkIdempotency = map[string]string{}
)

func resetShareLinkStoreForTests() {
	shareLinkMu.Lock()
	defer shareLinkMu.Unlock()
	shareLinksByTrip = map[string][]shareLink{}
	shareLinkByID = map[string]*shareLink{}
	shareLinkIdempotency = map[string]string{}
}

func registerShareLinkRoutes(v1 *gin.RouterGroup) {
	v1.POST("/trips/:tripId/share-links", createShareLink)
	v1.GET("/trips/:tripId/share-links", listShareLinks)
	v1.POST("/trips/:tripId/share-links/:linkId/revoke", revokeShareLink)
	v1.GET("/trips/:tripId/share/:token", verifyShareLink)
}

func createShareLink(c *gin.Context) {
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

	key := tripID + ":sl:" + idempotencyKey
	shareLinkMu.Lock()
	defer shareLinkMu.Unlock()

	if existingID, ok := shareLinkIdempotency[key]; ok {
		if sl, exists := shareLinkByID[existingID]; exists {
			response.JSON(c, http.StatusOK, sl)
			return
		}
	}

	rawToken := uuid.NewString()
	hash := sha256.Sum256([]byte(rawToken))

	sl := shareLink{
		ID:          uuid.NewString(),
		TripID:      tripID,
		TokenHash:   hex.EncodeToString(hash[:]),
		Token:       rawToken,
		AccessScope: "read_only",
		CreatedAt:   time.Now().UTC(),
	}

	shareLinksByTrip[tripID] = append(shareLinksByTrip[tripID], sl)
	shareLinkByID[sl.ID] = &shareLinksByTrip[tripID][len(shareLinksByTrip[tripID])-1]
	shareLinkIdempotency[key] = sl.ID

	response.JSON(c, http.StatusCreated, sl)
}

func listShareLinks(c *gin.Context) {
	tripID := c.Param("tripId")

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
		return
	}

	shareLinkMu.RLock()
	items := shareLinksByTrip[tripID]
	shareLinkMu.RUnlock()

	if items == nil {
		items = []shareLink{}
	}

	// Strip raw tokens from list view
	safe := make([]shareLink, len(items))
	copy(safe, items)
	for i := range safe {
		safe[i].Token = ""
	}

	response.JSON(c, http.StatusOK, safe)
}

func revokeShareLink(c *gin.Context) {
	tripID := c.Param("tripId")
	linkID := c.Param("linkId")

	shareLinkMu.Lock()
	defer shareLinkMu.Unlock()

	sl, ok := shareLinkByID[linkID]
	if !ok || sl.TripID != tripID {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "share link not found", gin.H{"linkId": linkID})
		return
	}

	if sl.RevokedAt != nil {
		response.Error(c, http.StatusConflict, perrors.CodeConflict, "share link already revoked", nil)
		return
	}

	now := time.Now().UTC()
	sl.RevokedAt = &now
	response.JSON(c, http.StatusOK, sl)
}

func verifyShareLink(c *gin.Context) {
	tripID := c.Param("tripId")
	rawToken := c.Param("token")

	hash := sha256.Sum256([]byte(rawToken))
	hashHex := hex.EncodeToString(hash[:])

	shareLinkMu.RLock()
	var matched *shareLink
	for _, sl := range shareLinksByTrip[tripID] {
		if sl.TokenHash == hashHex {
			matched = &sl
			break
		}
	}
	shareLinkMu.RUnlock()

	if matched == nil {
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "share link not found", nil)
		return
	}

	if matched.RevokedAt != nil {
		response.Error(c, http.StatusForbidden, perrors.CodeForbidden, "share link has been revoked", nil)
		return
	}

	if matched.ExpiresAt != nil && time.Now().After(*matched.ExpiresAt) {
		response.Error(c, http.StatusGone, perrors.CodeBadRequest, "share link has expired", nil)
		return
	}

	t, err := activeRepository.Get(c.Request.Context(), tripID)
	if err != nil {
		response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"trip":        t,
		"accessScope": matched.AccessScope,
	})
}
