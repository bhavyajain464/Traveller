package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/domain"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/repository"
)

type JourneySessionService struct {
	db           *database.DB
	billRepo     *repository.DailyBillRepository
	repo         *repository.JourneySessionRepository
	eventRepo    *repository.JourneyEventRepository
	stopService  *StopService
	fareService  *FareService
	routePlanner JourneyPlanner
}

func NewJourneySessionService(db *database.DB, repo *repository.JourneySessionRepository, billRepo *repository.DailyBillRepository, eventRepo *repository.JourneyEventRepository, stopService *StopService, fareService *FareService, routePlanner JourneyPlanner) *JourneySessionService {
	return &JourneySessionService{
		db:           db,
		billRepo:     billRepo,
		repo:         repo,
		eventRepo:    eventRepo,
		stopService:  stopService,
		fareService:  fareService,
		routePlanner: routePlanner,
	}
}

// CheckIn starts a new journey session and generates QR code
func (s *JourneySessionService) CheckIn(req models.CheckInRequest) (*models.JourneySession, *models.QRCodeTicket, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin check-in transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.ensureNoOverduePendingBillsWithExecutor(tx, req.UserID, time.Now()); err != nil {
		return nil, nil, err
	}

	// Find nearest stop if not provided, but don't fail check-in if no stop is nearby.
	stopID := req.StopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err == nil && len(stops) > 0 {
			stopID = &stops[0].ID
		}
	}

	// Generate QR code
	sessionID := uuid.New().String()
	qrCode := s.generateQRCode(sessionID, req.UserID)

	checkInStopID := ""
	if stopID != nil {
		checkInStopID = *stopID
	}

	// Create journey session
	session := &models.JourneySession{
		ID:            sessionID,
		UserID:        req.UserID,
		QRCode:        qrCode,
		CheckInTime:   time.Now(),
		CheckInStopID: checkInStopID,
		CheckInLat:    req.Latitude,
		CheckInLon:    req.Longitude,
		Status:        string(domain.JourneyStatusActive),
		RoutesUsed:    []string{},
		TotalDistance: 0,
		TotalFare:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreateWithExecutor(tx, session); err != nil {
		return nil, nil, err
	}

	if err := s.createSessionEvent(tx, session.ID, "checked_in", stringPtrIfNotEmpty(session.CheckInStopID), floatPtr(session.CheckInLat), floatPtr(session.CheckInLon), session.CheckInTime); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit check-in transaction: %w", err)
	}

	// Create QR ticket
	ticket := &models.QRCodeTicket{
		Code:          qrCode,
		UserID:        req.UserID,
		SessionID:     sessionID,
		CheckInTime:   session.CheckInTime,
		CheckInStopID: checkInStopID,
		ExpiresAt:     session.CheckInTime.Add(24 * time.Hour), // Valid for 24 hours
		IsValid:       true,
	}

	return session, ticket, nil
}

