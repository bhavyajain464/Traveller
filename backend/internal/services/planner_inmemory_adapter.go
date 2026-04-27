package services

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

// InMemoryJourneyPlannerAdapter owns an in-process timetable snapshot. Direct
// journey discovery now runs against that snapshot, while transfer-heavy cases
// still fall back to the SQL planner until the full in-memory engine lands.
type InMemoryJourneyPlannerAdapter struct {
	db            *database.DB
	fallback      JourneyPlannerAdapter
	fareService   *FareService
	maxWalkMeters float64
	maxTransfers  int
	snapshot      atomic.Pointer[PlannerSnapshot]
}

type plannerStopTime struct {
	StopTime         models.StopTime
	ArrivalSeconds   int
	DepartureSeconds int
}

type plannerStopVisit struct {
	TripID           string
	StopSequence     int
	DepartureSeconds int
}

type plannerFootpath struct {
	FromStopID string
	ToStopID   string
	Duration   int
	Distance   float64
	Indoor     bool
}

type roundCandidate struct {
	AtStopID       string
	ArrivalTime    time.Time
	ArrivalSeconds int
	Legs           []models.JourneyLeg
	Transfers      int
}

type raptorRoundState struct {
	bestByStop map[string]roundCandidate
	marked     map[string]struct{}
}

type scoredStop struct {
	stop     models.Stop
	distance float64
}

func NewInMemoryJourneyPlannerAdapter(db *database.DB, fallback JourneyPlannerAdapter, fareService *FareService) (*InMemoryJourneyPlannerAdapter, error) {
	adapter := &InMemoryJourneyPlannerAdapter{
		db:            db,
		fallback:      fallback,
		fareService:   fareService,
		maxWalkMeters: 3000,
		maxTransfers:  2,
	}

	if err := adapter.ReloadSnapshot(); err != nil {
		return nil, err
	}

	return adapter, nil
}

func (a *InMemoryJourneyPlannerAdapter) Engine() string {
	return "in_memory_snapshot(" + a.fallback.Engine() + ")"
}

func (a *InMemoryJourneyPlannerAdapter) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	snapshot := a.snapshot.Load()
	if snapshot == nil || snapshot.StopCount() == 0 {
		return a.fallback.PlanJourney(req)
	}

	requestedJourneyDate := time.Now()
	if req.Date != nil {
		requestedJourneyDate = *req.Date
	}

	resolvedJourneyDate := a.resolveJourneyDate(snapshot, requestedJourneyDate)
	departureTime := a.resolveDepartureTime(req.DepartureTime, requestedJourneyDate, resolvedJourneyDate)

	originStops := a.findCandidateStops(snapshot, req.FromLat, req.FromLon, 4)
	if len(originStops) == 0 {
		return nil, fmt.Errorf("no stops found near origin")
	}

	destStops := a.findCandidateStops(snapshot, req.ToLat, req.ToLon, 4)
	if len(destStops) == 0 {
		return nil, fmt.Errorf("no stops found near destination")
	}

	var allOptions []models.JourneyOption
	for _, originStop := range originStops {
		for _, destStop := range destStops {
			options := a.planRounds(snapshot, originStop.ID, destStop.ID, departureTime, resolvedJourneyDate)
			if len(options) == 0 {
				continue
			}

			options = a.addAccessAndEgressLegs(req, originStop, destStop, departureTime, options)
			allOptions = append(allOptions, options...)
		}
	}

	if len(allOptions) == 0 {
		return a.fallback.PlanJourney(req)
	}

	filtered := allOptions[:0]
	for _, option := range allOptions {
		if option.Duration != math.MaxInt {
			filtered = append(filtered, option)
		}
	}

	filtered = a.applyFares(filtered)
	filtered = a.removeDuplicateOptions(filtered)

	sort.Slice(filtered, func(i, j int) bool {
		optI := filtered[i]
		optJ := filtered[j]

		if optI.Duration != optJ.Duration {
			return optI.Duration < optJ.Duration
		}
		if optI.Transfers != optJ.Transfers {
			return optI.Transfers < optJ.Transfers
		}
		if optI.WalkingTime != optJ.WalkingTime {
			return optI.WalkingTime < optJ.WalkingTime
		}
		return optI.ArrivalTime.Before(optJ.ArrivalTime)
	})

	if len(filtered) > 10 {
		filtered = filtered[:10]
	}

	return filtered, nil
}

func (a *InMemoryJourneyPlannerAdapter) PlanJourneyBetweenStops(req StopJourneyRequest) ([]models.JourneyOption, error) {
	snapshot := a.snapshot.Load()
	if snapshot == nil || snapshot.StopCount() == 0 {
		return a.fallback.PlanJourneyBetweenStops(req)
	}

	fromStopID := strings.TrimSpace(req.FromStopID)
	toStopID := strings.TrimSpace(req.ToStopID)
	if fromStopID == "" || toStopID == "" {
		return nil, fmt.Errorf("from_stop_id and to_stop_id are required")
	}

	originStop, ok := snapshot.StopsByID[fromStopID]
	if !ok {
		return nil, fmt.Errorf("from stop %q not found in planner snapshot", fromStopID)
	}
	destStop, ok := snapshot.StopsByID[toStopID]
	if !ok {
		return nil, fmt.Errorf("to stop %q not found in planner snapshot", toStopID)
	}

	requestedJourneyDate := time.Now()
	if req.Date != nil {
		requestedJourneyDate = *req.Date
	}

	resolvedJourneyDate := a.resolveJourneyDate(snapshot, requestedJourneyDate)
	departureTime := a.resolveDepartureTime(req.DepartureTime, requestedJourneyDate, resolvedJourneyDate)

	options := a.planRounds(snapshot, originStop.ID, destStop.ID, departureTime, resolvedJourneyDate)
	if len(options) == 0 {
		return a.fallback.PlanJourneyBetweenStops(req)
	}

	options = a.applyFares(options)
	options = a.removeDuplicateOptions(options)

	sort.Slice(options, func(i, j int) bool {
		optI := options[i]
		optJ := options[j]

		if optI.Duration != optJ.Duration {
			return optI.Duration < optJ.Duration
		}
		if optI.Transfers != optJ.Transfers {
			return optI.Transfers < optJ.Transfers
		}
		if optI.WalkingTime != optJ.WalkingTime {
			return optI.WalkingTime < optJ.WalkingTime
		}
		return optI.ArrivalTime.Before(optJ.ArrivalTime)
	})

	if len(options) > 10 {
		options = options[:10]
	}

	return options, nil
}

