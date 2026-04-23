package services

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"

	"github.com/google/uuid"
)

type VehicleLocationService struct {
	db           *database.DB
	routeService *RouteService
	stopService  *StopService
	// In-memory cache for vehicle locations (mock data)
	vehicleLocations map[string]*models.VehicleLocation
	// Track route stops for each vehicle (for realistic movement)
	vehicleRouteStops map[string][]models.Stop
	// Track current stop index for each vehicle
	vehicleStopIndex map[string]int
	// Track when vehicle was created (for 30-second delay)
	vehicleCreatedAt map[string]time.Time
	// Track trip ID for each vehicle (for schedule-based positioning)
	vehicleTripID map[string]string
	mu            sync.RWMutex
}

func NewVehicleLocationService(db *database.DB, routeService *RouteService, stopService *StopService) *VehicleLocationService {
	service := &VehicleLocationService{
		db:                db,
		routeService:      routeService,
		stopService:       stopService,
		vehicleLocations:  make(map[string]*models.VehicleLocation),
		vehicleRouteStops: make(map[string][]models.Stop),
		vehicleStopIndex:  make(map[string]int),
		vehicleCreatedAt:  make(map[string]time.Time),
		vehicleTripID:     make(map[string]string),
	}

	// Initialize with mock vehicle locations
	service.initializeMockVehicles()

	// Start background goroutine to update vehicle positions
	go service.updateVehiclePositions()

	return service
}

// initializeMockVehicles creates initial mock vehicle locations
func (s *VehicleLocationService) initializeMockVehicles() {
	// Get some active routes from database
	routes, err := s.routeService.GetActiveRoutes(50) // Get 50 routes
	if err != nil || len(routes) == 0 {
		// If no routes, create some default mock locations
		s.createDefaultMockVehicles()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create mock vehicles for each route
	for i, route := range routes {
		if i >= 20 { // Limit to 20 vehicles for performance
			break
		}

		// Get stops for this route to create realistic positions
		stops, err := s.routeService.GetStops(route.ID)
		if err != nil || len(stops) == 0 {
			continue
		}

		// Pick a random stop along the route
		stopIndex := rand.Intn(len(stops))
		stop := stops[stopIndex]

		vehicleID := fmt.Sprintf("vehicle-%s-%d", route.ID, i)
		now := time.Now()
		s.vehicleLocations[vehicleID] = &models.VehicleLocation{
			VehicleID:    vehicleID,
			RouteID:      route.ID,
			Latitude:     stop.Latitude,
			Longitude:    stop.Longitude,
			Timestamp:    now,
			StopSequence: &stopIndex,
		}
		// Store route stops for this vehicle
		s.vehicleRouteStops[vehicleID] = stops
		s.vehicleStopIndex[vehicleID] = stopIndex
		s.vehicleCreatedAt[vehicleID] = now
	}
}

// createDefaultMockVehicles creates default mock vehicles if no routes found
func (s *VehicleLocationService) createDefaultMockVehicles() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create some mock vehicles at common Delhi locations
	mockVehicles := []struct {
		routeID   string
		routeType int
		lat       float64
		lon       float64
		name      string
	}{
		{"YELLOW_LINE", 1, 28.6304, 77.2177, "Yellow Line Metro"},
		{"BLUE_LINE", 1, 28.6129, 77.2295, "Blue Line Metro"},
		{"BUS_123", 3, 28.6250, 77.2200, "Bus Route 123"},
		{"BUS_456", 3, 28.6200, 77.2220, "Bus Route 456"},
		{"BUS_789", 3, 28.6150, 77.2250, "Bus Route 789"},
	}

	for i, mv := range mockVehicles {
		vehicleID := fmt.Sprintf("vehicle-%s-%d", mv.routeID, i)
		seq := i
		now := time.Now()
		s.vehicleLocations[vehicleID] = &models.VehicleLocation{
			VehicleID:    vehicleID,
			RouteID:      mv.routeID,
			Latitude:     mv.lat,
			Longitude:    mv.lon,
			Timestamp:    now,
			StopSequence: &seq,
		}
		// Try to get route stops, otherwise use empty slice
		stops, _ := s.routeService.GetStops(mv.routeID)
		s.vehicleRouteStops[vehicleID] = stops
		s.vehicleStopIndex[vehicleID] = 0
		s.vehicleCreatedAt[vehicleID] = now
	}
}

