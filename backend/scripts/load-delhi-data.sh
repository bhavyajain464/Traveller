#!/bin/bash

# Script to load Delhi GTFS data into the database

set -e

echo "=========================================="
echo "Loading Delhi GTFS Data"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if database is running
if ! docker ps | grep -q transit_postgres; then
    echo -e "${RED}Error: PostgreSQL container is not running.${NC}"
    echo "Please run: docker-compose up -d"
    exit 1
fi

# Check if GTFS directories exist
METRO_PATH="../DMRC_GTFS"
BUS_PATH="../GTFS (1)"

if [ ! -d "$METRO_PATH" ]; then
    echo -e "${YELLOW}Warning: Metro data directory not found: $METRO_PATH${NC}"
fi

if [ ! -d "$BUS_PATH" ]; then
    echo -e "${YELLOW}Warning: Bus data directory not found: $BUS_PATH${NC}"
fi

if [ ! -d "$METRO_PATH" ] && [ ! -d "$BUS_PATH" ]; then
    echo -e "${RED}Error: No GTFS data directories found${NC}"
    exit 1
fi

echo -e "${GREEN}Loading Delhi GTFS data...${NC}"
echo ""

# Load data using the Delhi loader
go run cmd/loader-delhi/main.go

echo ""
echo -e "${GREEN}✓ Data loading completed!${NC}"
echo ""
echo "You can now start the server:"
echo "  go run cmd/server/main.go"
echo ""

