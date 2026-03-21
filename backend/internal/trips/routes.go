package trips

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type trip struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Destination string    `json:"destinationText,omitempty"`
	StartDate   string    `json:"startDate"`
	EndDate     string    `json:"endDate"`
	Timezone    string    `json:"timezone"`
	Currency    string    `json:"currency"`
	Travelers   int       `json:"travelersCount"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type tripCreateInput struct {
	Name        string `json:"name"`
	Destination string `json:"destinationText"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Timezone    string `json:"timezone"`
	Currency    string `json:"currency"`
	Travelers   int    `json:"travelersCount"`
}

type tripPatchInput struct {
	Name        *string `json:"name"`
	Destination *string `json:"destinationText"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
	Timezone    *string `json:"timezone"`
	Currency    *string `json:"currency"`
	Travelers   *int    `json:"travelersCount"`
	Status      *string `json:"status"`
}

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips", listTrips)
	v1.POST("/trips", createTrip)
	v1.GET("/trips/:tripId", getTrip)
	v1.PATCH("/trips/:tripId", patchTrip)
	v1.GET("/trips/:tripId/members", func(c *gin.Context) {
		response.NotImplemented(c, "GET /trips/{tripId}/members")
	})
	v1.POST("/trips/:tripId/members", func(c *gin.Context) {
		response.NotImplemented(c, "POST /trips/{tripId}/members")
	})
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

	response.JSON(c, http.StatusCreated, t)
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

func validateCreateInput(in tripCreateInput) error {
	if in.Name == "" {
		return errText("name is required")
	}
	if in.Timezone == "" {
		return errText("timezone is required")
	}
	if in.Currency == "" || len(in.Currency) != 3 {
		return errText("currency must be ISO-4217 code")
	}
	if in.Travelers < 1 {
		return errText("travelersCount must be at least 1")
	}
	return validateDates(in.StartDate, in.EndDate)
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
