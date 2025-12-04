package models

// ContinuousLocationUpdate represents a continuous location update from user's device
type ContinuousLocationUpdate struct {
	SessionID string  `json:"session_id" binding:"required"`
	QRCode    string  `json:"qr_code"` // Alternative to session_id
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Timestamp *string `json:"timestamp,omitempty"` // Optional timestamp from device
}

// ContinuousLocationResponse represents the response to continuous location updates
type ContinuousLocationResponse struct {
	OnVehicle      bool    `json:"on_vehicle"`
	VehicleID      *string `json:"vehicle_id,omitempty"`
	RouteID        *string `json:"route_id,omitempty"`
	DistanceMeters float64 `json:"distance_meters"`
	Alighted       bool    `json:"alighted"`
	AlightingStopID *string `json:"alighting_stop_id,omitempty"`
	Message        string  `json:"message"`
}

