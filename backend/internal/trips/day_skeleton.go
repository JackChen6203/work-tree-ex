package trips

import (
	"time"

	"github.com/google/uuid"
)

// DaySkeleton represents a generated itinerary day for a trip.
type DaySkeleton struct {
	ID        string    `json:"id"`
	TripID    string    `json:"tripId"`
	TripDate  string    `json:"tripDate"`
	DayIndex  int       `json:"dayIndex"`
	SortOrder int       `json:"sortOrder"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// GenerateDaySkeletons creates day entries for each day in the trip date range.
func GenerateDaySkeletons(tripID, startDate, endDate string) []DaySkeleton {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	var days []DaySkeleton

	for d, idx := start, 0; !d.After(end); d, idx = d.AddDate(0, 0, 1), idx+1 {
		days = append(days, DaySkeleton{
			ID:        uuid.NewString(),
			TripID:    tripID,
			TripDate:  d.Format("2006-01-02"),
			DayIndex:  idx,
			SortOrder: idx,
			Version:   1,
			CreatedAt: now,
		})
	}

	return days
}
