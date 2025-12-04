DROP TRIGGER IF EXISTS trigger_update_stop_location ON stops;
DROP FUNCTION IF EXISTS update_stop_location();
DROP INDEX IF EXISTS idx_stops_parent;
DROP INDEX IF EXISTS idx_stops_zone;
DROP INDEX IF EXISTS idx_stops_code;
DROP INDEX IF EXISTS idx_stops_name;
DROP INDEX IF EXISTS idx_stops_location;
DROP TABLE IF EXISTS stops;


