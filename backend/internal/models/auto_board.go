package models

// AutoBoardRequest represents a request to automatically detect and board a vehicle
type AutoBoardRequest struct {
	SessionID string  `json:"session_id" binding:"required"`
	QRCode    string  `json:"qr_code"` // Alternative to session_id
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

