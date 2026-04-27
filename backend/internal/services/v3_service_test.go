package services

import (
	"context"
	"testing"
	"time"

	"indian-transit-backend/internal/models"
)

type testJourneyPlanner struct {
	options        []models.JourneyOption
	err            error
	engine         string
	coordCalls     int
	stopCalls      int
	lastJourneyReq models.JourneyRequest
	lastStopReq    StopJourneyRequest
}

func (p *testJourneyPlanner) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	p.coordCalls++
	p.lastJourneyReq = req
	return p.options, p.err
}

func (p *testJourneyPlanner) PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error) {
	p.stopCalls++
	p.lastStopReq = req
	return p.options, p.err
}

func (p *testJourneyPlanner) Engine() string {
	if p.engine != "" {
		return p.engine
	}
	return "test_planner"
}

type testPlaceProvider struct {
	suggestions []models.PlaceSearchSuggestion
	resolved    map[string]*models.PlaceSearchResult
}

type testRealtimeService struct {
	arrivals []StopArrival
	err      error
	lastStop string
	lastLimit int
}

func (p testPlaceProvider) Search(ctx context.Context, req PlaceSearchRequest) ([]models.PlaceSearchSuggestion, error) {
	return p.suggestions, nil
}

func (p testPlaceProvider) Resolve(ctx context.Context, id string) (*models.PlaceSearchResult, error) {
	return p.resolved[id], nil
}

func (p testPlaceProvider) Name() string {
	return "test_places"
}

func (s *testRealtimeService) GetStopArrivals(stopID string, limit int) ([]StopArrival, error) {
	s.lastStop = stopID
	s.lastLimit = limit
	return s.arrivals, s.err
}

func TestSearchLocationsIncludesMetaAndCoordinates(t *testing.T) {
	lat := 12.9716
	lon := 77.5946
	service := NewV3JourneyService(
		&testJourneyPlanner{},
		NewPlaceSearchService(testPlaceProvider{
			suggestions: []models.PlaceSearchSuggestion{
				{
					ID:          "stop-1",
					Title:       "Majestic",
					FeatureType: "transit_stop",
					Latitude:    &lat,
					Longitude:   &lon,
				},
			},
		}),
		nil,
		nil,
	)

	response, err := service.SearchLocations("Majestic", 5)
	if err != nil {
		t.Fatalf("SearchLocations returned error: %v", err)
	}

	if response.Meta.Provider != "test_places" {
		t.Fatalf("expected provider metadata, got %q", response.Meta.Provider)
	}
	if response.Meta.Query != "Majestic" {
		t.Fatalf("expected query metadata, got %q", response.Meta.Query)
	}
	if len(response.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(response.Locations))
	}
	if response.Locations[0].Type != "station" {
		t.Fatalf("expected station type, got %q", response.Locations[0].Type)
	}
	if response.Locations[0].Coordinates == nil {
		t.Fatal("expected coordinates to be populated")
	}
}

