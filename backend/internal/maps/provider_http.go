package maps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultProviderTimeout = 8 * time.Second

func loadProviderChainFromEnv() []MapProvider {
	googleKey := strings.TrimSpace(os.Getenv("GOOGLE_MAPS_API_KEY"))
	mapboxKey := strings.TrimSpace(os.Getenv("MAPBOX_API_KEY"))
	primary := strings.ToLower(strings.TrimSpace(os.Getenv("MAP_PRIMARY_PROVIDER")))
	if primary == "" {
		primary = "google"
	}

	googleProvider := MapProvider(&googleMapsProvider{
		apiKey:     googleKey,
		baseURL:    strings.TrimSpace(os.Getenv("GOOGLE_MAPS_BASE_URL")),
		httpClient: &http.Client{Timeout: defaultProviderTimeout},
	})
	mapboxProvider := MapProvider(&mapboxProvider{
		apiKey:     mapboxKey,
		baseURL:    strings.TrimSpace(os.Getenv("MAPBOX_BASE_URL")),
		httpClient: &http.Client{Timeout: defaultProviderTimeout},
	})

	available := make([]MapProvider, 0, 2)
	switch primary {
	case "mapbox":
		if mapboxKey != "" {
			available = append(available, mapboxProvider)
		}
		if googleKey != "" {
			available = append(available, googleProvider)
		}
	default:
		if googleKey != "" {
			available = append(available, googleProvider)
		}
		if mapboxKey != "" {
			available = append(available, mapboxProvider)
		}
	}
	return available
}

type googleMapsProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p *googleMapsProvider) Name() string { return "google_maps" }

func (p *googleMapsProvider) SearchPlaces(ctx context.Context, req PlaceSearchRequest) ([]NormalizedPlace, error) {
	base := p.defaultBaseURL()
	u, err := url.Parse(base + "/place/textsearch/json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", req.Query)
	q.Set("key", p.apiKey)
	if req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Lat != 0 || req.Lng != 0 {
		q.Set("location", fmt.Sprintf("%.6f,%.6f", req.Lat, req.Lng))
	}
	u.RawQuery = q.Encode()

	body, err := p.doRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			PlaceID          string   `json:"place_id"`
			Name             string   `json:"name"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if err := googleStatusToError(p.Name(), payload.Status, payload.ErrorMessage); err != nil {
		return nil, err
	}

	items := make([]NormalizedPlace, 0, len(payload.Results))
	for _, raw := range payload.Results {
		items = append(items, NormalizedPlace{
			ProviderPlaceID: raw.PlaceID,
			Name:            raw.Name,
			Address:         raw.FormattedAddress,
			Lat:             raw.Geometry.Location.Lat,
			Lng:             raw.Geometry.Location.Lng,
			Categories:      raw.Types,
		})
		if req.Limit > 0 && len(items) >= req.Limit {
			break
		}
	}
	return items, nil
}

func (p *googleMapsProvider) GetPlaceDetail(ctx context.Context, providerPlaceID string) (*NormalizedPlace, error) {
	base := p.defaultBaseURL()
	u, err := url.Parse(base + "/place/details/json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("place_id", providerPlaceID)
	q.Set("fields", "place_id,name,formatted_address,geometry,types,opening_hours")
	q.Set("key", p.apiKey)
	u.RawQuery = q.Encode()

	body, err := p.doRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var payload struct {
		Status string `json:"status"`
		Result struct {
			PlaceID          string   `json:"place_id"`
			Name             string   `json:"name"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
			OpeningHours struct {
				WeekdayText []string `json:"weekday_text"`
			} `json:"opening_hours"`
		} `json:"result"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if err := googleStatusToError(p.Name(), payload.Status, payload.ErrorMessage); err != nil {
		return nil, err
	}

	item := &NormalizedPlace{
		ProviderPlaceID: payload.Result.PlaceID,
		Name:            payload.Result.Name,
		Address:         payload.Result.FormattedAddress,
		Lat:             payload.Result.Geometry.Location.Lat,
		Lng:             payload.Result.Geometry.Location.Lng,
		Categories:      payload.Result.Types,
	}
	if len(payload.Result.OpeningHours.WeekdayText) > 0 {
		opening := strings.Join(payload.Result.OpeningHours.WeekdayText, "; ")
		item.OpeningHours = &opening
	}
	return item, nil
}

func (p *googleMapsProvider) EstimateRoute(ctx context.Context, req RouteEstimateRequest) (*NormalizedRoute, error) {
	base := p.defaultBaseURL()
	u, err := url.Parse(base + "/directions/json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("origin", fmt.Sprintf("%.6f,%.6f", req.OriginLat, req.OriginLng))
	q.Set("destination", fmt.Sprintf("%.6f,%.6f", req.DestinationLat, req.DestinationLng))
	q.Set("mode", normalizeRouteMode(req.Mode))
	q.Set("key", p.apiKey)
	u.RawQuery = q.Encode()

	body, err := p.doRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var payload struct {
		Status string `json:"status"`
		Routes []struct {
			Legs []struct {
				Distance struct {
					Value int `json:"value"`
				} `json:"distance"`
				Duration struct {
					Value int `json:"value"`
				} `json:"duration"`
			} `json:"legs"`
		} `json:"routes"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if err := googleStatusToError(p.Name(), payload.Status, payload.ErrorMessage); err != nil {
		return nil, err
	}
	if len(payload.Routes) == 0 || len(payload.Routes[0].Legs) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: "empty directions route"}
	}

	leg := payload.Routes[0].Legs[0]
	route := &NormalizedRoute{
		Mode:            normalizeRouteMode(req.Mode),
		DistanceMeters:  leg.Distance.Value,
		DurationSeconds: leg.Duration.Value,
	}
	if route.Mode == "transit" || route.Mode == "taxi" || route.Mode == "drive" {
		cost := math.Round((float64(route.DistanceMeters) / 1000 * 220) + 120)
		currency := "JPY"
		route.EstimatedCostAmount = &cost
		route.EstimatedCostCurrency = &currency
	}
	return route, nil
}

