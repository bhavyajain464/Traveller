package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"indian-transit-backend/internal/models"
)

const (
	PlaceSearchProviderStopLocal    = "stop_local"
	PlaceSearchProviderGooglePlaces = "google_places"
)

type PlaceSearchRequest struct {
	Query string
	Limit int
	Lat   *float64
	Lon   *float64
}

type PlaceSearchProvider interface {
	Search(ctx context.Context, req PlaceSearchRequest) ([]models.PlaceSearchSuggestion, error)
	Resolve(ctx context.Context, id string) (*models.PlaceSearchResult, error)
	Name() string
}

type PlaceSearchService struct {
	provider PlaceSearchProvider
}

func NewPlaceSearchService(provider PlaceSearchProvider) *PlaceSearchService {
	return &PlaceSearchService{provider: provider}
}

func (s *PlaceSearchService) Search(ctx context.Context, req PlaceSearchRequest) ([]models.PlaceSearchSuggestion, error) {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if req.Limit <= 0 || req.Limit > 10 {
		req.Limit = 6
	}

	return s.provider.Search(ctx, req)
}

func (s *PlaceSearchService) Resolve(ctx context.Context, id string) (*models.PlaceSearchResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	return s.provider.Resolve(ctx, id)
}

func (s *PlaceSearchService) ProviderName() string {
	if s.provider == nil {
		return ""
	}
	return s.provider.Name()
}

type StopPlaceSearchProvider struct {
	stopService *StopService
}

func NewStopPlaceSearchProvider(stopService *StopService) *StopPlaceSearchProvider {
	return &StopPlaceSearchProvider{stopService: stopService}
}

func (p *StopPlaceSearchProvider) Name() string {
	return PlaceSearchProviderStopLocal
}

func (p *StopPlaceSearchProvider) Search(ctx context.Context, req PlaceSearchRequest) ([]models.PlaceSearchSuggestion, error) {
	stops, err := p.stopService.Search(req.Query, req.Limit)
	if err != nil {
		return nil, err
	}

	suggestions := make([]models.PlaceSearchSuggestion, 0, len(stops))
	for _, stop := range stops {
		lat := stop.Latitude
		lon := stop.Longitude
		subtitle := stop.Code
		if subtitle == "" {
			subtitle = fmt.Sprintf("%.5f, %.5f", stop.Latitude, stop.Longitude)
		}
		suggestions = append(suggestions, models.PlaceSearchSuggestion{
			ID:          stop.ID,
			Title:       stop.Name,
			Subtitle:    subtitle,
			Provider:    p.Name(),
			FeatureType: "transit_stop",
			Latitude:    &lat,
			Longitude:   &lon,
		})
	}

	return suggestions, nil
}

func (p *StopPlaceSearchProvider) Resolve(ctx context.Context, id string) (*models.PlaceSearchResult, error) {
	stop, err := p.stopService.GetByID(id)
	if err != nil {
		return nil, err
	}

	subtitle := stop.Code
	if subtitle == "" {
		subtitle = fmt.Sprintf("%.5f, %.5f", stop.Latitude, stop.Longitude)
	}

	return &models.PlaceSearchResult{
		ID:          stop.ID,
		Title:       stop.Name,
		Subtitle:    subtitle,
		Provider:    p.Name(),
		FeatureType: "transit_stop",
		Latitude:    stop.Latitude,
		Longitude:   stop.Longitude,
	}, nil
}

type GooglePlacesProvider struct {
	apiKey     string
	httpClient *http.Client
	regionCode string
}

func NewGooglePlacesProvider(apiKey, regionCode string) *GooglePlacesProvider {
	return &GooglePlacesProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		regionCode: strings.ToLower(strings.TrimSpace(regionCode)),
	}
}

func (p *GooglePlacesProvider) Name() string {
	return PlaceSearchProviderGooglePlaces
}