func (a *InMemoryJourneyPlannerAdapter) Snapshot() *PlannerSnapshot {
	return a.snapshot.Load()
}

func (a *InMemoryJourneyPlannerAdapter) ReloadSnapshot() error {
	stops, err := a.loadStops()
	if err != nil {
		return fmt.Errorf("load planner stops snapshot: %w", err)
	}

	routes, err := a.loadRoutes()
	if err != nil {
		return fmt.Errorf("load planner routes snapshot: %w", err)
	}

	trips, err := a.loadTrips()
	if err != nil {
		return fmt.Errorf("load planner trips snapshot: %w", err)
	}

	footpaths, footpathsFromStopID, err := a.loadFootpaths()
	if err != nil {
		return fmt.Errorf("load planner footpaths snapshot: %w", err)
	}

	calendars, minServiceDate, maxServiceDate, err := a.loadCalendars()
	if err != nil {
		return fmt.Errorf("load planner calendars snapshot: %w", err)
	}

	stopTimesByTripID, stopVisitsByStopID, routeIDsByStopID, tripIDsByRouteID, stopModeTypes, err := a.loadStopTimesAndIndexes(routes)
	if err != nil {
		return fmt.Errorf("load planner stop times snapshot: %w", err)
	}

	stopsByID := make(map[string]models.Stop, len(stops))
	for _, stop := range stops {
		stopsByID[stop.ID] = stop
	}

	routesByID := make(map[string]models.Route, len(routes))
	for _, route := range routes {
		routesByID[route.ID] = route
	}

	tripsByID := make(map[string]models.Trip, len(trips))
	for _, trip := range trips {
		tripsByID[trip.ID] = trip
	}

	a.snapshot.Store(&PlannerSnapshot{
		Version:              time.Now().UTC().Format(time.RFC3339),
		LoadedAt:             time.Now().UTC(),
		Stops:                stops,
		Routes:               routes,
		Trips:                trips,
		Footpaths:            footpaths,
		StopsByID:            stopsByID,
		RoutesByID:           routesByID,
		TripsByID:            tripsByID,
		CalendarsByServiceID: calendars,
		StopTimesByTripID:    stopTimesByTripID,
		StopVisitsByStopID:   stopVisitsByStopID,
		RouteIDsByStopID:     routeIDsByStopID,
		TripIDsByRouteID:     tripIDsByRouteID,
		FootpathsFromStopID:  footpathsFromStopID,
		StopModeTypes:        stopModeTypes,
		MinServiceDate:       minServiceDate,
		MaxServiceDate:       maxServiceDate,
	})

	return nil
}

