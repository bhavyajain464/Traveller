# Quick Start Guide - Docker Setup

Get the Delhi Transit Backend running in minutes using Docker!

## Prerequisites

- Docker Desktop installed and running
- Go 1.19+ installed (for running the application)

## Step-by-Step Setup

### 1. Start Database and Redis

```bash
cd backend
docker compose up -d
```

This starts:
- PostgreSQL with PostGIS (port 5432)
- Redis (port 6379)
- Adminer (optional, port 8081)

### 2. Run Database Setup Script

```bash
./scripts/setup-database.sh
```

This script will:
- Wait for PostgreSQL to be ready
- Run all database migrations
- Verify the setup

### 3. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit if needed (defaults work with Docker setup)
# nano .env
```

### 4. Load Delhi GTFS Data

```bash
# Load both Metro and Bus data
./scripts/load-delhi-data.sh

# OR manually:
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/loader-delhi/main.go
```

### 5. Start the Server

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Verify Setup

### Check Database

```bash
# Connect to PostgreSQL
docker compose exec postgres psql -U traveller -d traveller

# List tables
\dt

# Check agencies
SELECT * FROM agencies;

# Exit
\q
```

### Check Redis

```bash
# Test Redis
docker compose exec redis redis-cli ping
# Should return: PONG
```

### Test API

```bash
# Health check (if implemented)
curl http://localhost:8080/health

# Search stops
curl "http://localhost:8080/api/v1/stops/search?q=Metro"

# Plan a journey
curl -X POST "http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910"
```

## Common Commands

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f postgres
```

### Stop Services

```bash
docker compose down
```

### Restart Services

```bash
docker compose restart
```

### Reset Everything (⚠️ Deletes all data)

```bash
docker compose down -v
docker compose up -d
./scripts/setup-database.sh
./scripts/load-delhi-data.sh
```

## Troubleshooting

### Port Already in Use

If port 5432 or 6379 is already in use:

```bash
# Check what's using the port
lsof -i :5432
lsof -i :6379

# Stop conflicting services or change ports in docker-compose.yml
```

### Database Connection Failed

```bash
# Check if container is running
docker compose ps

# Check logs
docker compose logs postgres

# Test connection
docker compose exec postgres pg_isready -U traveller -d traveller
```

### Migrations Not Running

```bash
# Run migrations manually
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/migrate/main.go
```

## Next Steps

1. **Explore the API**: Check `backend/api/openapi.yaml` for API documentation
2. **Load Data**: Make sure Delhi GTFS data is loaded
3. **Test Endpoints**: Try the journey planning and stop search APIs
4. **Read Documentation**: See `README_DOCKER.md` for detailed Docker guide

## Need Help?

- Check `README_DOCKER.md` for detailed Docker documentation
- Check `DELHI_MIGRATION_SUMMARY.md` for data loading information
- View container logs: `docker compose logs -f`
