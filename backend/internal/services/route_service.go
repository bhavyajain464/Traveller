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

func NewRouteService(db *database.DB) *RouteService {
	return &RouteService{db: db}
}

func (s *RouteService) GetByID(routeID string) (*models.Route, error) {
	query := `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
		route_type, route_url, route_color, route_text_color
		FROM routes WHERE route_id = $1`

	route := &models.Route{}
	err := s.db.QueryRow(query, routeID).Scan(
		&route.ID, &route.AgencyID, &route.ShortName, &route.LongName,
		&route.Description, &route.Type, &route.URL, &route.Color, &route.TextColor)
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
			FROM routes WHERE agency_id = $1 ORDER BY route_short_name LIMIT $2 OFFSET $3`
		args = []interface{}{agencyID, limit, offset}
	} else {
		query = `SELECT route_id, agency_id, route_short_name, route_long_name, route_desc, 
			route_type, route_url, route_color, route_text_color
			FROM routes ORDER BY route_short_name LIMIT $1 OFFSET $2`
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
		err := rows.Scan(
			&route.ID, &route.AgencyID, &route.ShortName, &route.LongName,
			&route.Description, &route.Type, &route.URL, &route.Color, &route.TextColor)
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
		WHERE route_short_name ILIKE $1 OR route_long_name ILIKE $1
		ORDER BY route_short_name LIMIT $2`

	searchPattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search routes: %w", err)
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		route := models.Route{}
		err := rows.Scan(
			&route.ID, &route.AgencyID, &route.ShortName, &route.LongName,
			&route.Description, &route.Type, &route.URL, &route.Color, &route.TextColor)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (s *RouteService) GetStops(routeID string) ([]models.Stop, error) {
	query := `SELECT DISTINCT s.stop_id, s.stop_code, s.stop_name, s.stop_desc, s.stop_lat, s.stop_lon, 
		s.zone_id, s.stop_url, s.location_type, s.parent_station, s.stop_timezone, s.wheelchair_boarding
		FROM stops s
		JOIN stop_times st ON s.stop_id = st.stop_id
		JOIN trips t ON st.trip_id = t.trip_id
		WHERE t.route_id = $1
		ORDER BY MIN(st.stop_sequence) OVER (PARTITION BY t.trip_id, s.stop_id)`

	rows, err := s.db.Query(query, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get route stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	seenStops := make(map[string]bool)

	for rows.Next() {
		stop := models.Stop{}
		var parentStation sql.NullString

		err := rows.Scan(
			&stop.ID, &stop.Code, &stop.Name, &stop.Description,
			&stop.Latitude, &stop.Longitude, &stop.ZoneID, &stop.URL,
			&stop.LocationType, &parentStation, &stop.Timezone, &stop.WheelchairBoarding)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
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

func (s *RouteService) GetTrips(routeID string, limit int) ([]models.Trip, error) {
	query := `SELECT trip_id, route_id, service_id, trip_headsign, trip_short_name, 
		direction_id, block_id, shape_id, wheelchair_accessible, bikes_allowed
		FROM trips WHERE route_id = $1 ORDER BY trip_id LIMIT $2`

	rows, err := s.db.Query(query, routeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get route trips: %w", err)
	}
	defer rows.Close()

	var trips []models.Trip
	for rows.Next() {
		trip := models.Trip{}
		var directionID, wheelchairAccessible, bikesAllowed sql.NullInt64

		err := rows.Scan(
			&trip.ID, &trip.RouteID, &trip.ServiceID, &trip.Headsign, &trip.ShortName,
			&directionID, &trip.BlockID, &trip.ShapeID, &wheelchairAccessible, &bikesAllowed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}

		if directionID.Valid {
			d := int(directionID.Int64)
			trip.DirectionID = &d
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


