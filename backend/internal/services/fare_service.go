package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"indian-transit-backend/internal/models"
	"indian-transit-backend/internal/repository"
)

type FareService struct {
	repo *repository.FareRepository
}

// Fare rules configuration
type FareRules struct {
	BaseFare             float64 // Minimum fare in INR
	FarePerKm            float64 // Fare per kilometer in INR
	FarePerStop          float64 // Alternative: fare per stop
	TransferFee          float64 // Additional fee for transfers
	ACBusMultiplier      float64 // Multiplier for AC buses
	ExpressBusMultiplier float64 // Multiplier for express buses
}

type farePolicyMetadata struct {
	ACMultiplier      *float64 `json:"ac_multiplier"`
	ExpressMultiplier *float64 `json:"express_multiplier"`
	TransferPolicy    string   `json:"transfer_policy"`
}

// Default fare rules are only an emergency fallback when seeded fare config
// is unavailable for an agency.
var DefaultFareRules = FareRules{
	BaseFare:             5.0, // Minimum ₹5
	FarePerKm:            2.0, // ₹2 per km
	FarePerStop:          0.0, // Not used if FarePerKm > 0
	TransferFee:          2.0, // ₹2 per transfer
	ACBusMultiplier:      1.5, // 1.5x for AC buses
	ExpressBusMultiplier: 1.2, // 1.2x for express buses
}

func NewFareService(repo *repository.FareRepository) *FareService {
	return &FareService{repo: repo}
}

// GetAgencyIDFromRoute gets the agency ID for a given route
func (s *FareService) GetAgencyIDFromRoute(routeID string) string {
	agencyID, err := s.repo.GetAgencyIDByRouteID(routeID)
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
	lat1, lon1, err := s.repo.GetStopCoordinates(fromStopID)
	if err != nil {
		fmt.Printf("[CalculateDistance] Error getting coordinates for stop %s: %v\n", fromStopID, err)
		return 0.5
	}

	lat2, lon2, err := s.repo.GetStopCoordinates(toStopID)
	if err != nil {
		fmt.Printf("[CalculateDistance] Error getting coordinates for stop %s: %v\n", toStopID, err)
		fmt.Printf("[CalculateDistance] Error getting coordinates for stops %s and %s: %v\n", fromStopID, toStopID, err)
		return 0.5
	}

	distanceKm := haversineDistance(lat1, lon1, lat2, lon2) / 1000.0
	fmt.Printf("[CalculateDistance] Stop %s to %s: %.3f km\n", fromStopID, toStopID, distanceKm)
	return distanceKm
}

// getRouteType determines route type from route ID or name
func (s *FareService) getRouteType(routeID string) string {
	shortName, longName, err := s.repo.GetRouteNames(routeID)
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
	product, err := s.repo.GetActiveFareProductByAgencyID(strings.ToUpper(agencyID))
	if err == nil && product != nil {
		policy := parseFarePolicyMetadata(product.Metadata)
		acMultiplier := 1.0
		expressMultiplier := 1.0
		if policy.ACMultiplier != nil {
			acMultiplier = *policy.ACMultiplier
		}
		if policy.ExpressMultiplier != nil {
			expressMultiplier = *policy.ExpressMultiplier
		}
		return FareRules{
			BaseFare:             product.BaseFare,
			FarePerKm:            product.FarePerKM,
			FarePerStop:          product.FarePerStop,
			TransferFee:          product.TransferFee,
			ACBusMultiplier:      acMultiplier,
			ExpressBusMultiplier: expressMultiplier,
		}
	}

	return DefaultFareRules
}

func (s *FareService) GetFareProductForAgency(agencyID string) (*models.FareProduct, error) {
	product, err := s.repo.GetActiveFareProductByAgencyID(strings.ToUpper(agencyID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return product, nil
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

	fmt.Printf("[CalculateRouteSegmentFare] Route: %s, Type: %s, Multiplier: %.2f, Distance: %.3f km, FarePerKm: %.2f\n",
		routeID, routeType, multiplier, distance, rules.FarePerKm)

	fare := distance * rules.FarePerKm * multiplier
	if fare < rules.BaseFare {
		fare = rules.BaseFare
	}

	finalFare := math.Round(fare*100) / 100
	fmt.Printf("[CalculateRouteSegmentFare] Final fare: ₹%.2f (distance: %.3f km × %.2f/km × %.2f multiplier, min: ₹%.2f)\n",
		finalFare, distance, rules.FarePerKm, multiplier, rules.BaseFare)
	return finalFare
}

func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

func parseFarePolicyMetadata(raw string) farePolicyMetadata {
	if raw == "" {
		return farePolicyMetadata{}
	}

	var metadata farePolicyMetadata
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return farePolicyMetadata{}
	}
	return metadata
}
