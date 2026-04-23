package services

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
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

type manualTransferLink struct {
	FromStopID string
	ToStopID   string
	Duration   int
}

var manualTransferLinks = []manualTransferLink{
	{
		// Practical interchange between Delhi Metro Blue/Magenta side and Noida Aqua line.
		FromStopID: "metro:81",  // Botanical Garden
		ToStopID:   "metro:500", // Noida Sector 51
		Duration:   12,
	},
}

func NewRoutePlanner(db *database.DB, stopService *StopService, routeService *RouteService, fareService *FareService) *RoutePlanner {
	return &RoutePlanner{
		db:            db,
		stopService:   stopService,
		routeService:  routeService,
		fareService:   fareService,
		maxTransfers:  3,
		maxWalkMeters: 3000.0, // adapt up to 3km so sparse areas like airport edges can still find transit
	}
}

func (rp *RoutePlanner) PlanJourney(req models.JourneyRequest) ([]models.JourneyOption, error) {
	requestedJourneyDate := time.Now()
	if req.Date != nil {
		requestedJourneyDate = *req.Date
	}

	resolvedJourneyDate, err := rp.resolveJourneyDate(requestedJourneyDate)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve service date: %w", err)
	}

	departureTime := rp.resolveDepartureTime(req.DepartureTime, requestedJourneyDate, resolvedJourneyDate)

	// Find nearest stops to origin and destination, ensuring mix of metro and bus.
	originStops, err := rp.findCandidateStops(req.FromLat, req.FromLon, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to find origin stops: %w", err)
	}
	if len(originStops) == 0 {
		return nil, fmt.Errorf("no stops found near origin")
	}

	destStops, err := rp.findCandidateStops(req.ToLat, req.ToLon, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to find destination stops: %w", err)
	}
	if len(destStops) == 0 {
		return nil, fmt.Errorf("no stops found near destination")
	}

	// Plan journeys from each origin stop to each destination stop
	var allOptions []models.JourneyOption

	for _, originStop := range originStops {
		for _, destStop := range destStops {
			options, err := rp.planBetweenStops(originStop.ID, destStop.ID, departureTime, resolvedJourneyDate)
			if err != nil {
				continue // Skip if no route found
			}

			// Add walking legs to/from stops
			for i := range options {
				originWalkTime := rp.calculateWalkTime(req.FromLat, req.FromLon, originStop.Latitude, originStop.Longitude)
				destWalkTime := rp.calculateWalkTime(destStop.Latitude, destStop.Longitude, req.ToLat, req.ToLon)
				firstTransitDeparture := options[i].Legs[0].DepartureTime
				lastTransitArrival := options[i].Legs[len(options[i].Legs)-1].ArrivalTime
				walkStartTime := firstTransitDeparture.Add(-time.Duration(originWalkTime) * time.Minute)

				// Can't make this itinerary if reaching the first stop would require leaving before the requested time.
				if walkStartTime.Before(departureTime) {
					options[i].Duration = math.MaxInt
					continue
				}

				// Add origin walking leg
				if originWalkTime > 0 {
					walkLeg := models.JourneyLeg{
						Mode:         "walking",
						FromStopID:   "",
						FromStopName: "Origin",
						ToStopID:     originStop.ID,
						ToStopName:   originStop.Name,
						DepartureTime: walkStartTime,
						ArrivalTime:   firstTransitDeparture,
						Duration:     originWalkTime,
						StopCount:    0,
					}
					options[i].Legs = append([]models.JourneyLeg{walkLeg}, options[i].Legs...)
					options[i].WalkingTime += originWalkTime
				}

				// Add destination walking leg
				if destWalkTime > 0 {
					destWalkStart := lastTransitArrival
					destWalkArrival := destWalkStart.Add(time.Duration(destWalkTime) * time.Minute)
					walkLeg := models.JourneyLeg{
						Mode:         "walking",
						FromStopID:   destStop.ID,
						FromStopName: destStop.Name,
						ToStopID:     "",
						ToStopName:   "Destination",
						DepartureTime: destWalkStart,
						ArrivalTime:   destWalkArrival,
						Duration:     destWalkTime,
						StopCount:    0,
					}
					options[i].Legs = append(options[i].Legs, walkLeg)
					options[i].WalkingTime += destWalkTime
					lastTransitArrival = destWalkArrival
				}

				options[i].DepartureTime = departureTime
				options[i].ArrivalTime = lastTransitArrival
				options[i].Duration = int(math.Ceil(options[i].ArrivalTime.Sub(departureTime).Minutes()))
			}

			allOptions = append(allOptions, options...)
		}
	}

	filtered := allOptions[:0]
	for _, option := range allOptions {
		if option.Duration != math.MaxInt {
			filtered = append(filtered, option)
		}
	}
	allOptions = filtered

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

		// Primary: total duration from requested departure to final arrival.
		if optI.Duration != optJ.Duration {
			return optI.Duration < optJ.Duration
		}

		// Secondary: Fewer transfers is better
		if optI.Transfers != optJ.Transfers {
			return optI.Transfers < optJ.Transfers
		}

		// Tertiary: Less walking time is better
		if optI.WalkingTime != optJ.WalkingTime {
			return optI.WalkingTime < optJ.WalkingTime
		}

		return optI.ArrivalTime.Before(optJ.ArrivalTime)
	})

	allOptions = rp.removeDuplicateOptions(allOptions)

	// Return the fastest 10 options.
	if len(allOptions) > 10 {
		allOptions = allOptions[:10]
	}

	return allOptions, nil
}