// updateVehiclePositions periodically updates vehicle positions (simulates movement)
func (s *VehicleLocationService) updateVehiclePositions() {
	ticker := time.NewTicker(1 * time.Second) // Update every 1 second for smoother movement
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		for vehicleID, vehicle := range s.vehicleLocations {
			createdAt := s.vehicleCreatedAt[vehicleID]
			timeSinceCreation := now.Sub(createdAt)

			// For first 30 seconds, vehicles stay at boarding location (boarding phase)
			// This ensures vehicles are at the correct location for boarding detection
			if timeSinceCreation < 30*time.Second {
				// Stay at same location, just update timestamp
				vehicle.Timestamp = now
				continue
			}

			// After 30 seconds, check if vehicle has trip_id for schedule-based positioning
			tripID, hasSchedule := s.vehicleTripID[vehicleID]
			if hasSchedule {
				// Use schedule-based positioning (more realistic)
				// Use elapsed time since creation (after 30 seconds) to calculate position
				elapsedTime := timeSinceCreation - 30*time.Second
				schedulePos, err := s.calculateVehiclePositionFromScheduleWithElapsedTime(tripID, elapsedTime)
				if err == nil && schedulePos != nil {
					vehicle.Latitude = schedulePos.Latitude
					vehicle.Longitude = schedulePos.Longitude
					if schedulePos.Bearing != nil {
						vehicle.Bearing = schedulePos.Bearing
					}
					if schedulePos.Speed != nil {
						vehicle.Speed = schedulePos.Speed
					}
					vehicle.Timestamp = now
					continue
				}
				// If schedule calculation fails, fall back to route-based movement
			}

			// After 30 seconds, move vehicle along route stops
			stops := s.vehicleRouteStops[vehicleID]
			if len(stops) > 0 {
				currentIndex := s.vehicleStopIndex[vehicleID]

				// Move to next stop if we've been at current stop for a while
				// Or if we're close to current stop, move towards next
				if currentIndex < len(stops)-1 {
					currentStop := stops[currentIndex]
					nextStop := stops[currentIndex+1]

					// Calculate distance to current stop
					distanceToCurrent := s.calculateDistance(vehicle.Latitude, vehicle.Longitude, currentStop.Latitude, currentStop.Longitude)

					// If vehicle is very far (>5km), teleport closer to speed up simulation
					// This handles cases where vehicle is created far from route stops
					if distanceToCurrent > 5000 {
						// Teleport to within ~1km of current stop to speed up simulation
						latDiff := currentStop.Latitude - vehicle.Latitude
						lonDiff := currentStop.Longitude - vehicle.Longitude
						// Move 95% of the way (leaving ~1km, which will be covered quickly)
						vehicle.Latitude += latDiff * 0.95
						vehicle.Longitude += lonDiff * 0.95
					} else if distanceToCurrent < 50 {
						// If at or close to current stop (< 50m), move towards next stop
						latDiff := nextStop.Latitude - vehicle.Latitude
						lonDiff := nextStop.Longitude - vehicle.Longitude

						// Move 1% of the way towards next stop each update (since we update every second)
						// This gives smooth movement: ~100 seconds to reach next stop
						vehicle.Latitude += latDiff * 0.01
						vehicle.Longitude += lonDiff * 0.01

						// If we've reached next stop, update index
						distanceToNext := s.calculateDistance(vehicle.Latitude, vehicle.Longitude, nextStop.Latitude, nextStop.Longitude)
						if distanceToNext < 50 {
							s.vehicleStopIndex[vehicleID] = currentIndex + 1
							seq := currentIndex + 1
							vehicle.StopSequence = &seq
						}
					} else {
						// Vehicle is moderately far from current stop - move towards current stop
						latDiff := currentStop.Latitude - vehicle.Latitude
						lonDiff := currentStop.Longitude - vehicle.Longitude
						// Move 10% towards current stop each update (faster movement when far)
						vehicle.Latitude += latDiff * 0.10
						vehicle.Longitude += lonDiff * 0.10
					}
				} else {
					// At last stop, stay there or wrap around
					lastStop := stops[len(stops)-1]
					vehicle.Latitude = lastStop.Latitude
					vehicle.Longitude = lastStop.Longitude
				}

				// Calculate bearing towards next stop
				if currentIndex < len(stops)-1 {
					nextStop := stops[currentIndex+1]
					bearing := s.calculateBearing(vehicle.Latitude, vehicle.Longitude, nextStop.Latitude, nextStop.Longitude)
					vehicle.Bearing = &bearing
				}

				// Set speed (metro: 30-50 km/h, bus: 20-40 km/h)
				route, _ := s.routeService.GetRouteByID(vehicle.RouteID)
				var speed float64
				if route != nil && route.Type == 1 {
					speed = 30 + rand.Float64()*20 // Metro: 30-50 km/h
				} else {
					speed = 20 + rand.Float64()*20 // Bus: 20-40 km/h
				}
				vehicle.Speed = &speed
			} else {
				// No route stops, use random movement
				latOffset := (rand.Float64() - 0.5) * 0.001
				lonOffset := (rand.Float64() - 0.5) * 0.001
				vehicle.Latitude += latOffset
				vehicle.Longitude += lonOffset
			}

			vehicle.Timestamp = now
		}
		s.mu.Unlock()
	}
}

