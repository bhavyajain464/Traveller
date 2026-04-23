CREATE TABLE IF NOT EXISTS shapes (
    shape_id VARCHAR(255) NOT NULL,
    shape_pt_lat DECIMAL(10, 8) NOT NULL,
    shape_pt_lon DECIMAL(11, 8) NOT NULL,
    shape_pt_sequence INTEGER NOT NULL,
    shape_dist_traveled DECIMAL(12, 3),
    shape_geog GEOGRAPHY(Point, 4326) GENERATED ALWAYS AS (
        ST_SetSRID(ST_MakePoint(shape_pt_lon::double precision, shape_pt_lat::double precision), 4326)::geography
    ) STORED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (shape_id, shape_pt_sequence)
);

CREATE INDEX idx_shapes_shape_id ON shapes(shape_id);
CREATE INDEX idx_shapes_geog ON shapes USING GIST(shape_geog);

CREATE TABLE IF NOT EXISTS fare_attributes (
    fare_id VARCHAR(255) PRIMARY KEY,
    price DECIMAL(10, 2) NOT NULL,
    currency_type VARCHAR(3) NOT NULL,
    payment_method INTEGER DEFAULT 0,
    transfers INTEGER DEFAULT 0,
    transfer_duration INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fare_attributes_currency ON fare_attributes(currency_type);

CREATE TABLE IF NOT EXISTS fare_rules (
    fare_id VARCHAR(255) NOT NULL,
    route_id VARCHAR(255),
    origin_id VARCHAR(255),
    destination_id VARCHAR(255),
    contains_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (fare_id) REFERENCES fare_attributes(fare_id) ON DELETE CASCADE,
    FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE CASCADE
);

CREATE INDEX idx_fare_rules_fare_id ON fare_rules(fare_id);
CREATE INDEX idx_fare_rules_route_id ON fare_rules(route_id);
CREATE INDEX idx_fare_rules_origin_destination ON fare_rules(origin_id, destination_id);

