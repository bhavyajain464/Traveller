#!/bin/bash

# Setup script for Delhi Transit Backend Database
# This script sets up the database using Docker Compose

set -e

echo "=========================================="
echo "Delhi Transit Backend - Database Setup"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed.${NC}"
    exit 1
fi

# Determine docker-compose command
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE="docker compose"
fi

echo -e "${GREEN}Step 1: Starting Docker containers...${NC}"
$DOCKER_COMPOSE up -d

echo ""
echo -e "${GREEN}Step 2: Waiting for PostgreSQL to be ready...${NC}"
# Wait for PostgreSQL to be ready
timeout=60
counter=0
until $DOCKER_COMPOSE exec -T postgres pg_isready -U postgres > /dev/null 2>&1; do
    if [ $counter -ge $timeout ]; then
        echo -e "${RED}Error: PostgreSQL did not become ready within ${timeout} seconds${NC}"
        exit 1
    fi
    echo -n "."
    sleep 2
    counter=$((counter + 2))
done
echo ""
echo -e "${GREEN}✓ PostgreSQL is ready${NC}"

echo ""
echo -e "${GREEN}Step 3: Running database migrations...${NC}"
# Run migrations
MIGRATION_FILES=$(ls migrations/*.up.sql 2>/dev/null | sort)
if [ -z "$MIGRATION_FILES" ]; then
    echo -e "${YELLOW}Warning: No migration files found in migrations/ directory${NC}"
else
    for migration in $MIGRATION_FILES; do
        echo "  Running: $(basename $migration)"
        # Copy migration to container and run it
        docker cp "$migration" transit_postgres:/tmp/$(basename $migration) 2>/dev/null || true
        $DOCKER_COMPOSE exec -T postgres psql -U postgres -d transit_db -f /tmp/$(basename $migration) 2>&1 | grep -v "already exists" || true
    done
    echo -e "${GREEN}✓ Migrations completed${NC}"
fi

echo ""
echo -e "${GREEN}Step 4: Verifying database setup...${NC}"
# Verify tables exist
TABLES=$($DOCKER_COMPOSE exec -T postgres psql -U postgres -d transit_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
if [ "$TABLES" -gt 0 ]; then
    echo -e "${GREEN}✓ Database tables created: $TABLES tables found${NC}"
else
    echo -e "${YELLOW}Warning: No tables found in database${NC}"
fi

echo ""
echo -e "${GREEN}Step 5: Checking Redis...${NC}"
if $DOCKER_COMPOSE exec -T redis redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis is ready${NC}"
else
    echo -e "${YELLOW}Warning: Redis may not be ready yet${NC}"
fi

echo ""
echo "=========================================="
echo -e "${GREEN}Database setup completed!${NC}"
echo "=========================================="
echo ""
echo "Database Connection:"
echo "  Host: localhost"
echo "  Port: 5432"
echo "  User: postgres"
echo "  Password: postgres"
echo "  Database: transit_db"
echo ""
echo "Redis Connection:"
echo "  Host: localhost"
echo "  Port: 6379"
echo ""
echo "pgAdmin (optional):"
echo "  URL: http://localhost:5050"
echo "  Email: admin@transit.local"
echo "  Password: admin"
echo ""
echo "Next steps:"
echo "  1. Copy .env.example to .env and update if needed"
echo "  2. Load GTFS data: go run cmd/loader-delhi/main.go"
echo "  3. Start the server: go run cmd/server/main.go"
echo ""
echo "To stop containers: docker-compose down"
echo "To view logs: docker-compose logs -f"
echo ""

