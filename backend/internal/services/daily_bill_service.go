package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/repository"
)

type DailyBillService struct {
	billRepo            *repository.DailyBillRepository
	sessionRepo         *repository.JourneySessionRepository
	fareTransactionRepo *repository.FareTransactionRepository
}

func NewDailyBillService(billRepo *repository.DailyBillRepository, sessionRepo *repository.JourneySessionRepository, fareTransactionRepo *repository.FareTransactionRepository) *DailyBillService {
	return &DailyBillService{
		billRepo:            billRepo,
		sessionRepo:         sessionRepo,
		fareTransactionRepo: fareTransactionRepo,
	}
}

// GetDailyBill retrieves or creates daily bill for a user and date
func (s *DailyBillService) GetDailyBill(userID string, billDate time.Time) (*models.DailyBill, error) {
	date := time.Date(billDate.Year(), billDate.Month(), billDate.Day(), 0, 0, 0, 0, billDate.Location())

	if err := s.syncDailyBill(userID, date); err != nil {
		return nil, err
	}

	bill, err := s.billRepo.GetByUserAndDate(userID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily bill: %w", err)
	}

	journeys, err := s.sessionRepo.ListCompletedByUserAndRange(userID, date, date.AddDate(0, 0, 1))
	if err == nil {
		bill.Journeys = journeys
	}

	return bill, nil
}

// GetPendingBills returns all pending bills for a user
func (s *DailyBillService) GetPendingBills(userID string) ([]models.DailyBill, error) {
	bills, err := s.billRepo.ListPendingByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending bills: %w", err)
	}

	return bills, nil
}

// GetBillByID retrieves a bill by ID.
func (s *DailyBillService) GetBillByID(billID string) (*models.DailyBill, error) {
	bill, err := s.billRepo.GetByID(billID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("bill not found")
		}
		return nil, fmt.Errorf("failed to get bill: %w", err)
	}
	return bill, nil
}

// MarkBillAsPaid marks a bill as paid
func (s *DailyBillService) MarkBillAsPaid(billID string, paymentID string, paymentMethod string) error {
	bill, err := s.billRepo.GetByID(billID)
	if err != nil {
		return fmt.Errorf("failed to load bill before payment reconciliation: %w", err)
	}

	now := time.Now()
	if err := s.billRepo.MarkPaid(billID, paymentID, paymentMethod, now); err != nil {
		return fmt.Errorf("failed to mark bill as paid: %w", err)
	}
	if err := s.fareTransactionRepo.MarkReconciledByUserAndDate(bill.UserID, bill.BillDate, paymentID, now); err != nil {
		return fmt.Errorf("failed to reconcile fare transactions: %w", err)
	}

	return nil
}

func (s *DailyBillService) WaiveFareTransaction(transactionID string, reason string) (*models.FareTransaction, error) {
	txModel, err := s.fareTransactionRepo.GetByID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fare transaction: %w", err)
	}

	tx, err := s.billRepoDB().Begin()
	if err != nil {
		return nil, fmt.Errorf("begin fare waiver transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	waiver, err := s.fareTransactionRepo.CreateWaiver(tx, txModel, reason, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit fare waiver transaction: %w", err)
	}
	return waiver, nil
}

func (s *DailyBillService) ReverseFareTransaction(transactionID string, reason string) (*models.FareTransaction, error) {
	txModel, err := s.fareTransactionRepo.GetByID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fare transaction: %w", err)
	}

	tx, err := s.billRepoDB().Begin()
	if err != nil {
		return nil, fmt.Errorf("begin fare reversal transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	reversal, err := s.fareTransactionRepo.CreateReversal(tx, txModel, reason, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit fare reversal transaction: %w", err)
	}
	return reversal, nil
}

// GenerateDailyBills generates bills for all users for a specific date
func (s *DailyBillService) GenerateDailyBills(billDate time.Time) error {
	date := time.Date(billDate.Year(), billDate.Month(), billDate.Day(), 0, 0, 0, 0, billDate.Location())
	nextDate := date.AddDate(0, 0, 1)

	userIDs, err := s.sessionRepo.ListDistinctCompletedUserIDsInRange(date, nextDate)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, userID := range userIDs {
		if err := s.syncDailyBill(userID, date); err != nil {
			fmt.Printf("Warning: Failed to generate bill for user %s: %v\n", userID, err)
		}
	}

	return nil
}

// Helper functions
func (s *DailyBillService) createDailyBill(userID string, billDate time.Time) (*models.DailyBill, error) {
	nextDate := billDate.AddDate(0, 0, 1)

	totalJourneys, totalDistance, totalFare, err := s.sessionRepo.AggregateCompletedByUserAndRange(userID, billDate, nextDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate bill totals: %w", err)
	}

	billID := uuid.New().String()
	now := time.Now()

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

	if err := s.billRepo.Create(bill); err != nil {
		return nil, fmt.Errorf("failed to create daily bill: %w", err)
	}

	journeys, err := s.sessionRepo.ListCompletedByUserAndRange(userID, billDate, nextDate)
	if err == nil {
		bill.Journeys = journeys
	}

	return bill, nil
}

func (s *DailyBillService) getJourneysForBill(userID string, billDate time.Time) ([]models.JourneySession, error) {
	return s.sessionRepo.ListCompletedByUserAndRange(userID, billDate, billDate.AddDate(0, 0, 1))
}

func (s *DailyBillService) syncDailyBill(userID string, billDate time.Time) error {
	nextDate := billDate.AddDate(0, 0, 1)

	totalJourneys, totalDistance, _, err := s.sessionRepo.AggregateCompletedByUserAndRange(userID, billDate, nextDate)
	if err != nil {
		return fmt.Errorf("failed to calculate bill journey totals: %w", err)
	}

	totalFare, err := s.fareTransactionRepo.AggregateBillableAmountByUserAndDate(userID, billDate)
	if err != nil {
		return fmt.Errorf("failed to calculate capped fare total: %w", err)
	}

	now := time.Now()
	billID := uuid.New().String()
	if err := s.billRepo.UpsertPendingTotals(billID, userID, billDate, totalJourneys, totalDistance, totalFare, now); err != nil {
		return fmt.Errorf("failed to upsert daily bill totals: %w", err)
	}

	return nil
}

func (s *DailyBillService) billRepoDB() *database.DB {
	return s.billRepo.DB()
}