func (rp *RoutePlanner) resolveDepartureTime(explicit *time.Time, requestedDate, resolvedDate time.Time) time.Time {
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

	// When we fall back to a different GTFS service date, using the current wall-clock
	// time can easily push us past the end of service and produce no results at all.
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

func (rp *RoutePlanner) planBetweenStops(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	var allOptions []models.JourneyOption

	// Find direct routes (multiple options from different routes/times)
	directOptions, err := rp.findDirectRoutes(fromStopID, toStopID, departureTime, journeyDate)
	if err == nil && len(directOptions) > 0 {
		allOptions = append(allOptions, directOptions...)
	}

	// Transfers are significantly more expensive to search, so only use them
	// when a stop pair has no direct service.
	if len(allOptions) == 0 {
		transferOptions, err := rp.findSingleTransferRoutes(fromStopID, toStopID, departureTime, journeyDate)
		if err == nil && len(transferOptions) > 0 {
			allOptions = append(allOptions, transferOptions...)
		}
	}

	if len(allOptions) == 0 {
		manualOptions, err := rp.findJourneysViaManualTransfers(fromStopID, toStopID, departureTime, journeyDate)
		if err == nil && len(manualOptions) > 0 {
			allOptions = append(allOptions, manualOptions...)
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

	// Return up to 10 options
	if len(uniqueOptions) > 10 {
		uniqueOptions = uniqueOptions[:10]
	}

	return uniqueOptions, nil
}

func (rp *RoutePlanner) findDirectRoutes(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	query := `SELECT route_id, route_short_name, route_long_name, route_type, trip_id, departure_time, arrival_time, stop_count
	FROM (
		SELECT 
			t.route_id,
			r.route_short_name,
			r.route_long_name,
			r.route_type,
			st1.trip_id,
			st1.departure_time,
			st2.arrival_time,
			(st2.stop_sequence - st1.stop_sequence) AS stop_count,
			ROW_NUMBER() OVER (PARTITION BY t.route_id, st1.departure_time ORDER BY st2.arrival_time) AS rn
		FROM stop_times st1
		JOIN stop_times st2 ON st1.trip_id = st2.trip_id
		JOIN trips t ON st1.trip_id = t.trip_id
		JOIN routes r ON t.route_id = r.route_id
		JOIN calendar cal ON t.service_id = cal.service_id
		WHERE st1.stop_id = ?
			AND st2.stop_id = ?
			AND st1.stop_sequence < st2.stop_sequence
			AND cal.start_date <= ?
			AND cal.end_date >= ?
			AND (
				(EXTRACT(DOW FROM ?::date) = 0 AND cal.sunday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 1 AND cal.monday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 2 AND cal.tuesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 3 AND cal.wednesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 4 AND cal.thursday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 5 AND cal.friday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 6 AND cal.saturday = 1)
			)
			AND st1.departure_time >= ?
	) ranked
	WHERE rn = 1
	ORDER BY departure_time, route_id
	LIMIT 20`

	departureTimeStr := departureTime.Format("15:04:05")
	journeyDateStr := journeyDate.Format("2006-01-02")
	rows, err := rp.db.Query(
		query,
		fromStopID,
		toStopID,
		journeyDateStr,
		journeyDateStr,
		journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr,
		departureTimeStr,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	fromStop, err := rp.stopService.GetByID(fromStopID)
	if err != nil {
		return nil, err
	}

	toStop, err := rp.stopService.GetByID(toStopID)
	if err != nil {
		return nil, err
	}

	var options []models.JourneyOption
	for rows.Next() {
		var routeID, routeShortName, routeLongName, tripID, depTime, arrTime string
		var routeType, stopCount int

		err := rows.Scan(&routeID, &routeShortName, &routeLongName, &routeType, &tripID, &depTime, &arrTime, &stopCount)
		if err != nil {
			continue
		}

		depTimeToday, err := parseGTFSTimeOnDate(journeyDate, depTime)
		if err != nil {
			continue
		}

		arrTimeToday, err := parseGTFSTimeOnDate(journeyDate, arrTime)
		if err != nil {
			continue
		}

		if depTimeToday.Before(departureTime) {
			continue
		}

		duration := int(math.Ceil(arrTimeToday.Sub(depTimeToday).Minutes()))

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

func (rp *RoutePlanner) findJourneysViaManualTransfers(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	var allOptions []models.JourneyOption

	for _, link := range manualTransferLinks {
		firstSegment, err := rp.planSegmentWithoutManualTransfer(fromStopID, link.FromStopID, departureTime, journeyDate)
		if err != nil || len(firstSegment) == 0 {
			continue
		}

		fromTransferStop, err := rp.stopService.GetByID(link.FromStopID)
		if err != nil {
			continue
		}

		toTransferStop, err := rp.stopService.GetByID(link.ToStopID)
		if err != nil {
			continue
		}

		for _, firstOption := range firstSegment {
			transferDeparture := firstOption.ArrivalTime
			transferArrival := transferDeparture.Add(time.Duration(link.Duration) * time.Minute)

			secondSegment, err := rp.planSegmentWithoutManualTransfer(link.ToStopID, toStopID, transferArrival, journeyDate)
			if err != nil || len(secondSegment) == 0 {
				continue
			}

			transferLeg := models.JourneyLeg{
				Mode:          "transfer",
				RouteName:     "Manual transfer",
				FromStopID:    link.FromStopID,
				FromStopName:  fromTransferStop.Name,
				ToStopID:      link.ToStopID,
				ToStopName:    toTransferStop.Name,
				DepartureTime: transferDeparture,
				ArrivalTime:   transferArrival,
				Duration:      link.Duration,
				StopCount:     0,
			}

			for _, secondOption := range secondSegment {
				combinedLegs := append([]models.JourneyLeg{}, firstOption.Legs...)
				combinedLegs = append(combinedLegs, transferLeg)
				combinedLegs = append(combinedLegs, secondOption.Legs...)

				combined := models.JourneyOption{
					Duration:      int(math.Ceil(secondOption.ArrivalTime.Sub(firstOption.DepartureTime).Minutes())),
					Transfers:     firstOption.Transfers + secondOption.Transfers + 1,
					WalkingTime:   firstOption.WalkingTime + secondOption.WalkingTime + link.Duration,
					Legs:          combinedLegs,
					DepartureTime: firstOption.DepartureTime,
					ArrivalTime:   secondOption.ArrivalTime,
				}
				allOptions = append(allOptions, combined)
			}
		}
	}

	if len(allOptions) == 0 {
		return nil, fmt.Errorf("no manual transfer route found")
	}

	return allOptions, nil
}

func (rp *RoutePlanner) planSegmentWithoutManualTransfer(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	var options []models.JourneyOption

	directOptions, err := rp.findDirectRoutes(fromStopID, toStopID, departureTime, journeyDate)
	if err == nil && len(directOptions) > 0 {
		options = append(options, directOptions...)
	}

	if len(options) == 0 {
		transferOptions, err := rp.findSingleTransferRoutes(fromStopID, toStopID, departureTime, journeyDate)
		if err == nil && len(transferOptions) > 0 {
			options = append(options, transferOptions...)
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no segment route found")
	}

	options = rp.removeDuplicateOptions(options)
	sort.Slice(options, func(i, j int) bool {
		if options[i].Duration != options[j].Duration {
			return options[i].Duration < options[j].Duration
		}
		if options[i].Transfers != options[j].Transfers {
			return options[i].Transfers < options[j].Transfers
		}
		return options[i].ArrivalTime.Before(options[j].ArrivalTime)
	})

	if len(options) > 4 {
		options = options[:4]
	}

	return options, nil
}

func (rp *RoutePlanner) findSingleTransferRoutes(fromStopID, toStopID string, departureTime time.Time, journeyDate time.Time) ([]models.JourneyOption, error) {
	query := `SELECT
		route1_id, route1_short_name, route1_long_name, route1_type,
		transfer_stop_id, transfer_stop_name,
		first_departure_time, first_arrival_time, first_stop_count,
		route2_id, route2_short_name, route2_long_name, route2_type,
		second_departure_time, second_arrival_time, second_stop_count
	FROM (
		SELECT
			t1.route_id AS route1_id,
			r1.route_short_name AS route1_short_name,
			r1.route_long_name AS route1_long_name,
			r1.route_type AS route1_type,
			transfer.stop_id AS transfer_stop_id,
			s_transfer.stop_name AS transfer_stop_name,
			st1.departure_time AS first_departure_time,
			transfer.arrival_time AS first_arrival_time,
			(transfer.stop_sequence - st1.stop_sequence) AS first_stop_count,
			t2.route_id AS route2_id,
			r2.route_short_name AS route2_short_name,
			r2.route_long_name AS route2_long_name,
			r2.route_type AS route2_type,
			st3.departure_time AS second_departure_time,
			st4.arrival_time AS second_arrival_time,
			(st4.stop_sequence - st3.stop_sequence) AS second_stop_count,
			ROW_NUMBER() OVER (
				PARTITION BY t1.route_id, transfer.stop_id, t2.route_id
				ORDER BY st1.departure_time, st4.arrival_time
			) AS rn
		FROM stop_times st1
		JOIN stop_times transfer
			ON st1.trip_id = transfer.trip_id
			AND st1.stop_sequence < transfer.stop_sequence
		JOIN trips t1 ON st1.trip_id = t1.trip_id
		JOIN routes r1 ON t1.route_id = r1.route_id
		JOIN calendar cal1 ON t1.service_id = cal1.service_id
		JOIN stops s_transfer ON transfer.stop_id = s_transfer.stop_id
		JOIN stop_times st3 ON st3.stop_id = transfer.stop_id
		JOIN stop_times st4
			ON st3.trip_id = st4.trip_id
			AND st3.stop_sequence < st4.stop_sequence
		JOIN trips t2 ON st3.trip_id = t2.trip_id
		JOIN routes r2 ON t2.route_id = r2.route_id
		JOIN calendar cal2 ON t2.service_id = cal2.service_id
		WHERE st1.stop_id = ?
			AND st4.stop_id = ?
			AND t1.route_id <> t2.route_id
			AND st1.departure_time >= ?
			AND st3.departure_time >= transfer.arrival_time
			AND cal1.start_date <= ?
			AND cal1.end_date >= ?
			AND cal2.start_date <= ?
			AND cal2.end_date >= ?
			AND (
				(EXTRACT(DOW FROM ?::date) = 0 AND cal1.sunday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 1 AND cal1.monday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 2 AND cal1.tuesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 3 AND cal1.wednesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 4 AND cal1.thursday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 5 AND cal1.friday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 6 AND cal1.saturday = 1)
			)
			AND (
				(EXTRACT(DOW FROM ?::date) = 0 AND cal2.sunday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 1 AND cal2.monday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 2 AND cal2.tuesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 3 AND cal2.wednesday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 4 AND cal2.thursday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 5 AND cal2.friday = 1) OR
				(EXTRACT(DOW FROM ?::date) = 6 AND cal2.saturday = 1)
			)
	) ranked
	WHERE rn = 1
	ORDER BY first_departure_time, second_arrival_time
	LIMIT 80`

	journeyDateStr := journeyDate.Format("2006-01-02")
	departureTimeStr := departureTime.Format("15:04:05")
	rows, err := rp.db.Query(
		query,
		fromStopID,
		toStopID,
		departureTimeStr,
		journeyDateStr,
		journeyDateStr,
		journeyDateStr,
		journeyDateStr,
		journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr,
		journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr, journeyDateStr,
	)
	if err != nil {
		return nil, fmt.Errorf("transfer query failed: %w", err)
	}
	defer rows.Close()

	fromStop, err := rp.stopService.GetByID(fromStopID)
	if err != nil {
		return nil, err
	}

	toStop, err := rp.stopService.GetByID(toStopID)
	if err != nil {
		return nil, err
	}

	var options []models.JourneyOption
	for rows.Next() {
		var route1ID, route1ShortName, route1LongName string
		var route1Type, firstStopCount int
		var transferStopID, transferStopName string
		var firstDepartureTime, firstArrivalTime string
		var route2ID, route2ShortName, route2LongName string
		var route2Type, secondStopCount int
		var secondDepartureTime, secondArrivalTime string

		err := rows.Scan(
			&route1ID, &route1ShortName, &route1LongName, &route1Type,
			&transferStopID, &transferStopName,
			&firstDepartureTime, &firstArrivalTime, &firstStopCount,
			&route2ID, &route2ShortName, &route2LongName, &route2Type,
			&secondDepartureTime, &secondArrivalTime, &secondStopCount,
		)
		if err != nil {
			continue
		}

		firstDeparture, err := parseGTFSTimeOnDate(journeyDate, firstDepartureTime)
		if err != nil {
			continue
		}
		firstArrival, err := parseGTFSTimeOnDate(journeyDate, firstArrivalTime)
		if err != nil {
			continue
		}
		secondDeparture, err := parseGTFSTimeOnDate(journeyDate, secondDepartureTime)
		if err != nil {
			continue
		}
		secondArrival, err := parseGTFSTimeOnDate(journeyDate, secondArrivalTime)
		if err != nil {
			continue
		}

		if firstDeparture.Before(departureTime) {
			continue
		}

		// Allow a small transfer buffer so unrealistic same-second transfers are filtered out.
		if secondDeparture.Before(firstArrival.Add(2 * time.Minute)) {
			continue
		}

		firstRouteName := route1ShortName
		if firstRouteName == "" {
			firstRouteName = route1LongName
		}

		secondRouteName := route2ShortName
		if secondRouteName == "" {
			secondRouteName = route2LongName
		}

		option := models.JourneyOption{
			Duration:      int(math.Ceil(secondArrival.Sub(firstDeparture).Minutes())),
			Transfers:     1,
			WalkingTime:   0,
			DepartureTime: firstDeparture,
			ArrivalTime:   secondArrival,
			Legs: []models.JourneyLeg{
				{
					Mode:          routeTypeToMode(route1Type),
					RouteID:       route1ID,
					RouteName:     firstRouteName,
					FromStopID:    fromStopID,
					FromStopName:  fromStop.Name,
					ToStopID:      transferStopID,
					ToStopName:    transferStopName,
					DepartureTime: firstDeparture,
					ArrivalTime:   firstArrival,
					Duration:      int(math.Ceil(firstArrival.Sub(firstDeparture).Minutes())),
					StopCount:     firstStopCount,
				},
				{
					Mode:          routeTypeToMode(route2Type),
					RouteID:       route2ID,
					RouteName:     secondRouteName,
					FromStopID:    transferStopID,
					FromStopName:  transferStopName,
					ToStopID:      toStopID,
					ToStopName:    toStop.Name,
					DepartureTime: secondDeparture,
					ArrivalTime:   secondArrival,
					Duration:      int(math.Ceil(secondArrival.Sub(secondDeparture).Minutes())),
					StopCount:     secondStopCount,
				},
			},
		}
		options = append(options, option)
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no transfer routes found")
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
	WHERE st.stop_id = ?`

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
	query := `SELECT s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon,
		s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
	FROM stops s
	JOIN stop_times st ON s.stop_id = st.stop_id
	JOIN trips t ON st.trip_id = t.trip_id
	JOIN (
		SELECT MIN(st2.stop_sequence) AS origin_seq
		FROM stop_times st2
		JOIN trips t2 ON st2.trip_id = t2.trip_id
		WHERE t2.route_id = ? AND st2.stop_id = ?
	) origin
	WHERE t.route_id = ? AND st.stop_sequence > origin.origin_seq
	GROUP BY s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon,
		s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
	ORDER BY MIN(st.stop_sequence)`

	rows, err := rp.db.Query(query, routeID, stopID, routeID)
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

func (rp *RoutePlanner) findCandidateStops(lat, lon float64, limit int) ([]models.Stop, error) {
	radii := []float64{1000, 2000, 3000, 5000}
	var lastStops []models.Stop

	for _, radius := range radii {
		stops, err := rp.findNearbyStopsWithModeMix(lat, lon, radius, limit)
		if err != nil {
			continue
		}
		if len(stops) > 0 {
			lastStops = stops
		}
		if len(stops) >= minInt(limit, 4) {
			return stops, nil
		}
	}

	if len(lastStops) == 0 {
		return nil, fmt.Errorf("no stops found")
	}

	return lastStops, nil
}

func (rp *RoutePlanner) resolveJourneyDate(requested time.Time) (time.Time, error) {
	if rp.hasServiceOnDate(requested) {
		return requested, nil
	}

	var minDate, maxDate time.Time
	err := rp.db.QueryRow(`SELECT MIN(start_date), MAX(end_date) FROM calendar`).Scan(&minDate, &maxDate)
	if err != nil {
		return time.Time{}, err
	}

	if requested.Before(minDate) {
		for i := 0; i < 14; i++ {
			candidate := minDate.AddDate(0, 0, i)
			if rp.hasServiceOnDate(candidate) {
				return candidate, nil
			}
		}
	}

	for i := 0; i < 14; i++ {
		candidate := maxDate.AddDate(0, 0, -i)
		if rp.hasServiceOnDate(candidate) {
			return candidate, nil
		}
	}

	return requested, nil
}

func (rp *RoutePlanner) hasServiceOnDate(date time.Time) bool {
	var count int
	dateStr := date.Format("2006-01-02")
	query := `SELECT COUNT(*)
	FROM calendar
	WHERE start_date <= ?
		AND end_date >= ?
		AND (
			(EXTRACT(DOW FROM ?::date) = 0 AND sunday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 1 AND monday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 2 AND tuesday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 3 AND wednesday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 4 AND thursday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 5 AND friday = 1) OR
			(EXTRACT(DOW FROM ?::date) = 6 AND saturday = 1)
		)`

	if err := rp.db.QueryRow(query, dateStr, dateStr, dateStr, dateStr, dateStr, dateStr, dateStr, dateStr, dateStr).Scan(&count); err != nil {
		return false
	}

	return count > 0
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
		WHERE st.stop_id = ? AND r.route_type IN (1, 3)`

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

func routeTypeToMode(routeType int) string {
	if routeType == 1 {
		return "metro"
	}
	return "bus"
}

func parseGTFSTimeOnDate(date time.Time, value string) (time.Time, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid GTFS time %q", value)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	second, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, err
	}

	dayOffset := hour / 24
	hour = hour % 24

	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		hour,
		minute,
		second,
		0,
		date.Location(),
	).AddDate(0, 0, dayOffset), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sameCalendarDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
