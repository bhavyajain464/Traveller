package services

import (
	"database/sql"
	"fmt"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type RouteService struct {
	db *database.DB
}

type RouteDetail struct {
	Route         models.Route  `json:"route"`
	Mode          string        `json:"mode"`
	StopCount     int           `json:"stop_count"`
	TripCount     int           `json:"trip_count"`
	FirstStopName string        `json:"first_stop_name,omitempty"`
	LastStopName  string        `json:"last_stop_name,omitempty"`
	Stops         []models.Stop `json:"stops"`
	Trips         []models.Trip `json:"trips"`
}

func NewRouteService(db *database.DB) *RouteService {
	return &RouteService{db: db}
}

func (s *RouteService) GetByID(routeID string) (*models.Route, error) {
	query := `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
		route_type, route_url, route_color, route_text_color
		FROM routes WHERE route_id = ?`

	route := &models.Route{}
	err := scanRouteRow(
		s.db.QueryRow(query, routeID),
		route,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	return route, nil
}

func (s *RouteService) List(limit, offset int, agencyID string) ([]models.Route, error) {
	var query string
	var args []interface{}

	if agencyID != "" {
		query = `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
			route_type, route_url, route_color, route_text_color
			FROM routes WHERE agency_id = ? ORDER BY route_short_name LIMIT ? OFFSET ?`
		args = []interface{}{agencyID, limit, offset}
	} else {
		query = `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
			route_type, route_url, route_color, route_text_color
			FROM routes ORDER BY route_short_name LIMIT ? OFFSET ?`
		args = []interface{}{limit, offset}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		route := models.Route{}
		err := scanRouteRow(rows, &route)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (s *RouteService) Search(query string, limit int) ([]models.Route, error) {
	sqlQuery := `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
		route_type, route_url, route_color, route_text_color
		FROM routes 
		WHERE LOWER(route_id) LIKE LOWER(?) OR LOWER(route_short_name) LIKE LOWER(?) OR LOWER(route_long_name) LIKE LOWER(?)
		ORDER BY route_short_name LIMIT ?`

	searchPattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchPattern, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search routes: %w", err)
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		route := models.Route{}
		err := scanRouteRow(rows, &route)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (s *RouteService) GetDetail(routeID string, tripLimit int) (*RouteDetail, error) {
	route, err := s.GetByID(routeID)
	if err != nil {
		return nil, err
	}

	stops, err := s.GetStops(routeID)
	if err != nil {
		return nil, err
	}

	trips, err := s.GetTrips(routeID, tripLimit)
	if err != nil {
		return nil, err
	}

	tripCount, err := s.getTripCount(routeID)
	if err != nil {
		return nil, err
	}

	detail := &RouteDetail{
		Route:     *route,
		Mode:      routeModeLabel(route.Type),
		StopCount: len(stops),
		TripCount: tripCount,
		Stops:     stops,
		Trips:     trips,
	}

	if len(stops) > 0 {
		detail.FirstStopName = stops[0].Name
		detail.LastStopName = stops[len(stops)-1].Name
	}

	return detail, nil
}

func (s *RouteService) GetStops(routeID string) ([]models.Stop, error) {
	query := `SELECT s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon, 
		s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
		FROM stops s
		JOIN stop_times st ON s.stop_id = st.stop_id
		JOIN trips t ON st.trip_id = t.trip_id
		WHERE t.route_id = ?
		GROUP BY s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon,
			s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
		ORDER BY MIN(st.stop_sequence)`

	rows, err := s.db.Query(query, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get route stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	seenStops := make(map[string]bool)

	for rows.Next() {
		stop := models.Stop{}
		err := scanStopRow(rows, &stop, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
		}

		if !seenStops[stop.ID] {
			stops = append(stops, stop)
			seenStops[stop.ID] = true
		}
	}

	return stops, nil
}

func (s *RouteService) GetTrips(routeID string, limit int) ([]models.Trip, error) {
	query := `SELECT trip_id, route_id, service_id, trip_headsign, trip_short_name, 
		direction_id, block_id, shape_id, wheelchair_accessible, bikes_allowed
		FROM trips WHERE route_id = ? ORDER BY trip_id LIMIT ?`

	rows, err := s.db.Query(query, routeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get route trips: %w", err)
	}
	defer rows.Close()

	var trips []models.Trip
	for rows.Next() {
		trip := models.Trip{}
		var directionID, wheelchairAccessible, bikesAllowed sql.NullInt64
		var headsign, shortName, blockID, shapeID sql.NullString

		err := rows.Scan(
			&trip.ID, &trip.RouteID, &trip.ServiceID, &headsign, &shortName,
			&directionID, &blockID, &shapeID, &wheelchairAccessible, &bikesAllowed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}

		if headsign.Valid {
			trip.Headsign = headsign.String
		}
		if shortName.Valid {
			trip.ShortName = shortName.String
		}
		if directionID.Valid {
			d := int(directionID.Int64)
			trip.DirectionID = &d
		}
		if blockID.Valid {
			trip.BlockID = blockID.String
		}
		if shapeID.Valid {
			trip.ShapeID = shapeID.String
		}
		if wheelchairAccessible.Valid {
			w := int(wheelchairAccessible.Int64)
			trip.WheelchairAccessible = &w
		}
		if bikesAllowed.Valid {
			b := int(bikesAllowed.Int64)
			trip.BikesAllowed = &b
		}

		trips = append(trips, trip)
	}

	return trips, nil
}

func (s *RouteService) getTripCount(routeID string) (int, error) {
	query := `SELECT COUNT(*) FROM trips WHERE route_id = ?`

	var tripCount int
	if err := s.db.QueryRow(query, routeID).Scan(&tripCount); err != nil {
		return 0, fmt.Errorf("failed to count route trips: %w", err)
	}

	return tripCount, nil
}

type routeScanner interface {
	Scan(dest ...any) error
}

func scanRouteRow(scanner routeScanner, route *models.Route) error {
	var shortName, longName, description sql.NullString
	var routeURL, color, textColor sql.NullString

	if err := scanner.Scan(
		&route.ID,
		&route.AgencyID,
		&shortName,
		&longName,
		&description,
		&route.Type,
		&routeURL,
		&color,
		&textColor,
	); err != nil {
		return err
	}

	if shortName.Valid {
		route.ShortName = shortName.String
	}
	if longName.Valid {
		route.LongName = longName.String
	}
	if description.Valid {
		route.Description = description.String
	}
	if routeURL.Valid {
		route.URL = routeURL.String
	}
	if color.Valid {
		route.Color = color.String
	}
	if textColor.Valid {
		route.TextColor = textColor.String
	}

	return nil
}

func routeModeLabel(routeType int) string {
	switch routeType {
	case 0:
		return "tram"
	case 1:
		return "metro"
	case 2:
		return "rail"
	case 3:
		return "bus"
	case 4:
		return "ferry"
	case 5:
		return "cable tram"
	case 6:
		return "gondola"
	case 7:
		return "funicular"
	default:
		return "transit"
	}
}

// GetRouteByID is an alias for GetByID (for consistency)
func (s *RouteService) GetRouteByID(routeID string) (*models.Route, error) {
	return s.GetByID(routeID)
}

// GetActiveRoutes returns a list of active routes (for vehicle tracking)
func (s *RouteService) GetActiveRoutes(limit int) ([]models.Route, error) {
	return s.List(limit, 0, "")
}
