package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/domain"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/repository"
)

type RouteBoardingService struct {
	repo                   *repository.RouteBoardingRepository
	segmentRepo            *repository.JourneySegmentRepository
	eventRepo              *repository.JourneyEventRepository
	fareTransactionRepo    *repository.FareTransactionRepository
	stopService            *StopService
	fareService            *FareService
	sessionService         *JourneySessionService
	vehicleLocationService *VehicleLocationService
}

func NewRouteBoardingService(repo *repository.RouteBoardingRepository, segmentRepo *repository.JourneySegmentRepository, eventRepo *repository.JourneyEventRepository, fareTransactionRepo *repository.FareTransactionRepository, stopService *StopService, fareService *FareService, sessionService *JourneySessionService, vehicleLocationService *VehicleLocationService) *RouteBoardingService {
	return &RouteBoardingService{
		repo:                   repo,
		segmentRepo:            segmentRepo,
		eventRepo:              eventRepo,
		fareTransactionRepo:    fareTransactionRepo,
		stopService:            stopService,
		fareService:            fareService,
		sessionService:         sessionService,
		vehicleLocationService: vehicleLocationService,
	}
}

// BoardRoute records when user boards a route (called when QR is validated on vehicle)
func (s *RouteBoardingService) BoardRoute(req models.BoardRouteRequest) (*models.RouteBoarding, error) {
	tx, err := s.sessionService.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin board transaction: %w", err)
	}
	defer tx.Rollback()

	boarding, err := s.BoardRouteWithExecutor(tx, req)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit board transaction: %w", err)
	}
	return boarding, nil
}

func (s *RouteBoardingService) BoardRouteWithExecutor(exec repository.DBTX, req models.BoardRouteRequest) (*models.RouteBoarding, error) {
	// Get session by QR code or session ID
	var session *models.JourneySession
	var err error

	if req.SessionID != "" {
		session, err = s.sessionService.repo.GetByIDWithExecutor(exec, req.SessionID)
	} else if req.QRCode != "" {
		session, err = s.sessionService.repo.GetByQRCodeWithExecutor(exec, req.QRCode)
	} else {
		return nil, fmt.Errorf("session_id or qr_code is required")
	}

	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	if !domain.IsActiveJourneyStatus(session.Status) {
		return nil, fmt.Errorf("journey session is not active")
	}

	// Find nearest stop if not provided
	stopID := req.BoardingStopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err != nil || len(stops) == 0 {
			return nil, fmt.Errorf("no nearby stop found")
		}
		stopID = &stops[0].ID
	}

	// Check if user is already on a route (has active boarding)
	activeBoarding, err := s.GetActiveBoardingWithExecutor(exec, session.ID)
	if err == nil && activeBoarding != nil {
		return nil, fmt.Errorf("user is already on route %s. Please alight first", activeBoarding.RouteID)
	}

	// Create boarding record
	boardingID := uuid.New().String()
	now := time.Now()
	boarding := &models.RouteBoarding{
		ID:             boardingID,
		SessionID:      session.ID,
		RouteID:        req.RouteID,
		BoardingStopID: *stopID,
		BoardingTime:   now,
		BoardingLat:    req.Latitude,
		BoardingLon:    req.Longitude,
		Distance:       0,
		Fare:           0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateWithExecutor(exec, boarding); err != nil {
		return nil, err
	}

	if err := s.createSegmentAndBoardEvent(exec, boarding, session.ID, boarding.BoardingTime); err != nil {
		return nil, err
	}

	return boarding, nil
}

// AutoDetectAndBoard automatically detects which vehicle user is on and records boarding
func (s *RouteBoardingService) AutoDetectAndBoard(sessionID string, userLat, userLon float64) (*models.RouteBoarding, *models.VehicleLocationMatch, error) {
	tx, err := s.sessionService.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin auto-board transaction: %w", err)
	}
	defer tx.Rollback()

	boarding, match, err := s.autoDetectAndBoardWithExecutor(tx, sessionID, userLat, userLon)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit auto-board transaction: %w", err)
	}
	return boarding, match, nil
}

