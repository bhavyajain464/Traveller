package models

// TrackingHeartbeatResponse represents the rider-facing tracking state for a ticket/session.
type TrackingHeartbeatResponse struct {
	SessionID      string          `json:"session_id"`
	OnVehicle      bool            `json:"on_vehicle"`
	AutoBoarded    bool            `json:"auto_boarded"`
	Alighted       bool            `json:"alighted"`
	TrackingState  string          `json:"tracking_state"`
	Message        string          `json:"message"`
	VehicleID      *string         `json:"vehicle_id,omitempty"`
	RouteID        *string         `json:"route_id,omitempty"`
	DistanceMeters float64         `json:"distance_meters"`
	ActiveBoarding *RouteBoarding  `json:"active_boarding,omitempty"`
	Boardings      []RouteBoarding `json:"boardings"`
}
