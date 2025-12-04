package services

import (
	"math"
	"strings"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type FareService struct {
	db *database.DB
}

// Fare rules configuration
type FareRules struct {
	BaseFare          float64 // Minimum fare in INR
	FarePerKm         float64 // Fare per kilometer in INR
	FarePerStop       float64 // Alternative: fare per stop
	TransferFee       float64 // Additional fee for transfers
	ACBusMultiplier   float64 // Multiplier for AC buses
	ExpressBusMultiplier float64 // Multiplier for express buses
}

// Default fare rules (approximate - should be configured per agency)
var DefaultFareRules = FareRules{
	BaseFare:           5.0,   // Minimum ₹5
	FarePerKm:          2.0,    // ₹2 per km
	FarePerStop:        0.0,   // Not used if FarePerKm > 0
	TransferFee:        2.0,    // ₹2 per transfer
	ACBusMultiplier:    1.5,    // 1.5x for AC buses
	ExpressBusMultiplier: 1.2,  // 1.2x for express buses
}

// Delhi Metro fare rules
var DMRCFareRules = FareRules{
	BaseFare:           10.0,  // Minimum ₹10 for metro
	FarePerKm:          2.5,   // ₹2.5 per km for metro
	FarePerStop:        0.0,
	TransferFee:        0.0,   // No transfer fee within metro system
	ACBusMultiplier:    1.0,   // Not applicable
	ExpressBusMultiplier: 1.0, // Not applicable
}

// Delhi Bus fare rules (DIMTS + DTC)
var DelhiBusFareRules = FareRules{
	BaseFare:           5.0,   // Minimum ₹5
	FarePerKm:          1.5,   // ₹1.5 per km
	FarePerStop:        0.0,
	TransferFee:        2.0,   // ₹2 per transfer
	ACBusMultiplier:    1.5,   // 1.5x for AC buses
	ExpressBusMultiplier: 1.2, // 1.2x for express buses
}

func NewFareService(db *database.DB) *FareService {
	return &FareService{db: db}
}

// GetAgencyIDFromRoute gets the agency ID for a given route
func (s *FareService) GetAgencyIDFromRoute(routeID string) string {
	query := `SELECT agency_id FROM routes WHERE route_id = $1`
	var agencyID string
	err := s.db.QueryRow(query, routeID).Scan(&agencyID)
	if err != nil {
		return "" // Return empty string if route not found
	}
	return agencyID
}

// CalculateFareForJourney calculates fare for a complete journey
func (s *FareService) CalculateFareForJourney(journey models.JourneyOption, rules FareRules) float64 {
	totalFare := 0.0

	for _, leg := range journey.Legs {
		if leg.Mode == "walking" {
			continue // Walking is free
		}

		legFare := s.calculateLegFare(leg, rules)
		totalFare += legFare
	}

	// Add transfer fees
	if journey.Transfers > 0 {
		totalFare += rules.TransferFee * float64(journey.Transfers)
	}

	// Ensure minimum fare
	if totalFare < rules.BaseFare {
		totalFare = rules.BaseFare
	}

	return math.Round(totalFare*100) / 100 // Round to 2 decimal places
}

// calculateLegFare calculates fare for a single leg
func (s *FareService) calculateLegFare(leg models.JourneyLeg, rules FareRules) float64 {
	if leg.Mode == "walking" {
		return 0.0
	}

	// Get route type to apply multiplier
	routeType := s.getRouteType(leg.RouteID)
	multiplier := 1.0

	switch routeType {
	case "AC", "Vayu Vajra":
		multiplier = rules.ACBusMultiplier
	case "Express", "Big 10":
		multiplier = rules.ExpressBusMultiplier
	}

	// Calculate distance-based fare
	var fare float64
	if rules.FarePerKm > 0 {
		distance := s.CalculateDistance(leg.FromStopID, leg.ToStopID)
		fare = distance * rules.FarePerKm
	} else if rules.FarePerStop > 0 {
		fare = float64(leg.StopCount) * rules.FarePerStop
	} else {
		// Fallback: use base fare
		fare = rules.BaseFare
	}

	return fare * multiplier
}

// CalculateDistance calculates distance between two stops in kilometers
func (s *FareService) CalculateDistance(fromStopID, toStopID string) float64 {
	query := `SELECT 
		ST_Distance(
			(SELECT location FROM stops WHERE stop_id = $1)::geography,
			(SELECT location FROM stops WHERE stop_id = $2)::geography
		) / 1000.0 as distance_km`

	var distance float64
	err := s.db.QueryRow(query, fromStopID, toStopID).Scan(&distance)
	if err != nil {
		// Fallback: estimate based on stop count (rough approximation)
		// Assume average 500m between stops
		return 0.5
	}

	return distance
}

// getRouteType determines route type from route ID or name
func (s *FareService) getRouteType(routeID string) string {
	// Check route name for indicators
	query := `SELECT route_short_name, route_long_name FROM routes WHERE route_id = $1`
	var shortName, longName string
	err := s.db.QueryRow(query, routeID).Scan(&shortName, &longName)
	if err != nil {
		return "Ordinary"
	}

	// Check for AC/Vayu Vajra indicators
	if containsIgnoreCase(shortName, "V-") || containsIgnoreCase(longName, "Vayu Vajra") || containsIgnoreCase(longName, "AC") {
		return "AC"
	}

	// Check for Express indicators
	if containsIgnoreCase(shortName, "E") || containsIgnoreCase(longName, "Express") || containsIgnoreCase(longName, "Big 10") {
		return "Express"
	}

	return "Ordinary"
}

// GetRouteFare gets fare information for a specific route
func (s *FareService) GetRouteFare(routeID string, fromStopID, toStopID string, rules FareRules) (float64, error) {
	if fromStopID == "" || toStopID == "" {
		// Return base fare if stops not specified
		return rules.BaseFare, nil
	}

	distance := s.CalculateDistance(fromStopID, toStopID)
	routeType := s.getRouteType(routeID)

	multiplier := 1.0
	switch routeType {
	case "AC":
		multiplier = rules.ACBusMultiplier
	case "Express":
		multiplier = rules.ExpressBusMultiplier
	}

	fare := distance * rules.FarePerKm * multiplier
	if fare < rules.BaseFare {
		fare = rules.BaseFare
	}

	return math.Round(fare*100) / 100, nil
}

// GetFareRulesForAgency gets fare rules for a specific agency
func (s *FareService) GetFareRulesForAgency(agencyID string) FareRules {
	// Return agency-specific fare rules
	switch strings.ToUpper(agencyID) {
	case "DMRC":
		return DMRCFareRules
	case "DIMTS", "DTC":
		return DelhiBusFareRules
	default:
		return DefaultFareRules
	}
}

// CalculateRouteSegmentFare calculates fare for a single route segment (boarding to alighting)
func (s *FareService) CalculateRouteSegmentFare(routeID, fromStopID, toStopID string, distance float64, rules FareRules) float64 {
	routeType := s.getRouteType(routeID)
	multiplier := 1.0

	switch routeType {
	case "AC", "Vayu Vajra":
		multiplier = rules.ACBusMultiplier
	case "Express", "Big 10":
		multiplier = rules.ExpressBusMultiplier
	}

	fare := distance * rules.FarePerKm * multiplier
	if fare < rules.BaseFare {
		fare = rules.BaseFare
	}

	return math.Round(fare*100) / 100
}

func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

