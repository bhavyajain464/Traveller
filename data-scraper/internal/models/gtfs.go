package models

import "time"

// GTFS Agency
type Agency struct {
	AgencyID       string `csv:"agency_id"`
	AgencyName     string `csv:"agency_name"`
	AgencyURL      string `csv:"agency_url"`
	AgencyTimezone string `csv:"agency_timezone"`
	AgencyLang     string `csv:"agency_lang,omitempty"`
	AgencyPhone    string `csv:"agency_phone,omitempty"`
}

// GTFS Route
type Route struct {
	RouteID        string `csv:"route_id"`
	AgencyID       string `csv:"agency_id"`
	RouteShortName string `csv:"route_short_name"`
	RouteLongName  string `csv:"route_long_name"`
	RouteDesc      string `csv:"route_desc,omitempty"`
	RouteType      int    `csv:"route_type"` // 0=Tram, 1=Subway, 2=Rail, 3=Bus
	RouteURL       string `csv:"route_url,omitempty"`
	RouteColor     string `csv:"route_color,omitempty"`
	RouteTextColor string `csv:"route_text_color,omitempty"`
}

// GTFS Stop
type Stop struct {
	StopID             string  `csv:"stop_id"`
	StopCode           string  `csv:"stop_code,omitempty"`
	StopName           string  `csv:"stop_name"`
	StopDesc           string  `csv:"stop_desc,omitempty"`
	StopLat            float64 `csv:"stop_lat"`
	StopLon            float64 `csv:"stop_lon"`
	ZoneID             string  `csv:"zone_id,omitempty"`
	StopURL            string  `csv:"stop_url,omitempty"`
	LocationType       int     `csv:"location_type,omitempty"` // 0=Stop, 1=Station
	ParentStation      string  `csv:"parent_station,omitempty"`
	WheelchairBoarding int     `csv:"wheelchair_boarding,omitempty"`
}

// GTFS Trip
type Trip struct {
	RouteID              string `csv:"route_id"`
	ServiceID            string `csv:"service_id"`
	TripID               string `csv:"trip_id"`
	TripHeadsign         string `csv:"trip_headsign,omitempty"`
	TripShortName        string `csv:"trip_short_name,omitempty"`
	DirectionID          int    `csv:"direction_id,omitempty"`
	BlockID              string `csv:"block_id,omitempty"`
	ShapeID              string `csv:"shape_id,omitempty"`
	WheelchairAccessible int    `csv:"wheelchair_accessible,omitempty"`
}

// GTFS StopTime
type StopTime struct {
	TripID            string  `csv:"trip_id"`
	ArrivalTime       string  `csv:"arrival_time"`   // HH:MM:SS
	DepartureTime     string  `csv:"departure_time"` // HH:MM:SS
	StopID            string  `csv:"stop_id"`
	StopSequence      int     `csv:"stop_sequence"`
	StopHeadsign      string  `csv:"stop_headsign,omitempty"`
	PickupType        int     `csv:"pickup_type,omitempty"`   // 0=Regular, 1=None
	DropOffType       int     `csv:"drop_off_type,omitempty"` // 0=Regular, 1=None
	ShapeDistTraveled float64 `csv:"shape_dist_traveled,omitempty"`
}

// GTFS Calendar
type Calendar struct {
	ServiceID string `csv:"service_id"`
	Monday    int    `csv:"monday"`     // 0 or 1
	Tuesday   int    `csv:"tuesday"`    // 0 or 1
	Wednesday int    `csv:"wednesday"`  // 0 or 1
	Thursday  int    `csv:"thursday"`   // 0 or 1
	Friday    int    `csv:"friday"`     // 0 or 1
	Saturday  int    `csv:"saturday"`   // 0 or 1
	Sunday    int    `csv:"sunday"`     // 0 or 1
	StartDate string `csv:"start_date"` // YYYYMMDD
	EndDate   string `csv:"end_date"`   // YYYYMMDD
}

// Custom Fare (extension)
type Fare struct {
	FareID        string  `csv:"fare_id"`
	Price         float64 `csv:"price"`
	CurrencyType  string  `csv:"currency_type"`
	PaymentMethod int     `csv:"payment_method"` // 0=On board, 1=Before boarding
	Transfers     int     `csv:"transfers,omitempty"`
	AgencyID      string  `csv:"agency_id,omitempty"`
}

// BMRCL Station
type BMRCLStation struct {
	StationID   string
	StationName string
	Line        string // "Green", "Purple", etc.
	Latitude    float64
	Longitude   float64
	StationCode string
	Order       int // Order on the line
}

// BMRCL Route
type BMRCLRoute struct {
	RouteID     string
	RouteName   string
	FromStation string
	ToStation   string
	Line        string
	Stations    []BMRCLStation
}

// IRCTC Station
type IRCTCStation struct {
	StationCode string
	StationName string
	Latitude    float64
	Longitude   float64
	Zone        string // Railway zone
}

// IRCTC Train
type IRCTCTrain struct {
	TrainNumber string
	TrainName   string
	FromStation string
	ToStation   string
	Stations    []IRCTCStation
	Days        []time.Weekday
}

