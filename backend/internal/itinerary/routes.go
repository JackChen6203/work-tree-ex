package itinerary

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type itineraryItem struct {
	ID                    string   `json:"id"`
	DayID                 string   `json:"dayId"`
	Title                 string   `json:"title"`
	ItemType              string   `json:"itemType"`
	StartAt               *string  `json:"startAt,omitempty"`
	EndAt                 *string  `json:"endAt,omitempty"`
	AllDay                bool     `json:"allDay"`
	SortOrder             int      `json:"sortOrder"`
	Note                  *string  `json:"note,omitempty"`
	PlaceID               *string  `json:"placeId,omitempty"`
	Lat                   *float64 `json:"lat,omitempty"`
	Lng                   *float64 `json:"lng,omitempty"`
	EstimatedCostAmount   *float64 `json:"estimatedCostAmount,omitempty"`
	EstimatedCostCurrency *string  `json:"estimatedCostCurrency,omitempty"`
	Version               int      `json:"version"`
}

type itineraryDay struct {
	DayID     string          `json:"dayId"`
	Date      string          `json:"date"`
	SortOrder int             `json:"sortOrder"`
	Items     []itineraryItem `json:"items"`
}

type itemCreateInput struct {
	DayID    string   `json:"dayId"`
	Title    string   `json:"title"`
	ItemType string   `json:"itemType"`
	StartAt  *string  `json:"startAt"`
	EndAt    *string  `json:"endAt"`
	AllDay   bool     `json:"allDay"`
	Note     *string  `json:"note"`
	PlaceID  *string  `json:"placeId"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

type itemPatchInput struct {
	DayID     *string  `json:"dayId"`
	Title     *string  `json:"title"`
	StartAt   *string  `json:"startAt"`
	EndAt     *string  `json:"endAt"`
	AllDay    *bool    `json:"allDay"`
	Note      *string  `json:"note"`
	SortOrder *int     `json:"sortOrder"`
	PlaceID   *string  `json:"placeId"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
}

type reorderInput struct {
	Operations []struct {
		ItemID          string `json:"itemId"`
		TargetDayID     string `json:"targetDayId"`
		TargetSortOrder int    `json:"targetSortOrder"`
	} `json:"operations"`
}

var (
	itineraryMu           sync.RWMutex
	daysByTrip            = map[string][]itineraryDay{}
	itemByID              = map[string]itineraryItem{}
	itemTripByID          = map[string]string{}
	itemCreateIdempotency = map[string]string{}
	reorderIdempotency    = map[string]string{}
)

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/trips/:tripId/days", listDays)
	v1.POST("/trips/:tripId/items", createItem)
	v1.PATCH("/trips/:tripId/items/:itemId", patchItem)
	v1.DELETE("/trips/:tripId/items/:itemId", deleteItem)
	v1.POST("/trips/:tripId/items/reorder", reorderItems)
}

func listDays(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	itineraryMu.Lock()
	ensureSeededLocked(tripID)
	items := cloneDays(daysByTrip[tripID])
	itineraryMu.Unlock()

	response.JSON(c, http.StatusOK, items)
}

func createItem(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in itemCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(in.DayID) == "" || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.ItemType) == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "dayId, title, and itemType are required", nil)
		return
	}
	if !isValidItemType(in.ItemType) {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "itemType must be place_visit/meal/transit/hotel/free_time/custom", nil)
		return
	}
	if len(in.Title) > 200 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "title must not exceed 200 characters", nil)
		return
	}
	if in.Note != nil && len(*in.Note) > 5000 {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "note must not exceed 5000 characters", nil)
		return
	}

	itineraryMu.Lock()
	if existingID, ok := itemCreateIdempotency[idempotencyKey]; ok {
		existing := itemByID[existingID]
		itineraryMu.Unlock()
		response.JSON(c, http.StatusCreated, existing)
		return
	}

	ensureSeededLocked(tripID)
	item := itineraryItem{
		ID:        uuid.NewString(),
		DayID:     in.DayID,
		Title:     in.Title,
		ItemType:  in.ItemType,
		StartAt:   in.StartAt,
		EndAt:     in.EndAt,
		AllDay:    in.AllDay,
		SortOrder: nextSortLocked(tripID, in.DayID),
		Note:      in.Note,
		PlaceID:   in.PlaceID,
		Lat:       in.Lat,
		Lng:       in.Lng,
		Version:   1,
	}

	if !attachItemLocked(tripID, item) {
		itineraryMu.Unlock()
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "dayId not found", gin.H{"dayId": in.DayID})
		return
	}

	itemByID[item.ID] = item
	itemTripByID[item.ID] = tripID
	itemCreateIdempotency[idempotencyKey] = item.ID
	itineraryMu.Unlock()

	// Detect time overlaps in the same day
	warnings := detectTimeOverlapsLocked(tripID, item.DayID)

	result := gin.H{"item": item}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}

	response.JSON(c, http.StatusCreated, result)
}

