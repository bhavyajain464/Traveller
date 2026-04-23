CREATE TABLE IF NOT EXISTS trips (
    trip_id VARCHAR(255) PRIMARY KEY,
    route_id VARCHAR(255) NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    trip_headsign VARCHAR(255),
    trip_short_name VARCHAR(50),
    direction_id INTEGER,
    block_id VARCHAR(255),
    shape_id VARCHAR(255),
    wheelchair_accessible INTEGER,
    bikes_allowed INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE CASCADE,
    FOREIGN KEY (service_id) REFERENCES calendar(service_id) ON DELETE CASCADE
);

CREATE INDEX idx_trips_route_id ON trips(route_id);
CREATE INDEX idx_trips_service_id ON trips(service_id);
CREATE INDEX idx_trips_direction ON trips(direction_id);
CREATE INDEX idx_trips_block ON trips(block_id);