func (s *RouteBoardingService) autoDetectAndBoardWithExecutor(exec repository.DBTX, sessionID string, userLat, userLon float64) (*models.RouteBoarding, *models.VehicleLocationMatch, error) {
	// Get session
	session, err := s.sessionService.repo.GetByIDWithExecutor(exec, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid session: %w", err)
	}

	if !domain.IsActiveJourneyStatus(session.Status) {
		return nil, nil, fmt.Errorf("journey session is not active")
	}

	// Check if user is already on a route
	activeBoarding, err := s.GetActiveBoardingWithExecutor(exec, session.ID)
	if err == nil && activeBoarding != nil {
		return nil, nil, fmt.Errorf("user is already on route %s. Please alight first", activeBoarding.RouteID)
	}

	// Detect which vehicle user is on
	if s.vehicleLocationService == nil {
		return nil, nil, fmt.Errorf("vehicle location service not available")
	}

	match, err := s.vehicleLocationService.DetectTransportMode(userLat, userLon)
	if err != nil {
		return nil, nil, fmt.Errorf("could not detect transport mode: %w", err)
	}

	// Find nearest stop
	stops, err := s.stopService.FindNearby(userLat, userLon, 500, 1)
	if err != nil || len(stops) == 0 {
		return nil, nil, fmt.Errorf("no nearby stop found")
	}
	stopID := stops[0].ID

	// Create boarding record with vehicle ID
	boardingID := uuid.New().String()
	now := time.Now()
	vehicleID := match.VehicleLocation.VehicleID
	boarding := &models.RouteBoarding{
		ID:             boardingID,
		SessionID:      session.ID,
		RouteID:        match.RouteID,
		VehicleID:      &vehicleID,
		BoardingStopID: stopID,
		BoardingTime:   now,
		BoardingLat:    userLat,
		BoardingLon:    userLon,
		Distance:       0,
		Fare:           0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateWithExecutor(exec, boarding); err != nil {
		return nil, nil, err
	}

	if err := s.createSegmentAndBoardEvent(exec, boarding, session.ID, boarding.BoardingTime); err != nil {
		return nil, nil, err
	}

	return boarding, match, nil
}

// AlightRoute records when user alights from a route
func (s *RouteBoardingService) AlightRoute(req models.AlightRouteRequest) (*models.RouteBoarding, error) {
	tx, err := s.sessionService.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin alight transaction: %w", err)
	}
	defer tx.Rollback()

	boarding, err := s.AlightRouteWithExecutor(tx, req)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit alight transaction: %w", err)
	}
	return boarding, nil
}

func (s *RouteBoardingService) AlightRouteWithExecutor(exec repository.DBTX, req models.AlightRouteRequest) (*models.RouteBoarding, error) {
	// Get boarding record
	boarding, err := s.GetBoardingByIDWithExecutor(exec, req.BoardingID)
	if err != nil {
		return nil, fmt.Errorf("boarding not found: %w", err)
	}

	if boarding.AlightingTime != nil {
		return nil, fmt.Errorf("user has already alighted from this route")
	}

	// Find nearest stop if not provided
	stopID := req.AlightingStopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err != nil || len(stops) == 0 {
			return nil, fmt.Errorf("no nearby stop found")
		}
		stopID = &stops[0].ID
	}

	// Calculate distance and fare for this route segment
	distance := s.fareService.CalculateDistance(boarding.BoardingStopID, *stopID)
	fmt.Printf("[AlightRoute] Boarding stop: %s, Alighting stop: %s, Distance: %.3f km\n",
		boarding.BoardingStopID, *stopID, distance)

	agencyID := s.fareService.GetAgencyIDFromRoute(boarding.RouteID)
	if agencyID == "" {
		agencyID = "DIMTS" // Default to Delhi bus if not found
	}
	fmt.Printf("[AlightRoute] Route: %s, Agency: %s\n", boarding.RouteID, agencyID)

	rules := s.fareService.GetFareRulesForAgency(agencyID)
	fmt.Printf("[AlightRoute] Fare rules: BaseFare=%.2f, FarePerKm=%.2f\n", rules.BaseFare, rules.FarePerKm)

	fare := s.fareService.CalculateRouteSegmentFare(boarding.RouteID, boarding.BoardingStopID, *stopID, distance, rules)
	fmt.Printf("[AlightRoute] Calculated fare: ₹%.2f\n", fare)

	// Update boarding record
	now := time.Now()
	alightingTime := now
	boarding.AlightingTime = &alightingTime
	boarding.AlightingStopID = stopID
	alightingLat := req.Latitude
	alightingLon := req.Longitude
	boarding.AlightingLat = &alightingLat
	boarding.AlightingLon = &alightingLon
	boarding.Distance = distance
	boarding.Fare = fare
	boarding.UpdatedAt = now

	if err := s.repo.UpdateWithExecutor(exec, boarding); err != nil {
		return nil, err
	}

	if err := s.completeSegmentAndCreateAlightEvent(exec, boarding, now); err != nil {
		return nil, err
	}
	if err := s.createFareTransaction(exec, boarding, agencyID, now); err != nil {
		return nil, err
	}

	return boarding, nil
}

