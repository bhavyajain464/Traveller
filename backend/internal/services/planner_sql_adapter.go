package services

import (
	"fmt"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

// SQLJourneyPlannerAdapter keeps the existing SQL/GTFS routing implementation
// behind the new adapter boundary.
type SQLJourneyPlannerAdapter struct {
	planner     *RoutePlanner
	stopService *StopService
}

func NewSQLJourneyPlannerAdapter(db *database.DB, stopService *StopService, routeService *RouteService, fareService *FareService) *SQLJourneyPlannerAdapter {
	return &SQLJourneyPlannerAdapter{
		planner:     NewRoutePlanner(db, stopService, routeService, fareService),
		stopService: stopService,
	}
}

func (a *SQLJourneyPlannerAdapter) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	return a.planner.PlanJourney(req)
}

func (a *SQLJourneyPlannerAdapter) PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error) {
	fromStop, err := a.stopService.GetByID(req.FromStopID)
	if err != nil {
		return nil, fmt.Errorf("get from stop: %w", err)
	}

	toStop, err := a.stopService.GetByID(req.ToStopID)
	if err != nil {
		return nil, fmt.Errorf("get to stop: %w", err)
	}

	return a.planner.PlanJourney(models.JourneyRequest{
		FromLat:       fromStop.Latitude,
		FromLon:       fromStop.Longitude,
		ToLat:         toStop.Latitude,
		ToLon:         toStop.Longitude,
		DepartureTime: req.DepartureTime,
		ArrivalTime:   req.ArrivalTime,
		Date:          req.Date,
	})
}

func (a *SQLJourneyPlannerAdapter) Engine() string {
	return a.planner.Engine()
}
