-- Add vehicle_id column to route_boardings table
ALTER TABLE route_boardings ADD COLUMN vehicle_id VARCHAR(255);

CREATE INDEX idx_route_boardings_vehicle_id ON route_boardings(vehicle_id);