// GetBoardingsForSession returns all route boardings for a journey session
func (s *RouteBoardingService) GetBoardingsForSession(sessionID string) ([]models.RouteBoarding, error) {
	return s.repo.ListBySessionID(sessionID)
}

// GetActiveBoarding returns the currently active boarding (user is on a route)
func (s *RouteBoardingService) GetActiveBoarding(sessionID string) (*models.RouteBoarding, error) {
	return s.repo.GetActiveBySessionID(sessionID)
}

func (s *RouteBoardingService) GetActiveBoardingWithExecutor(exec repository.DBTX, sessionID string) (*models.RouteBoarding, error) {
	return s.repo.GetActiveBySessionIDWithExecutor(exec, sessionID)
}

// CalculateFareFromBoardings calculates total fare from actual route boardings
func (s *RouteBoardingService) CalculateFareFromBoardings(sessionID string) (float64, float64, []string, error) {
	return s.CalculateFareFromBoardingsWithExecutor(s.repoDB(), sessionID)
}

func (s *RouteBoardingService) CalculateFareFromBoardingsWithExecutor(exec repository.DBTX, sessionID string) (float64, float64, []string, error) {
	boardings, err := s.repo.ListBySessionIDWithExecutor(exec, sessionID)
	if err != nil {
		return 0, 0, nil, err
	}

	totalDistance := 0.0
	totalFare := 0.0
	routesUsed := make([]string, 0)

	for _, boarding := range boardings {
		if boarding.AlightingTime != nil { // Only count completed segments
			totalDistance += boarding.Distance
			totalFare += boarding.Fare
			if boarding.RouteID != "" {
				routesUsed = append(routesUsed, boarding.RouteID)
			}
		}
	}

	return totalDistance, totalFare, routesUsed, nil
}

// GetBoardingByID gets a boarding record by ID
func (s *RouteBoardingService) GetBoardingByID(boardingID string) (*models.RouteBoarding, error) {
	return s.GetBoardingByIDWithExecutor(s.repoDB(), boardingID)
}

func (s *RouteBoardingService) GetBoardingByIDWithExecutor(exec repository.DBTX, boardingID string) (*models.RouteBoarding, error) {
	boarding, err := s.repo.GetByIDWithExecutor(exec, boardingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get boarding: %w", err)
	}
	return boarding, nil
}

func (s *RouteBoardingService) repoDB() repository.DBTX {
	return s.sessionService.db
}

