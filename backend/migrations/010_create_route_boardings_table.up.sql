CREATE TABLE IF NOT EXISTS route_boardings (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    route_id VARCHAR(255) NOT NULL,
    boarding_stop_id VARCHAR(255),
    alighting_stop_id VARCHAR(255),
    boarding_time TIMESTAMP NOT NULL,
    alighting_time TIMESTAMP,
    boarding_lat DECIMAL(10, 8) NOT NULL,
    boarding_lon DECIMAL(11, 8) NOT NULL,
    alighting_lat DECIMAL(10, 8),
    alighting_lon DECIMAL(11, 8),
    distance DECIMAL(10, 2) DEFAULT 0,
    fare DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES journey_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE CASCADE,
    FOREIGN KEY (boarding_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL,
    FOREIGN KEY (alighting_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_route_boardings_session_id ON route_boardings(session_id);
CREATE INDEX IF NOT EXISTS idx_route_boardings_route_id ON route_boardings(route_id);
CREATE INDEX IF NOT EXISTS idx_route_boardings_boarding_time ON route_boardings(boarding_time);
CREATE INDEX IF NOT EXISTS idx_route_boardings_session_active ON route_boardings(session_id, alighting_time) WHERE alighting_time IS NULL;


