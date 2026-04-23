#!/bin/bash
# This script runs all migrations in order
# Kept for manual use. The Docker setup now runs migrations through cmd/migrate.

set -e

echo "Initializing database..."

# Get the migrations directory
MIGRATIONS_DIR="${MIGRATIONS_DIR:-./migrations}"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "Migrations directory not found: $MIGRATIONS_DIR"
    exit 1
fi

# Run all .up.sql files in order
for migration in $(ls $MIGRATIONS_DIR/*.up.sql | sort); do
    echo "Running migration: $(basename $migration)"
    PGPASSWORD="${DB_PASSWORD:-traveller}" psql \
        -h "${DB_HOST:-localhost}" \
        -p "${DB_PORT:-5432}" \
        -U "${DB_USER:-traveller}" \
        -d "${DB_NAME:-traveller}" \
        -f "$migration"
done

echo "Database initialization complete!"
