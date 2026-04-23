package models

import "time"

// User represents a user account
type User struct {
	ID             string     `json:"id" db:"id"`
	PhoneNumber    *string    `json:"phone_number,omitempty" db:"phone_number"`
	Email          *string    `json:"email,omitempty" db:"email"`
	Name           string     `json:"name" db:"name"`
	AvatarURL      *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	GoogleSub      *string    `json:"google_sub,omitempty" db:"google_sub"`
	AuthProvider   string     `json:"auth_provider" db:"auth_provider"`
	Status         string     `json:"status" db:"status"`                           // "active", "suspended", "inactive"
	PaymentMethod  *string    `json:"payment_method,omitempty" db:"payment_method"` // "upi", "card", "wallet"
	AutoPayEnabled bool       `json:"auto_pay_enabled" db:"auto_pay_enabled"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// CheckInRequest represents a check-in request
type CheckInRequest struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	StopID    *string `json:"stop_id,omitempty"` // Optional, can be inferred from location
}

// CheckOutRequest represents a check-out request
type CheckOutRequest struct {
	SessionID string  `json:"session_id"`
	QRCode    string  `json:"qr_code"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	StopID    *string `json:"stop_id,omitempty"`
}

// QRValidationRequest represents a QR code validation request (by conductor/validator)
type QRValidationRequest struct {
	QRCode  string `json:"qr_code"`
	RouteID string `json:"route_id"` // Route being validated
}
