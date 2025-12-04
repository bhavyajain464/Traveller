package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type UserService struct {
	db *database.DB
}

func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(phoneNumber, name string, email *string) (*models.User, error) {
	// Check if user already exists
	var existingID string
	err := s.db.QueryRow("SELECT id FROM users WHERE phone_number = $1", phoneNumber).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user with phone number %s already exists", phoneNumber)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Create new user
	userID := uuid.New().String()
	now := time.Now()
	user := &models.User{
		ID:             userID,
		PhoneNumber:    phoneNumber,
		Email:          email,
		Name:           name,
		Status:         "active",
		AutoPayEnabled: false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	query := `INSERT INTO users (id, phone_number, email, name, status, auto_pay_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.db.Exec(query,
		user.ID, user.PhoneNumber, user.Email, user.Name,
		user.Status, user.AutoPayEnabled, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	query := `SELECT id, phone_number, email, name, status, payment_method, auto_pay_enabled, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	var email sql.NullString
	var paymentMethod sql.NullString

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.PhoneNumber, &email,
		&user.Name, &user.Status, &paymentMethod,
		&user.AutoPayEnabled, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		user.Email = &email.String
	}
	if paymentMethod.Valid {
		user.PaymentMethod = &paymentMethod.String
	}

	return user, nil
}

// GetUserByPhone retrieves a user by phone number
func (s *UserService) GetUserByPhone(phoneNumber string) (*models.User, error) {
	query := `SELECT id, phone_number, email, name, status, payment_method, auto_pay_enabled, created_at, updated_at
		FROM users WHERE phone_number = $1`

	user := &models.User{}
	var email sql.NullString
	var paymentMethod sql.NullString

	err := s.db.QueryRow(query, phoneNumber).Scan(
		&user.ID, &user.PhoneNumber, &email,
		&user.Name, &user.Status, &paymentMethod,
		&user.AutoPayEnabled, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		user.Email = &email.String
	}
	if paymentMethod.Valid {
		user.PaymentMethod = &paymentMethod.String
	}

	return user, nil
}


// DeleteUserByPhone deletes a user by phone number (for testing/cleanup)
func (s *UserService) DeleteUserByPhone(phoneNumber string) error {
	query := `DELETE FROM users WHERE phone_number = $1`
	result, err := s.db.Exec(query, phoneNumber)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}