func TestPlanJourneyAddsMetaAndSectionTimings(t *testing.T) {
	queryTime := time.Date(2026, 4, 26, 8, 30, 0, 0, time.FixedZone("IST", 5*3600+1800))
	serviceDate := time.Date(2026, 4, 25, 8, 35, 0, 0, queryTime.Location())
	planner := &testJourneyPlanner{
		engine: "in_memory_snapshot(sql_gtfs)",
		options: []models.JourneyOption{
			{
				Duration:      45,
				Transfers:     1,
				WalkingTime:   8,
				DepartureTime: serviceDate,
				ArrivalTime:   serviceDate.Add(45 * time.Minute),
				Legs: []models.JourneyLeg{
					{
						Mode:          "walking",
						FromStopName:  "Origin",
						ToStopID:      "stop-a",
						ToStopName:    "Stop A",
						DepartureTime: serviceDate,
						ArrivalTime:   serviceDate.Add(5 * time.Minute),
						Duration:      5,
					},
					{
						Mode:          "metro",
						RouteID:       "route-1",
						RouteName:     "Green Line",
						FromStopID:    "stop-a",
						FromStopName:  "Stop A",
						ToStopID:      "stop-b",
						ToStopName:    "Stop B",
						DepartureTime: serviceDate.Add(10 * time.Minute),
						ArrivalTime:   serviceDate.Add(35 * time.Minute),
						Duration:      25,
					},
				},
			},
		},
	}

	service := NewV3JourneyService(
		planner,
		NewPlaceSearchService(testPlaceProvider{
			resolved: map[string]*models.PlaceSearchResult{
				"from-stop": {ID: "from-stop", Title: "From", Latitude: 12.0, Longitude: 77.0},
				"to-stop":   {ID: "to-stop", Title: "To", Latitude: 12.5, Longitude: 77.5},
			},
		}),
		nil,
		nil,
	)

	response, err := service.PlanJourney(V3JourneyQuery{
		From:            "from-stop",
		To:              "to-stop",
		Time:            queryTime,
		Mode:            "departure",
		Results:         3,
		Transportations: []string{"train", "bus", "walk"},
	})
	if err != nil {
		t.Fatalf("PlanJourney returned error: %v", err)
	}

	if response.Meta.Engine != "in_memory_snapshot(sql_gtfs)" {
		t.Fatalf("expected engine metadata, got %q", response.Meta.Engine)
	}
	if response.Meta.Request.Mode != "departure" {
		t.Fatalf("expected request mode metadata, got %q", response.Meta.Request.Mode)
	}
	if len(response.Meta.Request.Transportations) != 2 || response.Meta.Request.Transportations[0] != "metro" || response.Meta.Request.Transportations[1] != "bus" {
		t.Fatalf("expected normalized transportation filters, got %#v", response.Meta.Request.Transportations)
	}
	if !response.Meta.FallbackServiceDate {
		t.Fatal("expected fallback service date metadata to be true")
	}
	if response.Meta.ServiceDate != "2026-04-25" {
		t.Fatalf("expected service date metadata, got %q", response.Meta.ServiceDate)
	}
	if len(response.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(response.Connections))
	}
	if response.Connections[0].Sections[0].Walk == nil {
		t.Fatal("expected first section to be walk")
	}
	if response.Connections[0].Sections[0].Walk.Departure == "" || response.Connections[0].Sections[0].Walk.Arrival == "" {
		t.Fatal("expected walk section timings to be populated")
	}
	if planner.coordCalls != 1 || planner.stopCalls != 0 {
		t.Fatalf("expected coordinate planner path for generic places, got coord=%d stop=%d", planner.coordCalls, planner.stopCalls)
	}
}

func TestPlanJourneyUsesStopToStopPlannerForTransitStops(t *testing.T) {
	queryTime := time.Date(2026, 4, 26, 8, 30, 0, 0, time.UTC)
	planner := &testJourneyPlanner{
		options: []models.JourneyOption{
			{
				Duration:      15,
				Transfers:     0,
				DepartureTime: queryTime,
				ArrivalTime:   queryTime.Add(15 * time.Minute),
				Legs: []models.JourneyLeg{
					{
						Mode:          "metro",
						RouteID:       "route-1",
						RouteName:     "Green Line",
						FromStopID:    "stop-a",
						FromStopName:  "Stop A",
						ToStopID:      "stop-b",
						ToStopName:    "Stop B",
						DepartureTime: queryTime,
						ArrivalTime:   queryTime.Add(15 * time.Minute),
						Duration:      15,
					},
				},
			},
		},
	}

	service := NewV3JourneyService(
		planner,
		NewPlaceSearchService(testPlaceProvider{
			resolved: map[string]*models.PlaceSearchResult{
				"metro:stop-a": {ID: "metro:stop-a", Title: "Stop A", FeatureType: "transit_stop", Latitude: 12.0, Longitude: 77.0},
				"metro:stop-b": {ID: "metro:stop-b", Title: "Stop B", FeatureType: "transit_stop", Latitude: 12.5, Longitude: 77.5},
			},
		}),
		nil,
		nil,
	)

	_, err := service.PlanJourney(V3JourneyQuery{
		From:    "metro:stop-a",
		To:      "metro:stop-b",
		Time:    queryTime,
		Mode:    "departure",
		Results: 3,
	})
	if err != nil {
		t.Fatalf("PlanJourney returned error: %v", err)
	}

	if planner.stopCalls != 1 || planner.coordCalls != 0 {
		t.Fatalf("expected stop-based planner path, got coord=%d stop=%d", planner.coordCalls, planner.stopCalls)
	}
	if planner.lastStopReq.FromStopID != "metro:stop-a" || planner.lastStopReq.ToStopID != "metro:stop-b" {
		t.Fatalf("unexpected stop planner request: %#v", planner.lastStopReq)
	}
}

