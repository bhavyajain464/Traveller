package services

import "indian-transit-backend/internal/models"

// JourneyPlannerService is the stable service-layer entry point for routing.
// It delegates to a concrete adapter while keeping handlers and higher-level
// workflows isolated from implementation churn.
type JourneyPlannerService struct {
	adapter JourneyPlannerAdapter
}

func NewJourneyPlannerService(adapter JourneyPlannerAdapter) *JourneyPlannerService {
	return &JourneyPlannerService{adapter: adapter}
}

func (s *JourneyPlannerService) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	return s.adapter.PlanJourney(req)
}

func (s *JourneyPlannerService) PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error) {
	return s.adapter.PlanJourneyBetweenStops(req)
}

func (s *JourneyPlannerService) Engine() string {
	return s.adapter.Engine()
}
