package services

import (
	"database/sql"
	"fmt"
	"math"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type StopService struct {
	db *database.DB
}

func NewStopService(db *database.DB) *StopService {
	return &StopService{db: db}
}

func (s *StopService) GetByID(stopID string) (*models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops WHERE stop_id = ?`

	stop := &models.Stop{}
	err := scanStopRow(
		s.db.QueryRow(query, stopID),
		stop,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stop: %w", err)
	}

	return stop, nil
}

func (s *StopService) List(limit, offset int) ([]models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops ORDER BY stop_name LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		stop := models.Stop{}
		err := scanStopRow(rows, &stop, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
		}

		stops = append(stops, stop)
	}

	return stops, nil
}

func (s *StopService) Search(query string, limit int) ([]models.Stop, error) {
	sqlQuery := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops 
		WHERE LOWER(stop_name) LIKE LOWER(?) OR LOWER(stop_code) LIKE LOWER(?)
		ORDER BY stop_name LIMIT ?`

	searchPattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		stop := models.Stop{}
		err := scanStopRow(rows, &stop, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
		}

		stops = append(stops, stop)
	}

	return stops, nil
}

func (s *StopService) FindNearby(lat, lon float64, radiusMeters float64, limit int) ([]models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url,
		location_type, parent_station, stop_timezone, wheelchair_boarding, distance
		FROM (
			SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url,
				location_type, parent_station, stop_timezone, wheelchair_boarding,
				ST_Distance(stop_geog, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance
			FROM stops
		) nearby_stops
		WHERE distance <= ?
		ORDER BY distance
		LIMIT ?`

	rows, err := s.db.Query(query, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		stop := models.Stop{}
		var distance sql.NullFloat64

		err := scanStopRow(rows, &stop, &distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
		}

		stops = append(stops, stop)
	}

	return stops, nil
}

func (s *StopService) GetDepartures(stopID string, limit int) ([]StopDeparture, error) {
	query := `SELECT 
		st.trip_id,
		t.route_id,
		r.route_short_name,
		r.route_long_name,
		st.arrival_time,
		st.departure_time,
		t.trip_headsign,
		t.service_id
	FROM stop_times st
	JOIN trips t ON st.trip_id = t.trip_id
	JOIN routes r ON t.route_id = r.route_id
	WHERE st.stop_id = ?
	ORDER BY st.departure_time
	LIMIT ?`

	rows, err := s.db.Query(query, stopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get departures: %w", err)
	}
	defer rows.Close()

	var departures []StopDeparture
	for rows.Next() {
		dep := StopDeparture{}
		var routeShortName, routeLongName sql.NullString
		var headsign sql.NullString
		err := rows.Scan(
			&dep.TripID, &dep.RouteID, &routeShortName,
			&routeLongName, &dep.ArrivalTime, &dep.DepartureTime,
			&headsign, &dep.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan departure: %w", err)
		}
		if routeShortName.Valid {
			dep.RouteShortName = routeShortName.String
		}
		if routeLongName.Valid {
			dep.RouteLongName = routeLongName.String
		}
		if headsign.Valid {
			dep.Headsign = headsign.String
		}
		departures = append(departures, dep)
	}

	return departures, nil
}

type StopDeparture struct {
	TripID         string `json:"trip_id"`
	RouteID        string `json:"route_id"`
	RouteShortName string `json:"route_short_name"`
	RouteLongName  string `json:"route_long_name"`
	ArrivalTime    string `json:"arrival_time"`
	DepartureTime  string `json:"departure_time"`
	Headsign       string `json:"headsign"`
	ServiceID      string `json:"service_id"`
}

type stopScanner interface {
	Scan(dest ...any) error
}

func scanStopRow(scanner stopScanner, stop *models.Stop, distance *sql.NullFloat64) error {
	var code, description, zoneID, stopURL sql.NullString
	var parentStation, timezone sql.NullString
	var locationType, wheelchairBoarding sql.NullInt64

	destinations := []any{
		&stop.ID,
		&code,
		&stop.Name,
		&description,
		&stop.Latitude,
		&stop.Longitude,
		&zoneID,
		&stopURL,
		&locationType,
		&parentStation,
		&timezone,
		&wheelchairBoarding,
	}

	if distance != nil {
		destinations = append(destinations, distance)
	}

	if err := scanner.Scan(destinations...); err != nil {
		return err
	}

	if code.Valid {
		stop.Code = code.String
	}
	if description.Valid {
		stop.Description = description.String
	}
	if zoneID.Valid {
		stop.ZoneID = zoneID.String
	}
	if stopURL.Valid {
		stop.URL = stopURL.String
	}
	if locationType.Valid {
		stop.LocationType = int(locationType.Int64)
	}
	if parentStation.Valid {
		stop.ParentStation = parentStation.String
	}
	if timezone.Valid {
		stop.Timezone = timezone.String
	}
	if wheelchairBoarding.Valid {
		stop.WheelchairBoarding = int(wheelchairBoarding.Int64)
	}

	return nil
}

// Haversine distance calculation (fallback if PostGIS not available)
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c * 1000 // Return in meters
}
