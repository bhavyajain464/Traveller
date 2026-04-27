package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"indian-transit-backend/internal/models"
)

type V3JourneyService struct {
	planner     JourneyPlanner
	placeSearch *PlaceSearchService
	routeSvc    *RouteService
	realtimeSvc stationboardRealtimeService
}

type stationboardRealtimeService interface {
	GetStopArrivals(stopID string, limit int) ([]StopArrival, error)
}

func NewV3JourneyService(planner JourneyPlanner, placeSearch *PlaceSearchService, routeSvc *RouteService, realtimeSvc stationboardRealtimeService) *V3JourneyService {
	return &V3JourneyService{
		planner:     planner,
		placeSearch: placeSearch,
		routeSvc:    routeSvc,
		realtimeSvc: realtimeSvc,
	}
}

func (s *V3JourneyService) SearchLocations(query string, limit int) (*models.V3LocationsResponse, error) {
	suggestions, err := s.placeSearch.Search(
		context.Background(),
		PlaceSearchRequest{
			Query: query,
			Limit: limit,
		},
	)
	if err != nil {
		return nil, err
	}

	locations := make([]models.V3Location, 0, len(suggestions))
	for index, suggestion := range suggestions {
		location := models.V3Location{
			ID:    suggestion.ID,
			Name:  suggestion.Title,
			Type:  mapLocationType(suggestion.FeatureType),
			Score: float64(len(suggestions) - index),
		}
		if suggestion.Latitude != nil && suggestion.Longitude != nil {
			location.Coordinates = &models.V3Coordinates{
				Lat: *suggestion.Latitude,
				Lon: *suggestion.Longitude,
			}
		}
		locations = append(locations, location)
	}

	return &models.V3LocationsResponse{
		Locations: locations,
		Count:     len(locations),
		Meta: models.V3LocationsMeta{
			Provider: s.placeSearch.ProviderName(),
			Query:    query,
		},
	}, nil
}

type V3JourneyQuery struct {
	From            string
	To              string
	Time            time.Time
	Mode            string
	Results         int
	Transportations []string
}

func (s *V3JourneyService) PlanJourney(query V3JourneyQuery) (*models.V3ConnectionResponse, error) {
	if strings.TrimSpace(query.From) == "" || strings.TrimSpace(query.To) == "" {
		return nil, fmt.Errorf("from and to are required")
	}

	mode := strings.ToLower(strings.TrimSpace(query.Mode))
	if mode == "" {
		mode = "departure"
	}
	if mode != "departure" {
		return nil, fmt.Errorf("only departure mode is supported right now")
	}

	fromPlace, err := s.placeSearch.Resolve(context.Background(), query.From)
	if err != nil {
		return nil, fmt.Errorf("resolve from location: %w", err)
	}
	toPlace, err := s.placeSearch.Resolve(context.Background(), query.To)
	if err != nil {
		return nil, fmt.Errorf("resolve to location: %w", err)
	}

	options, err := s.planOptions(query, fromPlace, toPlace)
	if err != nil {
		return nil, err
	}

	options = filterJourneyOptionsByTransportations(options, query.Transportations)
	if len(options) == 0 {
		return nil, fmt.Errorf("no journey options found")
	}

	if query.Results <= 0 {
		query.Results = 5
	}
	if len(options) > query.Results {
		options = options[:query.Results]
	}

	connections := make([]models.V3Connection, 0, len(options))
	for index, option := range options {
		connection, mapErr := s.mapConnection(index, option)
		if mapErr != nil {
			return nil, mapErr
		}
		connections = append(connections, connection)
	}

	return &models.V3ConnectionResponse{
		Connections: connections,
		Count:       len(connections),
		Meta: models.V3JourneyMeta{
			Engine:              s.planner.Engine(),
			ServiceDate:         inferServiceDate(options),
			RequestedDate:       query.Time.Format("2006-01-02"),
			FallbackServiceDate: inferServiceDate(options) != "" && inferServiceDate(options) != query.Time.Format("2006-01-02"),
			Request: models.V3JourneyRequestEcho{
				From:            query.From,
				To:              query.To,
				Time:            query.Time.Format(time.RFC3339),
				Mode:            mode,
				Results:         query.Results,
				Transportations: normalizeTransportationFilters(query.Transportations),
			},
		},
	}, nil
}

