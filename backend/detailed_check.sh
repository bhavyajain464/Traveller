#!/bin/bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=traveller
DB_PASSWORD=traveller
DB_NAME=traveller

echo "=== Detailed Database Analysis ==="
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL'
-- Check row counts
SELECT 'TABLE STATISTICS:' as Info;
SELECT table_name || ': ' || COALESCE(n_live_tup, 0) as Counts
FROM pg_stat_user_tables
WHERE schemaname = 'public' AND table_name IN ('agencies', 'routes', 'stops', 'trips', 'calendar', 'stop_times');

-- Check sample data
SELECT '' as '';
SELECT '=== SAMPLE AGENCIES ===' as '';
SELECT * FROM agencies LIMIT 3;

SELECT '' as '';
SELECT '=== SAMPLE ROUTES ===' as '';
SELECT route_id, agency_id, route_short_name FROM routes LIMIT 3;

SELECT '' as '';
SELECT '=== SAMPLE STOPS ===' as '';
SELECT stop_id, stop_name FROM stops LIMIT 3;

SELECT '' as '';
SELECT '=== SAMPLE TRIPS ===' as '';
SELECT trip_id, route_id, service_id FROM trips LIMIT 3;
SQL
