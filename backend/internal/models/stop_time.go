package models

type StopTime struct {
	TripID            string   `json:"trip_id" db:"trip_id"`
	ArrivalTime       string   `json:"arrival_time" db:"arrival_time"`
	DepartureTime     string   `json:"departure_time" db:"departure_time"`
	StopID            string   `json:"stop_id" db:"stop_id"`
	StopSequence      int      `json:"stop_sequence" db:"stop_sequence"`
	StopHeadsign      string   `json:"stop_headsign" db:"stop_headsign"`
	PickupType        *int     `json:"pickup_type" db:"pickup_type"`
	DropOffType       *int     `json:"drop_off_type" db:"drop_off_type"`
	ShapeDistTraveled *float64 `json:"shape_dist_traveled" db:"shape_dist_traveled"`
	Timepoint         *int     `json:"timepoint" db:"timepoint"`
}
