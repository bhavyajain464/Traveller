package models

type Stop struct {
	ID                 string  `json:"id" db:"stop_id"`
	Code               string  `json:"code" db:"stop_code"`
	Name               string  `json:"name" db:"stop_name"`
	Description        string  `json:"description" db:"stop_desc"`
	Latitude           float64 `json:"latitude" db:"stop_lat"`
	Longitude          float64 `json:"longitude" db:"stop_lon"`
	ZoneID             string  `json:"zone_id" db:"zone_id"`
	URL                string  `json:"url" db:"stop_url"`
	LocationType       int     `json:"location_type" db:"location_type"`
	ParentStation      string  `json:"parent_station" db:"parent_station"`
	Timezone           string  `json:"timezone" db:"stop_timezone"`
	WheelchairBoarding int     `json:"wheelchair_boarding" db:"wheelchair_boarding"`
}
