#!/bin/bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=traveller
DB_PASSWORD=traveller
DB_NAME=traveller

echo "=== DATABASE COUNTS ==="
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL'
SELECT 'Agencies' as Table_Name, COUNT(*) as Row_Count FROM agencies
UNION ALL
SELECT 'Routes', COUNT(*) FROM routes
UNION ALL
SELECT 'Stops', COUNT(*) FROM stops
UNION ALL
SELECT 'Calendar', COUNT(*) FROM calendar
UNION ALL
SELECT 'Trips', COUNT(*) FROM trips
UNION ALL
SELECT 'Stop Times', COUNT(*) FROM stop_times;
SQL
