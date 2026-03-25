package maps

import (
	"fmt"
	"math"
	"net/http"
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

// --- Quota / Circuit Breaker ---

var (
	quotaCounter atomic.Int64
	quotaLimit   int64 = 1000
	quotaResetAt time.Time
	quotaMu      sync.Mutex
	circuitOpen  bool
)

func checkQuota() bool {
	quotaMu.Lock()
	defer quotaMu.Unlock()
	now := time.Now().UTC()
	if now.After(quotaResetAt) {
		quotaCounter.Store(0)
		quotaResetAt = now.Add(1 * time.Hour)
		circuitOpen = false
	}
	if circuitOpen {
		return false
	}
	current := quotaCounter.Add(1)
	if current > quotaLimit {
		circuitOpen = true
		return false
	}
	return true
}

// --- Place store for detail lookups ---

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
	if !checkQuota() {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed,
			"map provider quota exceeded, try again later", nil)
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

	response.JSON(c, http.StatusOK, filtered)
}

func estimateRoute(c *gin.Context) {
	if !checkQuota() {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed,
			"map provider quota exceeded, try again later", nil)
		return
	}

	var in routeEstimateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "invalid request body", gin.H{"error": err.Error()})
		return
	}

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

	route := NormalizedRoute{
		Mode:                  mode,
		DistanceMeters:        distanceMeters,
		DurationSeconds:       durationSeconds,
		EstimatedCostAmount:   costAmount,
		EstimatedCostCurrency: costCurrency,
	}

	response.JSON(c, http.StatusOK, route)
}

func geocode(c *gin.Context) {
	if !checkQuota() {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed,
			"map provider quota exceeded, try again later", nil)
		return
	}

	address := strings.TrimSpace(c.Query("address"))
	if address == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "address query is required", nil)
		return
	}

	result := NormalizedPlace{
		ProviderPlaceID: fmt.Sprintf("geocoded_%d", time.Now().UnixNano()%10000),
		Name:            address,
		Address:         address,
		Lat:             35.0116,
		Lng:             135.7681,
		Categories:      []string{},
	}

	response.JSON(c, http.StatusOK, result)
}

func reverseGeocode(c *gin.Context) {
	if !checkQuota() {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed,
			"map provider quota exceeded, try again later", nil)
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

	result := NormalizedPlace{
		ProviderPlaceID: fmt.Sprintf("rev_%d", time.Now().UnixNano()%10000),
		Name:            fmt.Sprintf("Location at %.4f, %.4f", lat, lng),
		Address:         fmt.Sprintf("%.4f, %.4f", lat, lng),
		Lat:             lat,
		Lng:             lng,
		Categories:      []string{},
	}

	response.JSON(c, http.StatusOK, result)
}

func getPlaceDetail(c *gin.Context) {
	if !checkQuota() {
		response.Error(c, http.StatusTooManyRequests, perrors.CodeMapProviderQuotaExceed,
			"map provider quota exceeded, try again later", nil)
		return
	}

	placeID := strings.TrimSpace(c.Param("placeId"))
	place, ok := placeStore[placeID]
	if !ok {
		// Partial warning: return minimal place with warning
		response.JSON(c, http.StatusOK, gin.H{
			"providerPlaceId": placeID,
			"name":            placeID,
			"address":         "",
			"lat":             0,
			"lng":             0,
			"categories":      []string{},
			"warnings":        []string{"place not found in provider, data may be incomplete"},
		})
		return
	}

	response.JSON(c, http.StatusOK, place)
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
