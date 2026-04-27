package models

type V3Location struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Score       float64        `json:"score,omitempty"`
	Coordinates *V3Coordinates `json:"coordinates,omitempty"`
}

type V3Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type V3LocationsResponse struct {
	Locations []V3Location `json:"locations"`
	Count     int          `json:"count"`
	Meta      V3LocationsMeta `json:"meta"`
}

type V3ConnectionResponse struct {
	Connections []V3Connection `json:"connections"`
	Count       int            `json:"count"`
	Meta        V3JourneyMeta  `json:"meta"`
}

type V3StationboardResponse struct {
	Station V3StationRef         `json:"station"`
	Entries []V3StationboardEntry `json:"entries"`
	Count   int                   `json:"count"`
	Meta    V3StationboardMeta    `json:"meta"`
}

type V3StationboardMeta struct {
	Limit int    `json:"limit"`
	Time  string `json:"time,omitempty"`
}

type V3StationboardEntry struct {
	TripID    string             `json:"trip_id,omitempty"`
	Stop      *V3StationStopInfo `json:"stop,omitempty"`
	Journey   V3JourneySection   `json:"journey"`
	Realtime  *V3RealtimeSection `json:"realtime,omitempty"`
	Departure string             `json:"departure,omitempty"`
	Arrival   string             `json:"arrival,omitempty"`
}

type V3StationStopInfo struct {
	Departure string `json:"departure,omitempty"`
	Arrival   string `json:"arrival,omitempty"`
}

type V3LocationsMeta struct {
	Provider string `json:"provider,omitempty"`
	Query    string `json:"query,omitempty"`
}

type V3JourneyMeta struct {
	Engine              string               `json:"engine,omitempty"`
	ServiceDate         string               `json:"service_date,omitempty"`
	RequestedDate       string               `json:"requested_date,omitempty"`
	FallbackServiceDate bool                 `json:"fallback_service_date,omitempty"`
	Request             V3JourneyRequestEcho `json:"request"`
}

type V3JourneyRequestEcho struct {
	From            string   `json:"from"`
	To              string   `json:"to"`
	Time            string   `json:"time"`
	Mode            string   `json:"mode"`
	Results         int      `json:"results"`
	Transportations []string `json:"transportations,omitempty"`
}

type V3Connection struct {
	ID        string                `json:"id"`
	Duration  string                `json:"duration"`
	Transfers int                   `json:"transfers"`
	From      V3ConnectionEndpoint  `json:"from"`
	To        V3ConnectionEndpoint  `json:"to"`
	Sections  []V3ConnectionSection `json:"sections"`
}

type V3ConnectionEndpoint struct {
	Station   V3StationRef `json:"station"`
	Departure string       `json:"departure,omitempty"`
	Arrival   string       `json:"arrival,omitempty"`
	Platform  string       `json:"platform,omitempty"`
}

type V3StationRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type V3ConnectionSection struct {
	Walk     *V3WalkSection     `json:"walk,omitempty"`
	Journey  *V3JourneySection  `json:"journey,omitempty"`
	Realtime *V3RealtimeSection `json:"realtime,omitempty"`
}

type V3WalkSection struct {
	From     V3StationRef `json:"from"`
	To       V3StationRef `json:"to"`
	Departure string       `json:"departure,omitempty"`
	Arrival   string       `json:"arrival,omitempty"`
	Duration string       `json:"duration"`
}

type V3JourneySection struct {
	ID        string       `json:"id,omitempty"`
	Name      string       `json:"name"`
	Operator  string       `json:"operator,omitempty"`
	Category  string       `json:"category,omitempty"`
	From      V3StationRef `json:"from"`
	To        V3StationRef `json:"to"`
	Departure string       `json:"departure,omitempty"`
	Arrival   string       `json:"arrival,omitempty"`
}

type V3RealtimeSection struct {
	Delay     int  `json:"delay"`
	Cancelled bool `json:"cancelled"`
}