// calculateBearing calculates bearing (direction) from point A to point B in degrees
func (s *VehicleLocationService) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	bearing = math.Mod(bearing+360, 360) // Normalize to 0-360

	return bearing
}

// FindNearbyVehicles finds vehicles near a given location
func (s *VehicleLocationService) FindNearbyVehicles(lat, lon float64, radiusMeters float64) ([]models.VehicleLocationMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []models.VehicleLocationMatch

	for _, vehicle := range s.vehicleLocations {
		distance := s.calculateDistance(lat, lon, vehicle.Latitude, vehicle.Longitude)

		if distance <= radiusMeters {
			// Get route details
			route, err := s.routeService.GetRouteByID(vehicle.RouteID)
			if err != nil {
				continue
			}

			// Calculate confidence based on distance (closer = higher confidence)
			confidence := 1.0 - (distance / radiusMeters)
			if confidence < 0 {
				confidence = 0
			}

			// Get agency ID for route
			agencyID := s.getAgencyIDForRoute(vehicle.RouteID)

			matches = append(matches, models.VehicleLocationMatch{
				VehicleLocation: *vehicle,
				RouteID:         vehicle.RouteID,
				RouteName:       route.ShortName,
				RouteType:       route.Type,
				AgencyID:        agencyID,
				Distance:        distance,
				Confidence:      confidence,
			})
		}
	}

	return matches, nil
}

// DetectTransportMode automatically detects which vehicle the user is on
// NOTE: When multiple vehicles are at the same location, this picks one arbitrarily.
// The EXACT vehicle confirmation happens AFTER 30 seconds when vehicles move,
// using continuous location matching via FindExactVehicleMatch.
func (s *VehicleLocationService) DetectTransportMode(userLat, userLon float64) (*models.VehicleLocationMatch, error) {
	// Search within 100 meters radius (increased tolerance for boarding)
	// At boarding time, user and vehicle should be at same location (within GPS accuracy)
	matches, err := s.FindNearbyVehicles(userLat, userLon, 100)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		// Debug: Check if there are any vehicles at all
		s.mu.RLock()
		totalVehicles := len(s.vehicleLocations)
		s.mu.RUnlock()
		if totalVehicles == 0 {
			return nil, fmt.Errorf("no vehicles available in system")
		}
		return nil, fmt.Errorf("no nearby vehicles found within 100m (user: %.6f, %.6f, total vehicles: %d)", userLat, userLon, totalVehicles)
	}

	// When multiple vehicles at same location, we can't confirm which one user boarded
	// Just pick the first/closest one - confirmation will happen after vehicles move
	bestMatch := &matches[0]
	for i := 1; i < len(matches); i++ {
		// Prefer closest match (highest confidence = closest distance)
		if matches[i].Confidence > bestMatch.Confidence {
			bestMatch = &matches[i]
		}
	}

	// At boarding time, accept any match within 100m (confidence > 0 means within radius)
	// Lower threshold for boarding since user and vehicle should be at same location
	if bestMatch.Confidence < 0 {
		return nil, fmt.Errorf("low confidence match (%.2f) - vehicle too far", bestMatch.Confidence)
	}

	return bestMatch, nil
}

