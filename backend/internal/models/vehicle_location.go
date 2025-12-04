package models

import "time"

// VehicleLocation represents the current location of a transport vehicle
type VehicleLocation struct {
	VehicleID    string    `json:"vehicle_id" db:"vehicle_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	TripID       *string   `json:"trip_id,omitempty" db:"trip_id"`
	Latitude     float64   `json:"latitude" db:"latitude"`
	Longitude    float64   `json:"longitude" db:"longitude"`
	Bearing      *float64  `json:"bearing,omitempty" db:"bearing"` // Direction in degrees (0-360)
	Speed        *float64  `json:"speed,omitempty" db:"speed"`     // Speed in km/h
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	StopSequence *int      `json:"stop_sequence,omitempty" db:"stop_sequence"`
}

// VehicleLocationMatch represents a match between user location and vehicle location
type VehicleLocationMatch struct {
	VehicleLocation VehicleLocation `json:"vehicle_location"`
	RouteID         string          `json:"route_id"`
	RouteName       string          `json:"route_name"`
	RouteType       int             `json:"route_type"` // 1=Metro, 3=Bus, etc.
	AgencyID        string          `json:"agency_id"`
	Distance        float64         `json:"distance"` // Distance in meters
	Confidence      float64         `json:"confidence"` // 0-1, how confident we are user is on this vehicle
}

// MockVehicleSetupRequest represents a request to set up mock vehicles
type MockVehicleSetupRequest struct {
	RouteID              string  `json:"route_id" binding:"required"`
	Latitude             float64 `json:"latitude" binding:"required"`
	Longitude            float64 `json:"longitude" binding:"required"`
	TripID               *string `json:"trip_id,omitempty"` // Optional: if provided, vehicle follows schedule
	StartMovingImmediately bool  `json:"start_moving_immediately,omitempty"` // If true, vehicle starts moving right away
}

