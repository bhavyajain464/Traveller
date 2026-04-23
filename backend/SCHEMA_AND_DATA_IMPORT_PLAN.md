# Schema And GTFS Import Plan

## Goal

Load the root `GTFS/` bus feed and `DMRC_GTFS/` metro feed into PostgreSQL/PostGIS without ID collisions, while keeping the schema extensible for more cities, operators, transport modes, realtime data, fares, and user journeys.

## Database Choice

Use PostgreSQL with PostGIS as the system of record.

- Relational tables model GTFS references cleanly: agencies, routes, trips, stops, stop times, service calendars, fares.
- PostGIS provides indexed geography queries for nearby stops, shapes, geofencing, and vehicle proximity.
- Redis remains a cache for active realtime state.

## Schema Layers

### 1. Feed Import Layer

This layer tracks where data came from and allows future feed versioning.

- `feed_imports`: one row per import run.
- `feed_sources`: bus, metro, future rail, ferry, tram, etc.
- Raw GTFS files should be archived outside the DB or in object storage.

The current first pass loads directly into canonical GTFS tables after cleaning. The next pass should add `feed_id` columns to canonical tables so multiple feed versions can coexist during blue/green swaps.

### 2. Canonical GTFS Layer

Existing tables stay as the first canonical layer:

- `agencies`
- `routes`
- `stops`
- `calendar`
- `trips`
- `stop_times`
- `shapes`
- `fare_attributes`
- `fare_rules`

PostGIS additions:

- `stops.stop_geog GEOGRAPHY(Point, 4326)`
- `shapes.shape_geog GEOGRAPHY(Point, 4326)`
- GiST indexes on both geography columns.

### 3. App Domain Layer

These tables support the Traveller product:

- `users`
- `journey_sessions`
- `route_boardings`
- `daily_bills`

Next production iteration should add:

- `journey_events`: immutable event log for check-in, board, alight, checkout, cancellation.
- `fare_transactions`: immutable charge/adjustment ledger.
- `payment_attempts`: payment provider state.
- `user_entitlements`: passes, discounts, subscriptions.

### 4. Realtime Layer

Redis is appropriate for short-lived realtime positions and trip updates.

Postgres should store durable operational data later:

- `realtime_feeds`
- `vehicle_positions_history` partitioned by day
- `trip_updates`
- `service_alerts`

## Cleaning Rules Before Load

The current root feeds cannot be merged raw because both bus and metro use compact IDs like `1`, `2`, `weekday`, etc. Cleaning must happen before loading.

### Namespacing

Prefix feed-local identifiers:

- Bus feed prefix: `bus:`
- Metro feed prefix: `metro:`

Fields to namespace:

- `routes.route_id`
- `stops.stop_id`
- `stops.parent_station`
- `stops.zone_id`
- `trips.trip_id`
- `trips.route_id`
- `trips.service_id`
- `trips.shape_id`
- `stop_times.trip_id`
- `stop_times.stop_id`
- `calendar.service_id`
- `shapes.shape_id`
- `fare_attributes.fare_id`
- `fare_rules.fare_id`
- `fare_rules.route_id`
- `fare_rules.origin_id`
- `fare_rules.destination_id`
- `fare_rules.contains_id`

Agency IDs are not namespaced because `DMRC`, `DIMTS`, and `DTC` are already stable operator identifiers.

### Normalization

- Trim whitespace and carriage returns from headers and values.
- Preserve GTFS times as strings because GTFS can exceed `24:00:00`.
- Normalize optional empty references to empty strings.
- Validate latitude and longitude ranges.
- Validate required references:
  - trips reference routes and services.
  - stop_times reference trips and stops.
  - fare_rules reference fare_attributes.

### Outputs

Cleaned files are written to:

- `backend/tmp/cleaned_gtfs/metro`
- `backend/tmp/cleaned_gtfs/bus`

These directories are ignored by git and can be regenerated.

## Import Flow

1. Run migrations.
2. Clean each feed into namespaced GTFS directories.
3. Validate cleaned feed counts and references.
4. Load cleaned metro and bus data into PostGIS.
5. Verify table counts.
6. Build route-planning and fare read models in later passes.

## Commands

```bash
cd backend
go run cmd/clean-gtfs/main.go -in ../DMRC_GTFS -out tmp/cleaned_gtfs/metro -prefix metro
go run cmd/clean-gtfs/main.go -in ../GTFS -out tmp/cleaned_gtfs/bus -prefix bus
DATABASE_URL="postgres://traveller:traveller@localhost:5432/traveller?sslmode=disable" go run cmd/loader-delhi/main.go -metro tmp/cleaned_gtfs/metro -bus tmp/cleaned_gtfs/bus
```
