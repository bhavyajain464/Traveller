CREATE TABLE IF NOT EXISTS stops (
    stop_id VARCHAR(255) PRIMARY KEY,
    stop_code VARCHAR(50),
    stop_name VARCHAR(255) NOT NULL,
    stop_desc TEXT,
    stop_lat DECIMAL(10, 8) NOT NULL,
    stop_lon DECIMAL(11, 8) NOT NULL,
    zone_id VARCHAR(50),
    stop_url VARCHAR(255),
    location_type INTEGER DEFAULT 0,
    parent_station VARCHAR(255),
    stop_timezone VARCHAR(50),
    wheelchair_boarding INTEGER,
    location GEOMETRY(POINT, 4326),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_station) REFERENCES stops(stop_id) ON DELETE SET NULL
);

-- Create PostGIS geometry from lat/lon
CREATE INDEX IF NOT EXISTS idx_stops_location ON stops USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_stops_name ON stops(stop_name);
CREATE INDEX IF NOT EXISTS idx_stops_code ON stops(stop_code);
CREATE INDEX IF NOT EXISTS idx_stops_zone ON stops(zone_id);
CREATE INDEX IF NOT EXISTS idx_stops_parent ON stops(parent_station);

-- Update location geometry from lat/lon
CREATE OR REPLACE FUNCTION update_stop_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.stop_lon, NEW.stop_lat), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_stop_location
    BEFORE INSERT OR UPDATE ON stops
    FOR EACH ROW
    EXECUTE FUNCTION update_stop_location();


