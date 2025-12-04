CREATE TABLE IF NOT EXISTS journey_sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    qr_code VARCHAR(255) UNIQUE NOT NULL,
    check_in_time TIMESTAMP NOT NULL,
    check_out_time TIMESTAMP,
    check_in_stop_id VARCHAR(255),
    check_out_stop_id VARCHAR(255),
    check_in_lat DECIMAL(10, 8) NOT NULL,
    check_in_lon DECIMAL(11, 8) NOT NULL,
    check_out_lat DECIMAL(10, 8),
    check_out_lon DECIMAL(11, 8),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled')),
    routes_used JSONB DEFAULT '[]'::jsonb,
    total_distance DECIMAL(10, 2) DEFAULT 0,
    total_fare DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (check_in_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL,
    FOREIGN KEY (check_out_stop_id) REFERENCES stops(stop_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_journey_sessions_user_id ON journey_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_journey_sessions_qr_code ON journey_sessions(qr_code);
CREATE INDEX IF NOT EXISTS idx_journey_sessions_status ON journey_sessions(status);
CREATE INDEX IF NOT EXISTS idx_journey_sessions_check_in_time ON journey_sessions(check_in_time);
CREATE INDEX IF NOT EXISTS idx_journey_sessions_user_date ON journey_sessions(user_id, (check_in_time::date));

