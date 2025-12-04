package services

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type RoutePlanner struct {
	db            *database.DB
	stopService   *StopService
	routeService  *RouteService
	fareService   *FareService
	maxTransfers  int
	maxWalkMeters float64
}

func NewRoutePlanner(db *database.DB, stopService *StopService, routeService *RouteService, fareService *FareService) *RoutePlanner {
	return &RoutePlanner{
		db:            db,
		stopService:   stopService,
		routeService:  routeService,
		fareService:   fareService,
		maxTransfers:  3,
		maxWalkMeters: 1000.0, // 1km max walking distance
	}
}

func (rp *RoutePlanner) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	// Find nearest stops to origin and destination, ensuring mix of metro and bus
	originStops, err := rp.findNearbyStopsWithModeMix(req.FromLat, req.FromLon, rp.maxWalkMeters, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to find origin stops: %w", err)
	}
	if len(originStops) == 0 {
		return nil, fmt.Errorf("no stops found near origin")
	}

	destStops, err := rp.findNearbyStopsWithModeMix(req.ToLat, req.ToLon, rp.maxWalkMeters, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to find destination stops: %w", err)
	}
	if len(destStops) == 0 {
		return nil, fmt.Errorf("no stops found near destination")
	}

	// Determine the date for the journey (for day-specific service plans)
	journeyDate := time.Now()
	if req.Date != nil {
		journeyDate = *req.Date
	}

	// Use current time if departure time not specified
	departureTime := time.Now()
	if req.DepartureTime != nil {
		departureTime = *req.DepartureTime
	}

	// Plan journeys from each origin stop to each destination stop
	var allOptions []models.JourneyOption

	for _, originStop := range originStops {
		for _, destStop := range destStops {
			options, err := rp.planBetweenStops(originStop.ID, destStop.ID, departureTime, journeyDate)
			if err != nil {
				continue // Skip if no route found
			}

			// Add walking legs to/from stops
			for i := range options {
				originWalkTime := rp.calculateWalkTime(req.FromLat, req.FromLon, originStop.Latitude, originStop.Longitude)
				destWalkTime := rp.calculateWalkTime(destStop.Latitude, destStop.Longitude, req.ToLat, req.ToLon)

				// Add origin walking leg
				if originWalkTime > 0 {
					walkLeg := models.JourneyLeg{
						Mode:         "walking",
						FromStopID:   "",
						FromStopName: "Origin",
						ToStopID:     originStop.ID,
						ToStopName:   originStop.Name,
						Duration:     originWalkTime,
						StopCount:    0,
					}
					options[i].Legs = append([]models.JourneyLeg{walkLeg}, options[i].Legs...)
					options[i].WalkingTime += originWalkTime
					options[i].Duration += originWalkTime
				}

				// Add destination walking leg
				if destWalkTime > 0 {
					walkLeg := models.JourneyLeg{
						Mode:         "walking",
						FromStopID:   destStop.ID,
						FromStopName: destStop.Name,
						ToStopID:     "",
						ToStopName:   "Destination",
						Duration:     destWalkTime,
						StopCount:    0,
					}
					options[i].Legs = append(options[i].Legs, walkLeg)
					options[i].WalkingTime += destWalkTime
					options[i].Duration += destWalkTime
				}

				// Update departure/arrival times
				options[i].DepartureTime = departureTime.Add(time.Duration(-originWalkTime) * time.Minute)
				options[i].ArrivalTime = options[i].DepartureTime.Add(time.Duration(options[i].Duration) * time.Minute)
			}

			allOptions = append(allOptions, options...)
		}
	}

	// Calculate fare for each option
	// Use default rules (will be overridden per-leg if needed)
	defaultRules := rp.fareService.GetFareRulesForAgency("DIMTS")
	for i := range allOptions {
		// For multi-modal journeys, we'll use the first leg's agency
		// In a more sophisticated implementation, we'd calculate fare per leg
		if len(allOptions[i].Legs) > 0 {
			firstRouteID := allOptions[i].Legs[0].RouteID
			if firstRouteID != "" {
				agencyID := rp.fareService.GetAgencyIDFromRoute(firstRouteID)
				if agencyID != "" {
					defaultRules = rp.fareService.GetFareRulesForAgency(agencyID)
				}
			}
		}
		fare := rp.fareService.CalculateFareForJourney(allOptions[i], defaultRules)
		allOptions[i].Fare = &fare
	}

	// Sort by fastest journey: prioritize total duration, then transfers, then walking time
	sort.Slice(allOptions, func(i, j int) bool {
		optI := allOptions[i]
		optJ := allOptions[j]

		// Primary: Total duration (including walking)
		totalDurationI := optI.Duration + optI.WalkingTime
		totalDurationJ := optJ.Duration + optJ.WalkingTime

		if totalDurationI != totalDurationJ {
			return totalDurationI < totalDurationJ
		}

		// Secondary: Fewer transfers is better
		if optI.Transfers != optJ.Transfers {
			return optI.Transfers < optJ.Transfers
		}

		// Tertiary: Less walking time is better
		if optI.WalkingTime != optJ.WalkingTime {
			return optI.WalkingTime < optJ.WalkingTime
		}

		// Quaternary: Shorter journey duration (excluding walking)
		return optI.Duration < optJ.Duration
	})

	// Return up to 15 options to show variety (metro, bus, different routes)
	if len(allOptions) > 15 {
		allOptions = allOptions[:15]
	}

	return allOptions, nil
}

