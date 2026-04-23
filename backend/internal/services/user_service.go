package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type UserService struct {
	db *database.DB
}

type GoogleProfile struct {
	Subject  string
	Email    string
	Name     string
	Picture  string
	Verified bool
}

func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user.
func (s *UserService) CreateUser(phoneNumber, name string, email *string) (*models.User, error) {
	trimmedPhone := strings.TrimSpace(phoneNumber)
	trimmedName := strings.TrimSpace(name)
	if trimmedPhone == "" {
		return nil, fmt.Errorf("phone number is required")
	}
	if trimmedName == "" {
		return nil, fmt.Errorf("name is required")
	}

	var existingID string
	err := s.db.QueryRow("SELECT id FROM users WHERE phone_number = ?", trimmedPhone).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user with phone number %s already exists", trimmedPhone)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	userID := uuid.New().String()
	now := time.Now().UTC()
	user := &models.User{
		ID:             userID,
		PhoneNumber:    stringPtr(trimmedPhone),
		Email:          normalizeOptionalString(email),
		Name:           trimmedName,
		AuthProvider:   "phone",
		Status:         "active",
		AutoPayEnabled: false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	query := `INSERT INTO users (
			id, phone_number, email, name, auth_provider, status, auto_pay_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(
		query,
		user.ID,
		user.PhoneNumber,
		user.Email,
		user.Name,
		user.AuthProvider,
		user.Status,
		user.AutoPayEnabled,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpsertGoogleUser creates or updates a user authenticated via Google.
func (s *UserService) UpsertGoogleUser(profile GoogleProfile) (*models.User, error) {
	subject := strings.TrimSpace(profile.Subject)
	name := strings.TrimSpace(profile.Name)
	email := strings.TrimSpace(strings.ToLower(profile.Email))
	picture := strings.TrimSpace(profile.Picture)
	if subject == "" {
		return nil, fmt.Errorf("google subject is required")
	}
	if !profile.Verified {
		return nil, fmt.Errorf("google account email is not verified")
	}
	if email == "" {
		return nil, fmt.Errorf("google account email is required")
	}
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	now := time.Now().UTC()
	existing, err := s.GetUserByGoogleSub(subject)
	if err != nil && err.Error() != "user not found" {
		return nil, err
	}

	emailPtr := stringPtr(email)
	picturePtr := optionalStringPtr(picture)

	if existing != nil {
		existing.Name = name
		existing.Email = emailPtr
		existing.AvatarURL = picturePtr
		existing.GoogleSub = stringPtr(subject)
		existing.AuthProvider = "google"
		existing.LastLoginAt = &now
		existing.UpdatedAt = now

		query := `UPDATE users
			SET email = ?, name = ?, avatar_url = ?, google_sub = ?, auth_provider = ?, last_login_at = ?, updated_at = ?
			WHERE id = ?`
		if _, err := s.db.Exec(query, existing.Email, existing.Name, existing.AvatarURL, existing.GoogleSub, existing.AuthProvider, existing.LastLoginAt, existing.UpdatedAt, existing.ID); err != nil {
			return nil, fmt.Errorf("failed to update google user: %w", err)
		}

		return existing, nil
	}

	user := &models.User{
		ID:             uuid.New().String(),
		Email:          emailPtr,
		Name:           name,
		AvatarURL:      picturePtr,
		GoogleSub:      stringPtr(subject),
		AuthProvider:   "google",
		Status:         "active",
		AutoPayEnabled: false,
		LastLoginAt:    &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	query := `INSERT INTO users (
			id, phone_number, email, name, avatar_url, google_sub, auth_provider, status, auto_pay_enabled, last_login_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(
		query,
		user.ID,
		user.PhoneNumber,
		user.Email,
		user.Name,
		user.AvatarURL,
		user.GoogleSub,
		user.AuthProvider,
		user.Status,
		user.AutoPayEnabled,
		user.LastLoginAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			byEmail, lookupErr := s.GetUserByEmail(email)
			if lookupErr == nil {
				byEmail.GoogleSub = stringPtr(subject)
				byEmail.AvatarURL = picturePtr
				byEmail.AuthProvider = "google"
				byEmail.LastLoginAt = &now
				byEmail.UpdatedAt = now
				updateQuery := `UPDATE users
					SET google_sub = ?, avatar_url = ?, auth_provider = ?, last_login_at = ?, updated_at = ?
					WHERE id = ?`
				if _, updateErr := s.db.Exec(updateQuery, byEmail.GoogleSub, byEmail.AvatarURL, byEmail.AuthProvider, byEmail.LastLoginAt, byEmail.UpdatedAt, byEmail.ID); updateErr == nil {
					return byEmail, nil
				}
			}
		}
		return nil, fmt.Errorf("failed to create google user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	return s.getUser("WHERE id = ?", userID)
}

// GetUserByPhone retrieves a user by phone number.
func (s *UserService) GetUserByPhone(phoneNumber string) (*models.User, error) {
	return s.getUser("WHERE phone_number = ?", phoneNumber)
}

// GetUserByGoogleSub retrieves a user by Google subject.
func (s *UserService) GetUserByGoogleSub(googleSub string) (*models.User, error) {
	return s.getUser("WHERE google_sub = ?", googleSub)
}

// GetUserByEmail retrieves a user by email address.
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	return s.getUser("WHERE LOWER(email) = LOWER(?)", email)
}

// DeleteUserByPhone deletes a user by phone number (for testing/cleanup).
func (s *UserService) DeleteUserByPhone(phoneNumber string) error {
	query := `DELETE FROM users WHERE phone_number = ?`
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

func (s *UserService) getUser(whereClause string, args ...any) (*models.User, error) {
	query := `SELECT id, phone_number, email, name, avatar_url, google_sub, auth_provider, status, payment_method, auto_pay_enabled, last_login_at, created_at, updated_at
		FROM users ` + whereClause

	user := &models.User{}
	var phoneNumber sql.NullString
	var email sql.NullString
	var avatarURL sql.NullString
	var googleSub sql.NullString
	var authProvider sql.NullString
	var paymentMethod sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, args...).Scan(
		&user.ID,
		&phoneNumber,
		&email,
		&user.Name,
		&avatarURL,
		&googleSub,
		&authProvider,
		&user.Status,
		&paymentMethod,
		&user.AutoPayEnabled,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if phoneNumber.Valid {
		user.PhoneNumber = &phoneNumber.String
	}
	if email.Valid {
		user.Email = &email.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if googleSub.Valid {
		user.GoogleSub = &googleSub.String
	}
	if authProvider.Valid {
		user.AuthProvider = authProvider.String
	}
	if paymentMethod.Valid {
		user.PaymentMethod = &paymentMethod.String
	}
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		user.LastLoginAt = &t
	}

	return user, nil
}

func stringPtr(value string) *string {
	return &value
}

func optionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	return optionalStringPtr(*value)
}
