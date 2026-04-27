package repository

import (
	"database/sql"
	"fmt"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type JourneySegmentRepository struct {
	db *database.DB
}

func NewJourneySegmentRepository(db *database.DB) *JourneySegmentRepository {
	return &JourneySegmentRepository{db: db}
}

func (r *JourneySegmentRepository) CreateWithExecutor(exec DBTX, segment *models.JourneySegment) error {
	query := `INSERT INTO journey_segments
		(id, session_id, route_boarding_id, segment_index, route_id, vehicle_id, from_stop_id, to_stop_id, boarded_at, alighted_at, distance_km, fare_amount, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?)`

	_, err := exec.Exec(query,
		segment.ID, segment.SessionID, segment.RouteBoardingID, segment.SegmentIndex, segment.RouteID, segment.VehicleID,
		segment.FromStopID, segment.ToStopID, segment.BoardedAt, segment.AlightedAt, segment.DistanceKM, segment.FareAmount,
		defaultJSON(segment.Metadata), segment.CreatedAt, segment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert journey segment: %w", err)
	}
	return nil
}

func (r *JourneySegmentRepository) UpdateWithExecutor(exec DBTX, segment *models.JourneySegment) error {
	query := `UPDATE journey_segments SET
		to_stop_id = ?,
		alighted_at = ?,
		distance_km = ?,
		fare_amount = ?,
		metadata = ?::jsonb,
		updated_at = ?
		WHERE id = ?`

	_, err := exec.Exec(query,
		segment.ToStopID, segment.AlightedAt, segment.DistanceKM, segment.FareAmount,
		defaultJSON(segment.Metadata), segment.UpdatedAt, segment.ID,
	)
	if err != nil {
		return fmt.Errorf("update journey segment: %w", err)
	}
	return nil
}

func (r *JourneySegmentRepository) CountBySessionIDWithExecutor(exec DBTX, sessionID string) (int, error) {
	query := `SELECT COUNT(*) FROM journey_segments WHERE session_id = ?`
	var count int
	if err := exec.QueryRow(query, sessionID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count journey segments: %w", err)
	}
	return count, nil
}

func (r *JourneySegmentRepository) GetByRouteBoardingIDWithExecutor(exec DBTX, routeBoardingID string) (*models.JourneySegment, error) {
	query := `SELECT id, session_id, route_boarding_id, segment_index, route_id, vehicle_id, from_stop_id, to_stop_id, boarded_at, alighted_at, distance_km, fare_amount, metadata, created_at, updated_at
		FROM journey_segments WHERE route_boarding_id = ?`

	row := exec.QueryRow(query, routeBoardingID)
	segment := &models.JourneySegment{}
	var routeID, vehicleID, fromStopID, toStopID sql.NullString
	var routeBoardingRef sql.NullString
	var alightedAt sql.NullTime
	if err := row.Scan(
		&segment.ID, &segment.SessionID, &routeBoardingRef, &segment.SegmentIndex, &routeID, &vehicleID,
		&fromStopID, &toStopID, &segment.BoardedAt, &alightedAt, &segment.DistanceKM, &segment.FareAmount,
		&segment.Metadata, &segment.CreatedAt, &segment.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if routeBoardingRef.Valid {
		segment.RouteBoardingID = &routeBoardingRef.String
	}
	if routeID.Valid {
		segment.RouteID = &routeID.String
	}
	if vehicleID.Valid {
		segment.VehicleID = &vehicleID.String
	}
	if fromStopID.Valid {
		segment.FromStopID = &fromStopID.String
	}
	if toStopID.Valid {
		segment.ToStopID = &toStopID.String
	}
	if alightedAt.Valid {
		t := alightedAt.Time
		segment.AlightedAt = &t
	}
	return segment, nil
}

type JourneyEventRepository struct {
	db *database.DB
}

func NewJourneyEventRepository(db *database.DB) *JourneyEventRepository {
	return &JourneyEventRepository{db: db}
}

func (r *JourneyEventRepository) CreateWithExecutor(exec DBTX, event *models.JourneyEvent) error {
	query := `INSERT INTO journey_events
		(id, session_id, route_boarding_id, segment_id, event_type, stop_id, latitude, longitude, occurred_at, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?)`

	_, err := exec.Exec(query,
		event.ID, event.SessionID, event.RouteBoardingID, event.SegmentID, event.EventType, event.StopID,
		event.Latitude, event.Longitude, event.OccurredAt, defaultJSON(event.Metadata), event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert journey event: %w", err)
	}
	return nil
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}

func nowPair() (time.Time, time.Time) {
	now := time.Now()
	return now, now
}