// VerifyUserOnVehicle checks if user is still on a specific vehicle by matching locations over time
// This is critical after 30 seconds when vehicles start moving - it verifies the exact vehicle match
// Allows matching within error tolerance (GPS accuracy + movement)
func (s *VehicleLocationService) VerifyUserOnVehicle(vehicleID string, userLat, userLon float64) (bool, float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vehicle, exists := s.vehicleLocations[vehicleID]
	if !exists {
		return false, 0, fmt.Errorf("vehicle not found")
	}

	distance := s.calculateDistance(userLat, userLon, vehicle.Latitude, vehicle.Longitude)

	// Check if vehicle has started moving (created more than 30 seconds ago)
	createdAt, exists := s.vehicleCreatedAt[vehicleID]
	now := time.Now()
	timeSinceCreation := now.Sub(createdAt)

	// Error tolerance: Allow matching within 100m
	// This accounts for GPS accuracy (10-50m) and vehicle movement between updates
	// Vehicles update every 1 second, user location every 5 seconds
	const maxMatchDistance = 100.0 // meters

	// If vehicle has started moving (after 30 seconds), use tolerance-based matching
	// This ensures we match the EXACT vehicle the user boarded within GPS accuracy
	if exists && timeSinceCreation >= 30*time.Second {
		// After movement starts, user must be within tolerance of this specific vehicle
		isOnVehicle := distance <= maxMatchDistance
		return isOnVehicle, distance, nil
	}

	// Before 30 seconds, vehicles are still at same location, so use normal threshold
	// User is on vehicle if within 100 meters
	isOnVehicle := distance <= maxMatchDistance

	return isOnVehicle, distance, nil
}

