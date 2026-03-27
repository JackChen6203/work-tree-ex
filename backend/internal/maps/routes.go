package maps

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type routeEstimateInput struct {
	Origin struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"origin"`
	Destination struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"destination"`
	Mode string `json:"mode"`
}

var (
	quotaCounter atomic.Int64
	quotaLimit   int64 = 10000
	quotaResetAt time.Time
	quotaMu      sync.Mutex

	rpsLimit  int64 = 20
	rpsWindow time.Time
	rpsCount  int64
	rpsMu     sync.Mutex

	limitsOnce sync.Once
)

func initProviderLimits() {
	limitsOnce.Do(func() {
		quotaLimit = int64(getEnvInt("MAP_DAILY_QUOTA", 10000))
		rpsLimit = int64(getEnvInt("MAP_RPS_LIMIT", 20))
		if quotaLimit <= 0 {
			quotaLimit = 10000
		}
		if rpsLimit <= 0 {
			rpsLimit = 20
		}
	})
}

func enforceProviderLimits() (allowed bool, reason string) {
	initProviderLimits()
	now := time.Now().UTC()

	rpsMu.Lock()
	currentWindow := now.Truncate(time.Second)
	if rpsWindow.IsZero() || !rpsWindow.Equal(currentWindow) {
		rpsWindow = currentWindow
		rpsCount = 0
	}
	if rpsCount >= rpsLimit {
		rpsMu.Unlock()
		return false, "rps"
	}
	rpsCount++
	rpsMu.Unlock()

	quotaMu.Lock()
	if quotaResetAt.IsZero() || !now.Before(quotaResetAt) {
		quotaCounter.Store(0)
		quotaResetAt = nextUTCMidnight(now)
	}
	quotaMu.Unlock()

	current := quotaCounter.Add(1)
	if current > quotaLimit {
		return false, "quota"
	}
	return true, ""
}

func nextUTCMidnight(now time.Time) time.Time {
	year, month, day := now.Date()
	return time.Date(year, month, day+1, 0, 0, 0, 0, time.UTC)
}

var placeStore = map[string]NormalizedPlace{
	"poi_kiyomizu":  {ProviderPlaceID: "poi_kiyomizu", Name: "Kiyomizu-dera", Address: "Kyoto Higashiyama Ward", Lat: 34.9949, Lng: 135.7850, Categories: []string{"temple", "landmark"}},
	"poi_ninenzaka": {ProviderPlaceID: "poi_ninenzaka", Name: "Ninenzaka", Address: "Kyoto Higashiyama Ward", Lat: 34.9984, Lng: 135.7809, Categories: []string{"shopping", "street"}},
	"poi_pontocho":  {ProviderPlaceID: "poi_pontocho", Name: "Pontocho Alley", Address: "Kyoto Nakagyo Ward", Lat: 35.0048, Lng: 135.7708, Categories: []string{"food", "nightlife"}},
}

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/maps/search", searchPlaces)
	v1.POST("/maps/routes", estimateRoute)
	v1.GET("/maps/geocode", geocode)
	v1.GET("/maps/reverse-geocode", reverseGeocode)
	v1.GET("/maps/places/:placeId", getPlaceDetail)
}

