package models

import "time"

type JourneySegment struct {
	ID              string     `json:"id" db:"id"`
	SessionID       string     `json:"session_id" db:"session_id"`
	RouteBoardingID *string    `json:"route_boarding_id,omitempty" db:"route_boarding_id"`
	SegmentIndex    int        `json:"segment_index" db:"segment_index"`
	RouteID         *string    `json:"route_id,omitempty" db:"route_id"`
	VehicleID       *string    `json:"vehicle_id,omitempty" db:"vehicle_id"`
	FromStopID      *string    `json:"from_stop_id,omitempty" db:"from_stop_id"`
	ToStopID        *string    `json:"to_stop_id,omitempty" db:"to_stop_id"`
	BoardedAt       time.Time  `json:"boarded_at" db:"boarded_at"`
	AlightedAt      *time.Time `json:"alighted_at,omitempty" db:"alighted_at"`
	DistanceKM      float64    `json:"distance_km" db:"distance_km"`
	FareAmount      float64    `json:"fare_amount" db:"fare_amount"`
	Metadata        string     `json:"metadata" db:"metadata"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type JourneyEvent struct {
	ID              string    `json:"id" db:"id"`
	SessionID       string    `json:"session_id" db:"session_id"`
	RouteBoardingID *string   `json:"route_boarding_id,omitempty" db:"route_boarding_id"`
	SegmentID       *string   `json:"segment_id,omitempty" db:"segment_id"`
	EventType       string    `json:"event_type" db:"event_type"`
	StopID          *string   `json:"stop_id,omitempty" db:"stop_id"`
	Latitude        *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude       *float64  `json:"longitude,omitempty" db:"longitude"`
	OccurredAt      time.Time `json:"occurred_at" db:"occurred_at"`
	Metadata        string    `json:"metadata" db:"metadata"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
