#!/bin/bash
# This script runs all migrations in order
# It's called automatically by PostgreSQL when the container starts

set -e

echo "Initializing database..."

# Get the migrations directory
MIGRATIONS_DIR="/docker-entrypoint-initdb.d/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "Migrations directory not found: $MIGRATIONS_DIR"
    exit 1
fi

# Run all .up.sql files in order
for migration in $(ls $MIGRATIONS_DIR/*.up.sql | sort); do
    echo "Running migration: $(basename $migration)"
    psql -U postgres -d transit_db -f "$migration"
done

echo "Database initialization complete!"