func patchItem(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	itemID := strings.TrimSpace(c.Param("itemId"))

	ifMatch := strings.TrimSpace(c.GetHeader("If-Match-Version"))
	if ifMatch == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "If-Match-Version header is required", nil)
		return
	}

	expectedVersion, err := strconv.Atoi(ifMatch)
	if err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "If-Match-Version must be integer", nil)
		return
	}

	var in itemPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	itineraryMu.Lock()
	item, ok := itemByID[itemID]
	if !ok || itemTripByID[itemID] != tripID {
		itineraryMu.Unlock()
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "itinerary item not found", gin.H{"itemId": itemID})
		return
	}
	if item.Version != expectedVersion {
		itineraryMu.Unlock()
		response.Error(c, http.StatusConflict, perrors.CodeVersionConflict, "item version conflict", nil)
		return
	}

	if in.Title != nil {
		item.Title = *in.Title
	}
	if in.StartAt != nil {
		item.StartAt = in.StartAt
	}
	if in.EndAt != nil {
		item.EndAt = in.EndAt
	}
	if in.AllDay != nil {
		item.AllDay = *in.AllDay
	}
	if in.Note != nil {
		item.Note = in.Note
	}
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	if in.PlaceID != nil {
		item.PlaceID = in.PlaceID
	}
	if in.Lat != nil {
		item.Lat = in.Lat
	}
	if in.Lng != nil {
		item.Lng = in.Lng
	}

	// Cross-day move: remove from source day, attach to target day
	if in.DayID != nil && *in.DayID != item.DayID {
		oldDayID := item.DayID
		newDayID := *in.DayID

		// Remove from source day
		removeItemFromDayLocked(tripID, oldDayID, item.ID)

		// Attach to target day
		item.DayID = newDayID
		item.SortOrder = nextSortLocked(tripID, newDayID)
		if !attachItemLocked(tripID, item) {
			// Rollback: re-attach to old day
			item.DayID = oldDayID
			attachItemLocked(tripID, item)
			itineraryMu.Unlock()
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "target dayId not found", gin.H{"dayId": newDayID})
			return
		}
	} else {
		replaceItemLocked(tripID, item)
	}

	item.Version++
	itemByID[item.ID] = item
	itineraryMu.Unlock()

	response.JSON(c, http.StatusOK, item)
}

func deleteItem(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	itemID := strings.TrimSpace(c.Param("itemId"))

	itineraryMu.Lock()
	item, ok := itemByID[itemID]
	if !ok || itemTripByID[itemID] != tripID {
		itineraryMu.Unlock()
		response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "itinerary item not found", gin.H{"itemId": itemID})
		return
	}

	detachItemLocked(tripID, item.DayID, itemID)
	delete(itemByID, itemID)
	delete(itemTripByID, itemID)
	itineraryMu.Unlock()

	response.NoContent(c)
}

func reorderItems(c *gin.Context) {
	tripID := strings.TrimSpace(c.Param("tripId"))
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "Idempotency-Key header is required", nil)
		return
	}

	var in reorderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	itineraryMu.Lock()
	if reorderIdempotency[idempotencyKey] == tripID {
		items := cloneDays(daysByTrip[tripID])
		itineraryMu.Unlock()
		response.JSON(c, http.StatusOK, items)
		return
	}

	ensureSeededLocked(tripID)
	for _, op := range in.Operations {
		item, ok := itemByID[op.ItemID]
		if !ok || itemTripByID[op.ItemID] != tripID {
			itineraryMu.Unlock()
			response.Error(c, http.StatusNotFound, perrors.CodeNotFound, "itinerary item not found", gin.H{"itemId": op.ItemID})
			return
		}
		detachItemLocked(tripID, item.DayID, item.ID)
		item.DayID = op.TargetDayID
		item.SortOrder = op.TargetSortOrder
		item.Version++
		itemByID[item.ID] = item
		if !attachItemLocked(tripID, item) {
			itineraryMu.Unlock()
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "targetDayId not found", gin.H{"targetDayId": op.TargetDayID})
			return
		}
	}

	for i := range daysByTrip[tripID] {
		sort.SliceStable(daysByTrip[tripID][i].Items, func(a, b int) bool {
			return daysByTrip[tripID][i].Items[a].SortOrder < daysByTrip[tripID][i].Items[b].SortOrder
		})
	}

	reorderIdempotency[idempotencyKey] = tripID
	items := cloneDays(daysByTrip[tripID])
	itineraryMu.Unlock()

	response.JSON(c, http.StatusOK, items)
}

