package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type DailyBillService struct {
	db *database.DB
}

func NewDailyBillService(db *database.DB) *DailyBillService {
	return &DailyBillService{db: db}
}

// GetDailyBill retrieves or creates daily bill for a user and date
func (s *DailyBillService) GetDailyBill(userID string, billDate time.Time) (*models.DailyBill, error) {
	date := time.Date(billDate.Year(), billDate.Month(), billDate.Day(), 0, 0, 0, 0, billDate.Location())

	query := `SELECT id, user_id, bill_date, total_journeys, total_distance, total_fare, status, 
		payment_id, payment_method, paid_at, created_at, updated_at
		FROM daily_bills WHERE user_id = ? AND bill_date = ?`

	bill := &models.DailyBill{}
	var paymentID, paymentMethod sql.NullString
	var paidAt sql.NullTime

	err := s.db.QueryRow(query, userID, date).Scan(
		&bill.ID, &bill.UserID, &bill.BillDate,
		&bill.TotalJourneys, &bill.TotalDistance, &bill.TotalFare,
		&bill.Status, &paymentID, &paymentMethod, &paidAt,
		&bill.CreatedAt, &bill.UpdatedAt)
	if err == sql.ErrNoRows {
		// Create new bill if doesn't exist
		return s.createDailyBill(userID, date)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily bill: %w", err)
	}

	if paymentID.Valid {
		bill.PaymentID = &paymentID.String
	}
	if paymentMethod.Valid {
		bill.PaymentMethod = &paymentMethod.String
	}
	if paidAt.Valid {
		bill.PaidAt = &paidAt.Time
	}

	// Load journeys for this bill
	journeys, err := s.getJourneysForBill(userID, date)
	if err == nil {
		bill.Journeys = journeys
	}

	return bill, nil
}

// GetPendingBills returns all pending bills for a user
func (s *DailyBillService) GetPendingBills(userID string) ([]models.DailyBill, error) {
	query := `SELECT id, user_id, bill_date, total_journeys, total_distance, total_fare, status, 
		payment_id, payment_method, paid_at, created_at, updated_at
		FROM daily_bills 
		WHERE user_id = ? AND status = 'pending'
		ORDER BY bill_date DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending bills: %w", err)
	}
	defer rows.Close()

	var bills []models.DailyBill
	for rows.Next() {
		bill := models.DailyBill{}
		var paymentID, paymentMethod sql.NullString
		var paidAt sql.NullTime

		err := rows.Scan(
			&bill.ID, &bill.UserID, &bill.BillDate,
			&bill.TotalJourneys, &bill.TotalDistance, &bill.TotalFare,
			&bill.Status, &paymentID, &paymentMethod, &paidAt,
			&bill.CreatedAt, &bill.UpdatedAt)
		if err != nil {
			continue
		}

		if paymentID.Valid {
			bill.PaymentID = &paymentID.String
		}
		if paymentMethod.Valid {
			bill.PaymentMethod = &paymentMethod.String
		}
		if paidAt.Valid {
			bill.PaidAt = &paidAt.Time
		}

		bills = append(bills, bill)
	}

	return bills, nil
}

// MarkBillAsPaid marks a bill as paid
func (s *DailyBillService) MarkBillAsPaid(billID string, paymentID string, paymentMethod string) error {
	query := `UPDATE daily_bills SET
		status = 'paid',
		payment_id = ?,
		payment_method = ?,
		paid_at = ?,
		updated_at = ?
		WHERE id = ?`

	now := time.Now()
	_, err := s.db.Exec(query, paymentID, paymentMethod, now, now, billID)
	if err != nil {
		return fmt.Errorf("failed to mark bill as paid: %w", err)
	}

	return nil
}

// GenerateDailyBills generates bills for all users for a specific date
func (s *DailyBillService) GenerateDailyBills(billDate time.Time) error {
	date := time.Date(billDate.Year(), billDate.Month(), billDate.Day(), 0, 0, 0, 0, billDate.Location())
	nextDate := date.AddDate(0, 0, 1)

	// Get all users who had journeys on this date
	query := `SELECT DISTINCT user_id FROM journey_sessions 
		WHERE check_in_time >= ? AND check_in_time < ? AND status = 'completed'`

	rows, err := s.db.Query(query, date, nextDate)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err == nil {
			userIDs = append(userIDs, userID)
		}
	}

	// Generate bills for each user
	for _, userID := range userIDs {
		_, err := s.GetDailyBill(userID, date)
		if err != nil {
			fmt.Printf("Warning: Failed to generate bill for user %s: %v\n", userID, err)
		}
	}

	return nil
}

// Helper functions
func (s *DailyBillService) createDailyBill(userID string, billDate time.Time) (*models.DailyBill, error) {
	nextDate := billDate.AddDate(0, 0, 1)

	// Calculate totals from journey sessions
	query := `SELECT COUNT(*), COALESCE(SUM(total_distance), 0), COALESCE(SUM(total_fare), 0)
		FROM journey_sessions
		WHERE user_id = ? AND check_in_time >= ? AND check_in_time < ? AND status = 'completed'`

	var totalJourneys int
	var totalDistance, totalFare float64
	err := s.db.QueryRow(query, userID, billDate, nextDate).Scan(&totalJourneys, &totalDistance, &totalFare)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate bill totals: %w", err)
	}

	billID := uuid.New().String()
	now := time.Now()

	insertQuery := `INSERT INTO daily_bills 
		(id, user_id, bill_date, total_journeys, total_distance, total_fare, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)`

	_, err = s.db.Exec(insertQuery, billID, userID, billDate, totalJourneys, totalDistance, totalFare, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create daily bill: %w", err)
	}

	bill := &models.DailyBill{
		ID:            billID,
		UserID:        userID,
		BillDate:      billDate,
		TotalJourneys: totalJourneys,
		TotalDistance: totalDistance,
		TotalFare:     totalFare,
		Status:        "pending",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Load journeys
	journeys, err := s.getJourneysForBill(userID, billDate)
	if err == nil {
		bill.Journeys = journeys
	}

	return bill, nil
}

func (s *DailyBillService) getJourneysForBill(userID string, billDate time.Time) ([]models.JourneySession, error) {
	nextDate := billDate.AddDate(0, 0, 1)

	query := `SELECT id, user_id, qr_code, check_in_time, check_out_time, check_in_stop_id, check_out_stop_id,
		check_in_lat, check_in_lon, check_out_lat, check_out_lon, status, routes_used, total_distance, total_fare,
		created_at, updated_at
		FROM journey_sessions
		WHERE user_id = ? AND check_in_time >= ? AND check_in_time < ? AND status = 'completed'
		ORDER BY check_in_time`

	rows, err := s.db.Query(query, userID, billDate, nextDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var journeys []models.JourneySession
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

		journeys = append(journeys, session)
	}

	return journeys, nil
}
