CREATE TABLE IF NOT EXISTS planner_footpaths (
    from_stop_id VARCHAR(255) NOT NULL,
    to_stop_id VARCHAR(255) NOT NULL,
    duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
    distance_meters DOUBLE PRECISION NOT NULL DEFAULT 0,
    indoor BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (from_stop_id, to_stop_id),
    FOREIGN KEY (from_stop_id) REFERENCES stops(stop_id) ON DELETE CASCADE,
    FOREIGN KEY (to_stop_id) REFERENCES stops(stop_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_planner_footpaths_from_stop_id ON planner_footpaths(from_stop_id);
CREATE INDEX IF NOT EXISTS idx_planner_footpaths_to_stop_id ON planner_footpaths(to_stop_id);

INSERT INTO planner_footpaths (from_stop_id, to_stop_id, duration_seconds, distance_meters, indoor)
VALUES
    ('metro:81', 'metro:500', 720, 850, FALSE),
    ('metro:500', 'metro:81', 720, 850, FALSE)
ON CONFLICT (from_stop_id, to_stop_id) DO UPDATE
SET duration_seconds = EXCLUDED.duration_seconds,
    distance_meters = EXCLUDED.distance_meters,
    indoor = EXCLUDED.indoor,
    updated_at = CURRENT_TIMESTAMP;
