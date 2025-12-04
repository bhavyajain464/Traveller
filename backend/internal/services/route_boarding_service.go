package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type RouteBoardingService struct {
	db            *database.DB
	stopService   *StopService
	fareService   *FareService
	sessionService *JourneySessionService
}

func NewRouteBoardingService(db *database.DB, stopService *StopService, fareService *FareService, sessionService *JourneySessionService) *RouteBoardingService {
	return &RouteBoardingService{
		db:            db,
		stopService:   stopService,
		fareService:   fareService,
		sessionService: sessionService,
	}
}

// BoardRoute records when user boards a route (called when QR is validated on vehicle)
func (s *RouteBoardingService) BoardRoute(req models.BoardRouteRequest) (*models.RouteBoarding, error) {
	// Get session by QR code or session ID
	var session *models.JourneySession
	var err error
	
	if req.SessionID != "" {
		session, err = s.sessionService.GetSessionByID(req.SessionID)
	} else if req.QRCode != "" {
		session, err = s.sessionService.GetSessionByQRCode(req.QRCode)
	} else {
		return nil, fmt.Errorf("session_id or qr_code is required")
	}
	
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	if session.Status != "active" {
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
	activeBoarding, err := s.GetActiveBoarding(session.ID)
	if err == nil && activeBoarding != nil {
		return nil, fmt.Errorf("user is already on route %s. Please alight first", activeBoarding.RouteID)
	}

	// Create boarding record
	boardingID := uuid.New().String()
	now := time.Now()
	boarding := &models.RouteBoarding{
		ID:            boardingID,
		SessionID:     session.ID,
		RouteID:       req.RouteID,
		BoardingStopID: *stopID,
		BoardingTime:  now,
		BoardingLat:   req.Latitude,
		BoardingLon:   req.Longitude,
		Distance:      0,
		Fare:          0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Insert into database
	query := `INSERT INTO route_boardings 
		(id, session_id, route_id, boarding_stop_id, boarding_time, boarding_lat, boarding_lon, distance, fare, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = s.db.Exec(query,
		boarding.ID, boarding.SessionID, boarding.RouteID, boarding.BoardingStopID,
		boarding.BoardingTime, boarding.BoardingLat, boarding.BoardingLon,
		boarding.Distance, boarding.Fare, boarding.CreatedAt, boarding.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create boarding record: %w", err)
	}

	return boarding, nil
}

// AlightRoute records when user alights from a route
func (s *RouteBoardingService) AlightRoute(req models.AlightRouteRequest) (*models.RouteBoarding, error) {
	// Get boarding record
	boarding, err := s.GetBoardingByID(req.BoardingID)
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
	agencyID := s.fareService.GetAgencyIDFromRoute(boarding.RouteID)
	if agencyID == "" {
		agencyID = "DIMTS" // Default to Delhi bus if not found
	}
	rules := s.fareService.GetFareRulesForAgency(agencyID)
	fare := s.fareService.CalculateRouteSegmentFare(boarding.RouteID, boarding.BoardingStopID, *stopID, distance, rules)

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

	updateQuery := `UPDATE route_boardings SET
		alighting_stop_id = $1,
		alighting_time = $2,
		alighting_lat = $3,
		alighting_lon = $4,
		distance = $5,
		fare = $6,
		updated_at = $7
		WHERE id = $8`

	_, err = s.db.Exec(updateQuery,
		boarding.AlightingStopID, boarding.AlightingTime,
		boarding.AlightingLat, boarding.AlightingLon,
		boarding.Distance, boarding.Fare,
		boarding.UpdatedAt, boarding.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update boarding record: %w", err)
	}

	return boarding, nil
}

// GetBoardingsForSession returns all route boardings for a journey session
func (s *RouteBoardingService) GetBoardingsForSession(sessionID string) ([]models.RouteBoarding, error) {
	query := `SELECT id, session_id, route_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE session_id = $1 ORDER BY boarding_time`

	rows, err := s.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get boardings: %w", err)
	}
	defer rows.Close()

	var boardings []models.RouteBoarding
	for rows.Next() {
		boarding := models.RouteBoarding{}
		var alightingStopID sql.NullString
		var alightingTime sql.NullTime
		var alightingLat, alightingLon sql.NullFloat64

		err := rows.Scan(
			&boarding.ID, &boarding.SessionID, &boarding.RouteID,
			&boarding.BoardingStopID, &alightingStopID,
			&boarding.BoardingTime, &alightingTime,
			&boarding.BoardingLat, &boarding.BoardingLon,
			&alightingLat, &alightingLon,
			&boarding.Distance, &boarding.Fare,
			&boarding.CreatedAt, &boarding.UpdatedAt)
		if err != nil {
			continue
		}

		if alightingStopID.Valid {
			boarding.AlightingStopID = &alightingStopID.String
		}
		if alightingTime.Valid {
			t := alightingTime.Time
			boarding.AlightingTime = &t
		}
		if alightingLat.Valid {
			lat := alightingLat.Float64
			boarding.AlightingLat = &lat
		}
		if alightingLon.Valid {
			lon := alightingLon.Float64
			boarding.AlightingLon = &lon
		}

		boardings = append(boardings, boarding)
	}

	return boardings, nil
}

// GetActiveBoarding returns the currently active boarding (user is on a route)
func (s *RouteBoardingService) GetActiveBoarding(sessionID string) (*models.RouteBoarding, error) {
	query := `SELECT id, session_id, route_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE session_id = $1 AND alighting_time IS NULL ORDER BY boarding_time DESC LIMIT 1`

	boarding := &models.RouteBoarding{}
	var alightingStopID sql.NullString
	var alightingTime sql.NullTime
	var alightingLat, alightingLon sql.NullFloat64

	err := s.db.QueryRow(query, sessionID).Scan(
		&boarding.ID, &boarding.SessionID, &boarding.RouteID,
		&boarding.BoardingStopID, &alightingStopID,
		&boarding.BoardingTime, &alightingTime,
		&boarding.BoardingLat, &boarding.BoardingLon,
		&alightingLat, &alightingLon,
		&boarding.Distance, &boarding.Fare,
		&boarding.CreatedAt, &boarding.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // No active boarding
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active boarding: %w", err)
	}

	return boarding, nil
}

// CalculateFareFromBoardings calculates total fare from actual route boardings
func (s *RouteBoardingService) CalculateFareFromBoardings(sessionID string) (float64, float64, []string, error) {
	boardings, err := s.GetBoardingsForSession(sessionID)
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
	query := `SELECT id, session_id, route_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE id = $1`

	boarding := &models.RouteBoarding{}
	var alightingStopID sql.NullString
	var alightingTime sql.NullTime
	var alightingLat, alightingLon sql.NullFloat64

	err := s.db.QueryRow(query, boardingID).Scan(
		&boarding.ID, &boarding.SessionID, &boarding.RouteID,
		&boarding.BoardingStopID, &alightingStopID,
		&boarding.BoardingTime, &alightingTime,
		&boarding.BoardingLat, &boarding.BoardingLon,
		&alightingLat, &alightingLon,
		&boarding.Distance, &boarding.Fare,
		&boarding.CreatedAt, &boarding.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("boarding not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get boarding: %w", err)
	}

	if alightingStopID.Valid {
		boarding.AlightingStopID = &alightingStopID.String
	}
	if alightingTime.Valid {
		t := alightingTime.Time
		boarding.AlightingTime = &t
	}
	if alightingLat.Valid {
		lat := alightingLat.Float64
		boarding.AlightingLat = &lat
	}
	if alightingLon.Valid {
		lon := alightingLon.Float64
		boarding.AlightingLon = &lon
	}

	return boarding, nil
}