func (p *GooglePlacesProvider) Search(ctx context.Context, req PlaceSearchRequest) ([]models.PlaceSearchSuggestion, error) {
	body := map[string]any{
		"input": req.Query,
	}
	if p.regionCode != "" {
		body["regionCode"] = p.regionCode
	}
	if req.Lat != nil && req.Lon != nil {
		body["locationBias"] = map[string]any{
			"circle": map[string]any{
				"center": map[string]any{
					"latitude":  *req.Lat,
					"longitude": *req.Lon,
				},
				"radius": 5000.0,
			},
		}
	}

	payload, err := p.doJSONRequest(ctx, http.MethodPost, "https://places.googleapis.com/v1/places:autocomplete", body, map[string]string{
		"X-Goog-Api-Key":   p.apiKey,
		"X-Goog-FieldMask": "suggestions.placePrediction.placeId,suggestions.placePrediction.text.text,suggestions.placePrediction.structuredFormat.mainText.text,suggestions.placePrediction.structuredFormat.secondaryText.text,suggestions.placePrediction.types",
		"Content-Type":     "application/json",
	})
	if err != nil {
		return nil, err
	}

	var response struct {
		Suggestions []struct {
			PlacePrediction *struct {
				PlaceID string `json:"placeId"`
				Text    struct {
					Text string `json:"text"`
				} `json:"text"`
				StructuredFormat struct {
					MainText struct {
						Text string `json:"text"`
					} `json:"mainText"`
					SecondaryText struct {
						Text string `json:"text"`
					} `json:"secondaryText"`
				} `json:"structuredFormat"`
				Types []string `json:"types"`
			} `json:"placePrediction"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("decode google autocomplete: %w", err)
	}

	suggestions := make([]models.PlaceSearchSuggestion, 0, min(req.Limit, len(response.Suggestions)))
	for _, suggestion := range response.Suggestions {
		if suggestion.PlacePrediction == nil || suggestion.PlacePrediction.PlaceID == "" {
			continue
		}
		title := suggestion.PlacePrediction.StructuredFormat.MainText.Text
		if title == "" {
			title = suggestion.PlacePrediction.Text.Text
		}
		suggestions = append(suggestions, models.PlaceSearchSuggestion{
			ID:          suggestion.PlacePrediction.PlaceID,
			Title:       title,
			Subtitle:    suggestion.PlacePrediction.StructuredFormat.SecondaryText.Text,
			Provider:    p.Name(),
			FeatureType: firstString(suggestion.PlacePrediction.Types),
		})
		if len(suggestions) >= req.Limit {
			break
		}
	}

	return suggestions, nil
}

func (p *GooglePlacesProvider) Resolve(ctx context.Context, id string) (*models.PlaceSearchResult, error) {
	endpoint := "https://places.googleapis.com/v1/places/" + url.PathEscape(id)
	payload, err := p.doJSONRequest(ctx, http.MethodGet, endpoint, nil, map[string]string{
		"X-Goog-Api-Key":   p.apiKey,
		"X-Goog-FieldMask": "id,displayName,formattedAddress,location,primaryType",
	})
	if err != nil {
		return nil, err
	}

	var place struct {
		ID               string `json:"id"`
		FormattedAddress string `json:"formattedAddress"`
		DisplayName      struct {
			Text string `json:"text"`
		} `json:"displayName"`
		PrimaryType string `json:"primaryType"`
		Location    struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
	}
	if err := json.Unmarshal(payload, &place); err != nil {
		return nil, fmt.Errorf("decode google place details: %w", err)
	}
	if place.ID == "" {
		return nil, fmt.Errorf("place not found")
	}

	return &models.PlaceSearchResult{
		ID:          place.ID,
		Title:       place.DisplayName.Text,
		Subtitle:    place.FormattedAddress,
		Provider:    p.Name(),
		FeatureType: place.PrimaryType,
		Latitude:    place.Location.Latitude,
		Longitude:   place.Location.Longitude,
	}, nil
}

func (p *GooglePlacesProvider) doJSONRequest(ctx context.Context, method, endpoint string, body any, headers map[string]string) ([]byte, error) {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal google places request: %w", err)
		}
		requestBody = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
	if err != nil {
		return nil, fmt.Errorf("build google places request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google places request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google places response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google places request failed: %s", strings.TrimSpace(string(payload)))
	}

	return payload, nil
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
