package search

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type searchResult struct {
	ProviderPlaceID string   `json:"providerPlaceId"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Lat             float64  `json:"lat"`
	Lng             float64  `json:"lng"`
	Categories      []string `json:"categories"`
	Tags            []string `json:"tags,omitempty"`
	Score           float64  `json:"score"`
}

// Mock data for search
var mockPlaces = []searchResult{
	{ProviderPlaceID: "poi_kiyomizu", Name: "Kiyomizu-dera", Address: "Kyoto Higashiyama", Lat: 34.9949, Lng: 135.7850, Categories: []string{"temple"}, Tags: []string{"landmark", "culture", "kyoto"}, Score: 0.98},
	{ProviderPlaceID: "poi_fushimi", Name: "Fushimi Inari Shrine", Address: "Kyoto Fushimi", Lat: 34.9671, Lng: 135.7727, Categories: []string{"shrine"}, Tags: []string{"landmark", "hiking", "kyoto"}, Score: 0.97},
	{ProviderPlaceID: "poi_arashiyama", Name: "Arashiyama Bamboo Grove", Address: "Kyoto Ukyo", Lat: 35.0094, Lng: 135.6674, Categories: []string{"nature"}, Tags: []string{"nature", "walk", "kyoto"}, Score: 0.95},
	{ProviderPlaceID: "poi_nishiki", Name: "Nishiki Market", Address: "Kyoto Nakagyo", Lat: 35.0050, Lng: 135.7648, Categories: []string{"food"}, Tags: []string{"food", "market", "kyoto"}, Score: 0.93},
	{ProviderPlaceID: "poi_ginkaku", Name: "Ginkaku-ji (Silver Pavilion)", Address: "Kyoto Sakyo", Lat: 35.0270, Lng: 135.7983, Categories: []string{"temple"}, Tags: []string{"landmark", "garden", "kyoto"}, Score: 0.91},
}

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/search/places", searchPlaces)
	v1.GET("/search/suggestions", getSuggestions)
}

func searchPlaces(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	tags := strings.TrimSpace(c.Query("tags"))

	if q == "" && tags == "" {
		response.Error(c, http.StatusBadRequest, perrors.CodeBadRequest, "q or tags query param required", nil)
		return
	}

	results := make([]searchResult, 0)
	queryLower := strings.ToLower(q)
	tagList := []string{}
	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			tagList = append(tagList, strings.ToLower(strings.TrimSpace(t)))
		}
	}

	for _, place := range mockPlaces {
		matched := false

		// Full-text match on name/address
		if q != "" {
			nameLower := strings.ToLower(place.Name)
			addrLower := strings.ToLower(place.Address)
			if strings.Contains(nameLower, queryLower) || strings.Contains(addrLower, queryLower) {
				matched = true
			}
		}

		// Tag-based match
		if len(tagList) > 0 {
			for _, tag := range tagList {
				for _, pt := range place.Tags {
					if pt == tag {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
		}

		if matched {
			results = append(results, place)
		}
	}

	// Empty results → empty array + suggestion hint, never 404
	hint := ""
	if len(results) == 0 {
		hint = "No results found. Try broader search terms or popular tags like 'landmark', 'food', 'nature'."
	}

	response.JSON(c, http.StatusOK, gin.H{
		"results":        results,
		"total":          len(results),
		"suggestionHint": hint,
	})
}

func getSuggestions(c *gin.Context) {
	// Recent / favorite / similar place suggestions (mock: return top popular places)
	suggestions := make([]searchResult, 0, 3)
	for i, place := range mockPlaces {
		if i >= 3 {
			break
		}
		suggestions = append(suggestions, place)
	}

	response.JSON(c, http.StatusOK, gin.H{
		"suggestions": suggestions,
		"source":      "popular",
	})
}