// CheckOut completes a journey session and calculates fare based on actual routes taken
func (s *JourneySessionService) CheckOut(req models.CheckOutRequest, routeBoardingService *RouteBoardingService) (*models.JourneySession, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin check-out transaction: %w", err)
	}
	defer tx.Rollback()

	session, err := s.repo.GetByQRCodeWithExecutor(tx, req.QRCode)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code: %w", err)
	}

	if !domain.IsActiveJourneyStatus(session.Status) {
		return nil, fmt.Errorf("journey session is not active")
	}

	// Find nearest stop if not provided. If we don't have one, continue with location-only checkout.
	stopID := req.StopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err == nil && len(stops) > 0 {
			stopID = &stops[0].ID
		}
	}

	// Check if user is still on a route (has active boarding)
	// If yes, alight from that route first
	if routeBoardingService != nil {
		activeBoarding, err := routeBoardingService.GetActiveBoardingWithExecutor(tx, session.ID)
		if err == nil && activeBoarding != nil && stopID != nil {
			// Alight from current route
			alightReq := models.AlightRouteRequest{
				BoardingID:      activeBoarding.ID,
				AlightingStopID: stopID,
				Latitude:        req.Latitude,
				Longitude:       req.Longitude,
			}
			_, err = routeBoardingService.AlightRouteWithExecutor(tx, alightReq)
			if err != nil {
				return nil, fmt.Errorf("failed to alight from current route: %w", err)
			}
		}

		// Calculate fare from actual route boardings
		totalDistance, totalFare, routesUsed, err := routeBoardingService.CalculateFareFromBoardingsWithExecutor(tx, session.ID)
		if err == nil && len(routesUsed) > 0 {
			// Use actual tracked routes
			session.TotalDistance = totalDistance
			session.TotalFare = totalFare
			session.RoutesUsed = routesUsed
		} else {
			// Source of truth is tracked boardings/location flow.
			session.TotalDistance = 0
			session.TotalFare = 0
			session.RoutesUsed = []string{}
		}
	} else {
		session.TotalDistance = 0
		session.TotalFare = 0
		session.RoutesUsed = []string{}
	}

	// Update session
	now := time.Now()
	checkOutTime := now
	session.CheckOutTime = &checkOutTime
	session.CheckOutStopID = stopID
	checkOutLat := req.Latitude
	checkOutLon := req.Longitude
	session.CheckOutLat = &checkOutLat
	session.CheckOutLon = &checkOutLon
	session.Status = string(domain.JourneyStatusCompleted)
	session.UpdatedAt = now

	if err := s.repo.UpdateWithExecutor(tx, session); err != nil {
		return nil, err
	}

	if err := s.createSessionEvent(tx, session.ID, "checked_out", session.CheckOutStopID, session.CheckOutLat, session.CheckOutLon, now); err != nil {
		return nil, err
	}

	err = s.updateDailyBillWithExecutor(tx, session.UserID, session.CheckInTime)
	if err != nil {
		return nil, fmt.Errorf("update daily bill: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit check-out transaction: %w", err)
	}

	return session, nil
}

// ValidateQRCode validates a QR code for a route
func (s *JourneySessionService) ValidateQRCode(qrCode string, routeID string) (*models.QRCodeTicket, error) {
	session, err := s.GetSessionByQRCode(qrCode)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code")
	}

	if !domain.IsActiveJourneyStatus(session.Status) {
		return nil, fmt.Errorf("QR code is not active")
	}

	// Check if QR code has expired (24 hours)
	if time.Since(session.CheckInTime) > 24*time.Hour {
		return nil, fmt.Errorf("QR code has expired")
	}

	ticket := &models.QRCodeTicket{
		Code:          qrCode,
		UserID:        session.UserID,
		SessionID:     session.ID,
		CheckInTime:   session.CheckInTime,
		CheckInStopID: session.CheckInStopID,
		ExpiresAt:     session.CheckInTime.Add(24 * time.Hour),
		IsValid:       true,
	}

	return ticket, nil
}

// GetSessionByQRCode retrieves a session by QR code
func (s *JourneySessionService) GetSessionByQRCode(qrCode string) (*models.JourneySession, error) {
	session, err := s.repo.GetByQRCode(qrCode)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return session, nil
}

// GetActiveSessions returns all active sessions for a user
func (s *JourneySessionService) GetActiveSessions(userID string) ([]models.JourneySession, error) {
	return s.repo.GetActiveByUserID(userID)
}