func (p *googleMapsProvider) Geocode(ctx context.Context, address string) (*NormalizedPlace, error) {
	base := p.defaultBaseURL()
	u, err := url.Parse(base + "/geocode/json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("address", address)
	q.Set("key", p.apiKey)
	u.RawQuery = q.Encode()

	body, err := p.doRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			PlaceID          string   `json:"place_id"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if err := googleStatusToError(p.Name(), payload.Status, payload.ErrorMessage); err != nil {
		return nil, err
	}
	if len(payload.Results) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusNotFound, Message: "geocode result not found"}
	}

	r := payload.Results[0]
	return &NormalizedPlace{
		ProviderPlaceID: r.PlaceID,
		Name:            r.FormattedAddress,
		Address:         r.FormattedAddress,
		Lat:             r.Geometry.Location.Lat,
		Lng:             r.Geometry.Location.Lng,
		Categories:      r.Types,
	}, nil
}

func (p *googleMapsProvider) ReverseGeocode(ctx context.Context, lat, lng float64) (*NormalizedPlace, error) {
	base := p.defaultBaseURL()
	u, err := url.Parse(base + "/geocode/json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("latlng", fmt.Sprintf("%.6f,%.6f", lat, lng))
	q.Set("key", p.apiKey)
	u.RawQuery = q.Encode()

	body, err := p.doRequest(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			PlaceID          string   `json:"place_id"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
		} `json:"results"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if err := googleStatusToError(p.Name(), payload.Status, payload.ErrorMessage); err != nil {
		return nil, err
	}
	if len(payload.Results) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusNotFound, Message: "reverse geocode result not found"}
	}

	r := payload.Results[0]
	return &NormalizedPlace{
		ProviderPlaceID: r.PlaceID,
		Name:            r.FormattedAddress,
		Address:         r.FormattedAddress,
		Lat:             lat,
		Lng:             lng,
		Categories:      r.Types,
	}, nil
}

func (p *googleMapsProvider) defaultBaseURL() string {
	return strings.TrimRight(defaultIfEmpty(p.baseURL, "https://maps.googleapis.com/maps/api"), "/")
}

func (p *googleMapsProvider) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return nil, ProviderTimeoutError{Provider: p.Name()}
		}
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ProviderQuotaError{Provider: p.Name()}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: resp.StatusCode, Message: truncateBody(body)}
	}
	return body, nil
}

func googleStatusToError(provider, status, message string) error {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "", "OK", "ZERO_RESULTS":
		return nil
	case "OVER_QUERY_LIMIT":
		return ProviderQuotaError{Provider: provider}
	case "REQUEST_DENIED", "INVALID_REQUEST":
		return ProviderAPIError{Provider: provider, StatusCode: http.StatusBadRequest, Message: message}
	default:
		return ProviderAPIError{Provider: provider, StatusCode: http.StatusBadGateway, Message: defaultIfEmpty(message, status)}
	}
}

type mapboxProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p *mapboxProvider) Name() string { return "mapbox" }

func (p *mapboxProvider) SearchPlaces(ctx context.Context, req PlaceSearchRequest) ([]NormalizedPlace, error) {
	u, err := p.buildGeocodeURL(req.Query, req.Lat, req.Lng, req.Limit)
	if err != nil {
		return nil, err
	}

	body, err := p.doRequest(ctx, u)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Features []struct {
			ID        string    `json:"id"`
			Text      string    `json:"text"`
			PlaceName string    `json:"place_name"`
			Center    []float64 `json:"center"`
			PlaceType []string  `json:"place_type"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	items := make([]NormalizedPlace, 0, len(payload.Features))
	for _, feature := range payload.Features {
		lat, lng := featureToLatLng(feature.Center)
		items = append(items, NormalizedPlace{
			ProviderPlaceID: feature.ID,
			Name:            feature.Text,
			Address:         feature.PlaceName,
			Lat:             lat,
			Lng:             lng,
			Categories:      feature.PlaceType,
		})
		if req.Limit > 0 && len(items) >= req.Limit {
			break
		}
	}
	return items, nil
}