// FindExactVehicleMatch finds which specific vehicle matches user's location after vehicles have moved
// This is used after 30 seconds when vehicles diverge to CONFIRM the exact vehicle user is on
// This is the confirmation step - we can't confirm at boarding when multiple vehicles are at same location
// Allows matching within error tolerance (GPS accuracy + movement)
// Vehicles update every 1 second, user location checked every 5 seconds
func (s *VehicleLocationService) FindExactVehicleMatch(userLat, userLon float64, previouslyBoardedVehicleID *string) (*models.VehicleLocationMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var bestMatch *models.VehicleLocationMatch
	var bestDistance float64 = 1000 // Start with large distance
	var vehiclesHaveMoved bool

	// Error tolerance: Allow matching within 100m
	// This accounts for:
	// - GPS accuracy (typically 10-50m)
	// - Vehicle movement between updates (vehicles update every 1 second, user location every 5 seconds)
	// - User might be slightly ahead/behind vehicle position
	const maxMatchDistance = 100.0 // meters

	// First, check if user is still on the previously boarded vehicle
	if previouslyBoardedVehicleID != nil {
		vehicle, exists := s.vehicleLocations[*previouslyBoardedVehicleID]
		if exists {
			createdAt, hasTime := s.vehicleCreatedAt[*previouslyBoardedVehicleID]
			timeSinceCreation := now.Sub(createdAt)

			// Only check if vehicle has started moving (after 30 seconds)
			if hasTime && timeSinceCreation >= 30*time.Second {
				distance := s.calculateDistance(userLat, userLon, vehicle.Latitude, vehicle.Longitude)
				if distance <= maxMatchDistance {
					// User is still on the same vehicle (within tolerance)
					route, err := s.routeService.GetRouteByID(vehicle.RouteID)
					if err == nil {
						agencyID := s.getAgencyIDForRoute(vehicle.RouteID)
						confidence := 1.0 - (distance / maxMatchDistance)
						if confidence < 0 {
							confidence = 0
						}
						return &models.VehicleLocationMatch{
							VehicleLocation: *vehicle,
							RouteID:         vehicle.RouteID,
							RouteName:       route.ShortName,
							RouteType:       route.Type,
							AgencyID:        agencyID,
							Distance:        distance,
							Confidence:      confidence,
						}, nil
					}
				}
			}
		}
	}

	// If not on previously boarded vehicle, find the closest vehicle that has moved
	for _, vehicle := range s.vehicleLocations {
		createdAt, exists := s.vehicleCreatedAt[vehicle.VehicleID]
		if !exists {
			continue
		}

		timeSinceCreation := now.Sub(createdAt)

		// Only consider vehicles that have started moving (after 30 seconds)
		if timeSinceCreation >= 30*time.Second {
			vehiclesHaveMoved = true
			distance := s.calculateDistance(userLat, userLon, vehicle.Latitude, vehicle.Longitude)

			// Find the closest vehicle within error tolerance
			if distance <= maxMatchDistance && distance < bestDistance {
				route, err := s.routeService.GetRouteByID(vehicle.RouteID)
				if err != nil {
					continue
				}

				agencyID := s.getAgencyIDForRoute(vehicle.RouteID)
				// Confidence decreases as distance increases, but still valid within tolerance
				confidence := 1.0 - (distance / maxMatchDistance)
				if confidence < 0 {
					confidence = 0
				}

				bestMatch = &models.VehicleLocationMatch{
					VehicleLocation: *vehicle,
					RouteID:         vehicle.RouteID,
					RouteName:       route.ShortName,
					RouteType:       route.Type,
					AgencyID:        agencyID,
					Distance:        distance,
					Confidence:      confidence,
				}
				bestDistance = distance
			}
		}
	}

	if bestMatch == nil {
		if vehiclesHaveMoved {
			return nil, fmt.Errorf("no vehicle match found after movement - user location doesn't match any vehicle within %dm tolerance", int(maxMatchDistance))
		}
		return nil, fmt.Errorf("vehicles have not moved yet - confirmation pending")
	}

	return bestMatch, nil
}

// DetectAlightingStop checks if user and vehicle are both at a stop (for automatic alighting)
func (s *VehicleLocationService) DetectAlightingStop(vehicleID string, userLat, userLon float64) (*string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vehicle, exists := s.vehicleLocations[vehicleID]
	if !exists {
		return nil, fmt.Errorf("vehicle not found")
	}

	// Check if vehicle is at a stop (within 50m)
	stops := s.vehicleRouteStops[vehicleID]
	currentIndex := s.vehicleStopIndex[vehicleID]

	if currentIndex < len(stops) {
		stop := stops[currentIndex]
		vehicleDistanceToStop := s.calculateDistance(vehicle.Latitude, vehicle.Longitude, stop.Latitude, stop.Longitude)
		userDistanceToStop := s.calculateDistance(userLat, userLon, stop.Latitude, stop.Longitude)

		// If both vehicle and user are at the same stop (within 50m), user can alight
		if vehicleDistanceToStop < 50 && userDistanceToStop < 50 {
			return &stop.ID, nil
		}
	}

	// Also check nearby stops
	nearbyStops, err := s.stopService.FindNearby(userLat, userLon, 50, 5)
	if err == nil {
		for _, stop := range nearbyStops {
			vehicleDistanceToStop := s.calculateDistance(vehicle.Latitude, vehicle.Longitude, stop.Latitude, stop.Longitude)
			if vehicleDistanceToStop < 50 {
				return &stop.ID, nil
			}
		}
	}

	return nil, nil // No alighting stop detected
}

