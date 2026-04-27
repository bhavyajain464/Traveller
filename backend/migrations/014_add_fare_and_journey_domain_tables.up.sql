CREATE TABLE IF NOT EXISTS fare_products (
    id VARCHAR(255) PRIMARY KEY,
    agency_id VARCHAR(255),
    product_type VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'INR',
    base_fare DECIMAL(10, 2) DEFAULT 0,
    fare_per_km DECIMAL(10, 4) DEFAULT 0,
    fare_per_stop DECIMAL(10, 4) DEFAULT 0,
    transfer_fee DECIMAL(10, 2) DEFAULT 0,
    rule_version VARCHAR(100) NOT NULL DEFAULT 'v1',
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agency_id) REFERENCES agencies(agency_id) ON DELETE SET NULL
);

CREATE INDEX idx_fare_products_agency_id ON fare_products(agency_id);
CREATE INDEX idx_fare_products_type ON fare_products(product_type);
CREATE INDEX idx_fare_products_validity ON fare_products(valid_from, valid_until);

CREATE TABLE IF NOT EXISTS fare_zones (
    id VARCHAR(255) PRIMARY KEY,
    agency_id VARCHAR(255),
    zone_code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    parent_zone_id VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agency_id) REFERENCES agencies(agency_id) ON DELETE SET NULL,
    FOREIGN KEY (parent_zone_id) REFERENCES fare_zones(id) ON DELETE SET NULL,
    UNIQUE (agency_id, zone_code)
);

CREATE INDEX idx_fare_zones_agency_id ON fare_zones(agency_id);
CREATE INDEX idx_fare_zones_parent_zone_id ON fare_zones(parent_zone_id);

CREATE TABLE IF NOT EXISTS stop_zones (
    stop_id VARCHAR(255) NOT NULL,
    zone_id VARCHAR(255) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    effective_from TIMESTAMP,
    effective_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stop_id, zone_id),
    FOREIGN KEY (stop_id) REFERENCES stops(stop_id) ON DELETE CASCADE,
    FOREIGN KEY (zone_id) REFERENCES fare_zones(id) ON DELETE CASCADE
);

CREATE INDEX idx_stop_zones_zone_id ON stop_zones(zone_id);
CREATE INDEX idx_stop_zones_effective_dates ON stop_zones(effective_from, effective_until);

CREATE TABLE IF NOT EXISTS user_entitlements (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    fare_product_id VARCHAR(255),
    entitlement_type VARCHAR(100) NOT NULL,
    reference_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (fare_product_id) REFERENCES fare_products(id) ON DELETE SET NULL
);

CREATE INDEX idx_user_entitlements_user_id ON user_entitlements(user_id);
CREATE INDEX idx_user_entitlements_product_id ON user_entitlements(fare_product_id);
CREATE INDEX idx_user_entitlements_status_dates ON user_entitlements(status, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS fare_capping_rules (
    id VARCHAR(255) PRIMARY KEY,
    agency_id VARCHAR(255),
    fare_product_id VARCHAR(255),
    cap_type VARCHAR(50) NOT NULL CHECK (cap_type IN ('daily', 'weekly', 'monthly')),
    amount DECIMAL(10, 2) NOT NULL,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'INR',
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agency_id) REFERENCES agencies(agency_id) ON DELETE SET NULL,
    FOREIGN KEY (fare_product_id) REFERENCES fare_products(id) ON DELETE SET NULL
);

CREATE INDEX idx_fare_capping_rules_agency_id ON fare_capping_rules(agency_id);
CREATE INDEX idx_fare_capping_rules_product_id ON fare_capping_rules(fare_product_id);
CREATE INDEX idx_fare_capping_rules_type_validity ON fare_capping_rules(cap_type, valid_from, valid_until);

CREATE TABLE IF NOT EXISTS journey_segments (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    route_boarding_id VARCHAR(255),
    segment_index INTEGER NOT NULL,
    route_id VARCHAR(255),
    vehicle_id VARCHAR(255),
    from_stop_id VARCHAR(255),
    to_stop_id VARCHAR(255),
    boarded_at TIMESTAMP NOT NULL,
    alighted_at TIMESTAMP,
    distance_km DECIMAL(10, 3) DEFAULT 0,
    fare_amount DECIMAL(10, 2) DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES journey_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (route_boarding_id) REFERENCES route_boardings(id) ON DELETE SET NULL,
    FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE SET NULL,
    FOREIGN KEY (from_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL,
    FOREIGN KEY (to_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL,
    UNIQUE (session_id, segment_index),
    UNIQUE (route_boarding_id)
);

CREATE INDEX idx_journey_segments_session_id ON journey_segments(session_id);
CREATE INDEX idx_journey_segments_route_id ON journey_segments(route_id);
CREATE INDEX idx_journey_segments_boarded_at ON journey_segments(boarded_at);

CREATE TABLE IF NOT EXISTS journey_events (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    route_boarding_id VARCHAR(255),
    segment_id VARCHAR(255),
    event_type VARCHAR(100) NOT NULL,
    stop_id VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    occurred_at TIMESTAMP NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES journey_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (route_boarding_id) REFERENCES route_boardings(id) ON DELETE SET NULL,
    FOREIGN KEY (segment_id) REFERENCES journey_segments(id) ON DELETE SET NULL,
    FOREIGN KEY (stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL
);

CREATE INDEX idx_journey_events_session_time ON journey_events(session_id, occurred_at);
CREATE INDEX idx_journey_events_type_time ON journey_events(event_type, occurred_at);
CREATE INDEX idx_journey_events_segment_id ON journey_events(segment_id);

CREATE TABLE IF NOT EXISTS fare_transactions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    segment_id VARCHAR(255),
    route_boarding_id VARCHAR(255),
    fare_product_id VARCHAR(255),
    capping_rule_id VARCHAR(255),
    amount DECIMAL(10, 2) NOT NULL,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'INR',
    rule_version VARCHAR(100) NOT NULL DEFAULT 'v1',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'waived', 'reversed')),
    charged_at TIMESTAMP NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES journey_sessions(id) ON DELETE SET NULL,
    FOREIGN KEY (segment_id) REFERENCES journey_segments(id) ON DELETE SET NULL,
    FOREIGN KEY (route_boarding_id) REFERENCES route_boardings(id) ON DELETE SET NULL,
    FOREIGN KEY (fare_product_id) REFERENCES fare_products(id) ON DELETE SET NULL,
    FOREIGN KEY (capping_rule_id) REFERENCES fare_capping_rules(id) ON DELETE SET NULL
);

CREATE INDEX idx_fare_transactions_user_id ON fare_transactions(user_id);
CREATE INDEX idx_fare_transactions_session_id ON fare_transactions(session_id);
CREATE INDEX idx_fare_transactions_segment_id ON fare_transactions(segment_id);
CREATE INDEX idx_fare_transactions_status_time ON fare_transactions(status, charged_at);

CREATE INDEX idx_journey_sessions_user_active_time ON journey_sessions(user_id, status, check_in_time DESC);
CREATE INDEX idx_route_boardings_session_boarding_time ON route_boardings(session_id, boarding_time DESC);
CREATE INDEX idx_daily_bills_user_status_date ON daily_bills(user_id, status, bill_date DESC);