func (p *mapboxProvider) GetPlaceDetail(ctx context.Context, providerPlaceID string) (*NormalizedPlace, error) {
	items, err := p.SearchPlaces(ctx, PlaceSearchRequest{Query: providerPlaceID, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusNotFound, Message: "place not found"}
	}
	return &items[0], nil
}

func (p *mapboxProvider) EstimateRoute(ctx context.Context, req RouteEstimateRequest) (*NormalizedRoute, error) {
	profile := mapboxProfileForMode(req.Mode)
	base := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://api.mapbox.com"), "/")
	endpoint := fmt.Sprintf(
		"%s/directions/v5/mapbox/%s/%.6f,%.6f;%.6f,%.6f?overview=false&access_token=%s",
		base, profile, req.OriginLng, req.OriginLat, req.DestinationLng, req.DestinationLat, url.QueryEscape(p.apiKey),
	)

	body, err := p.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if len(payload.Routes) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: "empty route"}
	}

	mode := normalizeRouteMode(req.Mode)
	item := &NormalizedRoute{
		Mode:            mode,
		DistanceMeters:  int(math.Round(payload.Routes[0].Distance)),
		DurationSeconds: int(math.Round(payload.Routes[0].Duration)),
	}
	if mode == "transit" || mode == "taxi" || mode == "drive" {
		cost := math.Round((float64(item.DistanceMeters) / 1000 * 220) + 120)
		currency := "JPY"
		item.EstimatedCostAmount = &cost
		item.EstimatedCostCurrency = &currency
	}
	return item, nil
}

func (p *mapboxProvider) Geocode(ctx context.Context, address string) (*NormalizedPlace, error) {
	items, err := p.SearchPlaces(ctx, PlaceSearchRequest{Query: address, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusNotFound, Message: "geocode result not found"}
	}
	return &items[0], nil
}

func (p *mapboxProvider) ReverseGeocode(ctx context.Context, lat, lng float64) (*NormalizedPlace, error) {
	base := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://api.mapbox.com"), "/")
	endpoint := fmt.Sprintf(
		"%s/geocoding/v5/mapbox.places/%.6f,%.6f.json?limit=1&access_token=%s",
		base, lng, lat, url.QueryEscape(p.apiKey),
	)
	body, err := p.doRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Features []struct {
			ID        string    `json:"id"`
			Text      string    `json:"text"`
			PlaceName string    `json:"place_name"`
			Center    []float64 `json:"center"`
			PlaceType []string  `json:"place_type"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if len(payload.Features) == 0 {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusNotFound, Message: "reverse geocode result not found"}
	}

	feature := payload.Features[0]
	outLat, outLng := featureToLatLng(feature.Center)
	return &NormalizedPlace{
		ProviderPlaceID: feature.ID,
		Name:            feature.Text,
		Address:         feature.PlaceName,
		Lat:             outLat,
		Lng:             outLng,
		Categories:      feature.PlaceType,
	}, nil
}

func (p *mapboxProvider) buildGeocodeURL(query string, lat, lng float64, limit int) (string, error) {
	base := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://api.mapbox.com"), "/")
	u, err := url.Parse(base + "/geocoding/v5/mapbox.places/" + url.PathEscape(query) + ".json")
	if err != nil {
		return "", err
	}
	q := u.Query()
	if limit <= 0 {
		limit = 5
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("access_token", p.apiKey)
	if lat != 0 || lng != 0 {
		q.Set("proximity", fmt.Sprintf("%.6f,%.6f", lng, lat))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *mapboxProvider) doRequest(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return nil, ProviderTimeoutError{Provider: p.Name()}
		}
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ProviderQuotaError{Provider: p.Name()}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, ProviderAPIError{Provider: p.Name(), StatusCode: resp.StatusCode, Message: truncateBody(body)}
	}
	return body, nil
}

func normalizeRouteMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "walk", "walking":
		return "walk"
	case "drive", "driving":
		return "drive"
	case "taxi":
		return "taxi"
	default:
		return "transit"
	}
}

func mapboxProfileForMode(mode string) string {
	switch normalizeRouteMode(mode) {
	case "walk":
		return "walking"
	default:
		return "driving"
	}
}

func featureToLatLng(center []float64) (float64, float64) {
	if len(center) < 2 {
		return 0, 0
	}
	return center[1], center[0]
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncateBody(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return "provider error"
	}
	if len(raw) > 240 {
		return raw[:240]
	}
	return raw
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	type timeout interface {
		Timeout() bool
	}
	var te timeout
	return errors.As(err, &te) && te.Timeout()
}