func searchPlaces(c *gin.Context) {
	if ok, reason := enforceProviderLimits(); !ok {
		respondLimitError(c, reason)
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "q query is required", nil)
		return
	}

	limit := 5
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 20 {
			response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "limit must be an integer between 1 and 20", nil)
			return
		}
		limit = parsed
	}

	lat := parseFloatWithDefault(c.Query("lat"), 35.0116)
	lng := parseFloatWithDefault(c.Query("lng"), 135.7681)

	providers := loadProviderChainFromEnv()
	if len(providers) == 0 {
		response.JSON(c, http.StatusOK, mockSearchPlaces(q, limit, lat, lng))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultProviderTimeout)
	defer cancel()

	items, err := searchPlacesWithFallback(ctx, providers, PlaceSearchRequest{
		Query: q,
		Lat:   lat,
		Lng:   lng,
		Limit: limit,
	})
	if err != nil {
		respondProviderError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func estimateRoute(c *gin.Context) {
	if ok, reason := enforceProviderLimits(); !ok {
		respondLimitError(c, reason)
		return
	}

	var in routeEstimateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

	providers := loadProviderChainFromEnv()
	if len(providers) == 0 {
		response.JSON(c, http.StatusOK, mockEstimateRoute(in))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultProviderTimeout)
	defer cancel()

	route, err := estimateRouteWithFallback(ctx, providers, RouteEstimateRequest{
		OriginLat:      in.Origin.Lat,
		OriginLng:      in.Origin.Lng,
		DestinationLat: in.Destination.Lat,
		DestinationLng: in.Destination.Lng,
		Mode:           in.Mode,
	})
	if err != nil {
		respondProviderError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, route)
}

func geocode(c *gin.Context) {
	if ok, reason := enforceProviderLimits(); !ok {
		respondLimitError(c, reason)
		return
	}

	address := strings.TrimSpace(c.Query("address"))
	if address == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "address query is required", nil)
		return
	}

	providers := loadProviderChainFromEnv()
	if len(providers) == 0 {
		response.JSON(c, http.StatusOK, mockGeocode(address))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultProviderTimeout)
	defer cancel()

	item, err := geocodeWithFallback(ctx, providers, address)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func reverseGeocode(c *gin.Context) {
	if ok, reason := enforceProviderLimits(); !ok {
		respondLimitError(c, reason)
		return
	}

	latStr := strings.TrimSpace(c.Query("lat"))
	lngStr := strings.TrimSpace(c.Query("lng"))
	if latStr == "" || lngStr == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "lat and lng query params are required", nil)
		return
	}

	lat := parseFloatWithDefault(latStr, 0)
	lng := parseFloatWithDefault(lngStr, 0)

	providers := loadProviderChainFromEnv()
	if len(providers) == 0 {
		response.JSON(c, http.StatusOK, mockReverseGeocode(lat, lng))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultProviderTimeout)
	defer cancel()

	item, err := reverseGeocodeWithFallback(ctx, providers, lat, lng)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func getPlaceDetail(c *gin.Context) {
	if ok, reason := enforceProviderLimits(); !ok {
		respondLimitError(c, reason)
		return
	}

	placeID := strings.TrimSpace(c.Param("placeId"))
	if placeID == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "placeId is required", nil)
		return
	}

	providers := loadProviderChainFromEnv()
	if len(providers) == 0 {
		response.JSON(c, http.StatusOK, mockGetPlaceDetail(placeID))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultProviderTimeout)
	defer cancel()

	item, err := placeDetailWithFallback(ctx, providers, placeID)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func searchPlacesWithFallback(ctx context.Context, providers []MapProvider, req PlaceSearchRequest) ([]NormalizedPlace, error) {
	var lastErr error
	for _, provider := range providers {
		items, err := provider.SearchPlaces(ctx, req)
		if err == nil {
			return items, nil
		}
		lastErr = err
		if !shouldFallback(err) {
			break
		}
	}
	if lastErr == nil {
		return nil, errors.New("no provider available")
	}
	return nil, lastErr
}

func placeDetailWithFallback(ctx context.Context, providers []MapProvider, placeID string) (*NormalizedPlace, error) {
	var lastErr error
	for _, provider := range providers {
		item, err := provider.GetPlaceDetail(ctx, placeID)
		if err == nil {
			return item, nil
		}
		lastErr = err
		if !shouldFallback(err) {
			break
		}
	}
	if lastErr == nil {
		return nil, errors.New("no provider available")
	}
	return nil, lastErr
}

func estimateRouteWithFallback(ctx context.Context, providers []MapProvider, req RouteEstimateRequest) (*NormalizedRoute, error) {
	var lastErr error
	for _, provider := range providers {
		item, err := provider.EstimateRoute(ctx, req)
		if err == nil {
			return item, nil
		}
		lastErr = err
		if !shouldFallback(err) {
			break
		}
	}
	if lastErr == nil {
		return nil, errors.New("no provider available")
	}
	return nil, lastErr
}

func geocodeWithFallback(ctx context.Context, providers []MapProvider, address string) (*NormalizedPlace, error) {
	var lastErr error
	for _, provider := range providers {
		item, err := provider.Geocode(ctx, address)
		if err == nil {
			return item, nil
		}
		lastErr = err
		if !shouldFallback(err) {
			break
		}
	}
	if lastErr == nil {
		return nil, errors.New("no provider available")
	}
	return nil, lastErr
}

func reverseGeocodeWithFallback(ctx context.Context, providers []MapProvider, lat, lng float64) (*NormalizedPlace, error) {
	var lastErr error
	for _, provider := range providers {
		item, err := provider.ReverseGeocode(ctx, lat, lng)
		if err == nil {
			return item, nil
		}
		lastErr = err
		if !shouldFallback(err) {
			break
		}
	}
	if lastErr == nil {
		return nil, errors.New("no provider available")
	}
	return nil, lastErr
}

func shouldFallback(err error) bool {
	var timeoutErr ProviderTimeoutError
	var quotaErr ProviderQuotaError
	var apiErr ProviderAPIError
	return errors.As(err, &timeoutErr) || errors.As(err, &quotaErr) || errors.As(err, &apiErr)
}

func respondLimitError(c *gin.Context, reason string) {
	switch reason {
	case "quota":
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed, "map provider quota exceeded, try again later", nil)
	default:
		response.Error(c, http.StatusTooManyRequests, perrors.CodeRateLimitExceeded, "map provider rate limit exceeded, try again later", nil)
	}
}

