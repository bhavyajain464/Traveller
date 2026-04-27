package repository

import (
	"database/sql"
	"fmt"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type FareTransactionRepository struct {
	db *database.DB
}

func NewFareTransactionRepository(db *database.DB) *FareTransactionRepository {
	return &FareTransactionRepository{db: db}
}

func (r *FareTransactionRepository) CreateWithExecutor(exec DBTX, txModel *models.FareTransaction) error {
	query := `INSERT INTO fare_transactions
		(id, user_id, session_id, segment_id, route_boarding_id, fare_product_id, capping_rule_id, amount, currency_code, rule_version, status, reconciliation_status, payment_reference_id, charged_at, reconciled_at, adjustment_reason, reversed_from_transaction_id, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?)`

	_, err := exec.Exec(query,
		txModel.ID, txModel.UserID, txModel.SessionID, txModel.SegmentID, txModel.RouteBoardingID, txModel.FareProductID,
		txModel.CappingRuleID, txModel.Amount, txModel.CurrencyCode, txModel.RuleVersion, txModel.Status, txModel.ReconciliationStatus, txModel.PaymentReferenceID,
		txModel.ChargedAt, txModel.ReconciledAt, txModel.AdjustmentReason, txModel.ReversedFromTransactionID,
		defaultJSON(txModel.Metadata), txModel.CreatedAt, txModel.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert fare transaction: %w", err)
	}
	return nil
}

func (r *FareTransactionRepository) MarkReconciledByUserAndDate(userID string, billDate time.Time, paymentReferenceID string, now time.Time) error {
	nextDate := billDate.AddDate(0, 0, 1)
	query := `UPDATE fare_transactions
		SET reconciliation_status = 'reconciled',
			payment_reference_id = ?,
			reconciled_at = ?,
			updated_at = ?
		WHERE user_id = ?
			AND charged_at >= ?
			AND charged_at < ?
			AND status IN ('applied', 'pending')
			AND reconciliation_status IN ('unreconciled', 'failed')`

	_, err := r.db.Exec(query, paymentReferenceID, now, now, userID, billDate, nextDate)
	if err != nil {
		return fmt.Errorf("mark fare transactions reconciled: %w", err)
	}
	return nil
}

func (r *FareTransactionRepository) GetByID(transactionID string) (*models.FareTransaction, error) {
	query := `SELECT id, user_id, session_id, segment_id, route_boarding_id, fare_product_id, capping_rule_id, amount, currency_code, rule_version, status, reconciliation_status, payment_reference_id, charged_at, reconciled_at, adjustment_reason, reversed_from_transaction_id, metadata, created_at, updated_at
		FROM fare_transactions WHERE id = ?`

	row := r.db.QueryRow(query, transactionID)
	return scanFareTransaction(row)
}

func (r *FareTransactionRepository) CreateWaiver(exec DBTX, original *models.FareTransaction, reason string, now time.Time) (*models.FareTransaction, error) {
	originalID := original.ID
	reasonCopy := reason
	waiver := &models.FareTransaction{
		ID:                        "waiver_" + original.ID + "_" + now.Format("20060102150405"),
		UserID:                    original.UserID,
		SessionID:                 original.SessionID,
		SegmentID:                 original.SegmentID,
		RouteBoardingID:           original.RouteBoardingID,
		FareProductID:             original.FareProductID,
		CappingRuleID:             original.CappingRuleID,
		Amount:                    0,
		CurrencyCode:              original.CurrencyCode,
		RuleVersion:               original.RuleVersion,
		Status:                    "waived",
		ReconciliationStatus:      "waived",
		ChargedAt:                 now,
		ReconciledAt:              &now,
		AdjustmentReason:          &reasonCopy,
		ReversedFromTransactionID: &originalID,
		Metadata:                  `{"action":"waiver"}`,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if err := r.CreateWithExecutor(exec, waiver); err != nil {
		return nil, err
	}
	return waiver, nil
}

func (r *FareTransactionRepository) CreateReversal(exec DBTX, original *models.FareTransaction, reason string, now time.Time) (*models.FareTransaction, error) {
	originalID := original.ID
	reasonCopy := reason
	reversal := &models.FareTransaction{
		ID:                        "reversal_" + original.ID + "_" + now.Format("20060102150405"),
		UserID:                    original.UserID,
		SessionID:                 original.SessionID,
		SegmentID:                 original.SegmentID,
		RouteBoardingID:           original.RouteBoardingID,
		FareProductID:             original.FareProductID,
		CappingRuleID:             original.CappingRuleID,
		Amount:                    -original.Amount,
		CurrencyCode:              original.CurrencyCode,
		RuleVersion:               original.RuleVersion,
		Status:                    "reversed",
		ReconciliationStatus:      "reversed",
		ChargedAt:                 now,
		ReconciledAt:              &now,
		AdjustmentReason:          &reasonCopy,
		ReversedFromTransactionID: &originalID,
		Metadata:                  `{"action":"reversal"}`,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if err := r.CreateWithExecutor(exec, reversal); err != nil {
		return nil, err
	}
	return reversal, nil
}

func (r *FareTransactionRepository) AggregateBillableAmountByUserAndDate(userID string, billDate time.Time) (float64, error) {
	nextDate := billDate.AddDate(0, 0, 1)

	query := `WITH grouped AS (
		SELECT
			ft.fare_product_id,
			COALESCE(fp.agency_id, '') AS agency_id,
			SUM(ft.amount) AS total_amount
		FROM fare_transactions ft
		LEFT JOIN fare_products fp ON fp.id = ft.fare_product_id
		WHERE ft.user_id = ?
			AND ft.charged_at >= ?
			AND ft.charged_at < ?
			AND ft.status IN ('applied', 'pending', 'reversed')
			AND ft.reconciliation_status IN ('unreconciled', 'reconciled', 'failed', 'reversed')
		GROUP BY ft.fare_product_id, COALESCE(fp.agency_id, '')
	),
	capped AS (
		SELECT
			g.total_amount,
			(
				SELECT MIN(fcr.amount)
				FROM fare_capping_rules fcr
				WHERE fcr.cap_type = 'daily'
					AND (fcr.valid_from IS NULL OR fcr.valid_from <= ?)
					AND (fcr.valid_until IS NULL OR fcr.valid_until >= ?)
					AND (
						(fcr.fare_product_id IS NOT NULL AND fcr.fare_product_id = g.fare_product_id)
						OR (fcr.fare_product_id IS NULL AND fcr.agency_id IS NOT NULL AND fcr.agency_id = NULLIF(g.agency_id, ''))
					)
			) AS cap_amount
		FROM grouped g
	)
	SELECT COALESCE(SUM(
		CASE
			WHEN cap_amount IS NOT NULL AND total_amount > cap_amount THEN cap_amount
			ELSE total_amount
		END
	), 0)
	FROM capped`

	var total float64
	if err := r.db.QueryRow(query, userID, billDate, nextDate, billDate, billDate).Scan(&total); err != nil {
		return 0, fmt.Errorf("aggregate billable fare amount: %w", err)
	}
	return total, nil
}

func scanFareTransaction(scanner interface{ Scan(dest ...any) error }) (*models.FareTransaction, error) {
	txModel := &models.FareTransaction{}
	var sessionID, segmentID, routeBoardingID, fareProductID, cappingRuleID sql.NullString
	var paymentReferenceID, adjustmentReason, reversedFromTransactionID sql.NullString
	var reconciledAt sql.NullTime
	err := scanner.Scan(
		&txModel.ID, &txModel.UserID, &sessionID, &segmentID, &routeBoardingID, &fareProductID, &cappingRuleID,
		&txModel.Amount, &txModel.CurrencyCode, &txModel.RuleVersion, &txModel.Status, &txModel.ReconciliationStatus,
		&paymentReferenceID, &txModel.ChargedAt, &reconciledAt, &adjustmentReason, &reversedFromTransactionID,
		&txModel.Metadata, &txModel.CreatedAt, &txModel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sessionID.Valid {
		txModel.SessionID = &sessionID.String
	}
	if segmentID.Valid {
		txModel.SegmentID = &segmentID.String
	}
	if routeBoardingID.Valid {
		txModel.RouteBoardingID = &routeBoardingID.String
	}
	if fareProductID.Valid {
		txModel.FareProductID = &fareProductID.String
	}
	if cappingRuleID.Valid {
		txModel.CappingRuleID = &cappingRuleID.String
	}
	if paymentReferenceID.Valid {
		txModel.PaymentReferenceID = &paymentReferenceID.String
	}
	if reconciledAt.Valid {
		t := reconciledAt.Time
		txModel.ReconciledAt = &t
	}
	if adjustmentReason.Valid {
		txModel.AdjustmentReason = &adjustmentReason.String
	}
	if reversedFromTransactionID.Valid {
		txModel.ReversedFromTransactionID = &reversedFromTransactionID.String
	}
	return txModel, nil
}
