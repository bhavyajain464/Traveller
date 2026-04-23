package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"indian-transit-backend/internal/database"
)

type RealtimeService struct {
	db    *database.DB
	redis *redis.Client
	ctx   context.Context
}

func NewRealtimeService(db *database.DB, redisClient *redis.Client) *RealtimeService {
	return &RealtimeService{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

type VehiclePosition struct {
	TripID      string    `json:"trip_id"`
	RouteID     string    `json:"route_id"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Bearing     float64   `json:"bearing,omitempty"`
	Speed       float64   `json:"speed,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	StopID      string    `json:"stop_id,omitempty"`
	CurrentStop int       `json:"current_stop_sequence,omitempty"`
}

type TripUpdate struct {
	TripID          string           `json:"trip_id"`
	RouteID         string           `json:"route_id"`
	StopTimeUpdates []StopTimeUpdate `json:"stop_time_updates"`
	VehicleID       string           `json:"vehicle_id,omitempty"`
	Timestamp       time.Time        `json:"timestamp"`
	Delay           int              `json:"delay,omitempty"` // in seconds
}

type StopTimeUpdate struct {
	StopID               string    `json:"stop_id"`
	StopSequence         int       `json:"stop_sequence"`
	ArrivalTime          time.Time `json:"arrival_time,omitempty"`
	DepartureTime        time.Time `json:"departure_time,omitempty"`
	ArrivalDelay         int       `json:"arrival_delay,omitempty"`   // in seconds
	DepartureDelay       int       `json:"departure_delay,omitempty"` // in seconds
	ScheduleRelationship string    `json:"schedule_relationship,omitempty"`
}

func (s *RealtimeService) UpdateVehiclePosition(position VehiclePosition) error {
	if s.redis == nil {
		return fmt.Errorf("Redis client not available")
	}

	key := fmt.Sprintf("vehicle:%s", position.TripID)

	data, err := json.Marshal(position)
	if err != nil {
		return fmt.Errorf("failed to marshal vehicle position: %w", err)
	}

	// Cache for 5 minutes
	err = s.redis.Set(s.ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to cache vehicle position: %w", err)
	}

	// Also store in a set for route-based queries
	if position.RouteID != "" {
		routeKey := fmt.Sprintf("route:vehicles:%s", position.RouteID)
		s.redis.SAdd(s.ctx, routeKey, position.TripID)
		s.redis.Expire(s.ctx, routeKey, 5*time.Minute)
	}

	return nil
}

func (s *RealtimeService) GetVehiclePosition(tripID string) (*VehiclePosition, error) {
	if s.redis == nil {
		return nil, nil // Redis not available, return nil
	}

	key := fmt.Sprintf("vehicle:%s", tripID)

	data, err := s.redis.Get(s.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle position: %w", err)
	}

	var position VehiclePosition
	err = json.Unmarshal([]byte(data), &position)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vehicle position: %w", err)
	}

	return &position, nil
}

func (s *RealtimeService) UpdateTripUpdate(update TripUpdate) error {
	if s.redis == nil {
		return fmt.Errorf("Redis client not available")
	}

	key := fmt.Sprintf("trip:%s", update.TripID)

	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal trip update: %w", err)
	}

	// Cache for 5 minutes
	err = s.redis.Set(s.ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to cache trip update: %w", err)
	}

	return nil
}

func (s *RealtimeService) GetTripUpdate(tripID string) (*TripUpdate, error) {
	if s.redis == nil {
		return nil, nil // Redis not available, return nil
	}

	key := fmt.Sprintf("trip:%s", tripID)

	data, err := s.redis.Get(s.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trip update: %w", err)
	}

	var update TripUpdate
	err = json.Unmarshal([]byte(data), &update)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal trip update: %w", err)
	}

	return &update, nil
}

func (s *RealtimeService) GetStopArrivals(stopID string, limit int) ([]StopArrival, error) {
	// Get scheduled departures first
	query := `SELECT 
		st.trip_id,
		t.route_id,
		r.route_short_name,
		r.route_long_name,
		st.arrival_time,
		st.departure_time,
		t.trip_headsign
	FROM stop_times st
	JOIN trips t ON st.trip_id = t.trip_id
	JOIN routes r ON t.route_id = r.route_id
	JOIN calendar cal ON t.service_id = cal.service_id
	WHERE st.stop_id = ?
		AND cal.start_date <= CURRENT_DATE
		AND cal.end_date >= CURRENT_DATE
		AND (
			(EXTRACT(DOW FROM CURRENT_DATE) = 0 AND cal.sunday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 1 AND cal.monday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 2 AND cal.tuesday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 3 AND cal.wednesday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 4 AND cal.thursday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 5 AND cal.friday = 1) OR
			(EXTRACT(DOW FROM CURRENT_DATE) = 6 AND cal.saturday = 1)
		)
		AND st.departure_time >= TO_CHAR(CURRENT_TIME, 'HH24:MI:SS')
	ORDER BY st.departure_time
	LIMIT ?`

	rows, err := s.db.Query(query, stopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get stop arrivals: %w", err)
	}
	defer rows.Close()

	var arrivals []StopArrival
	for rows.Next() {
		var arrival StopArrival
		err := rows.Scan(
			&arrival.TripID, &arrival.RouteID, &arrival.RouteShortName,
			&arrival.RouteLongName, &arrival.ScheduledArrival, &arrival.ScheduledDeparture,
			&arrival.Headsign)
		if err != nil {
			continue
		}

		// Try to get real-time update
		update, err := s.GetTripUpdate(arrival.TripID)
		if err == nil && update != nil {
			// Apply real-time delays
			for _, stopUpdate := range update.StopTimeUpdates {
				if stopUpdate.StopID == stopID {
					if stopUpdate.ArrivalDelay != 0 {
						arrival.RealTimeArrival = arrival.ScheduledArrival.Add(time.Duration(stopUpdate.ArrivalDelay) * time.Second)
						arrival.HasRealTime = true
					}
					if stopUpdate.DepartureDelay != 0 {
						arrival.RealTimeDeparture = arrival.ScheduledDeparture.Add(time.Duration(stopUpdate.DepartureDelay) * time.Second)
						arrival.HasRealTime = true
					}
					break
				}
			}
		}

		arrivals = append(arrivals, arrival)
	}

	return arrivals, nil
}

type StopArrival struct {
	TripID             string    `json:"trip_id"`
	RouteID            string    `json:"route_id"`
	RouteShortName     string    `json:"route_short_name"`
	RouteLongName      string    `json:"route_long_name"`
	ScheduledArrival   time.Time `json:"scheduled_arrival"`
	ScheduledDeparture time.Time `json:"scheduled_departure"`
	RealTimeArrival    time.Time `json:"realtime_arrival,omitempty"`
	RealTimeDeparture  time.Time `json:"realtime_departure,omitempty"`
	Headsign           string    `json:"headsign"`
	HasRealTime        bool      `json:"has_realtime"`
}
