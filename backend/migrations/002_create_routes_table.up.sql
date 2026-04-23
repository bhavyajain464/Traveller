CREATE TABLE IF NOT EXISTS routes (
    route_id VARCHAR(255) PRIMARY KEY,
    agency_id VARCHAR(255) NOT NULL,
    route_short_name VARCHAR(50),
    route_long_name VARCHAR(255),
    route_desc TEXT,
    route_type INTEGER NOT NULL,
    route_url VARCHAR(255),
    route_color VARCHAR(6),
    route_text_color VARCHAR(6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agency_id) REFERENCES agencies(agency_id) ON DELETE CASCADE
);

CREATE INDEX idx_routes_agency_id ON routes(agency_id);
CREATE INDEX idx_routes_short_name ON routes(route_short_name);
CREATE INDEX idx_routes_type ON routes(route_type);


