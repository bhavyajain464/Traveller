package models

type Trip struct {
	ID                   string `json:"id" db:"trip_id"`
	RouteID              string `json:"route_id" db:"route_id"`
	ServiceID            string `json:"service_id" db:"service_id"`
	Headsign             string `json:"headsign" db:"trip_headsign"`
	ShortName            string `json:"short_name" db:"trip_short_name"`
	DirectionID          *int   `json:"direction_id" db:"direction_id"`
	BlockID              string `json:"block_id" db:"block_id"`
	ShapeID              string `json:"shape_id" db:"shape_id"`
	WheelchairAccessible *int   `json:"wheelchair_accessible" db:"wheelchair_accessible"`
	BikesAllowed         *int   `json:"bikes_allowed" db:"bikes_allowed"`
}