func (rp *RoutePlanner) planBetweenStops(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	var allOptions []models.JourneyOption

	// Find direct routes (multiple options from different routes/times)
	directOptions, err := rp.findDirectRoutes(fromStopID, toStopID, departureTime, journeyDate)
	if err == nil && len(directOptions) > 0 {
		allOptions = append(allOptions, directOptions...)
	}

	// Also try with transfers to get alternative routes (only if no direct routes found)
	if len(allOptions) == 0 {
		transferOptions, err := rp.findRoutesWithTransfers(fromStopID, toStopID, departureTime, journeyDate, 0)
		if err == nil && len(transferOptions) > 0 {
			allOptions = append(allOptions, transferOptions...)
		}
	}

	if len(allOptions) == 0 {
		return nil, fmt.Errorf("no route found between stops %s and %s", fromStopID, toStopID)
	}

	// Remove duplicates and sort by fastest journey
	uniqueOptions := rp.removeDuplicateOptions(allOptions)
	sort.Slice(uniqueOptions, func(i, j int) bool {
		optI := uniqueOptions[i]
		optJ := uniqueOptions[j]

		// Primary: Total duration (including walking)
		totalDurationI := optI.Duration + optI.WalkingTime
		totalDurationJ := optJ.Duration + optJ.WalkingTime

		if totalDurationI != totalDurationJ {
			return totalDurationI < totalDurationJ
		}

		// Secondary: Fewer transfers is better
		if optI.Transfers != optJ.Transfers {
			return optI.Transfers < optJ.Transfers
		}

		// Tertiary: Less walking time is better
		if optI.WalkingTime != optJ.WalkingTime {
			return optI.WalkingTime < optJ.WalkingTime
		}

		// Quaternary: Shorter journey duration (excluding walking)
		return optI.Duration < optJ.Duration
	})

	// Return up to 10 options
	if len(uniqueOptions) > 10 {
		uniqueOptions = uniqueOptions[:10]
	}

	return uniqueOptions, nil
}

