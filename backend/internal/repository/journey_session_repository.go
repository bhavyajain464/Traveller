package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type JourneySessionRepository struct {
	db *database.DB
}

func NewJourneySessionRepository(db *database.DB) *JourneySessionRepository {
	return &JourneySessionRepository{db: db}
}

func (r *JourneySessionRepository) Create(session *models.JourneySession) error {
	return r.CreateWithExecutor(r.db, session)
}

func (r *JourneySessionRepository) CreateWithExecutor(exec DBTX, session *models.JourneySession) error {
	query := `INSERT INTO journey_sessions
		(id, user_id, qr_code, check_in_time, check_in_stop_id, check_in_lat, check_in_lon, status, routes_used, total_distance, total_fare, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	routesJSON, err := json.Marshal(session.RoutesUsed)
	if err != nil {
		return fmt.Errorf("marshal routes used: %w", err)
	}

	var checkInStopValue any
	if session.CheckInStopID != "" {
		checkInStopValue = session.CheckInStopID
	}

	_, err = exec.Exec(query,
		session.ID, session.UserID, session.QRCode, session.CheckInTime,
		checkInStopValue, session.CheckInLat, session.CheckInLon,
		session.Status, routesJSON, session.TotalDistance, session.TotalFare,
		session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert journey session: %w", err)
	}

	return nil
}

func (r *JourneySessionRepository) Update(session *models.JourneySession) error {
	return r.UpdateWithExecutor(r.db, session)
}

func (r *JourneySessionRepository) UpdateWithExecutor(exec DBTX, session *models.JourneySession) error {
	query := `UPDATE journey_sessions SET
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

	routesJSON, err := json.Marshal(session.RoutesUsed)
	if err != nil {
		return fmt.Errorf("marshal routes used: %w", err)
	}

	_, err = exec.Exec(query,
		session.CheckOutTime, session.CheckOutStopID,
		session.CheckOutLat, session.CheckOutLon,
		session.Status, routesJSON,
		session.TotalDistance, session.TotalFare,
		session.UpdatedAt, session.ID)
	if err != nil {
		return fmt.Errorf("update journey session: %w", err)
	}

	return nil
}

func (r *JourneySessionRepository) GetByQRCode(qrCode string) (*models.JourneySession, error) {
	return r.GetByQRCodeWithExecutor(r.db, qrCode)
}

func (r *JourneySessionRepository) GetByQRCodeWithExecutor(exec DBTX, qrCode string) (*models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE qr_code = ?`

	return r.getOne(exec, query, qrCode)
}

func (r *JourneySessionRepository) GetByID(sessionID string) (*models.JourneySession, error) {
	return r.GetByIDWithExecutor(r.db, sessionID)
}

func (r *JourneySessionRepository) GetByIDWithExecutor(exec DBTX, sessionID string) (*models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE id = ?`

	return r.getOne(exec, query, sessionID)
}

func (r *JourneySessionRepository) GetActiveByUserID(userID string) ([]models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE user_id = ? AND status = 'active' ORDER BY check_in_time DESC`

	return r.list(query, userID)
}

func (r *JourneySessionRepository) ListByUserID(userID string, limit int) ([]models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions WHERE user_id = ? ORDER BY check_in_time DESC LIMIT ?`

	return r.list(query, userID, limit)
}

func (r *JourneySessionRepository) ListCompletedByUserAndRange(userID string, start time.Time, end time.Time) ([]models.JourneySession, error) {
	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions
		WHERE user_id = ? AND check_in_time >= ? AND check_in_time < ? AND status = 'completed'
		ORDER BY check_in_time`

	return r.list(query, userID, start, end)
}

func (r *JourneySessionRepository) AggregateCompletedByUserAndRange(userID string, start time.Time, end time.Time) (int, float64, float64, error) {
	return r.AggregateCompletedByUserAndRangeWithExecutor(r.db, userID, start, end)
}

func (r *JourneySessionRepository) AggregateCompletedByUserAndRangeWithExecutor(exec DBTX, userID string, start time.Time, end time.Time) (int, float64, float64, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(total_distance), 0), COALESCE(SUM(total_fare), 0)
		FROM journey_sessions
		WHERE user_id = ? AND check_in_time >= ? AND check_in_time < ? AND status = 'completed'`

	var totalJourneys int
	var totalDistance, totalFare float64
	if err := exec.QueryRow(query, userID, start, end).Scan(&totalJourneys, &totalDistance, &totalFare); err != nil {
		return 0, 0, 0, fmt.Errorf("aggregate completed journey sessions: %w", err)
	}

	return totalJourneys, totalDistance, totalFare, nil
}

func (r *JourneySessionRepository) ListDistinctCompletedUserIDsInRange(start time.Time, end time.Time) ([]string, error) {
	query := `SELECT DISTINCT user_id FROM journey_sessions
		WHERE check_in_time >= ? AND check_in_time < ? AND status = 'completed'`

	rows, err := r.db.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query distinct completed users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err == nil {
			userIDs = append(userIDs, userID)
		}
	}

	return userIDs, nil
}

func (r *JourneySessionRepository) getOne(exec DBTX, query string, args ...any) (*models.JourneySession, error) {
	row := exec.QueryRow(query, args...)
	session, err := scanJourneySession(row)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *JourneySessionRepository) list(query string, args ...any) ([]models.JourneySession, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.JourneySession
	for rows.Next() {
		session, err := scanJourneySession(rows)
		if err != nil {
			continue
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJourneySession(scanner rowScanner) (*models.JourneySession, error) {
	session := &models.JourneySession{}
	var checkOutTime sql.NullTime
	var checkInStopID sql.NullString
	var checkOutStopID sql.NullString
	var checkOutLat, checkOutLon sql.NullFloat64
	var routesJSON string

	err := scanner.Scan(
		&session.ID, &session.UserID, &session.QRCode,
		&session.CheckInTime, &checkOutTime, &checkInStopID, &checkOutStopID,
		&session.CheckInLat, &session.CheckInLon, &checkOutLat, &checkOutLon,
		&session.Status, &routesJSON, &session.TotalDistance, &session.TotalFare,
		&session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan journey session: %w", err)
	}

	if checkInStopID.Valid {
		session.CheckInStopID = checkInStopID.String
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

	_ = json.Unmarshal([]byte(routesJSON), &session.RoutesUsed)
	return session, nil
}
