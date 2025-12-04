package models

import "time"

// RouteBoarding represents a single route boarding event during a journey session
type RouteBoarding struct {
	ID              string     `json:"id" db:"id"`
	SessionID       string     `json:"session_id" db:"session_id"`
	RouteID         string     `json:"route_id" db:"route_id"`
	BoardingStopID  string     `json:"boarding_stop_id" db:"boarding_stop_id"`
	AlightingStopID *string    `json:"alighting_stop_id,omitempty" db:"alighting_stop_id"`
	BoardingTime    time.Time  `json:"boarding_time" db:"boarding_time"`
	AlightingTime   *time.Time `json:"alighting_time,omitempty" db:"alighting_time"`
	BoardingLat     float64    `json:"boarding_lat" db:"boarding_lat"`
	BoardingLon     float64    `json:"boarding_lon" db:"boarding_lon"`
	AlightingLat    *float64   `json:"alighting_lat,omitempty" db:"alighting_lat"`
	AlightingLon    *float64   `json:"alighting_lon,omitempty" db:"alighting_lon"`
	Distance        float64    `json:"distance" db:"distance"` // Distance traveled on this route in km
	Fare            float64    `json:"fare" db:"fare"`         // Fare for this route segment
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// BoardRouteRequest represents a request to board a route
type BoardRouteRequest struct {
	SessionID      string  `json:"session_id"`
	QRCode         string  `json:"qr_code"`
	RouteID        string  `json:"route_id" binding:"required"`
	BoardingStopID *string `json:"boarding_stop_id,omitempty"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
}

// AlightRouteRequest represents a request to alight from a route
type AlightRouteRequest struct {
	BoardingID      string  `json:"boarding_id" binding:"required"`
	AlightingStopID *string `json:"alighting_stop_id,omitempty"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}
