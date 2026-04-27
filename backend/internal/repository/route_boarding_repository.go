package repository

import (
	"database/sql"
	"fmt"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type RouteBoardingRepository struct {
	db *database.DB
}

func NewRouteBoardingRepository(db *database.DB) *RouteBoardingRepository {
	return &RouteBoardingRepository{db: db}
}

func (r *RouteBoardingRepository) Create(boarding *models.RouteBoarding) error {
	return r.CreateWithExecutor(r.db, boarding)
}

func (r *RouteBoardingRepository) CreateWithExecutor(exec DBTX, boarding *models.RouteBoarding) error {
	query := `INSERT INTO route_boardings
		(id, session_id, route_id, vehicle_id, boarding_stop_id, boarding_time, boarding_lat, boarding_lon, distance, fare, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := exec.Exec(query,
		boarding.ID, boarding.SessionID, boarding.RouteID, boarding.VehicleID, boarding.BoardingStopID,
		boarding.BoardingTime, boarding.BoardingLat, boarding.BoardingLon,
		boarding.Distance, boarding.Fare, boarding.CreatedAt, boarding.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert route boarding: %w", err)
	}

	return nil
}

func (r *RouteBoardingRepository) Update(boarding *models.RouteBoarding) error {
	return r.UpdateWithExecutor(r.db, boarding)
}

func (r *RouteBoardingRepository) UpdateWithExecutor(exec DBTX, boarding *models.RouteBoarding) error {
	query := `UPDATE route_boardings SET
		alighting_stop_id = ?,
		alighting_time = ?,
		alighting_lat = ?,
		alighting_lon = ?,
		distance = ?,
		fare = ?,
		updated_at = ?
		WHERE id = ?`

	_, err := exec.Exec(query,
		boarding.AlightingStopID, boarding.AlightingTime,
		boarding.AlightingLat, boarding.AlightingLon,
		boarding.Distance, boarding.Fare,
		boarding.UpdatedAt, boarding.ID)
	if err != nil {
		return fmt.Errorf("update route boarding: %w", err)
	}

	return nil
}

func (r *RouteBoardingRepository) GetByID(boardingID string) (*models.RouteBoarding, error) {
	return r.GetByIDWithExecutor(r.db, boardingID)
}

func (r *RouteBoardingRepository) GetByIDWithExecutor(exec DBTX, boardingID string) (*models.RouteBoarding, error) {
	query := `SELECT id, session_id, route_id, vehicle_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE id = ?`

	return r.getOne(exec, query, boardingID)
}

func (r *RouteBoardingRepository) GetActiveBySessionID(sessionID string) (*models.RouteBoarding, error) {
	return r.GetActiveBySessionIDWithExecutor(r.db, sessionID)
}

func (r *RouteBoardingRepository) GetActiveBySessionIDWithExecutor(exec DBTX, sessionID string) (*models.RouteBoarding, error) {
	query := `SELECT id, session_id, route_id, vehicle_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE session_id = ? AND alighting_time IS NULL ORDER BY boarding_time DESC LIMIT 1`

	boarding, err := r.getOne(exec, query, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return boarding, nil
}

func (r *RouteBoardingRepository) ListBySessionID(sessionID string) ([]models.RouteBoarding, error) {
	return r.ListBySessionIDWithExecutor(r.db, sessionID)
}

func (r *RouteBoardingRepository) ListBySessionIDWithExecutor(exec DBTX, sessionID string) ([]models.RouteBoarding, error) {
	query := `SELECT id, session_id, route_id, vehicle_id, boarding_stop_id, alighting_stop_id, boarding_time, alighting_time,
		boarding_lat, boarding_lon, alighting_lat, alighting_lon, distance, fare, created_at, updated_at
		FROM route_boardings WHERE session_id = ? ORDER BY boarding_time`

	rows, err := exec.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query route boardings: %w", err)
	}
	defer rows.Close()

	var boardings []models.RouteBoarding
	for rows.Next() {
		boarding, err := scanRouteBoarding(rows)
		if err != nil {
			continue
		}
		boardings = append(boardings, *boarding)
	}

	return boardings, nil
}

func (r *RouteBoardingRepository) getOne(exec DBTX, query string, args ...any) (*models.RouteBoarding, error) {
	row := exec.QueryRow(query, args...)
	boarding, err := scanRouteBoarding(row)
	if err != nil {
		return nil, err
	}
	return boarding, nil
}

func scanRouteBoarding(scanner rowScanner) (*models.RouteBoarding, error) {
	boarding := &models.RouteBoarding{}
	var vehicleID sql.NullString
	var alightingStopID sql.NullString
	var alightingTime sql.NullTime
	var alightingLat, alightingLon sql.NullFloat64

	err := scanner.Scan(
		&boarding.ID, &boarding.SessionID, &boarding.RouteID, &vehicleID,
		&boarding.BoardingStopID, &alightingStopID,
		&boarding.BoardingTime, &alightingTime,
		&boarding.BoardingLat, &boarding.BoardingLon,
		&alightingLat, &alightingLon,
		&boarding.Distance, &boarding.Fare,
		&boarding.CreatedAt, &boarding.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if vehicleID.Valid {
		boarding.VehicleID = &vehicleID.String
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