// calculateDistance calculates distance between two coordinates in meters (Haversine formula)
func (s *VehicleLocationService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // Earth radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// getAgencyIDForRoute gets the agency ID for a route
func (s *VehicleLocationService) getAgencyIDForRoute(routeID string) string {
	query := `SELECT agency_id FROM routes WHERE route_id = ?`
	var agencyID string
	err := s.db.QueryRow(query, routeID).Scan(&agencyID)
	if err != nil {
		return "DIMTS" // Default
	}
	return agencyID
}

// AddMockVehicle adds a mock vehicle at a specific location (for testing)
// If startMovingImmediately is true, vehicle will start moving right away (by setting createdAt to 30 seconds ago)
func (s *VehicleLocationService) AddMockVehicle(routeID string, lat, lon float64) string {
	return s.AddMockVehicleWithOptions(routeID, lat, lon, false)
}

// AddMockVehicleWithOptions adds a mock vehicle with options
func (s *VehicleLocationService) AddMockVehicleWithOptions(routeID string, lat, lon float64, startMovingImmediately bool) string {
	return s.AddMockVehicleWithTrip(routeID, lat, lon, nil, startMovingImmediately)
}

// AddMockVehicleWithTrip adds a mock vehicle with optional trip_id for schedule-based positioning
func (s *VehicleLocationService) AddMockVehicleWithTrip(routeID string, lat, lon float64, tripID *string, startMovingImmediately bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	newVehicleID := uuid.New().String()
	now := time.Now()

	// If startMovingImmediately, set createdAt to 30 seconds ago so vehicle starts moving right away
	var createdAt time.Time
	if startMovingImmediately {
		createdAt = now.Add(-30 * time.Second)
	} else {
		createdAt = now
	}

	bearing := rand.Float64() * 360
	speed := 30.0

	vehicle := &models.VehicleLocation{
		VehicleID: newVehicleID,
		RouteID:   routeID,
		Latitude:  lat,
		Longitude: lon,
		Bearing:   &bearing,
		Speed:     &speed,
		Timestamp: now,
	}

	// If trip_id provided, use schedule-based positioning
	// But start at the provided location first (for boarding)
	// Schedule-based positioning will take over after vehicle starts moving
	if tripID != nil {
		vehicle.TripID = tripID
		s.vehicleTripID[newVehicleID] = *tripID
		// Keep initial location as provided (for boarding)
		// Schedule-based positioning will be used in update loop
	}

	s.vehicleLocations[newVehicleID] = vehicle

	// Get route stops for realistic movement (fallback if no trip_id)
	stops, err := s.routeService.GetStops(routeID)
	if err == nil && len(stops) > 0 {
		// Find closest stop to starting location
		closestIndex := 0
		minDist := s.calculateDistance(lat, lon, stops[0].Latitude, stops[0].Longitude)
		for i, stop := range stops {
			dist := s.calculateDistance(lat, lon, stop.Latitude, stop.Longitude)
			if dist < minDist {
				minDist = dist
				closestIndex = i
			}
		}
		s.vehicleRouteStops[newVehicleID] = stops
		s.vehicleStopIndex[newVehicleID] = closestIndex
		seq := closestIndex
		s.vehicleLocations[newVehicleID].StopSequence = &seq
	} else {
		s.vehicleRouteStops[newVehicleID] = []models.Stop{}
		s.vehicleStopIndex[newVehicleID] = 0
	}
	s.vehicleCreatedAt[newVehicleID] = createdAt

	return newVehicleID
}

// GetVehicleLocation gets current location of a specific vehicle
func (s *VehicleLocationService) GetVehicleLocation(vehicleID string) (*models.VehicleLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vehicle, exists := s.vehicleLocations[vehicleID]
	if !exists {
		return nil, fmt.Errorf("vehicle not found")
	}

	return vehicle, nil
}

// calculateVehiclePositionFromSchedule calculates vehicle position based on GTFS schedule
// This uses the actual trip schedule (arrival_time, departure_time) to determine
// where the vehicle should be at the current time, assuming constant speed between stops
func (s *VehicleLocationService) calculateVehiclePositionFromSchedule(tripID string, currentTime time.Time) (*models.VehicleLocation, error) {
	// Parse current time to HH:MM:SS format
	currentTimeStr := currentTime.Format("15:04:05")

	// Get stop_times for this trip, ordered by sequence
	query := `SELECT st.stop_id, st.stop_sequence, st.arrival_time, st.departure_time,
		s.stop_lat, s.stop_lon
		FROM stop_times st
		JOIN stops s ON st.stop_id = s.stop_id
		WHERE st.trip_id = ?
		ORDER BY st.stop_sequence`

	rows, err := s.db.Query(query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stop times: %w", err)
	}
	defer rows.Close()

	type StopTimeData struct {
		StopID        string
		StopSequence  int
		ArrivalTime   string
		DepartureTime string
		Latitude      float64
		Longitude     float64
	}

	var stopTimes []StopTimeData
	for rows.Next() {
		var st StopTimeData
		err := rows.Scan(&st.StopID, &st.StopSequence, &st.ArrivalTime, &st.DepartureTime,
			&st.Latitude, &st.Longitude)
		if err != nil {
			continue
		}
		stopTimes = append(stopTimes, st)
	}

	if len(stopTimes) == 0 {
		return nil, fmt.Errorf("no stop times found for trip %s", tripID)
	}

	// Find which segment the vehicle is in (between which two stops)
	for i := 0; i < len(stopTimes)-1; i++ {
		currentStop := stopTimes[i]
		nextStop := stopTimes[i+1]

		// Check if current time is between departure from current stop and arrival at next stop
		if currentTimeStr >= currentStop.DepartureTime && currentTimeStr <= nextStop.ArrivalTime {
			// Vehicle is between currentStop and nextStop
			// Calculate progress percentage based on time
			departureSec := parseTimeToSeconds(currentStop.DepartureTime)
			arrivalSec := parseTimeToSeconds(nextStop.ArrivalTime)
			currentSec := parseTimeToSeconds(currentTimeStr)

			if arrivalSec <= departureSec {
				// Handle overnight trips (arrival time < departure time means next day)
				arrivalSec += 24 * 3600 // Add 24 hours
			}

			if currentSec < departureSec {
				// Before departure, vehicle is at current stop
				return &models.VehicleLocation{
					Latitude:  currentStop.Latitude,
					Longitude: currentStop.Longitude,
				}, nil
			}

			if currentSec > arrivalSec {
				// After arrival, vehicle is at next stop
				return &models.VehicleLocation{
					Latitude:  nextStop.Latitude,
					Longitude: nextStop.Longitude,
				}, nil
			}

			// Interpolate position between stops
			timeProgress := float64(currentSec-departureSec) / float64(arrivalSec-departureSec)

			latDiff := nextStop.Latitude - currentStop.Latitude
			lonDiff := nextStop.Longitude - currentStop.Longitude

			lat := currentStop.Latitude + (latDiff * timeProgress)
			lon := currentStop.Longitude + (lonDiff * timeProgress)

			// Calculate bearing
			bearing := s.calculateBearing(currentStop.Latitude, currentStop.Longitude,
				nextStop.Latitude, nextStop.Longitude)

			// Calculate speed (distance / time)
			distance := s.calculateDistance(currentStop.Latitude, currentStop.Longitude,
				nextStop.Latitude, nextStop.Longitude)
			timeDiff := float64(arrivalSec - departureSec)
			speed := (distance / timeDiff) * 3.6 // Convert m/s to km/h

			return &models.VehicleLocation{
				Latitude:  lat,
				Longitude: lon,
				Bearing:   &bearing,
				Speed:     &speed,
			}, nil
		}
	}

	// If before first stop departure, vehicle is at first stop
	if currentTimeStr < stopTimes[0].DepartureTime {
		return &models.VehicleLocation{
			Latitude:  stopTimes[0].Latitude,
			Longitude: stopTimes[0].Longitude,
		}, nil
	}

	// If after last stop arrival, vehicle is at last stop
	return &models.VehicleLocation{
		Latitude:  stopTimes[len(stopTimes)-1].Latitude,
		Longitude: stopTimes[len(stopTimes)-1].Longitude,
	}, nil
}

// calculateVehiclePositionFromScheduleWithElapsedTime calculates vehicle position based on elapsed time
// This uses elapsed time since vehicle creation (after 30 seconds) to simulate movement along schedule
func (s *VehicleLocationService) calculateVehiclePositionFromScheduleWithElapsedTime(tripID string, elapsedTime time.Duration) (*models.VehicleLocation, error) {
	// Get stop_times for this trip, ordered by sequence
	query := `SELECT st.stop_id, st.stop_sequence, st.arrival_time, st.departure_time,
		s.stop_lat, s.stop_lon
		FROM stop_times st
		JOIN stops s ON st.stop_id = s.stop_id
		WHERE st.trip_id = ?
		ORDER BY st.stop_sequence`

	rows, err := s.db.Query(query, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stop times: %w", err)
	}
	defer rows.Close()

	type StopTimeData struct {
		StopID        string
		StopSequence  int
		ArrivalTime   string
		DepartureTime string
		Latitude      float64
		Longitude     float64
	}

	var stopTimes []StopTimeData
	for rows.Next() {
		var st StopTimeData
		err := rows.Scan(&st.StopID, &st.StopSequence, &st.ArrivalTime, &st.DepartureTime,
			&st.Latitude, &st.Longitude)
		if err != nil {
			continue
		}
		stopTimes = append(stopTimes, st)
	}

	if len(stopTimes) == 0 {
		return nil, fmt.Errorf("no stop times found for trip %s", tripID)
	}

	// Calculate total trip duration from schedule
	firstDepartureSec := parseTimeToSeconds(stopTimes[0].DepartureTime)
	lastArrivalSec := parseTimeToSeconds(stopTimes[len(stopTimes)-1].ArrivalTime)
	if lastArrivalSec < firstDepartureSec {
		lastArrivalSec += 24 * 3600 // Handle overnight trips
	}
	totalTripDurationSec := lastArrivalSec - firstDepartureSec

	// Convert elapsed time to seconds
	elapsedSec := int(elapsedTime.Seconds())

	// If elapsed time exceeds trip duration, vehicle is at last stop
	if elapsedSec >= totalTripDurationSec {
		lastStop := stopTimes[len(stopTimes)-1]
		return &models.VehicleLocation{
			Latitude:  lastStop.Latitude,
			Longitude: lastStop.Longitude,
		}, nil
	}

	// Find which segment the vehicle is in based on elapsed time
	// Calculate cumulative time to each stop
	cumulativeTime := 0
	for i := 0; i < len(stopTimes)-1; i++ {
		currentStop := stopTimes[i]
		nextStop := stopTimes[i+1]

		departureSec := parseTimeToSeconds(currentStop.DepartureTime)
		arrivalSec := parseTimeToSeconds(nextStop.ArrivalTime)
		if arrivalSec < departureSec {
			arrivalSec += 24 * 3600
		}

		segmentDuration := arrivalSec - departureSec

		// Check if elapsed time falls within this segment
		if elapsedSec >= cumulativeTime && elapsedSec < cumulativeTime+segmentDuration {
			// Vehicle is in this segment
			segmentProgress := float64(elapsedSec-cumulativeTime) / float64(segmentDuration)

			// Interpolate position between stops
			latDiff := nextStop.Latitude - currentStop.Latitude
			lonDiff := nextStop.Longitude - currentStop.Longitude

			lat := currentStop.Latitude + (latDiff * segmentProgress)
			lon := currentStop.Longitude + (lonDiff * segmentProgress)

			// Calculate bearing
			bearing := s.calculateBearing(currentStop.Latitude, currentStop.Longitude,
				nextStop.Latitude, nextStop.Longitude)

			// Calculate speed (distance / time)
			distance := s.calculateDistance(currentStop.Latitude, currentStop.Longitude,
				nextStop.Latitude, nextStop.Longitude)
			speed := (distance / float64(segmentDuration)) * 3.6 // Convert m/s to km/h

			return &models.VehicleLocation{
				Latitude:  lat,
				Longitude: lon,
				Bearing:   &bearing,
				Speed:     &speed,
			}, nil
		}

		cumulativeTime += segmentDuration
	}

	// If before first departure, vehicle is at first stop
	return &models.VehicleLocation{
		Latitude:  stopTimes[0].Latitude,
		Longitude: stopTimes[0].Longitude,
	}, nil
}

// parseTimeToSeconds converts HH:MM:SS time string to seconds since midnight
func parseTimeToSeconds(timeStr string) int {
	var h, m, s int
	fmt.Sscanf(timeStr, "%d:%d:%d", &h, &m, &s)
	return h*3600 + m*60 + s
}
