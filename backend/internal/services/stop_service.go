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
		FROM stops WHERE stop_id = $1`

	stop := &models.Stop{}
	var parentStation sql.NullString

	err := s.db.QueryRow(query, stopID).Scan(
		&stop.ID, &stop.Code, &stop.Name, &stop.Description,
		&stop.Latitude, &stop.Longitude, &stop.ZoneID, &stop.URL,
		&stop.LocationType, &parentStation, &stop.Timezone, &stop.WheelchairBoarding)
	if err != nil {
		return nil, fmt.Errorf("failed to get stop: %w", err)
	}

	if parentStation.Valid {
		stop.ParentStation = parentStation.String
	}

	return stop, nil
}

func (s *StopService) List(limit, offset int) ([]models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops ORDER BY stop_name LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
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

		if parentStation.Valid {
			stop.ParentStation = parentStation.String
		}

		stops = append(stops, stop)
	}

	return stops, nil
}

func (s *StopService) Search(query string, limit int) ([]models.Stop, error) {
	sqlQuery := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding
		FROM stops 
		WHERE stop_name ILIKE $1 OR stop_code ILIKE $1
		ORDER BY stop_name LIMIT $2`

	searchPattern := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
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

		if parentStation.Valid {
			stop.ParentStation = parentStation.String
		}

		stops = append(stops, stop)
	}

	return stops, nil
}

func (s *StopService) FindNearby(lat, lon float64, radiusMeters float64, limit int) ([]models.Stop, error) {
	query := `SELECT stop_id, stop_code, stop_name, stop_desc, stop_lat, stop_lon, zone_id, stop_url, 
		location_type, parent_station, stop_timezone, wheelchair_boarding,
		ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as distance
		FROM stops 
		WHERE ST_DWithin(location::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY distance
		LIMIT $4`

	rows, err := s.db.Query(query, lon, lat, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		stop := models.Stop{}
		var parentStation sql.NullString
		var distance sql.NullFloat64

		err := rows.Scan(
			&stop.ID, &stop.Code, &stop.Name, &stop.Description,
			&stop.Latitude, &stop.Longitude, &stop.ZoneID, &stop.URL,
			&stop.LocationType, &parentStation, &stop.Timezone, &stop.WheelchairBoarding,
			&distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stop: %w", err)
		}

		if parentStation.Valid {
			stop.ParentStation = parentStation.String
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
		cal.service_id
	FROM stop_times st
	JOIN trips t ON st.trip_id = t.trip_id
	JOIN routes r ON t.route_id = r.route_id
	JOIN calendar cal ON t.service_id = cal.service_id
	WHERE st.stop_id = $1
		AND cal.start_date <= CURRENT_DATE::text
		AND cal.end_date >= CURRENT_DATE::text
		AND (
			(EXTRACT(DOW FROM CURRENT_DATE) = 0 AND cal.sunday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 1 AND cal.monday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 2 AND cal.tuesday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 3 AND cal.wednesday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 4 AND cal.thursday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 5 AND cal.friday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 6 AND cal.saturday = 1)
		)
		AND st.departure_time >= CURRENT_TIME::text
	ORDER BY st.departure_time
	LIMIT $2`

	rows, err := s.db.Query(query, stopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get departures: %w", err)
	}
	defer rows.Close()

	var departures []StopDeparture
	for rows.Next() {
		dep := StopDeparture{}
		err := rows.Scan(
			&dep.TripID, &dep.RouteID, &dep.RouteShortName,
			&dep.RouteLongName, &dep.ArrivalTime, &dep.DepartureTime,
			&dep.Headsign, &dep.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan departure: %w", err)
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

