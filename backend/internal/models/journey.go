package models

import "time"

type JourneyRequest struct {
	FromLat       float64    `json:"from_lat"`
	FromLon       float64    `json:"from_lon"`
	ToLat         float64    `json:"to_lat"`
	ToLon         float64    `json:"to_lon"`
	DepartureTime *time.Time `json:"departure_time,omitempty"`
	ArrivalTime   *time.Time `json:"arrival_time,omitempty"`
	Date          *time.Time `json:"date,omitempty"` // Date for the journey (for day-specific service plans)
}

type JourneyOption struct {
	Duration      int          `json:"duration"` // in minutes
	Transfers     int          `json:"transfers"`
	WalkingTime   int          `json:"walking_time"` // in minutes
	Legs          []JourneyLeg `json:"legs"`
	DepartureTime time.Time    `json:"departure_time"`
	ArrivalTime   time.Time    `json:"arrival_time"`
	Fare          *float64     `json:"fare,omitempty"` // Fare in INR
}

type JourneyLeg struct {
	Mode          string    `json:"mode"` // "walking", "bus", "metro", etc.
	RouteID       string    `json:"route_id,omitempty"`
	RouteName     string    `json:"route_name,omitempty"`
	FromStopID    string    `json:"from_stop_id"`
	FromStopName  string    `json:"from_stop_name"`
	ToStopID      string    `json:"to_stop_id"`
	ToStopName    string    `json:"to_stop_name"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	Duration      int       `json:"duration"` // in minutes
	StopCount     int       `json:"stop_count"`
}
