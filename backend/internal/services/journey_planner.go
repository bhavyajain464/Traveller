package services

import (
	"time"

	"indian-transit-backend/internal/models"
)

// JourneyPlanner is the routing boundary for future SQL, in-memory, Neo4j, or
// external planner adapters.
type JourneyPlanner interface {
	PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error)
	PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error)
	Engine() string
}

// JourneyPlannerAdapter is the concrete adapter contract behind the journey
// planner service.
type JourneyPlannerAdapter interface {
	PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error)
	PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error)
	Engine() string
}

type StopJourneyRequest struct {
	FromStopID    string
	ToStopID      string
	DepartureTime *time.Time
	ArrivalTime   *time.Time
	Date          *time.Time
}

// PlannerSnapshot is the in-process timetable state owned by snapshot-backed
// planner adapters.
type PlannerSnapshot struct {
	Version              string
	LoadedAt             time.Time
	Stops                []models.Stop
	Routes               []models.Route
	Trips                []models.Trip
	Footpaths            []plannerFootpath
	StopsByID            map[string]models.Stop
	RoutesByID           map[string]models.Route
	TripsByID            map[string]models.Trip
	CalendarsByServiceID map[string]models.Calendar
	StopTimesByTripID    map[string][]plannerStopTime
	StopVisitsByStopID   map[string][]plannerStopVisit
	RouteIDsByStopID     map[string][]string
	TripIDsByRouteID     map[string][]string
	FootpathsFromStopID  map[string][]plannerFootpath
	StopModeTypes        map[string]map[int]struct{}
	MinServiceDate       time.Time
	MaxServiceDate       time.Time
}

func (s *PlannerSnapshot) StopCount() int {
	if s == nil {
		return 0
	}
	return len(s.Stops)
}

func (s *PlannerSnapshot) RouteCount() int {
	if s == nil {
		return 0
	}
	return len(s.Routes)
}
