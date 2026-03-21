package maps

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

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

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/maps/search", searchPlaces)
	v1.POST("/maps/routes", estimateRoute)
}

func searchPlaces(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "q query is required", nil)
		return
	}

	lat := parseFloatWithDefault(c.Query("lat"), 35.0116)
	lng := parseFloatWithDefault(c.Query("lng"), 135.7681)

	items := []gin.H{
		{
			"providerPlaceId": "poi_kiyomizu",
			"name":            "Kiyomizu-dera",
			"address":         "Kyoto Higashiyama Ward",
			"lat":             lat + 0.004,
			"lng":             lng + 0.005,
			"categories":      []string{"temple", "landmark"},
		},
		{
			"providerPlaceId": "poi_ninenzaka",
			"name":            "Ninenzaka",
			"address":         "Kyoto Higashiyama Ward",
			"lat":             lat + 0.006,
			"lng":             lng + 0.003,
			"categories":      []string{"shopping", "street"},
		},
		{
			"providerPlaceId": "poi_pontocho",
			"name":            "Pontocho Alley",
			"address":         "Kyoto Nakagyo Ward",
			"lat":             lat + 0.002,
			"lng":             lng + 0.001,
			"categories":      []string{"food", "nightlife"},
		},
	}

	filtered := make([]gin.H, 0, len(items))
	queryLower := strings.ToLower(q)
	for _, item := range items {
		name := strings.ToLower(item["name"].(string))
		if strings.Contains(name, queryLower) || strings.Contains(queryLower, "kyoto") {
			filtered = append(filtered, item)
		}
	}

	response.JSON(c, http.StatusOK, filtered)
}

func estimateRoute(c *gin.Context) {
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
		cost := math.Round((float64(distanceMeters)/1000*220)+120) // mocked estimate in JPY
		costAmount = &cost
		currency := "JPY"
		costCurrency = &currency
	}

	response.JSON(c, http.StatusOK, gin.H{
		"mode":                  mode,
		"distanceMeters":        distanceMeters,
		"durationSeconds":       durationSeconds,
		"estimatedCostAmount":   costAmount,
		"estimatedCostCurrency": costCurrency,
		"provider":              "mock-map-adapter",
		"snapshotToken":         fmt.Sprintf("rt_%d", distanceMeters+durationSeconds),
	})
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
