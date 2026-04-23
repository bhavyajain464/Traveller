package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type JourneySessionService struct {
	db           *database.DB
	stopService  *StopService
	fareService  *FareService
	routePlanner *RoutePlanner
}

func NewJourneySessionService(db *database.DB, stopService *StopService, fareService *FareService, routePlanner *RoutePlanner) *JourneySessionService {
	return &JourneySessionService{
		db:           db,
		stopService:  stopService,
		fareService:  fareService,
		routePlanner: routePlanner,
	}
}

// CheckIn starts a new journey session and generates QR code
func (s *JourneySessionService) CheckIn(req models.CheckInRequest) (*models.JourneySession, *models.QRCodeTicket, error) {
	// Find nearest stop if not provided
	stopID := req.StopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err != nil || len(stops) == 0 {
			return nil, nil, fmt.Errorf("no nearby stop found")
		}
		stopID = &stops[0].ID
	}

	// Generate QR code
	sessionID := uuid.New().String()
	qrCode := s.generateQRCode(sessionID, req.UserID)

	// Create journey session
	session := &models.JourneySession{
		ID:            sessionID,
		UserID:        req.UserID,
		QRCode:        qrCode,
		CheckInTime:   time.Now(),
		CheckInStopID: *stopID,
		CheckInLat:    req.Latitude,
		CheckInLon:    req.Longitude,
		Status:        "active",
		RoutesUsed:    []string{},
		TotalDistance: 0,
		TotalFare:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Insert into database
	query := `INSERT INTO journey_sessions 
		(id, user_id, qr_code, check_in_time, check_in_stop_id, check_in_lat, check_in_lon, status, routes_used, total_distance, total_fare, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	routesJSON, _ := json.Marshal(session.RoutesUsed)
	_, err := s.db.Exec(query,
		session.ID, session.UserID, session.QRCode, session.CheckInTime,
		session.CheckInStopID, session.CheckInLat, session.CheckInLon,
		session.Status, routesJSON, session.TotalDistance, session.TotalFare,
		session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create journey session: %w", err)
	}

	// Create QR ticket
	ticket := &models.QRCodeTicket{
		Code:          qrCode,
		UserID:        req.UserID,
		SessionID:     sessionID,
		CheckInTime:   session.CheckInTime,
		CheckInStopID: *stopID,
		ExpiresAt:     session.CheckInTime.Add(24 * time.Hour), // Valid for 24 hours
		IsValid:       true,
	}

	return session, ticket, nil
}

// CheckOut completes a journey session and calculates fare based on actual routes taken
func (s *JourneySessionService) CheckOut(req models.CheckOutRequest, routeBoardingService *RouteBoardingService) (*models.JourneySession, error) {
	// Get session
	session, err := s.GetSessionByQRCode(req.QRCode)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code: %w", err)
	}

	if session.Status != "active" {
		return nil, fmt.Errorf("journey session is not active")
	}

	// Find nearest stop if not provided
	stopID := req.StopID
	if stopID == nil || *stopID == "" {
		stops, err := s.stopService.FindNearby(req.Latitude, req.Longitude, 500, 1)
		if err != nil || len(stops) == 0 {
			return nil, fmt.Errorf("no nearby stop found")
		}
		stopID = &stops[0].ID
	}

	// Check if user is still on a route (has active boarding)
	// If yes, alight from that route first
	if routeBoardingService != nil {
		activeBoarding, err := routeBoardingService.GetActiveBoarding(session.ID)
		if err == nil && activeBoarding != nil {
			// Alight from current route
			alightReq := models.AlightRouteRequest{
				BoardingID:      activeBoarding.ID,
				AlightingStopID: stopID,
				Latitude:        req.Latitude,
				Longitude:       req.Longitude,
			}
			_, err = routeBoardingService.AlightRoute(alightReq)
			if err != nil {
				return nil, fmt.Errorf("failed to alight from current route: %w", err)
			}
		}

		// Calculate fare from actual route boardings
		totalDistance, totalFare, routesUsed, err := routeBoardingService.CalculateFareFromBoardings(session.ID)
		if err == nil && len(routesUsed) > 0 {
			// Use actual tracked routes
			session.TotalDistance = totalDistance
			session.TotalFare = totalFare
			session.RoutesUsed = routesUsed
		} else {
			// Fallback: infer journey if no routes tracked
			session.TotalDistance, session.TotalFare, session.RoutesUsed = s.inferJourneyDetails(session, req)
		}
	} else {
		// Fallback: infer journey if route boarding service not available
		session.TotalDistance, session.TotalFare, session.RoutesUsed = s.inferJourneyDetails(session, req)
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
	session.Status = "completed"
	session.UpdatedAt = now

	// Update database
	routesJSON, _ := json.Marshal(session.RoutesUsed)
	updateQuery := `UPDATE journey_sessions SET
		check_out_time = ?,
		check_out_stop_id = ?,
		check_out_lat = ?,
		check_out_lon = ?,
		status = ?,
		routes_used = ?,
		total_distance = ?,
		total_fare = ?,
		updated_at = ?
		WHERE id = ?`

	_, err = s.db.Exec(updateQuery,
		session.CheckOutTime, session.CheckOutStopID,
		session.CheckOutLat, session.CheckOutLon,
		session.Status, routesJSON,
		session.TotalDistance, session.TotalFare,
		session.UpdatedAt, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update journey session: %w", err)
	}

	// Update daily bill
	err = s.updateDailyBill(session.UserID, session.CheckInTime)
	if err != nil {
		// Log error but don't fail checkout
		fmt.Printf("Warning: Failed to update daily bill: %v\n", err)
	}

	return session, nil
}

// ValidateQRCode validates a QR code for a route
func (s *JourneySessionService) ValidateQRCode(qrCode string, routeID string) (*models.QRCodeTicket, error) {
	session, err := s.GetSessionByQRCode(qrCode)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code")
	}

	if session.Status != "active" {
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
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE qr_code = ?`

	session := &models.JourneySession{}
	var checkOutTime sql.NullTime
	var checkOutStopID sql.NullString
	var checkOutLat, checkOutLon sql.NullFloat64
	var routesJSON string

	err := s.db.QueryRow(query, qrCode).Scan(
		&session.ID, &session.UserID, &session.QRCode,
		&session.CheckInTime, &checkOutTime, &session.CheckInStopID, &checkOutStopID,
		&session.CheckInLat, &session.CheckInLon, &checkOutLat, &checkOutLon,
		&session.Status, &routesJSON, &session.TotalDistance, &session.TotalFare,
		&session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if checkOutTime.Valid {
		t := checkOutTime.Time
		session.CheckOutTime = &t
	}
	if checkOutStopID.Valid {
		session.CheckOutStopID = &checkOutStopID.String
	}
	if checkOutLat.Valid {
		lat := checkOutLat.Float64
		session.CheckOutLat = &lat
	}
	if checkOutLon.Valid {
		lon := checkOutLon.Float64
		session.CheckOutLon = &lon
	}

	json.Unmarshal([]byte(routesJSON), &session.RoutesUsed)

	return session, nil
}

// GetActiveSessions returns all active sessions for a user
func (s *JourneySessionService) GetActiveSessions(userID string) ([]models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE user_id = ? AND status = 'active' ORDER BY check_in_time DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.JourneySession
	for rows.Next() {
		session := models.JourneySession{}
		var checkOutTime sql.NullTime
		var checkOutStopID sql.NullString
		var checkOutLat, checkOutLon sql.NullFloat64
		var routesJSON string

		err := rows.Scan(
			&session.ID, &session.UserID, &session.QRCode,
			&session.CheckInTime, &checkOutTime, &session.CheckInStopID, &checkOutStopID,
			&session.CheckInLat, &session.CheckInLon, &checkOutLat, &checkOutLon,
			&session.Status, &routesJSON, &session.TotalDistance, &session.TotalFare,
			&session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			continue
		}

		if checkOutTime.Valid {
			t := checkOutTime.Time
			session.CheckOutTime = &t
		}
		json.Unmarshal([]byte(routesJSON), &session.RoutesUsed)

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetSessionByID retrieves a session by ID
func (s *JourneySessionService) GetSessionByID(sessionID string) (*models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE id = ?`

	session := &models.JourneySession{}
	var checkOutTime sql.NullTime
	var checkOutStopID sql.NullString
	var checkOutLat, checkOutLon sql.NullFloat64
	var routesJSON string

	err := s.db.QueryRow(query, sessionID).Scan(
		&session.ID, &session.UserID, &session.QRCode,
		&session.CheckInTime, &checkOutTime, &session.CheckInStopID, &checkOutStopID,
		&session.CheckInLat, &session.CheckInLon, &checkOutLat, &checkOutLon,
		&session.Status, &routesJSON, &session.TotalDistance, &session.TotalFare,
		&session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if checkOutTime.Valid {
		t := checkOutTime.Time
		session.CheckOutTime = &t
	}
	if checkOutStopID.Valid {
		session.CheckOutStopID = &checkOutStopID.String
	}
	if checkOutLat.Valid {
		lat := checkOutLat.Float64
		session.CheckOutLat = &lat
	}
	if checkOutLon.Valid {
		lon := checkOutLon.Float64
		session.CheckOutLon = &lon
	}

	json.Unmarshal([]byte(routesJSON), &session.RoutesUsed)

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
	billDate := time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(), 0, 0, 0, 0, journeyDate.Location())
	nextDate := billDate.AddDate(0, 0, 1)

	// Get all completed journeys for the day
	query := `SELECT COUNT(*), COALESCE(SUM(total_distance), 0), COALESCE(SUM(total_fare), 0)
		FROM journey_sessions
		WHERE user_id = ? AND check_in_time >= ? AND check_in_time < ? AND status = 'completed'`

	var totalJourneys int
	var totalDistance, totalFare float64
	err := s.db.QueryRow(query, userID, billDate, nextDate).Scan(&totalJourneys, &totalDistance, &totalFare)
	if err != nil {
		return err
	}

	// Upsert daily bill
	upsertQuery := `INSERT INTO daily_bills (id, user_id, bill_date, total_journeys, total_distance, total_fare, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)
		ON CONFLICT (user_id, bill_date) DO UPDATE SET
			total_journeys = EXCLUDED.total_journeys,
			total_distance = EXCLUDED.total_distance,
			total_fare = EXCLUDED.total_fare,
			updated_at = EXCLUDED.updated_at`

	billID := uuid.New().String()
	now := time.Now()
	_, err = s.db.Exec(upsertQuery, billID, userID, billDate, totalJourneys, totalDistance, totalFare, now, now)
	return err
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