func TestPlanJourneyUsesCoordinatePlannerForBusStyleTransitStops(t *testing.T) {
	queryTime := time.Date(2026, 4, 26, 8, 30, 0, 0, time.UTC)
	planner := &testJourneyPlanner{
		options: []models.JourneyOption{
			{
				Duration:      18,
				Transfers:     0,
				DepartureTime: queryTime,
				ArrivalTime:   queryTime.Add(18 * time.Minute),
				Legs: []models.JourneyLeg{
					{
						Mode:          "bus",
						RouteID:       "route-bus",
						RouteName:     "Bus 1",
						FromStopID:    "bus:1",
						FromStopName:  "Bus Stop 1",
						ToStopID:      "bus:2",
						ToStopName:    "Bus Stop 2",
						DepartureTime: queryTime,
						ArrivalTime:   queryTime.Add(18 * time.Minute),
						Duration:      18,
					},
				},
			},
		},
	}

	service := NewV3JourneyService(
		planner,
		NewPlaceSearchService(testPlaceProvider{
			resolved: map[string]*models.PlaceSearchResult{
				"bus:1": {ID: "bus:1", Title: "Bus Stop 1", FeatureType: "transit_stop", Latitude: 28.62675, Longitude: 77.30935},
				"bus:2": {ID: "bus:2", Title: "Bus Stop 2", FeatureType: "transit_stop", Latitude: 28.61891, Longitude: 77.07517},
			},
		}),
		nil,
		nil,
	)

	_, err := service.PlanJourney(V3JourneyQuery{
		From:    "bus:1",
		To:      "bus:2",
		Time:    queryTime,
		Mode:    "departure",
		Results: 3,
	})
	if err != nil {
		t.Fatalf("PlanJourney returned error: %v", err)
	}

	if planner.coordCalls != 1 || planner.stopCalls != 0 {
		t.Fatalf("expected coordinate planner path for bus-style stop selection, got coord=%d stop=%d", planner.coordCalls, planner.stopCalls)
	}
	if planner.lastJourneyReq.FromLat != 28.62675 || planner.lastJourneyReq.ToLon != 77.07517 {
		t.Fatalf("unexpected coordinate planner request: %#v", planner.lastJourneyReq)
	}
}

func TestStationboardMapsRealtimeArrivals(t *testing.T) {
	now := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)
	realtime := &testRealtimeService{
		arrivals: []StopArrival{
			{
				TripID:             "trip-1",
				RouteID:            "route-1",
				RouteShortName:     "Blue",
				RouteLongName:      "Blue Line",
				ScheduledArrival:   now.Add(2 * time.Minute),
				ScheduledDeparture: now.Add(3 * time.Minute),
				RealTimeArrival:    now.Add(4 * time.Minute),
				RealTimeDeparture:  now.Add(5 * time.Minute),
				Headsign:           "Dwarka",
				HasRealTime:        true,
			},
		},
	}

	service := NewV3JourneyService(
		&testJourneyPlanner{},
		NewPlaceSearchService(testPlaceProvider{
			resolved: map[string]*models.PlaceSearchResult{
				"metro:108": {ID: "metro:108", Title: "Janak Puri West", FeatureType: "transit_stop", Latitude: 28.629637, Longitude: 77.077866},
			},
		}),
		nil,
		realtime,
	)

	response, err := service.Stationboard("metro:108", 6, now)
	if err != nil {
		t.Fatalf("Stationboard returned error: %v", err)
	}

	if realtime.lastStop != "metro:108" || realtime.lastLimit != 6 {
		t.Fatalf("unexpected realtime request stop=%q limit=%d", realtime.lastStop, realtime.lastLimit)
	}
	if response.Station.ID != "metro:108" || response.Station.Name != "Janak Puri West" {
		t.Fatalf("unexpected station response: %#v", response.Station)
	}
	if len(response.Entries) != 1 {
		t.Fatalf("expected 1 stationboard entry, got %d", len(response.Entries))
	}
	if response.Entries[0].Journey.Name != "Blue" {
		t.Fatalf("unexpected journey name: %#v", response.Entries[0].Journey)
	}
	if response.Entries[0].Realtime == nil || response.Entries[0].Realtime.Delay != 120 {
		t.Fatalf("expected 120-second realtime delay, got %#v", response.Entries[0].Realtime)
	}
}
