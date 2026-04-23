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
    stop_geog GEOGRAPHY(Point, 4326) GENERATED ALWAYS AS (
        ST_SetSRID(ST_MakePoint(stop_lon::double precision, stop_lat::double precision), 4326)::geography
    ) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_station) REFERENCES stops(stop_id) ON DELETE SET NULL
);

CREATE INDEX idx_stops_name ON stops(stop_name);
CREATE INDEX idx_stops_code ON stops(stop_code);
CREATE INDEX idx_stops_zone ON stops(zone_id);
CREATE INDEX idx_stops_parent ON stops(parent_station);
CREATE INDEX idx_stops_lat_lon ON stops(stop_lat, stop_lon);
CREATE INDEX idx_stops_geog ON stops USING GIST(stop_geog);

