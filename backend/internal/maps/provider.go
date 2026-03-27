package maps

import (
	"context"
	"fmt"
)

// MapProvider defines the interface for map service providers.
type MapProvider interface {
	SearchPlaces(ctx context.Context, req PlaceSearchRequest) ([]NormalizedPlace, error)
	GetPlaceDetail(ctx context.Context, providerPlaceID string) (*NormalizedPlace, error)
	EstimateRoute(ctx context.Context, req RouteEstimateRequest) (*NormalizedRoute, error)
	Geocode(ctx context.Context, address string) (*NormalizedPlace, error)
	ReverseGeocode(ctx context.Context, lat, lng float64) (*NormalizedPlace, error)
	Name() string
}

// PlaceSearchRequest holds parameters for a place search.
type PlaceSearchRequest struct {
	Query string  `json:"query"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Limit int     `json:"limit"`
}

// RouteEstimateRequest holds parameters for a route estimation.
type RouteEstimateRequest struct {
	OriginLat      float64 `json:"originLat"`
	OriginLng      float64 `json:"originLng"`
	DestinationLat float64 `json:"destinationLat"`
	DestinationLng float64 `json:"destinationLng"`
	Mode           string  `json:"mode"` // walk | transit | drive | taxi
}

// NormalizedPlace is the standardized place DTO across providers.
type NormalizedPlace struct {
	ProviderPlaceID string   `json:"providerPlaceId"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Lat             float64  `json:"lat"`
	Lng             float64  `json:"lng"`
	Categories      []string `json:"categories"`
	OpeningHours    *string  `json:"openingHours,omitempty"`
}

// NormalizedRoute is the standardized route DTO across providers.
type NormalizedRoute struct {
	Mode                  string   `json:"mode"`
	DistanceMeters        int      `json:"distanceMeters"`
	DurationSeconds       int      `json:"durationSeconds"`
	EstimatedCostAmount   *float64 `json:"estimatedCostAmount,omitempty"`
	EstimatedCostCurrency *string  `json:"estimatedCostCurrency,omitempty"`
}

type ProviderTimeoutError struct {
	Provider string
}

func (e ProviderTimeoutError) Error() string {
	return fmt.Sprintf("%s provider timed out", e.Provider)
}

type ProviderQuotaError struct {
	Provider string
}

func (e ProviderQuotaError) Error() string {
	return fmt.Sprintf("%s provider quota exceeded", e.Provider)
}

type ProviderAPIError struct {
	Provider   string
	StatusCode int
	Message    string
}

func (e ProviderAPIError) Error() string {
	return fmt.Sprintf("%s provider api error: status=%d message=%s", e.Provider, e.StatusCode, e.Message)
}
