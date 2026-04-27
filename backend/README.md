# Traveller Backend Service

A Go-based backend service for **Traveller** - an Indian public transport app similar to SBB's EasyRide system. The main feature is a **check-in/check-out system** where users check in to start a journey (get QR code), use it across multiple transport modes, check out when done, and receive a daily bill the next day.

## Features

- **Check-In/Check-Out System**: QR code-based journey tracking
- **Daily Billing**: Automatic aggregation of all journeys per day
- **GTFS Data Processing**: Parse and load GTFS feeds into PostgreSQL/PostGIS
- **Journey Planning**: Point-to-point route planning with transfers
- **Multi-Modal Support**: Support for multiple transport agencies (buses, metro, trains)
- **Real-time Information**: GTFS-RT support with Redis caching
- **Geospatial Queries**: Location-based queries using PostGIS
- **REST API**: Comprehensive REST API for all transit data

## Architecture Alignment

The backend is now being reshaped toward the `end goal` SBB-style architecture while still running as a single deployable service today.

Current runtime layering:

```text
Client apps / tools
        |
Gin HTTP router
        |
Application bootstrap (internal/app)
        |
Domain services (journey planner, realtime, fares, ticketing flows)
        |
PostgreSQL/PostGIS + Redis
```

Routing is now wired behind a planner service and adapter boundary. The default adapter is an in-memory snapshot-backed planner that owns the loaded timetable state in process, performs direct and single-transfer search from in-memory timetable indexes, and shapes that search as routing rounds rather than SQL joins. The SQL/GTFS adapter remains as a fallback for cases beyond the current in-memory rounds while we keep moving toward the end-goal RAPTOR engine.

Code structure for this transition:

```text
backend/internal/
├── app/           # bootstrap, service container, route registration
├── domain/        # provider-neutral domain concepts and lifecycle enums
├── repository/    # database access and row scanning for domain modules
├── handlers/      # HTTP adapters
├── services/      # current business logic modules
├── gtfs/          # timetable ingestion and parsing
├── database/      # PostgreSQL connection utilities
├── config/        # environment-backed config
└── models/        # persistence/API structs still being migrated upward
```

This gives us the service boundaries needed for the end-goal architecture now, while keeping existing GTFS and check-in/check-out flows working during the transition.

## Prerequisites

- Go 1.19 or higher
- PostgreSQL 16 + PostGIS
- Redis (optional, for caching and real-time data)

## Installation

1. Clone the repository:
```bash
cd backend
```

2. Install dependencies:
```bash
go mod download
```

3. Start PostgreSQL/PostGIS and Redis with Docker:
```bash
docker compose up -d postgres redis adminer
```

4. Run migrations:
```bash
cd backend
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/migrate/main.go
```

The migration runner now resolves `migrations/` safely even if a script launches it from outside `backend`, but the supported manual command is still to run it from the backend module directory.

5. Load Delhi GTFS data:
```bash
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/loader-delhi/main.go -metro ../DMRC_GTFS -bus ../GTFS
```

## Configuration

Set environment variables or use defaults:

```bash
# Server
export SERVER_PORT=8080
export SERVER_HOST=0.0.0.0

# Database
export DATABASE_URL=
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=traveller
export DB_PASSWORD=traveller
export DB_NAME=traveller
export DB_SSLMODE=disable

# Redis (optional)
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=
export REDIS_DB=0

# Auth
export GOOGLE_CLIENT_ID=
export SESSION_TOKEN_SECRET=change-me
export SESSION_DURATION_HOURS=720

# Planner
export PLANNER_ADAPTER=in_memory  # or sql

# GTFS Data
export GTFS_DATA_PATH=../DMRC_GTFS
```

`DATABASE_URL` (if set) takes precedence over the individual `DB_*` variables. This backend expects **PostgreSQL/PostGIS**.
When running locally, the backend also auto-loads `.env`, `.env.local`, `backend/.env`, and `backend/.env.local` if present.
If startup fails with a missing Phase 1 table such as `journey_events`, run the backend migrations against that same database config before restarting the server.

## Running the Server

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Running the Scheduler (Daily Bill Generation)

```bash
go run cmd/scheduler/main.go
```

This runs daily at 1 AM to generate bills for the previous day. Can also be triggered manually via API.

## API Endpoints

### Check-In/Check-Out (Core Feature)

