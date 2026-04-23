package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type AuthService struct {
	db                 *database.DB
	userService        *UserService
	googleClientID     string
	sessionTokenSecret string
	sessionDuration    time.Duration
	httpClient         *http.Client
}

type AuthSession struct {
	ID        string       `json:"id"`
	Token     string       `json:"token,omitempty"`
	Provider  string       `json:"provider"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

type googleTokenInfoResponse struct {
	AZP           string `json:"azp"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Exp           string `json:"exp"`
	Iss           string `json:"iss"`
}

func NewAuthService(db *database.DB, userService *UserService, cfg config.AuthConfig) *AuthService {
	duration := time.Duration(cfg.SessionDurationHours) * time.Hour
	if duration <= 0 {
		duration = 30 * 24 * time.Hour
	}

	return &AuthService{
		db:                 db,
		userService:        userService,
		googleClientID:     strings.TrimSpace(cfg.GoogleClientID),
		sessionTokenSecret: cfg.SessionTokenSecret,
		sessionDuration:    duration,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (*AuthSession, error) {
	profile, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userService.UpsertGoogleUser(*profile)
	if err != nil {
		return nil, err
	}

	return s.createSession(user, "google")
}

func (s *AuthService) GetSessionByToken(token string) (*AuthSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("missing auth token")
	}

	query := `SELECT s.id, s.provider, s.expires_at, u.id, u.phone_number, u.email, u.name, u.avatar_url, u.google_sub, u.auth_provider, u.status, u.payment_method, u.auto_pay_enabled, u.last_login_at, u.created_at, u.updated_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > NOW()`

	session := &AuthSession{}
	user := &models.User{}

	var phoneNumber sql.NullString
	var email sql.NullString
	var avatarURL sql.NullString
	var googleSub sql.NullString
	var authProvider sql.NullString
	var paymentMethod sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, s.hashToken(token)).Scan(
		&session.ID,
		&session.Provider,
		&session.ExpiresAt,
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
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
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

	session.User = user
	return session, nil
}

func (s *AuthService) Logout(token string) error {
	query := `UPDATE auth_sessions SET revoked_at = NOW(), updated_at = NOW() WHERE token_hash = ? AND revoked_at IS NULL`
	if _, err := s.db.Exec(query, s.hashToken(token)); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

func (s *AuthService) createSession(user *models.User, provider string) (*AuthSession, error) {
	rawToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &AuthSession{
		ID:        uuid.New().String(),
		Token:     rawToken,
		Provider:  provider,
		ExpiresAt: time.Now().UTC().Add(s.sessionDuration),
		User:      user,
	}

	query := `INSERT INTO auth_sessions (id, user_id, token_hash, provider, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	if _, err := s.db.Exec(query, session.ID, user.ID, s.hashToken(rawToken), provider, session.ExpiresAt, now, now); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (s *AuthService) verifyGoogleIDToken(ctx context.Context, idToken string) (*GoogleProfile, error) {
	if s.googleClientID == "" {
		return nil, fmt.Errorf("google sign-in is not configured on the backend")
	}

	token := strings.TrimSpace(idToken)
	if token == "" {
		return nil, fmt.Errorf("google credential is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://oauth2.googleapis.com/tokeninfo?id_token="+token, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build google verification request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify google token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read google verification response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token rejected")
	}

	var payload googleTokenInfoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode google verification response: %w", err)
	}

	if payload.Aud != s.googleClientID {
		return nil, fmt.Errorf("google token audience mismatch")
	}

	if payload.EmailVerified != "true" {
		return nil, fmt.Errorf("google account email is not verified")
	}

	if payload.Iss != "https://accounts.google.com" && payload.Iss != "accounts.google.com" {
		return nil, fmt.Errorf("google token issuer mismatch")
	}

	if payload.Sub == "" || payload.Email == "" {
		return nil, fmt.Errorf("google token missing user identity")
	}

	return &GoogleProfile{
		Subject:  payload.Sub,
		Email:    payload.Email,
		Name:     payload.Name,
		Picture:  payload.Picture,
		Verified: payload.EmailVerified == "true",
	}, nil
}

func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *AuthService) hashToken(token string) string {
	key := s.sessionTokenSecret
	if strings.TrimSpace(key) == "" {
		key = "traveller-dev-session-secret"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(token))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum[:])
}
