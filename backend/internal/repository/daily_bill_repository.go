package repository

import (
	"database/sql"
	"fmt"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type DailyBillRepository struct {
	db *database.DB
}

func NewDailyBillRepository(db *database.DB) *DailyBillRepository {
	return &DailyBillRepository{db: db}
}

func (r *DailyBillRepository) DB() *database.DB {
	return r.db
}

func (r *DailyBillRepository) GetByUserAndDate(userID string, billDate time.Time) (*models.DailyBill, error) {
	query := `SELECT id, user_id, bill_date, total_journeys, total_distance, total_fare, status,
		payment_id, payment_method, paid_at, created_at, updated_at
		FROM daily_bills WHERE user_id = ? AND bill_date = ?`

	return r.getOne(query, userID, billDate)
}

func (r *DailyBillRepository) GetByID(billID string) (*models.DailyBill, error) {
	query := `SELECT id, user_id, bill_date, total_journeys, total_distance, total_fare, status,
		payment_id, payment_method, paid_at, created_at, updated_at
		FROM daily_bills WHERE id = ?`

	return r.getOne(query, billID)
}

func (r *DailyBillRepository) ListPendingByUserID(userID string) ([]models.DailyBill, error) {
	query := `SELECT id, user_id, bill_date, total_journeys, total_distance, total_fare, status,
		payment_id, payment_method, paid_at, created_at, updated_at
		FROM daily_bills
		WHERE user_id = ? AND status = 'pending'
		ORDER BY bill_date DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("query pending daily bills: %w", err)
	}
	defer rows.Close()

	var bills []models.DailyBill
	for rows.Next() {
		bill, err := scanDailyBill(rows)
		if err != nil {
			continue
		}
		bills = append(bills, *bill)
	}

	return bills, nil
}

func (r *DailyBillRepository) Create(bill *models.DailyBill) error {
	query := `INSERT INTO daily_bills
		(id, user_id, bill_date, total_journeys, total_distance, total_fare, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		bill.ID, bill.UserID, bill.BillDate, bill.TotalJourneys, bill.TotalDistance,
		bill.TotalFare, bill.Status, bill.CreatedAt, bill.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert daily bill: %w", err)
	}

	return nil
}

func (r *DailyBillRepository) UpsertPendingTotals(
	billID string,
	userID string,
	billDate time.Time,
	totalJourneys int,
	totalDistance float64,
	totalFare float64,
	now time.Time,
) error {
	return r.UpsertPendingTotalsWithExecutor(r.db, billID, userID, billDate, totalJourneys, totalDistance, totalFare, now)
}

func (r *DailyBillRepository) UpsertPendingTotalsWithExecutor(
	exec DBTX,
	billID string,
	userID string,
	billDate time.Time,
	totalJourneys int,
	totalDistance float64,
	totalFare float64,
	now time.Time,
) error {
	query := `INSERT INTO daily_bills (id, user_id, bill_date, total_journeys, total_distance, total_fare, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)
		ON CONFLICT (user_id, bill_date) DO UPDATE SET
			total_journeys = EXCLUDED.total_journeys,
			total_distance = EXCLUDED.total_distance,
			total_fare = EXCLUDED.total_fare,
			updated_at = EXCLUDED.updated_at`

	_, err := exec.Exec(query, billID, userID, billDate, totalJourneys, totalDistance, totalFare, now, now)
	if err != nil {
		return fmt.Errorf("upsert daily bill totals: %w", err)
	}

	return nil
}

func (r *DailyBillRepository) MarkPaid(billID string, paymentID string, paymentMethod string, now time.Time) error {
	query := `UPDATE daily_bills SET
		status = 'paid',
		payment_id = ?,
		payment_method = ?,
		paid_at = ?,
		updated_at = ?
		WHERE id = ?`

	_, err := r.db.Exec(query, paymentID, paymentMethod, now, now, billID)
	if err != nil {
		return fmt.Errorf("mark daily bill paid: %w", err)
	}

	return nil
}

func (r *DailyBillRepository) CountPendingByUserID(userID string) (int, error) {
	return r.CountPendingByUserIDWithExecutor(r.db, userID)
}

func (r *DailyBillRepository) CountPendingByUserIDWithExecutor(exec DBTX, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM daily_bills WHERE user_id = ? AND status = 'pending'`
	var count int
	if err := exec.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending daily bills: %w", err)
	}
	return count, nil
}

func (r *DailyBillRepository) CountPendingBeforeDate(userID string, beforeDate time.Time) (int, error) {
	return r.CountPendingBeforeDateWithExecutor(r.db, userID, beforeDate)
}

func (r *DailyBillRepository) CountPendingBeforeDateWithExecutor(exec DBTX, userID string, beforeDate time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM daily_bills WHERE user_id = ? AND status = 'pending' AND bill_date < ?`
	var count int
	if err := exec.QueryRow(query, userID, beforeDate).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending daily bills before date: %w", err)
	}
	return count, nil
}

func (r *DailyBillRepository) getOne(query string, args ...any) (*models.DailyBill, error) {
	row := r.db.QueryRow(query, args...)
	bill, err := scanDailyBill(row)
	if err != nil {
		return nil, err
	}
	return bill, nil
}

func scanDailyBill(scanner rowScanner) (*models.DailyBill, error) {
	bill := &models.DailyBill{}
	var paymentID, paymentMethod sql.NullString
	var paidAt sql.NullTime

	err := scanner.Scan(
		&bill.ID, &bill.UserID, &bill.BillDate,
		&bill.TotalJourneys, &bill.TotalDistance, &bill.TotalFare,
		&bill.Status, &paymentID, &paymentMethod, &paidAt,
		&bill.CreatedAt, &bill.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if paymentID.Valid {
		bill.PaymentID = &paymentID.String
	}
	if paymentMethod.Valid {
		bill.PaymentMethod = &paymentMethod.String
	}
	if paidAt.Valid {
		t := paidAt.Time
		bill.PaidAt = &t
	}

	return bill, nil
}