func ensureSeededLocked(tripID string) {
	if _, ok := daysByTrip[tripID]; ok {
		return
	}

	daysByTrip[tripID] = []itineraryDay{
		{DayID: "day-1", Date: "2026-04-14", SortOrder: 1, Items: []itineraryItem{}},
		{DayID: "day-2", Date: "2026-04-15", SortOrder: 2, Items: []itineraryItem{}},
	}

	seed := []itineraryItem{
		{ID: "i-1", DayID: "day-1", Title: "清水寺晨間參拜", ItemType: "place_visit", AllDay: false, SortOrder: 1, Version: 1},
		{ID: "i-2", DayID: "day-1", Title: "二年坂咖啡與散策", ItemType: "meal", AllDay: false, SortOrder: 2, Version: 1},
		{ID: "i-3", DayID: "day-2", Title: "嵐山竹林晨拍", ItemType: "place_visit", AllDay: false, SortOrder: 1, Version: 1},
	}
	for _, item := range seed {
		attachItemLocked(tripID, item)
		itemByID[item.ID] = item
		itemTripByID[item.ID] = tripID
	}
}

func attachItemLocked(tripID string, item itineraryItem) bool {
	for i := range daysByTrip[tripID] {
		if daysByTrip[tripID][i].DayID == item.DayID {
			daysByTrip[tripID][i].Items = append(daysByTrip[tripID][i].Items, item)
			return true
		}
	}
	return false
}

func detachItemLocked(tripID, dayID, itemID string) {
	for i := range daysByTrip[tripID] {
		if daysByTrip[tripID][i].DayID != dayID {
			continue
		}
		filtered := make([]itineraryItem, 0, len(daysByTrip[tripID][i].Items))
		for _, candidate := range daysByTrip[tripID][i].Items {
			if candidate.ID != itemID {
				filtered = append(filtered, candidate)
			}
		}
		daysByTrip[tripID][i].Items = filtered
		return
	}
}

func removeItemFromDayLocked(tripID, dayID, itemID string) {
	for i := range daysByTrip[tripID] {
		if daysByTrip[tripID][i].DayID != dayID {
			continue
		}
		items := daysByTrip[tripID][i].Items
		for j := range items {
			if items[j].ID == itemID {
				daysByTrip[tripID][i].Items = append(items[:j], items[j+1:]...)
				return
			}
		}
	}
}

func replaceItemLocked(tripID string, item itineraryItem) {
	for i := range daysByTrip[tripID] {
		for j := range daysByTrip[tripID][i].Items {
			if daysByTrip[tripID][i].Items[j].ID == item.ID {
				daysByTrip[tripID][i].Items[j] = item
				return
			}
		}
	}
}

func nextSortLocked(tripID, dayID string) int {
	maxSort := 0
	for _, day := range daysByTrip[tripID] {
		if day.DayID != dayID {
			continue
		}
		for _, item := range day.Items {
			if item.SortOrder > maxSort {
				maxSort = item.SortOrder
			}
		}
	}
	return maxSort + 1
}

func cloneDays(days []itineraryDay) []itineraryDay {
	result := make([]itineraryDay, len(days))
	for i := range days {
		result[i] = itineraryDay{
			DayID:     days[i].DayID,
			Date:      days[i].Date,
			SortOrder: days[i].SortOrder,
			Items:     append([]itineraryItem{}, days[i].Items...),
		}
	}
	return result
}

func isValidItemType(itemType string) bool {
	switch strings.TrimSpace(itemType) {
	case "place_visit", "meal", "transit", "hotel", "free_time", "custom":
		return true
	default:
		return false
	}
}

func detectTimeOverlapsLocked(tripID, dayID string) []string {
	var warnings []string
	var timedItems []itineraryItem

	for _, day := range daysByTrip[tripID] {
		if day.DayID != dayID {
			continue
		}
		for _, item := range day.Items {
			if item.StartAt != nil && item.EndAt != nil {
				timedItems = append(timedItems, item)
			}
		}
	}

	for i := 0; i < len(timedItems); i++ {
		for j := i + 1; j < len(timedItems); j++ {
			a := timedItems[i]
			b := timedItems[j]
			// simple string comparison works for ISO timestamps
			if *a.StartAt < *b.EndAt && *b.StartAt < *a.EndAt {
				warnings = append(warnings, "time overlap between '"+a.Title+"' and '"+b.Title+"'")
			}
		}
	}

	return warnings
}
