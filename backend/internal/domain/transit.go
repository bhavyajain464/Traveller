package domain

// TransportMode is the provider-neutral transit mode vocabulary that the
// system should use above raw GTFS route_type values.
type TransportMode string

const (
	TransportModeTrain       TransportMode = "train"
	TransportModeBus         TransportMode = "bus"
	TransportModeTram        TransportMode = "tram"
	TransportModeMetro       TransportMode = "metro"
	TransportModeBoat        TransportMode = "boat"
	TransportModeCableCar    TransportMode = "cable_car"
	TransportModeFunicular   TransportMode = "funicular"
	TransportModeWalking     TransportMode = "walking"
	TransportModeReplacement TransportMode = "replacement_bus"
)

// JourneyStatus provides a shared lifecycle vocabulary for journey sessions.
type JourneyStatus string

const (
	JourneyStatusActive    JourneyStatus = "active"
	JourneyStatusCompleted JourneyStatus = "completed"
	JourneyStatusCancelled JourneyStatus = "cancelled"
)

type BillingStatus string

const (
	BillingStatusPending BillingStatus = "pending"
	BillingStatusPaid    BillingStatus = "paid"
	BillingStatusFailed  BillingStatus = "failed"
)

// ServiceName identifies the logical backend services that make up the target
// architecture, even while they currently run inside one deployable binary.
type ServiceName string

const (
	ServiceJourneyPlanner ServiceName = "journey_planner"
	ServiceRealtimeEngine ServiceName = "realtime_engine"
	ServiceTimetable      ServiceName = "timetable"
	ServiceTicketing      ServiceName = "ticketing"
	ServiceNotifications  ServiceName = "notifications"
)

func IsActiveJourneyStatus(status string) bool {
	return status == string(JourneyStatusActive)
}

func IsValidJourneyStatus(status string) bool {
	switch JourneyStatus(status) {
	case JourneyStatusActive, JourneyStatusCompleted, JourneyStatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidBillingStatus(status string) bool {
	switch BillingStatus(status) {
	case BillingStatusPending, BillingStatusPaid, BillingStatusFailed:
		return true
	default:
		return false
	}
}

func IsValidCoordinate(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func HasSessionReference(sessionID, qrCode string) bool {
	return sessionID != "" || qrCode != ""
}
