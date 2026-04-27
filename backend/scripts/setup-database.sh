#!/bin/bash

# Setup script for Traveller Backend Database (PostgreSQL + PostGIS)
# This script sets up the database using Docker Compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCKER_COMPOSE_FILE="$BACKEND_DIR/docker-compose.yml"

echo "=========================================="
echo "Traveller Backend - Database Setup"
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
$DOCKER_COMPOSE -f "$DOCKER_COMPOSE_FILE" up -d

echo ""
echo -e "${GREEN}Step 2: Waiting for PostgreSQL/PostGIS to be ready...${NC}"
timeout=60
counter=0
until $DOCKER_COMPOSE -f "$DOCKER_COMPOSE_FILE" exec -T postgres pg_isready -U traveller -d traveller > /dev/null 2>&1; do
    if [ $counter -ge $timeout ]; then
        echo -e "${RED}Error: PostgreSQL did not become ready within ${timeout} seconds${NC}"
        exit 1
    fi
    echo -n "."
    sleep 2
    counter=$((counter + 2))
done
echo ""
echo -e "${GREEN}✓ PostgreSQL/PostGIS is ready${NC}"

echo ""
echo -e "${GREEN}Step 3: Running database migrations...${NC}"
(cd "$BACKEND_DIR" && DATABASE_URL=${DATABASE_URL:-postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable} go run cmd/migrate/main.go)
echo -e "${GREEN}✓ Migrations completed${NC}"

echo ""
echo -e "${GREEN}Step 4: Verifying database setup...${NC}"
# Verify tables exist
TABLES=$($DOCKER_COMPOSE -f "$DOCKER_COMPOSE_FILE" exec -T postgres psql -U traveller -d traveller -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
if [ "$TABLES" -gt 0 ]; then
    echo -e "${GREEN}✓ Database tables created: $TABLES tables found${NC}"
else
    echo -e "${YELLOW}Warning: No tables found in database${NC}"
fi

echo ""
echo -e "${GREEN}Step 5: Checking Redis...${NC}"
if $DOCKER_COMPOSE -f "$DOCKER_COMPOSE_FILE" exec -T redis redis-cli ping > /dev/null 2>&1; then
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
echo "  User: traveller"
echo "  Password: traveller"
echo "  Database: traveller"
echo ""
echo "Redis Connection:"
echo "  Host: localhost"
echo "  Port: 6379"
echo ""
echo "Adminer (optional):"
echo "  URL: http://localhost:8081"
echo ""
echo "Next steps:"
echo "  1. Copy .env.example to .env and update if needed"
echo "  2. Load GTFS data: go run cmd/loader-delhi/main.go"
echo "  3. Start the server: go run cmd/server/main.go"
echo ""
echo "To stop containers: $DOCKER_COMPOSE -f \"$DOCKER_COMPOSE_FILE\" down"
echo "To view logs: $DOCKER_COMPOSE -f \"$DOCKER_COMPOSE_FILE\" logs -f"
echo ""