func (rp *RoutePlanner) findDirectRoutes(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	query := `SELECT DISTINCT ON (t.route_id, st1.departure_time) 
		t.route_id, r.route_short_name, r.route_long_name, 
		r.route_type, st1.trip_id, st1.departure_time, st2.arrival_time, st2.stop_sequence - st1.stop_sequence as stop_count
	FROM stop_times st1
	JOIN stop_times st2 ON st1.trip_id = st2.trip_id
	JOIN trips t ON st1.trip_id = t.trip_id
	JOIN routes r ON t.route_id = r.route_id
	JOIN calendar cal ON t.service_id = cal.service_id
	WHERE st1.stop_id = $1 
		AND st2.stop_id = $2
		AND st1.stop_sequence < st2.stop_sequence
		AND cal.start_date <= $4::date
		AND cal.end_date >= $4::date - INTERVAL '1 year'
		AND (
			(EXTRACT(DOW FROM $4::date) = 0 AND cal.sunday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 1 AND cal.monday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 2 AND cal.tuesday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 3 AND cal.wednesday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 4 AND cal.thursday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 5 AND cal.friday = 1) OR
			(EXTRACT(DOW FROM $4::date) = 6 AND cal.saturday = 1)
		)
		AND st1.departure_time >= $3
	ORDER BY st1.departure_time, t.route_id
	LIMIT 20`

	departureTimeStr := departureTime.Format("15:04:05")
	journeyDateStr := journeyDate.Format("2006-01-02")
	rows, err := rp.db.Query(query, fromStopID, toStopID, departureTimeStr, journeyDateStr)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var options []models.JourneyOption
	for rows.Next() {
		var routeID, routeShortName, routeLongName, tripID, depTime, arrTime string
		var routeType, stopCount int

		err := rows.Scan(&routeID, &routeShortName, &routeLongName, &routeType, &tripID, &depTime, &arrTime, &stopCount)
		if err != nil {
			continue
		}

		fromStop, err := rp.stopService.GetByID(fromStopID)
		if err != nil {
			continue
		}

		toStop, err := rp.stopService.GetByID(toStopID)
		if err != nil {
			continue
		}

		depTimeParsed, err := time.Parse("15:04:05", depTime)
		if err != nil {
			continue
		}

		arrTimeParsed, err := time.Parse("15:04:05", arrTime)
		if err != nil {
			continue
		}

		// Adjust times to the journey date
		depTimeToday := time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(),
			depTimeParsed.Hour(), depTimeParsed.Minute(), depTimeParsed.Second(), 0, journeyDate.Location())
		arrTimeToday := time.Date(journeyDate.Year(), journeyDate.Month(), journeyDate.Day(),
			arrTimeParsed.Hour(), arrTimeParsed.Minute(), arrTimeParsed.Second(), 0, journeyDate.Location())

		if arrTimeToday.Before(depTimeToday) {
			arrTimeToday = arrTimeToday.Add(24 * time.Hour)
		}

		duration := int(arrTimeToday.Sub(depTimeToday).Minutes())

		// Determine transport mode based on route_type
		// GTFS route_type: 1 = Metro/Subway, 3 = Bus
		mode := "bus"
		if routeType == 1 {
			mode = "metro"
		}

		// Use route_long_name if route_short_name is empty
		routeName := routeShortName
		if routeName == "" {
			routeName = routeLongName
		}

		leg := models.JourneyLeg{
			Mode:          mode,
			RouteID:       routeID,
			RouteName:     routeName,
			FromStopID:    fromStopID,
			FromStopName:  fromStop.Name,
			ToStopID:      toStopID,
			ToStopName:    toStop.Name,
			DepartureTime: depTimeToday,
			ArrivalTime:   arrTimeToday,
			Duration:      duration,
			StopCount:     stopCount,
		}

		option := models.JourneyOption{
			Duration:      duration,
			Transfers:     0,
			WalkingTime:   0,
			Legs:          []models.JourneyLeg{leg},
			DepartureTime: depTimeToday,
			ArrivalTime:   arrTimeToday,
		}

		options = append(options, option)
	}

	return options, nil
}

