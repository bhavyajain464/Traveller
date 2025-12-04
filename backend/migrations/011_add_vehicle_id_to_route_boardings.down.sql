-- Remove vehicle_id column from route_boardings table
ALTER TABLE route_boardings DROP COLUMN IF EXISTS vehicle_id;

DROP INDEX IF EXISTS idx_route_boardings_vehicle_id;

