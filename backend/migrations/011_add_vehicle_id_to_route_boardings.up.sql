-- Add vehicle_id column to route_boardings table
ALTER TABLE route_boardings ADD COLUMN IF NOT EXISTS vehicle_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_route_boardings_vehicle_id ON route_boardings(vehicle_id);

