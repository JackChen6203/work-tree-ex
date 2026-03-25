package trips

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type trip struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Destination  string    `json:"destinationText,omitempty"`
	Destinations []string  `json:"destinations,omitempty"`
	StartDate    string    `json:"startDate"`
	EndDate      string    `json:"endDate"`
	Timezone     string    `json:"timezone"`
	Currency     string    `json:"currency"`
	Travelers    int       `json:"travelersCount"`
	Status       string    `json:"status"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type tripCreateInput struct {
	Name         string   `json:"name"`
	Destination  string   `json:"destinationText"`
	Destinations []string `json:"destinations"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	Timezone     string   `json:"timezone"`
	Currency     string   `json:"currency"`
	Travelers    int      `json:"travelersCount"`
}

type tripPatchInput struct {
	Name         *string  `json:"name"`
	Destination  *string  `json:"destinationText"`
	Destinations []string `json:"destinations"`
	StartDate    *string  `json:"startDate"`
	EndDate      *string  `json:"endDate"`
	Timezone     *string  `json:"timezone"`
	Currency     *string  `json:"currency"`
	Travelers    *int     `json:"travelersCount"`
	Status       *string  `json:"status"`
}

type tripMember struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId,omitempty"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	JoinedAt    time.Time `json:"joinedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

type addTripMemberInput struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

type patchTripMemberInput struct {
	Role string `json:"role"`
}

var (
	membersMu      sync.RWMutex
	tripMembers    = map[string][]tripMember{}
	idempotentAdds = map[string]tripMember{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips", listTrips)
	v1.POST("/trips", createTrip)
	v1.GET("/trips/:tripId", getTrip)
	v1.PATCH("/trips/:tripId", patchTrip)
	v1.GET("/trips/:tripId/members", listTripMembers)
	v1.POST("/trips/:tripId/members", addTripMember)
	v1.PATCH("/trips/:tripId/members/:memberId", patchTripMember)
	v1.DELETE("/trips/:tripId/members/:memberId", removeTripMember)
	registerInvitationRoutes(v1)
	registerShareLinkRoutes(v1)
}

func resetMemberStoreForTests() {
	membersMu.Lock()
	defer membersMu.Unlock()
	tripMembers = map[string][]tripMember{}
	idempotentAdds = map[string]tripMember{}
	resetInvitationStoreForTests()
	resetShareLinkStoreForTests()
}

func listTrips(c *gin.Context) {
	items, err := activeRepository.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list trips", nil)
		return
	}

	response.JSON(c, http.StatusOK, items)
}

func createTrip(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in tripCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if err := validateCreateInput(in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, err.Error(), nil)
		return
	}

	t, err := activeRepository.Create(c.Request.Context(), in, idempotencyKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to create trip", nil)
		return
	}

	days := GenerateDaySkeletons(t.ID, in.StartDate, in.EndDate)

	response.JSON(c, http.StatusCreated, gin.H{
		"trip": t,
		"days": days,
	})
}

func getTrip(c *gin.Context) {
	tripID := c.Param("tripId")

	t, err := activeRepository.Get(c.Request.Context(), tripID)
	if err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to get trip", nil)
		return
	}

	if t.ID == "" {
		response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
		return
	}

	response.JSON(c, http.StatusOK, t)
}

func patchTrip(c *gin.Context) {
	tripID := c.Param("tripId")
	ifMatch := c.GetHeader("If-Match-Version")
	if ifMatch == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "If-Match-Version header is required", nil)
		return
	}

	version, err := strconv.Atoi(ifMatch)
	if err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "If-Match-Version must be an integer", nil)
		return
	}

	var in tripPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	preview, err := activeRepository.Get(c.Request.Context(), tripID)
	if err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load trip", nil)
		return
	}

	if preview.Status == "archived" {
		response.Error(c, http.StatusForbidden, perrors.CodeForbidden, "archived trip cannot be edited", nil)
		return
	}

	if in.Name != nil && len(*in.Name) > 200 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "name must not exceed 200 characters", nil)
		return
	}

	if in.Status != nil {
		if !isValidStatusTransition(preview.Status, *in.Status) {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid status transition", gin.H{"current": preview.Status, "requested": *in.Status})
			return
		}
	}

	if in.StartDate != nil {
		preview.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		preview.EndDate = *in.EndDate
	}

	if err := validateDates(preview.StartDate, preview.EndDate); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeInvalidDateRange, err.Error(), nil)
		return
	}

	t, err := activeRepository.Update(c.Request.Context(), tripID, version, in)
	if err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		if errors.Is(err, ErrVersionConflict) {
			response.Error(c, http.StatusConflict, perrors.CodeVersionConflict, "trip version conflict", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to update trip", nil)
		return
	}

	response.JSON(c, http.StatusOK, t)
}

