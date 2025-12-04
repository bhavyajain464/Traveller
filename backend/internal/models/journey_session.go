package models

import "time"

// JourneySession represents an active journey session (check-in to check-out)
type JourneySession struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	QRCode          string    `json:"qr_code" db:"qr_code"`
	CheckInTime     time.Time `json:"check_in_time" db:"check_in_time"`
	CheckOutTime    *time.Time `json:"check_out_time,omitempty" db:"check_out_time"`
	CheckInStopID   string    `json:"check_in_stop_id" db:"check_in_stop_id"`
	CheckOutStopID  *string   `json:"check_out_stop_id,omitempty" db:"check_out_stop_id"`
	CheckInLat      float64   `json:"check_in_lat" db:"check_in_lat"`
	CheckInLon      float64   `json:"check_in_lon" db:"check_in_lon"`
	CheckOutLat     *float64  `json:"check_out_lat,omitempty" db:"check_out_lat"`
	CheckOutLon     *float64  `json:"check_out_lon,omitempty" db:"check_out_lon"`
	Status          string    `json:"status" db:"status"` // "active", "completed", "cancelled"
	RoutesUsed      []string  `json:"routes_used" db:"routes_used"` // JSON array of route IDs
	TotalDistance   float64   `json:"total_distance" db:"total_distance"` // in km
	TotalFare       float64   `json:"total_fare" db:"total_fare"` // in INR
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// DailyBill represents the aggregated bill for a user for a day
type DailyBill struct {
	ID              string           `json:"id" db:"id"`
	UserID          string           `json:"user_id" db:"user_id"`
	BillDate        time.Time        `json:"bill_date" db:"bill_date"`
	TotalJourneys   int              `json:"total_journeys" db:"total_journeys"`
	TotalDistance   float64           `json:"total_distance" db:"total_distance"`
	TotalFare       float64           `json:"total_fare" db:"total_fare"`
	Status          string           `json:"status" db:"status"` // "pending", "paid", "failed"
	PaymentID       *string          `json:"payment_id,omitempty" db:"payment_id"`
	PaymentMethod   *string          `json:"payment_method,omitempty" db:"payment_method"`
	PaidAt          *time.Time       `json:"paid_at,omitempty" db:"paid_at"`
	Journeys        []JourneySession `json:"journeys,omitempty"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// QRCodeTicket represents a QR code ticket for validation
type QRCodeTicket struct {
	Code            string    `json:"code"`
	UserID          string    `json:"user_id"`
	SessionID       string    `json:"session_id"`
	CheckInTime     time.Time `json:"check_in_time"`
	CheckInStopID   string    `json:"check_in_stop_id"`
	ExpiresAt       time.Time `json:"expires_at"`
	IsValid         bool      `json:"is_valid"`
}


