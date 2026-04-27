DROP INDEX IF EXISTS idx_daily_bills_user_status_date;
DROP INDEX IF EXISTS idx_route_boardings_session_boarding_time;
DROP INDEX IF EXISTS idx_journey_sessions_user_active_time;

DROP INDEX IF EXISTS idx_fare_transactions_status_time;
DROP INDEX IF EXISTS idx_fare_transactions_segment_id;
DROP INDEX IF EXISTS idx_fare_transactions_session_id;
DROP INDEX IF EXISTS idx_fare_transactions_user_id;
DROP TABLE IF EXISTS fare_transactions;

DROP INDEX IF EXISTS idx_journey_events_segment_id;
DROP INDEX IF EXISTS idx_journey_events_type_time;
DROP INDEX IF EXISTS idx_journey_events_session_time;
DROP TABLE IF EXISTS journey_events;

DROP INDEX IF EXISTS idx_journey_segments_boarded_at;
DROP INDEX IF EXISTS idx_journey_segments_route_id;
DROP INDEX IF EXISTS idx_journey_segments_session_id;
DROP TABLE IF EXISTS journey_segments;

DROP INDEX IF EXISTS idx_fare_capping_rules_type_validity;
DROP INDEX IF EXISTS idx_fare_capping_rules_product_id;
DROP INDEX IF EXISTS idx_fare_capping_rules_agency_id;
DROP TABLE IF EXISTS fare_capping_rules;

DROP INDEX IF EXISTS idx_user_entitlements_status_dates;
DROP INDEX IF EXISTS idx_user_entitlements_product_id;
DROP INDEX IF EXISTS idx_user_entitlements_user_id;
DROP TABLE IF EXISTS user_entitlements;

DROP INDEX IF EXISTS idx_stop_zones_effective_dates;
DROP INDEX IF EXISTS idx_stop_zones_zone_id;
DROP TABLE IF EXISTS stop_zones;

DROP INDEX IF EXISTS idx_fare_zones_parent_zone_id;
DROP INDEX IF EXISTS idx_fare_zones_agency_id;
DROP TABLE IF EXISTS fare_zones;

DROP INDEX IF EXISTS idx_fare_products_validity;
DROP INDEX IF EXISTS idx_fare_products_type;
DROP INDEX IF EXISTS idx_fare_products_agency_id;
DROP TABLE IF EXISTS fare_products;