func listTripMembers(c *gin.Context) {
	tripID := c.Param("tripId")
	roleFilter := strings.TrimSpace(c.Query("role"))
	if roleFilter != "" && !isValidMemberRole(roleFilter) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "role must be owner/editor/commenter/viewer", nil)
		return
	}
	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load trip", nil)
		return
	}

	var items []tripMember
	if getCollaborationPool() != nil {
		var err error
		items, err = listTripMembersPostgres(c.Request.Context(), tripID, roleFilter)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to list trip members", nil)
			return
		}
	} else {
		membersMu.RLock()
		items = make([]tripMember, 0, len(tripMembers[tripID]))
		for _, item := range tripMembers[tripID] {
			if roleFilter != "" && item.Role != roleFilter {
				continue
			}
			items = append(items, item)
		}
		membersMu.RUnlock()
	}

	response.JSON(c, http.StatusOK, items)
}

func addTripMember(c *gin.Context) {
	tripID := c.Param("tripId")
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load trip", nil)
		return
	}

	var in addTripMemberInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(in.UserID) == "" && strings.TrimSpace(in.Email) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "userId or email is required", nil)
		return
	}

	if !isValidMemberRole(in.Role) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "role must be owner/editor/commenter/viewer", nil)
		return
	}

	key := tripID + ":" + idempotencyKey
	membersMu.Lock()
	if existing, ok := idempotentAdds[key]; ok {
		membersMu.Unlock()
		response.JSON(c, http.StatusOK, existing)
		return
	}
	membersMu.Unlock()

	if getCollaborationPool() != nil {
		item, err := addTripMemberPostgres(c.Request.Context(), tripID, in)
		if err != nil {
			if errors.Is(err, ErrMemberAlreadyExists) {
				response.Error(c, http.StatusConflict, perrors.CodeConflict, "member already exists", gin.H{"email": in.Email, "userId": in.UserID})
				return
			}
			response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to add member", nil)
			return
		}
		membersMu.Lock()
		idempotentAdds[key] = item
		membersMu.Unlock()
		response.JSON(c, http.StatusCreated, item)
		return
	}

	membersMu.Lock()
	for _, m := range tripMembers[tripID] {
		if strings.TrimSpace(in.UserID) != "" && m.UserID == strings.TrimSpace(in.UserID) {
			membersMu.Unlock()
			response.Error(c, http.StatusConflict, perrors.CodeConflict, "member already exists", gin.H{"userId": in.UserID})
			return
		}
		if strings.TrimSpace(in.Email) != "" && strings.EqualFold(m.Email, strings.TrimSpace(in.Email)) {
			membersMu.Unlock()
			response.Error(c, http.StatusConflict, perrors.CodeConflict, "member already exists", gin.H{"email": in.Email})
			return
		}
	}

	now := time.Now().UTC()
	item := tripMember{
		ID:          uuid.NewString(),
		UserID:      strings.TrimSpace(in.UserID),
		Email:       strings.TrimSpace(in.Email),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Role:        strings.TrimSpace(in.Role),
		Status:      "active",
		JoinedAt:    now,
		CreatedAt:   now,
	}
	tripMembers[tripID] = append(tripMembers[tripID], item)
	idempotentAdds[key] = item
	membersMu.Unlock()

	response.JSON(c, http.StatusCreated, item)
}