func (a *InMemoryJourneyPlannerAdapter) loadStops() ([]models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url,
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stops := make([]models.Stop, 0, 4096)
	for rows.Next() {
		var stop models.Stop
		if err := scanStopRow(rows, &stop, nil); err != nil {
			return nil, err
		}
		stops = append(stops, stop)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stops, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadRoutes() ([]models.Route, error) {
	query := `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc,
		route_type, route_url, route_color, route_text_color
		FROM routes`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routes := make([]models.Route, 0, 2048)
	for rows.Next() {
		var route models.Route
		if err := scanRouteRow(rows, &route); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadTrips() ([]models.Trip, error) {
	query := `SELECT trip_id, route_id, service_id, trip_headsign, trip_short_name, direction_id,
		block_id, shape_id, wheelchair_accessible, bikes_allowed
		FROM trips`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trips := make([]models.Trip, 0, 4096)
	for rows.Next() {
		var trip models.Trip
		if err := scanTripRow(rows, &trip); err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trips, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadFootpaths() ([]plannerFootpath, map[string][]plannerFootpath, error) {
	query := `SELECT from_stop_id, to_stop_id, duration_seconds, distance_meters, indoor
		FROM planner_footpaths
		ORDER BY from_stop_id, to_stop_id`

	rows, err := a.db.Query(query)
	if err != nil {
		if strings.Contains(err.Error(), `relation "planner_footpaths" does not exist`) {
			return []plannerFootpath{}, map[string][]plannerFootpath{}, nil
		}
		return nil, nil, err
	}
	defer rows.Close()

	footpaths := make([]plannerFootpath, 0, 128)
	footpathsFromStopID := make(map[string][]plannerFootpath)
	for rows.Next() {
		var footpath plannerFootpath
		if err := rows.Scan(&footpath.FromStopID, &footpath.ToStopID, &footpath.Duration, &footpath.Distance, &footpath.Indoor); err != nil {
			return nil, nil, err
		}
		footpaths = append(footpaths, footpath)
		footpathsFromStopID[footpath.FromStopID] = append(footpathsFromStopID[footpath.FromStopID], footpath)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return footpaths, footpathsFromStopID, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadCalendars() (map[string]models.Calendar, time.Time, time.Time, error) {
	query := `SELECT service_id, monday, tuesday, wednesday, thursday, friday, saturday, sunday, start_date, end_date
		FROM calendar`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	defer rows.Close()

	calendars := make(map[string]models.Calendar)
	var minDate, maxDate time.Time
	for rows.Next() {
		var calendar models.Calendar
		if err := rows.Scan(
			&calendar.ServiceID,
			&calendar.Monday,
			&calendar.Tuesday,
			&calendar.Wednesday,
			&calendar.Thursday,
			&calendar.Friday,
			&calendar.Saturday,
			&calendar.Sunday,
			&calendar.StartDate,
			&calendar.EndDate,
		); err != nil {
			return nil, time.Time{}, time.Time{}, err
		}

		startDate, err := time.Parse("2006-01-02", calendar.StartDate)
		if err == nil && (minDate.IsZero() || startDate.Before(minDate)) {
			minDate = startDate
		}

		endDate, err := time.Parse("2006-01-02", calendar.EndDate)
		if err == nil && (maxDate.IsZero() || endDate.After(maxDate)) {
			maxDate = endDate
		}

		calendars[calendar.ServiceID] = calendar
	}

	if err := rows.Err(); err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	return calendars, minDate, maxDate, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadStopTimesAndIndexes(routes []models.Route) (map[string][]plannerStopTime, map[string][]plannerStopVisit, map[string][]string, map[string][]string, map[string]map[int]struct{}, error) {
	query := `SELECT trip_id, arrival_time, departure_time, stop_id, stop_sequence, stop_headsign,
		pickup_type, drop_off_type, shape_dist_traveled, timepoint
		FROM stop_times
		ORDER BY trip_id, stop_sequence`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer rows.Close()

	routeTypeByID := make(map[string]int, len(routes))
	for _, route := range routes {
		routeTypeByID[route.ID] = route.Type
	}

	tripRouteMap, err := a.loadTripRouteIDs()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	stopTimesByTripID := make(map[string][]plannerStopTime)
	stopVisitsByStopID := make(map[string][]plannerStopVisit)
	routeIDsByStopID := make(map[string][]string)
	tripIDsByRouteID := make(map[string][]string)
	stopModeTypes := make(map[string]map[int]struct{})
	seenRouteByStopID := make(map[string]map[string]struct{})
	seenTripByRouteID := make(map[string]map[string]struct{})

	for rows.Next() {
		var stopTime models.StopTime
		if err := scanStopTimeRow(rows, &stopTime); err != nil {
			return nil, nil, nil, nil, nil, err
		}

		arrivalSeconds, err := gtfsTimeToSeconds(stopTime.ArrivalTime)
		if err != nil {
			continue
		}
		departureSeconds, err := gtfsTimeToSeconds(stopTime.DepartureTime)
		if err != nil {
			continue
		}

		entry := plannerStopTime{
			StopTime:         stopTime,
			ArrivalSeconds:   arrivalSeconds,
			DepartureSeconds: departureSeconds,
		}
		stopTimesByTripID[stopTime.TripID] = append(stopTimesByTripID[stopTime.TripID], entry)
		stopVisitsByStopID[stopTime.StopID] = append(stopVisitsByStopID[stopTime.StopID], plannerStopVisit{
			TripID:           stopTime.TripID,
			StopSequence:     stopTime.StopSequence,
			DepartureSeconds: departureSeconds,
		})

		routeID := tripRouteMap[stopTime.TripID]
		if _, ok := seenRouteByStopID[stopTime.StopID]; !ok {
			seenRouteByStopID[stopTime.StopID] = make(map[string]struct{})
		}
		if _, ok := seenRouteByStopID[stopTime.StopID][routeID]; !ok && routeID != "" {
			seenRouteByStopID[stopTime.StopID][routeID] = struct{}{}
			routeIDsByStopID[stopTime.StopID] = append(routeIDsByStopID[stopTime.StopID], routeID)
		}

		if _, ok := seenTripByRouteID[routeID]; !ok {
			seenTripByRouteID[routeID] = make(map[string]struct{})
		}
		if _, ok := seenTripByRouteID[routeID][stopTime.TripID]; !ok && routeID != "" {
			seenTripByRouteID[routeID][stopTime.TripID] = struct{}{}
			tripIDsByRouteID[routeID] = append(tripIDsByRouteID[routeID], stopTime.TripID)
		}

		routeType := routeTypeByID[routeID]
		if _, ok := stopModeTypes[stopTime.StopID]; !ok {
			stopModeTypes[stopTime.StopID] = make(map[int]struct{})
		}
		stopModeTypes[stopTime.StopID][routeType] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	for stopID := range stopVisitsByStopID {
		sort.Slice(stopVisitsByStopID[stopID], func(i, j int) bool {
			if stopVisitsByStopID[stopID][i].DepartureSeconds == stopVisitsByStopID[stopID][j].DepartureSeconds {
				if stopVisitsByStopID[stopID][i].TripID == stopVisitsByStopID[stopID][j].TripID {
					return stopVisitsByStopID[stopID][i].StopSequence < stopVisitsByStopID[stopID][j].StopSequence
				}
				return stopVisitsByStopID[stopID][i].TripID < stopVisitsByStopID[stopID][j].TripID
			}
			return stopVisitsByStopID[stopID][i].DepartureSeconds < stopVisitsByStopID[stopID][j].DepartureSeconds
		})
	}

	for stopID := range routeIDsByStopID {
		sort.Strings(routeIDsByStopID[stopID])
	}

	for routeID := range tripIDsByRouteID {
		sort.Slice(tripIDsByRouteID[routeID], func(i, j int) bool {
			left := tripIDsByRouteID[routeID][i]
			right := tripIDsByRouteID[routeID][j]
			leftTimes := stopTimesByTripID[left]
			rightTimes := stopTimesByTripID[right]
			if len(leftTimes) == 0 || len(rightTimes) == 0 {
				return left < right
			}
			if leftTimes[0].DepartureSeconds == rightTimes[0].DepartureSeconds {
				return left < right
			}
			return leftTimes[0].DepartureSeconds < rightTimes[0].DepartureSeconds
		})
	}

	return stopTimesByTripID, stopVisitsByStopID, routeIDsByStopID, tripIDsByRouteID, stopModeTypes, nil
}

func (a *InMemoryJourneyPlannerAdapter) loadTripRouteIDs() (map[string]string, error) {
	query := `SELECT trip_id, route_id FROM trips`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tripRouteIDs := make(map[string]string)
	for rows.Next() {
		var tripID, routeID string
		if err := rows.Scan(&tripID, &routeID); err != nil {
			return nil, err
		}
		tripRouteIDs[tripID] = routeID
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tripRouteIDs, nil
}

func (a *InMemoryJourneyPlannerAdapter) findCandidateStops(snapshot *PlannerSnapshot, lat, lon float64, limit int) []models.Stop {
	radii := []float64{1000, 2000, 3000, 5000}
	var lastStops []models.Stop

	for _, radius := range radii {
		stops := a.findNearbyStopsWithModeMix(snapshot, lat, lon, radius, limit)
		if len(stops) > 0 {
			lastStops = stops
		}
		if len(stops) >= minInt(limit, 4) {
			return stops
		}
	}

	return lastStops
}

func (a *InMemoryJourneyPlannerAdapter) findNearbyStopsWithModeMix(snapshot *PlannerSnapshot, lat, lon, radiusMeters float64, limit int) []models.Stop {
	allStops := a.findNearestStops(snapshot, lat, lon, radiusMeters, limit*5)
	if len(allStops) == 0 {
		return nil
	}

	var metroStops []models.Stop
	var busStops []models.Stop
	var otherStops []models.Stop

	for _, stop := range allStops {
		hasMetro, hasBus := a.stopHasModeTypes(snapshot, stop.ID)
		if hasMetro && !hasBus {
			metroStops = append(metroStops, stop)
		} else if hasBus && !hasMetro {
			busStops = append(busStops, stop)
		} else if hasMetro && hasBus {
			metroStops = append(metroStops, stop)
			busStops = append(busStops, stop)
		} else {
			otherStops = append(otherStops, stop)
		}
	}

	var result []models.Stop
	seen := make(map[string]bool)
	targetPerMode := limit / 2
	if targetPerMode == 0 {
		targetPerMode = 1
	}

	metroAdded := 0
	for i := 0; i < len(metroStops) && metroAdded < targetPerMode && len(result) < limit; i++ {
		if !seen[metroStops[i].ID] {
			result = append(result, metroStops[i])
			seen[metroStops[i].ID] = true
			metroAdded++
		}
	}

	busAdded := 0
	for i := 0; i < len(busStops) && busAdded < targetPerMode && len(result) < limit; i++ {
		if !seen[busStops[i].ID] {
			result = append(result, busStops[i])
			seen[busStops[i].ID] = true
			busAdded++
		}
	}

	maxRemaining := len(metroStops)
	if len(busStops) > maxRemaining {
		maxRemaining = len(busStops)
	}

	for i := targetPerMode; i < maxRemaining && len(result) < limit; i++ {
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

	for _, stop := range otherStops {
		if len(result) >= limit {
			break
		}
		if !seen[stop.ID] {
			result = append(result, stop)
			seen[stop.ID] = true
		}
	}

	for _, stop := range allStops {
		if len(result) >= limit {
			break
		}
		if !seen[stop.ID] {
			result = append(result, stop)
			seen[stop.ID] = true
		}
	}

	return result
}

func (a *InMemoryJourneyPlannerAdapter) findNearestStops(snapshot *PlannerSnapshot, lat, lon, radiusMeters float64, limit int) []models.Stop {
	if snapshot == nil || limit <= 0 {
		return nil
	}

	scored := make([]scoredStop, 0, len(snapshot.Stops))
	for _, stop := range snapshot.Stops {
		distance := haversineDistance(lat, lon, stop.Latitude, stop.Longitude)
		if distance > radiusMeters {
			continue
		}
		scored = append(scored, scoredStop{stop: stop, distance: distance})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].distance == scored[j].distance {
			return scored[i].stop.ID < scored[j].stop.ID
		}
		return scored[i].distance < scored[j].distance
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	stops := make([]models.Stop, 0, len(scored))
	for _, candidate := range scored {
		stops = append(stops, candidate.stop)
	}

	return stops
}

func (a *InMemoryJourneyPlannerAdapter) stopHasModeTypes(snapshot *PlannerSnapshot, stopID string) (hasMetro, hasBus bool) {
	modeTypes := snapshot.StopModeTypes[stopID]
	if len(modeTypes) == 0 {
		return false, false
	}

	_, hasMetro = modeTypes[1]
	_, hasBus = modeTypes[3]
	return hasMetro, hasBus
}

func (a *InMemoryJourneyPlannerAdapter) resolveJourneyDate(snapshot *PlannerSnapshot, requested time.Time) time.Time {
	if a.hasServiceOnDate(snapshot, requested) {
		return requested
	}

	if !snapshot.MinServiceDate.IsZero() && requested.Before(snapshot.MinServiceDate) {
		for i := 0; i < 14; i++ {
			candidate := snapshot.MinServiceDate.AddDate(0, 0, i)
			if a.hasServiceOnDate(snapshot, candidate) {
				return candidate
			}
		}
	}

	if !snapshot.MaxServiceDate.IsZero() {
		for i := 0; i < 14; i++ {
			candidate := snapshot.MaxServiceDate.AddDate(0, 0, -i)
			if a.hasServiceOnDate(snapshot, candidate) {
				return candidate
			}
		}
	}

	return requested
}

func (a *InMemoryJourneyPlannerAdapter) resolveDepartureTime(explicit *time.Time, requestedDate, resolvedDate time.Time) time.Time {
	if explicit != nil {
		return time.Date(
			resolvedDate.Year(),
			resolvedDate.Month(),
			resolvedDate.Day(),
			explicit.Hour(),
			explicit.Minute(),
			explicit.Second(),
			0,
			resolvedDate.Location(),
		)
	}

	if !sameCalendarDate(requestedDate, resolvedDate) {
		return time.Date(
			resolvedDate.Year(),
			resolvedDate.Month(),
			resolvedDate.Day(),
			8, 0, 0, 0,
			resolvedDate.Location(),
		)
	}

	now := time.Now()
	return time.Date(
		resolvedDate.Year(),
		resolvedDate.Month(),
		resolvedDate.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		0,
		resolvedDate.Location(),
	)
}

func (a *InMemoryJourneyPlannerAdapter) hasServiceOnDate(snapshot *PlannerSnapshot, date time.Time) bool {
	for _, calendar := range snapshot.CalendarsByServiceID {
		if serviceActiveOnDate(calendar, date) {
			return true
		}
	}
	return false
}

// planRounds runs a RAPTOR-style multi-round search where each round boards one
// more transit vehicle, scans routes that touch improved stops, and then
// relaxes footpaths without consuming an extra transfer.
func (a *InMemoryJourneyPlannerAdapter) planRounds(snapshot *PlannerSnapshot, fromStopID, toStopID string, departureTime, journeyDate time.Time) []models.JourneyOption {
	initialCandidate := roundCandidate{
		AtStopID:       fromStopID,
		ArrivalTime:    departureTime,
		ArrivalSeconds: departureTime.Hour()*3600 + departureTime.Minute()*60 + departureTime.Second(),
	}

	globalBest := make(map[string]roundCandidate)
	initialRound := newRaptorRoundState()
	initialRound.record(initialCandidate)
	globalBest[fromStopID] = initialCandidate
	a.relaxFootpaths(snapshot, initialRound, globalBest)

	collected := make([]models.JourneyOption, 0, 16)
	if candidate, ok := initialRound.bestByStop[toStopID]; ok {
		collected = append(collected, a.candidateToOption(candidate))
	}

	currentRound := initialRound
	for round := 0; round <= a.maxTransfers; round++ {
		if len(currentRound.marked) == 0 {
			break
		}

		nextRound := a.scanTransitRound(snapshot, currentRound, globalBest, journeyDate, round > 0)
		if len(nextRound.bestByStop) == 0 {
			break
		}

		a.relaxFootpaths(snapshot, nextRound, globalBest)
		if candidate, ok := nextRound.bestByStop[toStopID]; ok {
			collected = append(collected, a.candidateToOption(candidate))
		}

		currentRound = nextRound
	}

	collected = paretoPruneJourneyOptions(a.removeDuplicateOptions(collected))
	sort.Slice(collected, func(i, j int) bool {
		if collected[i].ArrivalTime.Equal(collected[j].ArrivalTime) {
			if collected[i].Transfers == collected[j].Transfers {
				return collected[i].DepartureTime.After(collected[j].DepartureTime)
			}
			return collected[i].Transfers < collected[j].Transfers
		}
		return collected[i].ArrivalTime.Before(collected[j].ArrivalTime)
	})

	if len(collected) > 40 {
		collected = collected[:40]
	}

	return collected
}

func (a *InMemoryJourneyPlannerAdapter) addAccessAndEgressLegs(req models.JourneyRequest, originStop, destStop models.Stop, departureTime time.Time, options []models.JourneyOption) []models.JourneyOption {
	for i := range options {
		if len(options[i].Legs) == 0 {
			options[i].Duration = math.MaxInt
			continue
		}

		originWalkTime := a.calculateWalkTime(req.FromLat, req.FromLon, originStop.Latitude, originStop.Longitude)
		destWalkTime := a.calculateWalkTime(destStop.Latitude, destStop.Longitude, req.ToLat, req.ToLon)
		firstTransitDeparture := options[i].Legs[0].DepartureTime
		lastTransitArrival := options[i].Legs[len(options[i].Legs)-1].ArrivalTime
		walkStartTime := firstTransitDeparture.Add(-time.Duration(originWalkTime) * time.Minute)

		if walkStartTime.Before(departureTime) {
			options[i].Duration = math.MaxInt
			continue
		}

		if originWalkTime > 0 {
			walkLeg := models.JourneyLeg{
				Mode:          "walking",
				FromStopID:    "",
				FromStopName:  "Origin",
				ToStopID:      originStop.ID,
				ToStopName:    originStop.Name,
				DepartureTime: walkStartTime,
				ArrivalTime:   firstTransitDeparture,
				Duration:      originWalkTime,
				StopCount:     0,
			}
			options[i].Legs = append([]models.JourneyLeg{walkLeg}, options[i].Legs...)
			options[i].WalkingTime += originWalkTime
		}

		if destWalkTime > 0 {
			destWalkStart := lastTransitArrival
			destWalkArrival := destWalkStart.Add(time.Duration(destWalkTime) * time.Minute)
			walkLeg := models.JourneyLeg{
				Mode:          "walking",
				FromStopID:    destStop.ID,
				FromStopName:  destStop.Name,
				ToStopID:      "",
				ToStopName:    "Destination",
				DepartureTime: destWalkStart,
				ArrivalTime:   destWalkArrival,
				Duration:      destWalkTime,
				StopCount:     0,
			}
			options[i].Legs = append(options[i].Legs, walkLeg)
			options[i].WalkingTime += destWalkTime
			lastTransitArrival = destWalkArrival
		}

		options[i].DepartureTime = departureTime
		options[i].ArrivalTime = lastTransitArrival
		options[i].Duration = int(math.Ceil(options[i].ArrivalTime.Sub(departureTime).Minutes()))
	}

	return options
}

func (a *InMemoryJourneyPlannerAdapter) applyFares(options []models.JourneyOption) []models.JourneyOption {
	defaultRules := a.fareService.GetFareRulesForAgency("DIMTS")
	for i := range options {
		if len(options[i].Legs) > 0 {
			firstRouteID := options[i].Legs[0].RouteID
			if firstRouteID != "" {
				agencyID := a.fareService.GetAgencyIDFromRoute(firstRouteID)
				if agencyID != "" {
					defaultRules = a.fareService.GetFareRulesForAgency(agencyID)
				}
			}
		}

		fare := a.fareService.CalculateFareForJourney(options[i], defaultRules)
		options[i].Fare = &fare
	}
	return options
}

func (a *InMemoryJourneyPlannerAdapter) calculateWalkTime(lat1, lon1, lat2, lon2 float64) int {
	distance := haversineDistance(lat1, lon1, lat2, lon2)
	return int(math.Ceil(distance / 83.33))
}

func (a *InMemoryJourneyPlannerAdapter) removeDuplicateOptions(options []models.JourneyOption) []models.JourneyOption {
	seen := make(map[string]bool)
	var unique []models.JourneyOption

	for _, option := range options {
		key := ""
		for _, leg := range option.Legs {
			if leg.RouteID != "" {
				key += leg.RouteID + ":" + leg.FromStopID + "->" + leg.ToStopID + "@" + leg.DepartureTime.Format(time.RFC3339) + "|"
			}
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, option)
		}
	}

	return unique
}

func (a *InMemoryJourneyPlannerAdapter) scanTransitRound(snapshot *PlannerSnapshot, previous *raptorRoundState, globalBest map[string]roundCandidate, journeyDate time.Time, addTransferBuffer bool) *raptorRoundState {
	next := newRaptorRoundState()
	if snapshot == nil || previous == nil || len(previous.marked) == 0 {
		return next
	}

	for _, routeID := range a.collectMarkedRoutes(snapshot, previous.marked) {
		for _, tripID := range snapshot.TripIDsByRouteID[routeID] {
			trip, ok := snapshot.TripsByID[tripID]
			if !ok {
				continue
			}

			calendar, ok := snapshot.CalendarsByServiceID[trip.ServiceID]
			if !ok || !serviceActiveOnDate(calendar, journeyDate) {
				continue
			}

			stopTimes := snapshot.StopTimesByTripID[tripID]
			if len(stopTimes) < 2 {
				continue
			}

			boardIndex := -1
			var boardedFrom roundCandidate
			var readyTime time.Time
			for idx, stopTime := range stopTimes {
				candidate, ok := previous.bestByStop[stopTime.StopTime.StopID]
				if !ok {
					continue
				}

				candidateReadyTime := candidate.ArrivalTime
				candidateReadySeconds := candidate.ArrivalSeconds
				if addTransferBuffer && endsWithTransitLeg(candidate.Legs) {
					candidateReadyTime = candidateReadyTime.Add(2 * time.Minute)
					candidateReadySeconds += 120
				}
				if stopTime.DepartureSeconds < candidateReadySeconds {
					continue
				}
				if lastLeg := lastTransitLeg(candidate.Legs); lastLeg != nil && lastLeg.RouteID == trip.RouteID {
					continue
				}

				boardIndex = idx
				boardedFrom = candidate
				readyTime = candidateReadyTime
				break
			}

			if boardIndex < 0 {
				continue
			}

			for idx := boardIndex + 1; idx < len(stopTimes); idx++ {
				nextCandidate := a.buildCandidate(snapshot, boardedFrom, trip, stopTimes[boardIndex], stopTimes[idx], journeyDate, readyTime)
				if nextCandidate == nil {
					continue
				}
				recordIfImproves(next, globalBest, *nextCandidate)
			}
		}
	}

	return next
}

func (a *InMemoryJourneyPlannerAdapter) buildCandidate(snapshot *PlannerSnapshot, previous roundCandidate, trip models.Trip, fromTime, toTime plannerStopTime, journeyDate, readyTime time.Time) *roundCandidate {
	segment := a.buildDirectLeg(snapshot, trip, fromTime, toTime, journeyDate)
	if segment == nil {
		return nil
	}
	if segment.DepartureTime.Before(readyTime) {
		return nil
	}

	legs := append([]models.JourneyLeg{}, previous.Legs...)
	transfers := previous.Transfers

	if lastTransitLeg(previous.Legs) != nil {
		transfers++
		if segment.DepartureTime.After(previous.ArrivalTime) {
			transferStopName := previous.AtStopName(snapshot)
			legs = append(legs, models.JourneyLeg{
				Mode:          "transfer",
				RouteName:     "Transfer",
				FromStopID:    previous.AtStopID,
				FromStopName:  transferStopName,
				ToStopID:      previous.AtStopID,
				ToStopName:    transferStopName,
				DepartureTime: previous.ArrivalTime,
				ArrivalTime:   segment.DepartureTime,
				Duration:      int(math.Ceil(segment.DepartureTime.Sub(previous.ArrivalTime).Minutes())),
				StopCount:     0,
			})
		}
	}

	legs = append(legs, *segment)

	return &roundCandidate{
		AtStopID:       segment.ToStopID,
		ArrivalTime:    segment.ArrivalTime,
		ArrivalSeconds: toTime.ArrivalSeconds,
		Legs:           legs,
		Transfers:      transfers,
	}
}

func (a *InMemoryJourneyPlannerAdapter) relaxFootpaths(snapshot *PlannerSnapshot, state *raptorRoundState, globalBest map[string]roundCandidate) {
	if snapshot == nil || state == nil || len(state.bestByStop) == 0 {
		return
	}

	queue := make([]roundCandidate, 0, len(state.marked))
	for stopID := range state.marked {
		queue = append(queue, state.bestByStop[stopID])
	}
	if len(queue) == 0 {
		for _, candidate := range state.bestByStop {
			queue = append(queue, candidate)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, footpath := range snapshot.FootpathsFromStopID[current.AtStopID] {
			nextStop, ok := snapshot.StopsByID[footpath.ToStopID]
			if !ok {
				continue
			}

			arrivalTime := current.ArrivalTime.Add(time.Duration(footpath.Duration) * time.Second)
			legs := append([]models.JourneyLeg{}, current.Legs...)
			legs = append(legs, models.JourneyLeg{
				Mode:          "walking",
				FromStopID:    current.AtStopID,
				FromStopName:  current.AtStopName(snapshot),
				ToStopID:      footpath.ToStopID,
				ToStopName:    nextStop.Name,
				DepartureTime: current.ArrivalTime,
				ArrivalTime:   arrivalTime,
				Duration:      int(math.Ceil(float64(footpath.Duration) / 60.0)),
				StopCount:     0,
			})

			candidate := roundCandidate{
				AtStopID:       footpath.ToStopID,
				ArrivalTime:    arrivalTime,
				ArrivalSeconds: current.ArrivalSeconds + footpath.Duration,
				Legs:           legs,
				Transfers:      current.Transfers,
			}

			if !recordIfImproves(state, globalBest, candidate) {
				continue
			}
			queue = append(queue, candidate)
		}
	}
}

func (a *InMemoryJourneyPlannerAdapter) candidateToOption(candidate roundCandidate) models.JourneyOption {
	walkingTime := 0
	for _, leg := range candidate.Legs {
		if leg.Mode == "walking" || leg.Mode == "transfer" {
			walkingTime += leg.Duration
		}
	}

	departureTime := candidate.ArrivalTime
	if len(candidate.Legs) > 0 {
		departureTime = candidate.Legs[0].DepartureTime
	}

	return models.JourneyOption{
		Duration:      int(math.Ceil(candidate.ArrivalTime.Sub(departureTime).Minutes())),
		Transfers:     candidate.Transfers,
		WalkingTime:   walkingTime,
		Legs:          append([]models.JourneyLeg{}, candidate.Legs...),
		DepartureTime: departureTime,
		ArrivalTime:   candidate.ArrivalTime,
	}
}

func newRaptorRoundState() *raptorRoundState {
	return &raptorRoundState{
		bestByStop: make(map[string]roundCandidate),
		marked:     make(map[string]struct{}),
	}
}

func (s *raptorRoundState) record(candidate roundCandidate) bool {
	current, exists := s.bestByStop[candidate.AtStopID]
	if exists && !candidateStrictlyBetter(candidate, current) {
		return false
	}

	s.bestByStop[candidate.AtStopID] = candidate
	s.marked[candidate.AtStopID] = struct{}{}
	return true
}

func recordIfImproves(state *raptorRoundState, globalBest map[string]roundCandidate, candidate roundCandidate) bool {
	if state == nil {
		return false
	}

	currentRound, exists := state.bestByStop[candidate.AtStopID]
	if exists && !candidateStrictlyBetter(candidate, currentRound) {
		return false
	}

	global, exists := globalBest[candidate.AtStopID]
	if exists && !candidateStrictlyBetter(candidate, global) {
		return false
	}

	state.bestByStop[candidate.AtStopID] = candidate
	state.marked[candidate.AtStopID] = struct{}{}
	globalBest[candidate.AtStopID] = candidate
	return true
}

func candidateStrictlyBetter(left, right roundCandidate) bool {
	if left.ArrivalTime.Before(right.ArrivalTime) {
		return true
	}
	if left.ArrivalTime.After(right.ArrivalTime) {
		return false
	}

	leftWalking := candidateWalkingMinutes(left)
	rightWalking := candidateWalkingMinutes(right)
	if leftWalking != rightWalking {
		return leftWalking < rightWalking
	}

	leftDeparture := candidateDepartureTime(left)
	rightDeparture := candidateDepartureTime(right)
	if !leftDeparture.Equal(rightDeparture) {
		return leftDeparture.After(rightDeparture)
	}

	return left.Transfers < right.Transfers
}

func candidateWalkingMinutes(candidate roundCandidate) int {
	total := 0
	for _, leg := range candidate.Legs {
		if leg.Mode == "walking" || leg.Mode == "transfer" {
			total += leg.Duration
		}
	}
	return total
}

func candidateDepartureTime(candidate roundCandidate) time.Time {
	if len(candidate.Legs) > 0 {
		return candidate.Legs[0].DepartureTime
	}
	return candidate.ArrivalTime
}

func paretoPruneJourneyOptions(options []models.JourneyOption) []models.JourneyOption {
	filtered := make([]models.JourneyOption, 0, len(options))
	for i := range options {
		dominated := false
		for j := range options {
			if i == j {
				continue
			}
			if journeyOptionDominates(options[j], options[i]) {
				dominated = true
				break
			}
		}
		if !dominated {
			filtered = append(filtered, options[i])
		}
	}
	return filtered
}

func journeyOptionDominates(left, right models.JourneyOption) bool {
	leftArrivalBetter := left.ArrivalTime.Before(right.ArrivalTime) || left.ArrivalTime.Equal(right.ArrivalTime)
	leftTransfersBetter := left.Transfers <= right.Transfers
	leftWalkingBetter := left.WalkingTime <= right.WalkingTime
	leftDurationBetter := left.Duration <= right.Duration

	if !(leftArrivalBetter && leftTransfersBetter && leftWalkingBetter && leftDurationBetter) {
		return false
	}

	return left.ArrivalTime.Before(right.ArrivalTime) ||
		left.Transfers < right.Transfers ||
		left.WalkingTime < right.WalkingTime ||
		left.Duration < right.Duration
}

func (a *InMemoryJourneyPlannerAdapter) collectMarkedRoutes(snapshot *PlannerSnapshot, markedStops map[string]struct{}) []string {
	routeSet := make(map[string]struct{})
	for stopID := range markedStops {
		for _, routeID := range snapshot.RouteIDsByStopID[stopID] {
			if routeID == "" {
				continue
			}
			routeSet[routeID] = struct{}{}
		}
	}

	routes := make([]string, 0, len(routeSet))
	for routeID := range routeSet {
		routes = append(routes, routeID)
	}
	sort.Strings(routes)
	return routes
}

func (a *InMemoryJourneyPlannerAdapter) buildDirectLeg(snapshot *PlannerSnapshot, trip models.Trip, fromTime, toTime plannerStopTime, journeyDate time.Time) *models.JourneyLeg {
	if toTime.ArrivalSeconds <= fromTime.DepartureSeconds {
		return nil
	}

	fromStop, ok := snapshot.StopsByID[fromTime.StopTime.StopID]
	if !ok {
		return nil
	}
	toStop, ok := snapshot.StopsByID[toTime.StopTime.StopID]
	if !ok {
		return nil
	}

	departureAt, err := parseGTFSTimeOnDate(journeyDate, fromTime.StopTime.DepartureTime)
	if err != nil {
		return nil
	}
	arrivalAt, err := parseGTFSTimeOnDate(journeyDate, toTime.StopTime.ArrivalTime)
	if err != nil {
		return nil
	}

	route, ok := snapshot.RoutesByID[trip.RouteID]
	if !ok {
		return nil
	}
	routeName := route.ShortName
	if routeName == "" {
		routeName = route.LongName
	}

	duration := int(math.Ceil(arrivalAt.Sub(departureAt).Minutes()))
	return &models.JourneyLeg{
		Mode:          routeTypeToMode(route.Type),
		RouteID:       route.ID,
		RouteName:     routeName,
		FromStopID:    fromTime.StopTime.StopID,
		FromStopName:  fromStop.Name,
		ToStopID:      toTime.StopTime.StopID,
		ToStopName:    toStop.Name,
		DepartureTime: departureAt,
		ArrivalTime:   arrivalAt,
		Duration:      duration,
		StopCount:     toTime.StopTime.StopSequence - fromTime.StopTime.StopSequence,
	}
}

func lastTransitLeg(legs []models.JourneyLeg) *models.JourneyLeg {
	for i := len(legs) - 1; i >= 0; i-- {
		if legs[i].RouteID != "" {
			return &legs[i]
		}
	}
	return nil
}

func endsWithTransitLeg(legs []models.JourneyLeg) bool {
	if len(legs) == 0 {
		return false
	}
	return legs[len(legs)-1].RouteID != ""
}

func (c roundCandidate) AtStopName(snapshot *PlannerSnapshot) string {
	if snapshot == nil {
		return ""
	}
	stop, ok := snapshot.StopsByID[c.AtStopID]
	if !ok {
		return ""
	}
	return stop.Name
}

func serviceActiveOnDate(calendar models.Calendar, date time.Time) bool {
	startDate, err := time.Parse("2006-01-02", calendar.StartDate)
	if err != nil {
		return false
	}
	endDate, err := time.Parse("2006-01-02", calendar.EndDate)
	if err != nil {
		return false
	}

	queryDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)
	if queryDate.Before(startDate) || queryDate.After(endDate) {
		return false
	}

	switch date.Weekday() {
	case time.Sunday:
		return calendar.Sunday == 1
	case time.Monday:
		return calendar.Monday == 1
	case time.Tuesday:
		return calendar.Tuesday == 1
	case time.Wednesday:
		return calendar.Wednesday == 1
	case time.Thursday:
		return calendar.Thursday == 1
	case time.Friday:
		return calendar.Friday == 1
	case time.Saturday:
		return calendar.Saturday == 1
	default:
		return false
	}
}

func gtfsTimeToSeconds(value string) (int, error) {
	parts, err := parseGTFSTimeParts(value)
	if err != nil {
		return 0, err
	}
	return parts.hour*3600 + parts.minute*60 + parts.second, nil
}

type gtfsTimeParts struct {
	hour   int
	minute int
	second int
}

func parseGTFSTimeParts(value string) (gtfsTimeParts, error) {
	var parts gtfsTimeParts
	segments := [3]int{}
	n := 0
	current := 0

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == ':' {
			if n >= 3 {
				return gtfsTimeParts{}, fmt.Errorf("invalid GTFS time %q", value)
			}
			segments[n] = current
			n++
			current = 0
			continue
		}
		if ch < '0' || ch > '9' {
			return gtfsTimeParts{}, fmt.Errorf("invalid GTFS time %q", value)
		}
		current = current*10 + int(ch-'0')
	}

	if n != 2 {
		return gtfsTimeParts{}, fmt.Errorf("invalid GTFS time %q", value)
	}
	segments[n] = current
	parts.hour = segments[0]
	parts.minute = segments[1]
	parts.second = segments[2]
	return parts, nil
}

type tripScanner interface {
	Scan(dest ...any) error
}

func scanTripRow(scanner tripScanner, trip *models.Trip) error {
	var headsign, shortName, blockID, shapeID sql.NullString
	var directionID, wheelchairAccessible, bikesAllowed sql.NullInt64

	if err := scanner.Scan(
		&trip.ID,
		&trip.RouteID,
		&trip.ServiceID,
		&headsign,
		&shortName,
		&directionID,
		&blockID,
		&shapeID,
		&wheelchairAccessible,
		&bikesAllowed,
	); err != nil {
		return err
	}

	if headsign.Valid {
		trip.Headsign = headsign.String
	}
	if shortName.Valid {
		trip.ShortName = shortName.String
	}
	if directionID.Valid {
		value := int(directionID.Int64)
		trip.DirectionID = &value
	}
	if blockID.Valid {
		trip.BlockID = blockID.String
	}
	if shapeID.Valid {
		trip.ShapeID = shapeID.String
	}
	if wheelchairAccessible.Valid {
		value := int(wheelchairAccessible.Int64)
		trip.WheelchairAccessible = &value
	}
	if bikesAllowed.Valid {
		value := int(bikesAllowed.Int64)
		trip.BikesAllowed = &value
	}

	return nil
}

type stopTimeScanner interface {
	Scan(dest ...any) error
}

func scanStopTimeRow(scanner stopTimeScanner, stopTime *models.StopTime) error {
	var stopHeadsign sql.NullString
	var pickupType, dropOffType, timepoint sql.NullInt64
	var shapeDistTraveled sql.NullFloat64

	if err := scanner.Scan(
		&stopTime.TripID,
		&stopTime.ArrivalTime,
		&stopTime.DepartureTime,
		&stopTime.StopID,
		&stopTime.StopSequence,
		&stopHeadsign,
		&pickupType,
		&dropOffType,
		&shapeDistTraveled,
		&timepoint,
	); err != nil {
		return err
	}

	if stopHeadsign.Valid {
		stopTime.StopHeadsign = stopHeadsign.String
	}
	if pickupType.Valid {
		value := int(pickupType.Int64)
		stopTime.PickupType = &value
	}
	if dropOffType.Valid {
		value := int(dropOffType.Int64)
		stopTime.DropOffType = &value
	}
	if shapeDistTraveled.Valid {
		value := shapeDistTraveled.Float64
		stopTime.ShapeDistTraveled = &value
	}
	if timepoint.Valid {
		value := int(timepoint.Int64)
		stopTime.Timepoint = &value
	}

	return nil
}