func (rp *RoutePlanner) findRoutesWithTransfers(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time, currentTransfers int) ([]models.JourneyOption, error) {
	if currentTransfers >= rp.maxTransfers {
		return nil, fmt.Errorf("max transfers reached")
	}

	// Find all routes that pass through origin stop
	originRoutes, err := rp.getRoutesForStop(fromStopID)
	if err != nil {
		return nil, err
	}

	var options []models.JourneyOption

	for _, route := range originRoutes {
		// Get all stops on this route after origin
		stops, err := rp.getStopsOnRouteAfterStop(route.ID, fromStopID)
		if err != nil {
			continue
		}

		// Check if destination is on this route
		for _, stop := range stops {
			if stop.ID == toStopID {
				// Direct route found
				directOptions, _ := rp.findDirectRoutes(fromStopID, toStopID, departureTime, journeyDate)
				options = append(options, directOptions...)
				continue
			}

			// Check if we can transfer here
			if currentTransfers < rp.maxTransfers {
				// Find routes from this stop to destination
				transferOptions, err := rp.findRoutesWithTransfers(stop.ID, toStopID, departureTime, journeyDate, currentTransfers+1)
				if err == nil {
					// Combine with first leg
					firstLegOptions, _ := rp.findDirectRoutes(fromStopID, stop.ID, departureTime, journeyDate)
					for _, firstLeg := range firstLegOptions {
						for _, transferOption := range transferOptions {
							combined := models.JourneyOption{
								Duration:      firstLeg.Duration + transferOption.Duration,
								Transfers:     currentTransfers + 1,
								WalkingTime:   firstLeg.WalkingTime + transferOption.WalkingTime,
								Legs:          append(firstLeg.Legs, transferOption.Legs...),
								DepartureTime: firstLeg.DepartureTime,
								ArrivalTime:   transferOption.ArrivalTime,
							}
							options = append(options, combined)
						}
					}
				}
			}
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no route found")
	}

	return options, nil
}

func (rp *RoutePlanner) getRoutesForStop(stopID string) ([]models.Route, error) {
	query := `SELECT DISTINCT r.route_id, r.agency_id, r.route_short_name, r.route_long_name, 
		r.route_desc, r.route_type, r.route_url, r.route_color, r.route_text_color
	FROM routes r
	JOIN trips t ON r.route_id = t.route_id
	JOIN stop_times st ON t.trip_id = st.trip_id
	WHERE st.stop_id = $1`

	rows, err := rp.db.Query(query, stopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		route := models.Route{}
		err := rows.Scan(&route.ID, &route.AgencyID, &route.ShortName, &route.LongName,
			&route.Description, &route.Type, &route.URL, &route.Color, &route.TextColor)
		if err != nil {
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (rp *RoutePlanner) getStopsOnRouteAfterStop(routeID, stopID string) ([]models.Stop, error) {
	query := `SELECT DISTINCT s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon,
		s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
	FROM stops s
	JOIN stop_times st ON s.stop_id = st.stop_id
	JOIN trips t ON st.trip_id = t.trip_id
	WHERE t.route_id = $1
		AND st.stop_sequence > (SELECT MIN(st2.stop_sequence) FROM stop_times st2 
			JOIN trips t2 ON st2.trip_id = t2.trip_id 
			WHERE t2.route_id = $1 AND st2.stop_id = $2)
	ORDER BY MIN(st.stop_sequence) OVER (PARTITION BY t.trip_id, s.stop_id)`

	rows, err := rp.db.Query(query, routeID, stopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []models.Stop
	seenStops := make(map[string]bool)

	for rows.Next() {
		stop := models.Stop{}
		var parentStation sql.NullString

		err := rows.Scan(&stop.ID, &stop.Code, &stop.Name, &stop.Description,
			&stop.Latitude, &stop.Longitude, &stop.ZoneID, &stop.URL,
			&stop.LocationType, &parentStation, &stop.Timezone, &stop.WheelchairBoarding)
		if err != nil {
			continue
		}

		if !seenStops[stop.ID] {
			if parentStation.Valid {
				stop.ParentStation = parentStation.String
			}
			stops = append(stops, stop)
			seenStops[stop.ID] = true
		}
	}

	return stops, nil
}

func (rp *RoutePlanner) calculateWalkTime(lat1, lon1, lat2, lon2 float64) int {
	distance := haversineDistance(lat1, lon1, lat2, lon2)
	// Average walking speed: 5 km/h = 83.33 m/min
	walkTimeMinutes := int(math.Ceil(distance / 83.33))
	return walkTimeMinutes
}

// removeDuplicateOptions removes duplicate journey options based on route and stops
func (rp *RoutePlanner) removeDuplicateOptions(options []models.JourneyOption) []models.JourneyOption {
	seen := make(map[string]bool)
	var unique []models.JourneyOption

	for _, option := range options {
		// Create a key based on route IDs and stop sequence
		key := ""
		for _, leg := range option.Legs {
			if leg.RouteID != "" {
				key += leg.RouteID + ":" + leg.FromStopID + "->" + leg.ToStopID + "|"
			}
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, option)
		}
	}

	return unique
}

// findNearbyStopsWithModeMix finds nearby stops ensuring a mix of metro and bus stops
// It returns up to limit stops, prioritizing equal representation from both modes
func (rp *RoutePlanner) findNearbyStopsWithModeMix(lat, lon float64, radiusMeters float64, limit int) ([]models.Stop, error) {
	// Find all nearby stops - increase limit to ensure we get metro stops like Rajiv Chowk
	// Rajiv Chowk is ~330m away, so we need at least 30-40 stops to include it
	allStops, err := rp.stopService.FindNearby(lat, lon, radiusMeters, limit*5)
	if err != nil {
		return nil, err
	}

	if len(allStops) == 0 {
		return nil, fmt.Errorf("no stops found")
	}

	// Separate stops by mode (metro vs bus)
	var metroStops []models.Stop
	var busStops []models.Stop
	var otherStops []models.Stop

	for _, stop := range allStops {
		// Check if this stop has metro routes (route_type = 1) or bus routes (route_type = 3)
		hasMetro, hasBus := rp.stopHasModeTypes(stop.ID)

		if hasMetro && !hasBus {
			metroStops = append(metroStops, stop)
		} else if hasBus && !hasMetro {
			busStops = append(busStops, stop)
		} else if hasMetro && hasBus {
			// Stop serves both modes - add to both lists but only count once
			metroStops = append(metroStops, stop)
			busStops = append(busStops, stop)
		} else {
			otherStops = append(otherStops, stop)
		}
	}

	// Combine stops ensuring equal representation
	var result []models.Stop
	seen := make(map[string]bool)

	// Prioritize metro stops first, then bus stops, ensuring we get both modes
	// Take at least limit/2 from each mode if available
	targetPerMode := limit / 2
	if targetPerMode == 0 {
		targetPerMode = 1
	}

	// First, add metro stops (up to targetPerMode)
	metroAdded := 0
	for i := 0; i < len(metroStops) && metroAdded < targetPerMode && len(result) < limit; i++ {
		if !seen[metroStops[i].ID] {
			result = append(result, metroStops[i])
			seen[metroStops[i].ID] = true
			metroAdded++
		}
	}

	// Then, add bus stops (up to targetPerMode)
	busAdded := 0
	for i := 0; i < len(busStops) && busAdded < targetPerMode && len(result) < limit; i++ {
		if !seen[busStops[i].ID] {
			result = append(result, busStops[i])
			seen[busStops[i].ID] = true
			busAdded++
		}
	}

	// Fill remaining slots by interleaving remaining metro and bus stops
	maxRemaining := len(metroStops)
	if len(busStops) > maxRemaining {
		maxRemaining = len(busStops)
	}

	for i := targetPerMode; i < maxRemaining && len(result) < limit; i++ {
		// Alternate between metro and bus
		if i < len(metroStops) && !seen[metroStops[i].ID] {
			result = append(result, metroStops[i])
			seen[metroStops[i].ID] = true
			if len(result) >= limit {
				break
			}
		}

		if i < len(busStops) && !seen[busStops[i].ID] {
			result = append(result, busStops[i])
			seen[busStops[i].ID] = true
			if len(result) >= limit {
				break
			}
		}
	}

	// Fill remaining slots with other stops or more from metro/bus
	for _, stop := range allStops {
		if len(result) >= limit {
			break
		}
		if !seen[stop.ID] {
			result = append(result, stop)
			seen[stop.ID] = true
		}
	}

	return result, nil
}

// stopHasModeTypes checks if a stop has metro (route_type=1) or bus (route_type=3) routes
func (rp *RoutePlanner) stopHasModeTypes(stopID string) (hasMetro, hasBus bool) {
	query := `SELECT DISTINCT r.route_type 
		FROM routes r
		JOIN trips t ON r.route_id = t.route_id
		JOIN stop_times st ON t.trip_id = st.trip_id
		WHERE st.stop_id = $1 AND r.route_type IN (1, 3)`

	rows, err := rp.db.Query(query, stopID)
	if err != nil {
		return false, false
	}
	defer rows.Close()

	for rows.Next() {
		var routeType int
		if err := rows.Scan(&routeType); err != nil {
			continue
		}
		if routeType == 1 {
			hasMetro = true
		} else if routeType == 3 {
			hasBus = true
		}
	}

	return hasMetro, hasBus
}