// GetLatestActiveSession returns the most recent active session for a user, if any.
func (s *JourneySessionService) GetLatestActiveSession(userID string) (*models.JourneySession, error) {
	sessions, err := s.GetActiveSessions(userID)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

// ListSessions returns the most recent sessions for a user, newest first.
func (s *JourneySessionService) ListSessions(userID string, limit int) ([]models.JourneySession, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.ListByUserID(userID, limit)
}

// GetSessionByID retrieves a session by ID
func (s *JourneySessionService) GetSessionByID(sessionID string) (*models.JourneySession, error) {
	session, err := s.repo.GetByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return session, nil
}

// inferJourneyDetails infers journey details when routes are not tracked (fallback)
func (s *JourneySessionService) inferJourneyDetails(session *models.JourneySession, req models.CheckOutRequest) (float64, float64, []string) {
	journeyReq := models.JourneyRequest{
		FromLat:       session.CheckInLat,
		FromLon:       session.CheckInLon,
		ToLat:         req.Latitude,
		ToLon:         req.Longitude,
		DepartureTime: &session.CheckInTime,
	}

	options, err := s.routePlanner.PlanJourney(journeyReq)
	if err != nil || len(options) == 0 {
		// Fallback: calculate direct distance
		distance := s.calculateDirectDistance(session.CheckInLat, session.CheckInLon, req.Latitude, req.Longitude)
		rules := s.fareService.GetFareRulesForAgency("DIMTS") // Default to Delhi bus
		return distance, rules.BaseFare, []string{}
	}

	// Use best option
	bestOption := options[0]
	distance := s.calculateTotalDistance(bestOption)
	var fare float64
	if bestOption.Fare != nil {
		fare = *bestOption.Fare
	}

	routesUsed := make([]string, 0)
	for _, leg := range bestOption.Legs {
		if leg.RouteID != "" {
			routesUsed = append(routesUsed, leg.RouteID)
		}
	}

	return distance, fare, routesUsed
}

// updateDailyBill updates or creates daily bill for user
func (s *JourneySessionService) updateDailyBill(userID string, journeyDate time.Time) error {
	return s.updateDailyBillWithExecutor(s.db, userID, journeyDate)
}

func (s *JourneySessionService) updateDailyBillWithExecutor(exec repository.DBTX, userID string, journeyDate time.Time) error {
	billDate := time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(), 0, 0, 0, 0, journeyDate.Location())
	nextDate := billDate.AddDate(0, 0, 1)

	totalJourneys, totalDistance, totalFare, err := s.repo.AggregateCompletedByUserAndRangeWithExecutor(exec, userID, billDate, nextDate)
	if err != nil {
		return err
	}

	billID := uuid.New().String()
	now := time.Now()
	return s.billRepo.UpsertPendingTotalsWithExecutor(exec, billID, userID, billDate, totalJourneys, totalDistance, totalFare, now)
}

func (s *JourneySessionService) ensureNoOverduePendingBills(userID string, now time.Time) error {
	return s.ensureNoOverduePendingBillsWithExecutor(s.db, userID, now)
}

func (s *JourneySessionService) ensureNoOverduePendingBillsWithExecutor(exec repository.DBTX, userID string, now time.Time) error {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := s.billRepo.CountPendingBeforeDateWithExecutor(exec, userID, todayStart)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("payment required for overdue bills before starting a new journey")
	}
	return nil
}

func (s *JourneySessionService) createSessionEvent(exec repository.DBTX, sessionID, eventType string, stopID *string, lat, lon *float64, occurredAt time.Time) error {
	event := &models.JourneyEvent{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		EventType:  eventType,
		StopID:     stopID,
		Latitude:   lat,
		Longitude:  lon,
		OccurredAt: occurredAt,
		Metadata:   "{}",
		CreatedAt:  occurredAt,
	}
	return s.eventRepo.CreateWithExecutor(exec, event)
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

// Helper functions
func (s *JourneySessionService) generateQRCode(sessionID, userID string) string {
	// Generate a unique QR code
	// Format: TRANSIT-{timestamp}-{sessionID}-{hash}
	timestamp := time.Now().Unix()
	return fmt.Sprintf("TRANSIT-%d-%s-%s", timestamp, safeCodePrefix(sessionID, 8), safeCodePrefix(userID, 8))
}

func safeCodePrefix(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func (s *JourneySessionService) calculateDirectDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return haversineDistance(lat1, lon1, lat2, lon2) / 1000.0
}

func (s *JourneySessionService) calculateTotalDistance(option models.JourneyOption) float64 {
	total := 0.0
	for _, leg := range option.Legs {
		if leg.Mode != "walking" && leg.FromStopID != "" && leg.ToStopID != "" {
			fromStop, _ := s.stopService.GetByID(leg.FromStopID)
			toStop, _ := s.stopService.GetByID(leg.ToStopID)
			if fromStop != nil && toStop != nil {
				total += haversineDistance(fromStop.Latitude, fromStop.Longitude, toStop.Latitude, toStop.Longitude) / 1000.0
			}
		}
	}
	return total
}