### Authentication

- `POST /api/v1/auth/google` - Exchange a Google ID token for a Traveller session
  - Body: `{credential}`
- `GET /api/v1/auth/me` - Resolve the current user from the bearer token
- `POST /api/v1/auth/logout` - Revoke the current bearer token

- `POST /api/v1/sessions/checkin` - Check-in and generate QR code
  - Body: `{user_id, latitude, longitude, stop_id?}`
- `POST /api/v1/sessions/checkout` - Check-out and calculate fare
  - Body: `{session_id/qr_code, latitude, longitude, stop_id?}`
- `POST /api/v1/sessions/validate-qr` - Validate QR code (for conductors)
  - Body: `{qr_code, route_id}`
- `GET /api/v1/sessions/users/:user_id/active` - Get active sessions

### Daily Billing

- `GET /api/v1/bills/users/:user_id?date=YYYY-MM-DD` - Get daily bill
- `GET /api/v1/bills/users/:user_id/pending` - Get all pending bills
- `POST /api/v1/bills/:bill_id/pay` - Mark bill as paid
  - Body: `{payment_id, payment_method}`
- `POST /api/v1/bills/generate?date=YYYY-MM-DD` - Generate daily bills (admin)

### Journey Planning

- `POST /api/v1/journeys/plan?from_lat=12.9716&from_lon=77.5946&to_lat=12.9352&to_lon=77.6245&departure_time=2024-01-01T08:00:00Z`

### Stops

- `GET /api/v1/stops` - List stops (with pagination)
- `GET /api/v1/stops/search?q=Kempegowda` - Search stops
- `GET /api/v1/stops/nearby?lat=12.9716&lon=77.5946&radius=500` - Find nearby stops
- `GET /api/v1/stops/:id` - Get stop details
- `GET /api/v1/stops/:id/departures` - Get next departures

### Routes

- `GET /api/v1/routes` - List routes
- `GET /api/v1/routes/search?q=327` - Search routes
- `GET /api/v1/routes/:id` - Get route details
- `GET /api/v1/routes/:id/stops` - Get stops on a route
- `GET /api/v1/routes/:id/trips` - Get trips for a route

### Real-time

- `GET /api/v1/realtime/stops/:id` - Get real-time arrivals for a stop
- `GET /api/v1/realtime/trips/:id` - Get real-time trip updates

### Fares

- `GET /api/v1/fares/calculate?route_id=327H&from_stop_id=3280&to_stop_id=3282` - Calculate fare
- `GET /api/v1/fares/routes/:id?from_stop_id=3280&to_stop_id=3282` - Get fare information

## User Flow

1. **Check-In**: User clicks button → `POST /sessions/checkin` → Receives QR code
2. **Use QR Code**: User shows QR code to conductor → `POST /sessions/validate-qr` → Validated
3. **Check-Out**: User clicks button → `POST /sessions/checkout` → Fare calculated
4. **Daily Bill**: Next day → System generates bill → `GET /bills/users/:user_id` → User views bill
5. **Payment**: User pays → `POST /bills/:bill_id/pay` → Bill marked as paid

## Project Structure

```
backend/
├── cmd/
│   ├── server/          # Main server application
│   ├── loader/          # GTFS data loader
│   └── scheduler/       # Daily bill generation scheduler
├── internal/
│   ├── app/            # Bootstrap and route registration
│   ├── config/         # Configuration management
│   ├── database/       # Database connection
│   ├── domain/         # Provider-neutral domain concepts
│   ├── models/         # Existing persistence/API models
│   ├── repository/     # Database access helpers and scanners
│   ├── gtfs/           # GTFS parser, validator, loader
│   ├── services/       # Business logic modules
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   └── utils/          # Helper functions (QR generation)
├── migrations/         # Database migrations
└── api/               # API definitions
```

## Development

### Running Tests
```bash
go test ./...
```

### Building
```bash
go build -o bin/server cmd/server/main.go
go build -o bin/loader cmd/loader/main.go
go build -o bin/scheduler cmd/scheduler/main.go
```

## Data Sources

Currently supports:
- BMTC (Bangalore Metropolitan Transport Corporation) GTFS feed

To add more feeds:
1. Place GTFS files in separate directories
2. Use the GTFS aggregator to merge multiple feeds
3. Load the merged data

## License

MIT