func (s *V3JourneyService) Stationboard(stationID string, limit int, at time.Time) (*models.V3StationboardResponse, error) {
	stationID = strings.TrimSpace(stationID)
	if stationID == "" {
		return nil, fmt.Errorf("station is required")
	}
	if s.realtimeSvc == nil {
		return nil, fmt.Errorf("stationboard service is not available")
	}
	if limit <= 0 {
		limit = 10
	}

	station, err := s.placeSearch.Resolve(context.Background(), stationID)
	if err != nil {
		return nil, fmt.Errorf("resolve station: %w", err)
	}

	arrivals, err := s.realtimeSvc.GetStopArrivals(stationID, limit)
	if err != nil {
		return nil, err
	}

	entries := make([]models.V3StationboardEntry, 0, len(arrivals))
	for _, arrival := range arrivals {
		entry := models.V3StationboardEntry{
			TripID: arrival.TripID,
			Stop: &models.V3StationStopInfo{
				Departure: arrival.ScheduledDeparture.Format(time.RFC3339),
				Arrival:   arrival.ScheduledArrival.Format(time.RFC3339),
			},
			Journey: models.V3JourneySection{
				ID:       arrival.RouteID,
				Name:     firstNonEmpty(arrival.RouteShortName, arrival.RouteLongName, arrival.RouteID),
				Operator: s.lookupOperator(arrival.RouteID),
				Category: s.lookupRouteMode(arrival.RouteID),
				From:     models.V3StationRef{ID: station.ID, Name: station.Title},
				To:       models.V3StationRef{Name: arrival.Headsign},
				Departure: arrival.ScheduledDeparture.Format(time.RFC3339),
				Arrival:   arrival.ScheduledArrival.Format(time.RFC3339),
			},
		}
		if arrival.HasRealTime {
			delay := int(arrival.RealTimeDeparture.Sub(arrival.ScheduledDeparture).Seconds())
			if arrival.RealTimeDeparture.IsZero() {
				delay = int(arrival.RealTimeArrival.Sub(arrival.ScheduledArrival).Seconds())
			}
			entry.Realtime = &models.V3RealtimeSection{
				Delay:     delay,
				Cancelled: false,
			}
			if !arrival.RealTimeDeparture.IsZero() {
				entry.Departure = arrival.RealTimeDeparture.Format(time.RFC3339)
			}
			if !arrival.RealTimeArrival.IsZero() {
				entry.Arrival = arrival.RealTimeArrival.Format(time.RFC3339)
			}
		}
		if entry.Departure == "" {
			entry.Departure = arrival.ScheduledDeparture.Format(time.RFC3339)
		}
		if entry.Arrival == "" {
			entry.Arrival = arrival.ScheduledArrival.Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}

	return &models.V3StationboardResponse{
		Station: models.V3StationRef{
			ID:   station.ID,
			Name: station.Title,
		},
		Entries: entries,
		Count:   len(entries),
		Meta: models.V3StationboardMeta{
			Limit: limit,
			Time:  at.Format(time.RFC3339),
		},
	}, nil
}

func (s *V3JourneyService) planOptions(query V3JourneyQuery, fromPlace, toPlace *models.PlaceSearchResult) ([]models.JourneyOption, error) {
	if shouldUseExactStopRouting(fromPlace) && shouldUseExactStopRouting(toPlace) {
		return s.planner.PlanJourneyBetweenStops(StopJourneyRequest{
			FromStopID:    fromPlace.ID,
			ToStopID:      toPlace.ID,
			DepartureTime: &query.Time,
			Date:          datePtr(query.Time),
		})
	}

	return s.planner.PlanJourney(models.JourneyRequest{
		FromLat:       fromPlace.Latitude,
		FromLon:       fromPlace.Longitude,
		ToLat:         toPlace.Latitude,
		ToLon:         toPlace.Longitude,
		DepartureTime: &query.Time,
		Date:          datePtr(query.Time),
	})
}

func (s *V3JourneyService) mapConnection(index int, option models.JourneyOption) (models.V3Connection, error) {
	if len(option.Legs) == 0 {
		return models.V3Connection{}, fmt.Errorf("journey option has no legs")
	}

	firstLeg := option.Legs[0]
	lastLeg := option.Legs[len(option.Legs)-1]

	sections := make([]models.V3ConnectionSection, 0, len(option.Legs))
	for _, leg := range option.Legs {
		if leg.Mode == "walking" || leg.Mode == "transfer" {
			sections = append(sections, models.V3ConnectionSection{
				Walk: &models.V3WalkSection{
					From:     models.V3StationRef{ID: leg.FromStopID, Name: leg.FromStopName},
					To:       models.V3StationRef{ID: leg.ToStopID, Name: leg.ToStopName},
					Departure: leg.DepartureTime.Format(time.RFC3339),
					Arrival:   leg.ArrivalTime.Format(time.RFC3339),
					Duration: formatDurationMinutes(leg.Duration),
				},
			})
			continue
		}

		operator := s.lookupOperator(leg.RouteID)

		sections = append(sections, models.V3ConnectionSection{
			Journey: &models.V3JourneySection{
				ID:        leg.RouteID,
				Name:      firstNonEmpty(leg.RouteName, leg.RouteID, leg.Mode),
				Operator:  operator,
				Category:  leg.Mode,
				From:      models.V3StationRef{ID: leg.FromStopID, Name: leg.FromStopName},
				To:        models.V3StationRef{ID: leg.ToStopID, Name: leg.ToStopName},
				Departure: leg.DepartureTime.Format(time.RFC3339),
				Arrival:   leg.ArrivalTime.Format(time.RFC3339),
			},
			Realtime: &models.V3RealtimeSection{
				Delay:     0,
				Cancelled: false,
			},
		})
	}

	return models.V3Connection{
		ID:        buildConnectionID(index, option),
		Duration:  formatDurationMinutes(option.Duration),
		Transfers: option.Transfers,
		From: models.V3ConnectionEndpoint{
			Station:   models.V3StationRef{ID: firstLeg.FromStopID, Name: firstLeg.FromStopName},
			Departure: firstLeg.DepartureTime.Format(time.RFC3339),
		},
		To: models.V3ConnectionEndpoint{
			Station: models.V3StationRef{ID: lastLeg.ToStopID, Name: lastLeg.ToStopName},
			Arrival: lastLeg.ArrivalTime.Format(time.RFC3339),
		},
		Sections: sections,
	}, nil
}

func filterJourneyOptionsByTransportations(options []models.JourneyOption, requested []string) []models.JourneyOption {
	if len(requested) == 0 {
		return options
	}

	normalizedRequested := normalizeTransportationFilters(requested)
	if len(normalizedRequested) == 0 {
		return options
	}

	allowed := make(map[string]struct{}, len(normalizedRequested))
	for _, mode := range normalizedRequested {
		allowed[mode] = struct{}{}
	}

	filtered := make([]models.JourneyOption, 0, len(options))
	for _, option := range options {
		ok := true
		for _, leg := range option.Legs {
			mode := strings.ToLower(strings.TrimSpace(leg.Mode))
			if mode == "" || mode == "walking" || mode == "transfer" {
				continue
			}
			if _, exists := allowed[mode]; !exists {
				ok = false
				break
			}
		}
		if ok {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func mapLocationType(featureType string) string {
	switch strings.ToLower(strings.TrimSpace(featureType)) {
	case "transit_stop", "transit_station", "train_station", "subway_station", "bus_station":
		return "station"
	default:
		return "location"
	}
}

func (s *V3JourneyService) lookupOperator(routeID string) string {
	if s.routeSvc == nil || strings.TrimSpace(routeID) == "" {
		return ""
	}

	route, err := s.routeSvc.GetByID(routeID)
	if err != nil || route == nil {
		return ""
	}

	return route.AgencyID
}

func (s *V3JourneyService) lookupRouteMode(routeID string) string {
	if s.routeSvc == nil || strings.TrimSpace(routeID) == "" {
		return ""
	}

	route, err := s.routeSvc.GetByID(routeID)
	if err != nil || route == nil {
		return ""
	}

	return routeTypeToMode(route.Type)
}

func buildConnectionID(index int, option models.JourneyOption) string {
	return fmt.Sprintf(
		"conn-%d-%d-%d",
		index,
		option.DepartureTime.Unix(),
		option.ArrivalTime.Unix(),
	)
}

func formatDurationMinutes(minutes int) string {
	if minutes < 0 {
		minutes = 0
	}
	duration := time.Duration(minutes) * time.Minute
	hours := int(duration / time.Hour)
	duration -= time.Duration(hours) * time.Hour
	mins := int(duration / time.Minute)
	return fmt.Sprintf("%02d:%02d:00", hours, mins)
}

func datePtr(value time.Time) *time.Time {
	date := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	return &date
}

func inferServiceDate(options []models.JourneyOption) string {
	for _, option := range options {
		for _, leg := range option.Legs {
			if leg.DepartureTime.IsZero() {
				continue
			}
			return leg.DepartureTime.Format("2006-01-02")
		}
		if !option.DepartureTime.IsZero() {
			return option.DepartureTime.Format("2006-01-02")
		}
	}

	return ""
}

func normalizeTransportationFilters(requested []string) []string {
	if len(requested) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(requested))
	normalized := make([]string, 0, len(requested))
	for _, raw := range requested {
		mode := normalizeTransportationFilter(raw)
		if mode == "" {
			continue
		}
		if _, exists := seen[mode]; exists {
			continue
		}
		seen[mode] = struct{}{}
		normalized = append(normalized, mode)
	}

	return normalized
}

func normalizeTransportationFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return ""
	case "train", "subway":
		return "metro"
	case "walk", "walking", "foot":
		return ""
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func shouldUseExactStopRouting(place *models.PlaceSearchResult) bool {
	if place == nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(place.FeatureType)) {
	case "transit_station", "train_station", "subway_station", "light_rail_station":
		return true
	case "bus_station":
		return !isBusStopID(place.ID)
	case "transit_stop":
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(place.ID)), "metro:") {
			return true
		}
		return false
	default:
		return false
	}
}

func isBusStopID(id string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(id)), "bus:")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