func respondProviderError(c *gin.Context, err error) {
	var timeoutErr ProviderTimeoutError
	if errors.As(err, &timeoutErr) {
		response.Error(c, http.StatusGatewayTimeout, perrors.CodeMapProviderTimeout, "map provider timed out", gin.H{"provider": timeoutErr.Provider})
		return
	}

	var quotaErr ProviderQuotaError
	if errors.As(err, &quotaErr) {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed, "map provider quota exceeded", gin.H{"provider": quotaErr.Provider})
		return
	}

	var apiErr ProviderAPIError
	if errors.As(err, &apiErr) {
		response.Error(c, http.StatusBadGateway, perrors.CodeInternalError, "map provider error", gin.H{"provider": apiErr.Provider})
		return
	}

	response.Error(c, http.StatusInternalServerError, perrors.CodeInternalError, "map provider error", nil)
}

func mockSearchPlaces(q string, limit int, lat, lng float64) []NormalizedPlace {
	items := []NormalizedPlace{
		{ProviderPlaceID: "poi_kiyomizu", Name: "Kiyomizu-dera", Address: "Kyoto Higashiyama Ward", Lat: lat + 0.004, Lng: lng + 0.005, Categories: []string{"temple", "landmark"}},
		{ProviderPlaceID: "poi_ninenzaka", Name: "Ninenzaka", Address: "Kyoto Higashiyama Ward", Lat: lat + 0.006, Lng: lng + 0.003, Categories: []string{"shopping", "street"}},
		{ProviderPlaceID: "poi_pontocho", Name: "Pontocho Alley", Address: "Kyoto Nakagyo Ward", Lat: lat + 0.002, Lng: lng + 0.001, Categories: []string{"food", "nightlife"}},
	}

	filtered := make([]NormalizedPlace, 0, len(items))
	queryLower := strings.ToLower(q)
	for _, item := range items {
		name := strings.ToLower(item.Name)
		if strings.Contains(name, queryLower) || strings.Contains(queryLower, "kyoto") {
			filtered = append(filtered, item)
			if len(filtered) == limit {
				break
			}
		}
	}
	return filtered
}

func mockEstimateRoute(in routeEstimateInput) NormalizedRoute {
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = "transit"
	}

	distanceKm := haversineKm(in.Origin.Lat, in.Origin.Lng, in.Destination.Lat, in.Destination.Lng)
	distanceMeters := int(distanceKm * 1000)
	if distanceMeters < 50 {
		distanceMeters = 50
	}

	speedKmh := map[string]float64{"walk": 4.5, "transit": 24, "drive": 32, "taxi": 30}[mode]
	if speedKmh == 0 {
		speedKmh = 24
	}
	durationSeconds := int((distanceKm / speedKmh) * 3600)
	if durationSeconds < 180 {
		durationSeconds = 180
	}

	var costAmount *float64
	var costCurrency *string
	if mode == "transit" || mode == "taxi" || mode == "drive" {
		cost := math.Round((float64(distanceMeters) / 1000 * 220) + 120)
		costAmount = &cost
		currency := "JPY"
		costCurrency = &currency
	}

	return NormalizedRoute{
		Mode:                  mode,
		DistanceMeters:        distanceMeters,
		DurationSeconds:       durationSeconds,
		EstimatedCostAmount:   costAmount,
		EstimatedCostCurrency: costCurrency,
	}
}

func mockGeocode(address string) NormalizedPlace {
	return NormalizedPlace{
		ProviderPlaceID: fmt.Sprintf("geocoded_%d", time.Now().UnixNano()%10000),
		Name:            address,
		Address:         address,
		Lat:             35.0116,
		Lng:             135.7681,
		Categories:      []string{},
	}
}

func mockReverseGeocode(lat, lng float64) NormalizedPlace {
	return NormalizedPlace{
		ProviderPlaceID: fmt.Sprintf("rev_%d", time.Now().UnixNano()%10000),
		Name:            fmt.Sprintf("Location at %.4f, %.4f", lat, lng),
		Address:         fmt.Sprintf("%.4f, %.4f", lat, lng),
		Lat:             lat,
		Lng:             lng,
		Categories:      []string{},
	}
}

func mockGetPlaceDetail(placeID string) any {
	place, ok := placeStore[placeID]
	if ok {
		return place
	}
	return gin.H{
		"providerPlaceId": placeID,
		"name":            placeID,
		"address":         "",
		"lat":             0,
		"lng":             0,
		"categories":      []string{},
		"warnings":        []string{"place not found in provider, data may be incomplete"},
	}
}

func parseFloatWithDefault(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return value
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(v float64) float64 { return v * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