func removeTripMember(c *gin.Context) {
	tripID := c.Param("tripId")
	memberID := strings.TrimSpace(c.Param("memberId"))
	if memberID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "memberId is required", nil)
		return
	}

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load trip", nil)
		return
	}

	if getCollaborationPool() != nil {
		err := removeTripMemberPostgres(c.Request.Context(), tripID, memberID)
		if err != nil {
			switch {
			case errors.Is(err, ErrLastOwner):
				response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "cannot remove the last owner", nil)
			case errors.Is(err, ErrMemberNotFound):
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "member not found", gin.H{"memberId": memberID})
			default:
				response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to remove trip member", nil)
			}
			return
		}
		response.NoContent(c)
		return
	}

	membersMu.Lock()
	defer membersMu.Unlock()

	current := tripMembers[tripID]
	for i := range current {
		if current[i].ID != memberID {
			continue
		}
		if current[i].Role == "owner" && countOwnersLocked(tripID) <= 1 {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "cannot remove the last owner", nil)
			return
		}
		tripMembers[tripID] = append(current[:i], current[i+1:]...)
		response.NoContent(c)
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "member not found", gin.H{"memberId": memberID})
}

func patchTripMember(c *gin.Context) {
	tripID := c.Param("tripId")
	memberID := strings.TrimSpace(c.Param("memberId"))
	if memberID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "memberId is required", nil)
		return
	}

	if _, err := activeRepository.Get(c.Request.Context(), tripID); err != nil {
		if errors.Is(err, ErrTripNotFound) {
			response.Error(c, http.StatusNotFound, perrors.CodeTripNotFound, "trip not found", gin.H{"tripId": tripID})
			return
		}
		response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to load trip", nil)
		return
	}

	var in patchTripMemberInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	if !isValidMemberRole(in.Role) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "role must be owner/editor/commenter/viewer", nil)
		return
	}

	if getCollaborationPool() != nil {
		item, err := patchTripMemberPostgres(c.Request.Context(), tripID, memberID, strings.TrimSpace(in.Role))
		if err != nil {
			switch {
			case errors.Is(err, ErrLastOwner):
				response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "cannot demote the last owner", nil)
			case errors.Is(err, ErrMemberNotFound):
				response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "member not found", gin.H{"memberId": memberID})
			default:
				response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "failed to patch member", nil)
			}
			return
		}
		response.JSON(c, http.StatusOK, item)
		return
	}

	membersMu.Lock()
	defer membersMu.Unlock()

	for i := range tripMembers[tripID] {
		if tripMembers[tripID][i].ID != memberID {
			continue
		}
		oldRole := tripMembers[tripID][i].Role
		newRole := strings.TrimSpace(in.Role)
		if oldRole == "owner" && newRole != "owner" && countOwnersLocked(tripID) <= 1 {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "cannot demote the last owner", nil)
			return
		}
		tripMembers[tripID][i].Role = newRole
		response.JSON(c, http.StatusOK, tripMembers[tripID][i])
		return
	}

	response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "member not found", gin.H{"memberId": memberID})
}

func isValidMemberRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "owner", "editor", "commenter", "viewer":
		return true
	default:
		return false
	}
}

func validateCreateInput(in tripCreateInput) error {
	if in.Name == "" {
		return errText("name is required")
	}
	if len(in.Name) > 200 {
		return errText("name must not exceed 200 characters")
	}
	if in.Timezone == "" {
		return errText("timezone is required")
	}
	if in.Currency == "" || len(in.Currency) != 3 {
		return errText("currency must be ISO-4217 code")
	}
	if in.Travelers < 1 || in.Travelers > 50 {
		return errText("travelersCount must be between 1 and 50")
	}
	return validateDates(in.StartDate, in.EndDate)
}

func isValidStatusTransition(current, next string) bool {
	switch current {
	case "draft":
		return next == "active"
	case "active":
		return next == "archived"
	default:
		return false
	}
}

func countOwnersLocked(tripID string) int {
	count := 0
	for _, m := range tripMembers[tripID] {
		if m.Role == "owner" {
			count++
		}
	}
	return count
}

func validateDates(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return errText("startDate must be in YYYY-MM-DD format")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return errText("endDate must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return errText("endDate must be on or after startDate")
	}
	return nil
}

type errText string

func (e errText) Error() string {
	return string(e)
}
