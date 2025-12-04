package services

import (
	"fmt"
	"indian-transit-backend/internal/models"
	"math"
	"sync"
)

// AutoAlightService handles automatic alighting detection
type AutoAlightService struct {
	boardingService        *RouteBoardingService
	vehicleLocationService *VehicleLocationService
	stopService            *StopService
	// Track consecutive "not on vehicle" checks per session
	// Key: sessionID, Value: count of consecutive checks where user is not on vehicle
	consecutiveNotOnVehicle map[string]int
	mu                      sync.RWMutex
}

func NewAutoAlightService(boardingService *RouteBoardingService, vehicleLocationService *VehicleLocationService, stopService *StopService) *AutoAlightService {
	return &AutoAlightService{
		boardingService:         boardingService,
		vehicleLocationService:  vehicleLocationService,
		stopService:             stopService,
		consecutiveNotOnVehicle: make(map[string]int),
	}
}

// ResetCounter resets the consecutive "not on vehicle" counter for a session
// This should be called when a new boarding is created to ensure clean state
func (s *AutoAlightService) ResetCounter(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consecutiveNotOnVehicle[sessionID] = 0
	fmt.Printf("[AutoAlight] Session %s: Counter reset (new boarding created)\n", sessionID)
}

// CheckAndAlight automatically detects if user should alight based on location matching
// Alights user if they're not on vehicle for 2-3 consecutive checks after leaving a stop
func (s *AutoAlightService) CheckAndAlight(sessionID string, userLat, userLon float64) (*models.RouteBoarding, error) {
	// Get active boarding
	activeBoarding, err := s.boardingService.GetActiveBoarding(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active boarding: %w", err)
	}
	if activeBoarding == nil {
		// Reset counter if no active boarding
		s.mu.Lock()
		delete(s.consecutiveNotOnVehicle, sessionID)
		s.mu.Unlock()
		return nil, nil // No active boarding
	}

	// Check if vehicle ID is available
	if activeBoarding.VehicleID == nil {
		// Debug: Log when vehicle ID is missing
		fmt.Printf("[AutoAlight] Session %s: No vehicle ID for route %s, cannot auto-alight\n", sessionID, activeBoarding.RouteID)
		return nil, nil // No vehicle tracking, can't auto-alight
	}

	// Verify user is still on the vehicle
	isOnVehicle, distance, err := s.vehicleLocationService.VerifyUserOnVehicle(*activeBoarding.VehicleID, userLat, userLon)
	if err != nil {
		// Debug: Log verification errors
		fmt.Printf("[AutoAlight] Session %s: Failed to verify vehicle %s: %v\n", sessionID, *activeBoarding.VehicleID, err)
		return nil, fmt.Errorf("failed to verify vehicle: %w", err)
	}
	
	// Debug: Log verification result
	fmt.Printf("[AutoAlight] Session %s: Route %s, Vehicle %s, OnVehicle: %v, Distance: %.1fm\n", 
		sessionID, activeBoarding.RouteID, *activeBoarding.VehicleID, isOnVehicle, distance)

	// CRITICAL: If user IS on vehicle, NEVER alight (return immediately)
	// This prevents false alighting when user is confirmed to be on vehicle
	if isOnVehicle {
		// User is on vehicle - reset counter and return (no alighting)
		s.mu.Lock()
		s.consecutiveNotOnVehicle[sessionID] = 0
		s.mu.Unlock()
		return nil, nil
	}

	// Track consecutive "not on vehicle" checks
	s.mu.Lock()
	// User is not on vehicle - increment counter
	s.consecutiveNotOnVehicle[sessionID]++
	consecutiveCount := s.consecutiveNotOnVehicle[sessionID]
	s.mu.Unlock()

	// If user is not on vehicle for 3 consecutive checks, auto-alight
	// This means user has alighted after vehicle left the stop
	// Check 1 and Check 2 will pass (user on vehicle at same location)
	// After vehicle leaves, checks start failing - alight after 3 consecutive failures
	// IMPORTANT: Only auto-alight if user is near a station/stop
	// NOTE: We already checked isOnVehicle above and returned early if true
	if consecutiveCount >= 3 {
		// Debug: Log when threshold is reached
		fmt.Printf("[AutoAlight] Session %s: Consecutive count reached %d, checking for nearby stops...\n", sessionID, consecutiveCount)
		
		// Check if user is near a stop/station (within 100m - lenient for GPS accuracy)
		nearbyStops, err := s.stopService.FindNearby(userLat, userLon, 100, 1)
		if err == nil && len(nearbyStops) > 0 {
			// User is near a stop - auto-alight
			stop := nearbyStops[0]
			alightingStopID := stop.ID
			
			fmt.Printf("[AutoAlight] Session %s: User near stop %s, triggering auto-alight for route %s\n", 
				sessionID, alightingStopID, activeBoarding.RouteID)
			
			alightReq := models.AlightRouteRequest{
				BoardingID:      activeBoarding.ID,
				AlightingStopID: &alightingStopID,
				Latitude:        userLat,
				Longitude:       userLon,
			}

			// Reset counter after alighting
			s.mu.Lock()
			delete(s.consecutiveNotOnVehicle, sessionID)
			s.mu.Unlock()

			return s.boardingService.AlightRoute(alightReq)
		} else {
			// Debug: Log when user is not near a stop
			if err != nil {
				fmt.Printf("[AutoAlight] Session %s: Error finding nearby stops: %v\n", sessionID, err)
			} else {
				fmt.Printf("[AutoAlight] Session %s: User not near any stop (lat: %.6f, lon: %.6f), waiting...\n", 
					sessionID, userLat, userLon)
			}
		}
		// If user is not near a stop, don't auto-alight yet (wait for user to reach a stop)
	} else {
		// Debug: Log current count
		fmt.Printf("[AutoAlight] Session %s: Consecutive count: %d (need 3)\n", sessionID, consecutiveCount)
	}

	// No alighting detected yet (need more checks or conditions not met)
	return nil, nil
}

// calculateDistance calculates distance between two coordinates in meters (Haversine formula)
func (s *AutoAlightService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // Earth radius in meters

	lat1Rad := lat1 * 3.141592653589793 / 180
	lat2Rad := lat2 * 3.141592653589793 / 180
	deltaLat := (lat2 - lat1) * 3.141592653589793 / 180
	deltaLon := (lon2 - lon1) * 3.141592653589793 / 180

	a := 0.5 - math.Cos(deltaLat)/2 + math.Cos(lat1Rad)*math.Cos(lat2Rad)*(1-math.Cos(deltaLon))/2

	return earthRadius * 2 * math.Asin(math.Sqrt(a))
}