func (s *RouteBoardingService) createSegmentAndBoardEvent(exec repository.DBTX, boarding *models.RouteBoarding, sessionID string, occurredAt time.Time) error {
	segmentCount, err := s.segmentRepo.CountBySessionIDWithExecutor(exec, sessionID)
	if err != nil {
		return err
	}

	var routeBoardingID *string
	if boarding.ID != "" {
		routeBoardingID = &boarding.ID
	}
	fromStopID := boarding.BoardingStopID
	segment := &models.JourneySegment{
		ID:              uuid.New().String(),
		SessionID:       sessionID,
		RouteBoardingID: routeBoardingID,
		SegmentIndex:    segmentCount + 1,
		RouteID:         &boarding.RouteID,
		VehicleID:       boarding.VehicleID,
		FromStopID:      &fromStopID,
		BoardedAt:       boarding.BoardingTime,
		DistanceKM:      0,
		FareAmount:      0,
		Metadata:        "{}",
		CreatedAt:       occurredAt,
		UpdatedAt:       occurredAt,
	}
	if err := s.segmentRepo.CreateWithExecutor(exec, segment); err != nil {
		return err
	}

	event := &models.JourneyEvent{
		ID:              uuid.New().String(),
		SessionID:       sessionID,
		RouteBoardingID: routeBoardingID,
		SegmentID:       &segment.ID,
		EventType:       "boarded",
		StopID:          &fromStopID,
		Latitude:        &boarding.BoardingLat,
		Longitude:       &boarding.BoardingLon,
		OccurredAt:      occurredAt,
		Metadata:        "{}",
		CreatedAt:       occurredAt,
	}
	return s.eventRepo.CreateWithExecutor(exec, event)
}

func (s *RouteBoardingService) completeSegmentAndCreateAlightEvent(exec repository.DBTX, boarding *models.RouteBoarding, occurredAt time.Time) error {
	segment, err := s.segmentRepo.GetByRouteBoardingIDWithExecutor(exec, boarding.ID)
	if err != nil {
		return err
	}

	segment.ToStopID = boarding.AlightingStopID
	segment.AlightedAt = boarding.AlightingTime
	segment.DistanceKM = boarding.Distance
	segment.FareAmount = boarding.Fare
	segment.UpdatedAt = occurredAt
	if err := s.segmentRepo.UpdateWithExecutor(exec, segment); err != nil {
		return err
	}

	event := &models.JourneyEvent{
		ID:              uuid.New().String(),
		SessionID:       boarding.SessionID,
		RouteBoardingID: &boarding.ID,
		SegmentID:       &segment.ID,
		EventType:       "alighted",
		StopID:          boarding.AlightingStopID,
		Latitude:        boarding.AlightingLat,
		Longitude:       boarding.AlightingLon,
		OccurredAt:      occurredAt,
		Metadata:        "{}",
		CreatedAt:       occurredAt,
	}
	return s.eventRepo.CreateWithExecutor(exec, event)
}

func (s *RouteBoardingService) createFareTransaction(exec repository.DBTX, boarding *models.RouteBoarding, agencyID string, occurredAt time.Time) error {
	segment, err := s.segmentRepo.GetByRouteBoardingIDWithExecutor(exec, boarding.ID)
	if err != nil {
		return err
	}

	product, err := s.fareService.GetFareProductForAgency(agencyID)
	if err != nil {
		return err
	}

	var fareProductID *string
	ruleVersion := "legacy-v1"
	currencyCode := "INR"
	if product != nil {
		fareProductID = &product.ID
		ruleVersion = product.RuleVersion
		currencyCode = product.CurrencyCode
	}

	sessionID := boarding.SessionID
	segmentID := segment.ID
	routeBoardingID := boarding.ID
	transaction := &models.FareTransaction{
		ID:                   uuid.New().String(),
		UserID:               mustSessionUserID(exec, s.sessionService.repo, boarding.SessionID),
		SessionID:            &sessionID,
		SegmentID:            &segmentID,
		RouteBoardingID:      &routeBoardingID,
		FareProductID:        fareProductID,
		Amount:               boarding.Fare,
		CurrencyCode:         currencyCode,
		RuleVersion:          ruleVersion,
		Status:               "applied",
		ReconciliationStatus: "unreconciled",
		ChargedAt:            occurredAt,
		Metadata:             fmt.Sprintf(`{"agency_id":"%s","route_id":"%s"}`, agencyID, boarding.RouteID),
		CreatedAt:            occurredAt,
		UpdatedAt:            occurredAt,
	}
	return s.fareTransactionRepo.CreateWithExecutor(exec, transaction)
}

func mustSessionUserID(exec repository.DBTX, repo *repository.JourneySessionRepository, sessionID string) string {
	session, err := repo.GetByIDWithExecutor(exec, sessionID)
	if err != nil || session == nil {
		return ""
	}
	return session.UserID
}
