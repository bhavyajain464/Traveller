# Docker Setup Guide for Delhi Transit Backend

This guide will help you set up the database and Redis using Docker Compose.

## Prerequisites

- Docker Desktop installed and running
- Docker Compose (included with Docker Desktop)

## Quick Start

### 1. Start Database and Redis

```bash
cd backend
docker compose up -d
```

This will start:
- **PostgreSQL with PostGIS** on port 5432
- **Redis** on port 6379
- **Adminer** (optional) on port 8081

### 2. Run Database Migrations

```bash
# Option 1: Use the setup script (recommended)
./scripts/setup-database.sh

# Option 2: Manual migration
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/migrate/main.go
```

### 3. Load Delhi GTFS Data

```bash
# Option 1: Use the loader script
./scripts/load-delhi-data.sh

# Option 2: Manual load
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/loader-delhi/main.go
```

### 4. Start the Backend Server

```bash
# Copy environment file
cp .env.example .env

# Start server
go run cmd/server/main.go
```

## Docker Services

### PostgreSQL (PostGIS)

- **Container**: `traveller_postgres`
- **Port**: 5432
- **Database**: `traveller`
- **User**: `traveller`
- **Password**: `traveller`
- **Image**: `postgis/postgis:16-3.4`

### Redis

- **Container**: `traveller_redis`
- **Port**: 6379
- **Image**: `redis:7-alpine`

### Adminer (Optional)

- **Container**: `traveller_adminer`
- **Port**: 8081
- **URL**: http://localhost:8081

## Environment Variables

Create a `.env` file in the `backend` directory (or copy from `.env.example`). You can optionally set `DATABASE_URL` (preferred) instead of individual `DB_*` variables:

```env
# Database (using Docker containers)
DB_HOST=localhost
DB_PORT=5432
DB_USER=traveller
DB_PASSWORD=traveller
DB_NAME=traveller
DB_SSLMODE=disable

# Redis (using Docker container)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
```

## Useful Commands

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f postgres
docker compose logs -f redis
```

### Stop Services

```bash
docker compose down
```

### Stop and Remove Volumes (⚠️ This deletes all data)

```bash
docker compose down -v
```

### Access PostgreSQL CLI

```bash
docker compose exec postgres psql -U traveller -d traveller
```

### Access Redis CLI

```bash
docker compose exec redis redis-cli
```

### Check Service Status

```bash
docker compose ps
```

### Restart Services

```bash
docker compose restart
```

## Database Migrations

Migrations are located in the `migrations/` directory. Run them with the Go migration command after the database is healthy.

To run migrations manually:

```bash
# List all migrations
ls migrations/*.up.sql

# Run all migrations
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/migrate/main.go
```

## Troubleshooting

### PostgreSQL not starting

```bash
# Check logs
docker compose logs postgres

# Check if port is already in use
lsof -i :5432

# Remove old container and volumes
docker compose down -v
docker compose up -d
```

### Redis connection issues

```bash
# Test Redis connection
docker compose exec redis redis-cli ping

# Should return: PONG
```

### Database connection from application

If you're running the Go application outside Docker, make sure:
1. Docker containers are running: `docker compose ps`
2. Port 5432 is accessible: `telnet localhost 5432`
3. Environment variables are set correctly in `.env`

### Reset Everything

```bash
# Stop and remove containers, networks, and volumes
docker compose down -v

# Remove images (optional)
docker compose down --rmi all

# Start fresh
docker compose up -d
./scripts/setup-database.sh
```

## Production Considerations

For production deployment:

1. **Change default passwords** in `docker-compose.yml`
2. **Use environment variables** for sensitive data
3. **Set up proper backups** for PostgreSQL volumes
4. **Configure Redis persistence** (already enabled with `--appendonly yes`)
5. **Use Docker secrets** or external secret management
6. **Set resource limits** for containers
7. **Use a reverse proxy** (nginx/traefik) for Adminer

## Data Persistence

Data is persisted in Docker volumes:
- `postgres_data`: PostgreSQL data files
- `redis_data`: Redis data files

These volumes persist even if containers are stopped. To remove them:

```bash
docker compose down -v
```

## Network

All services are connected via a Docker bridge network (`transit_network`), allowing them to communicate using service names:
- `postgres` (instead of `localhost`)
- `redis` (instead of `localhost`)

This is useful if you run the Go application in Docker as well.
