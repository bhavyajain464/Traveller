# Production Backend Revamp Plan

This backend is a solid GTFS-driven prototype, but an SBB-class app needs clearer domain boundaries, provider-neutral transport models, reliable fare products, and query paths that scale beyond a single city feed.

## Current Findings

- Entities are mostly raw GTFS tables plus app-specific journey/session tables. That is useful for ingestion, but production code needs a domain layer above GTFS for operators, transport modes, service calendars, fare products, disruption state, and user entitlements.
- Services currently mix API orchestration, SQL, fare calculation, route search, and billing side effects. This makes correctness hard to test and makes provider-specific behavior leak into generic code.
- Fare logic is hardcoded around Delhi agencies and simple distance pricing. A SBB-style clone needs configurable tariff zones, passes, half-fare/discount products, caps, transfer windows, city/network-specific rules, and settlement metadata.
- Journey planning uses direct SQL search and recursive transfer exploration. For scale, it should move toward a routing engine abstraction with precomputed stop-route indexes, realtime overlays, service-day awareness, and bounded search algorithms.
- Realtime, vehicle matching, journey sessions, route boardings, and billing are promising app-specific modules, but state transitions are not yet transactionally modeled as a ledger.
- Several schedule/billing queries were MySQL-oriented; the first PostgreSQL pass moved the hottest daily bill filters to explicit service-day ranges. The next pass should finish injected clock/timezone handling everywhere.

## Target Module Shape

- `internal/domain`: typed domain models and enums such as `TransportMode`, `JourneyStatus`, `BoardingStatus`, `FareProduct`, `TariffZone`, and `ServiceDay`.
- `internal/repository`: database access only. Repositories should accept `context.Context`, return domain models, and own scanning/null handling.
- `internal/services/journey`: check-in, boarding, alighting, checkout, and journey state transitions. Use transactions and idempotency keys.
- `internal/services/routing`: journey planning interface. Keep GTFS SQL implementation as one adapter, with room for RAPTOR/CSA or external router integration.
- `internal/services/fares`: tariff engine interface. Load fare rules/products from database, not constants.
- `internal/services/billing`: daily/monthly caps, payment state, invoice generation, and reconciliation.
- `internal/services/realtime`: GTFS-RT/SIRI/vehicle-location ingestion and freshness checks.

## Entity Roadmap

- Keep GTFS tables append/import-oriented and add feed/version metadata so routes, trips, stop times, shapes, and calendars can coexist across feed versions.
- Add normalized `transport_modes` or typed route-mode mapping. GTFS `route_type` alone is not enough for train, tram, bus, boat, cableway, funicular, metro, replacement bus, and walking transfers.
- Replace `journey_sessions.routes_used JSON` as the source of truth with immutable journey events or completed segments. JSON can remain a read-model/cache.
- Add fare tables: `fare_products`, `fare_rules`, `fare_zones`, `stop_zones`, `user_entitlements`, `fare_capping_rules`, and `fare_transactions`.
- Add operational state tables for disruptions, platform changes, cancellations, realtime trip updates, and vehicle positions with feed timestamps.

## Service Logic Roadmap

- Make all public service methods context-aware and give write paths database transactions.
- Model check-in, route boarding, alighting, checkout, bill generation, and payment as explicit state transitions.
- Make boarding/alighting idempotent using request IDs or device event IDs so mobile retries do not double-charge.
- Move distance and fare calculation behind interfaces and store the rule/version used for each charged segment.
- Add validation at request boundaries: coordinate ranges, required identifiers, active session invariants, and operator/route compatibility.
- Replace `fmt.Printf` with structured logging and request/session identifiers.

## Scale Roadmap

- Avoid `DATE(check_in_time)` filters; query `[day_start, next_day)` with indexed UTC timestamps and a service timezone.
- Add compound indexes for the highest-traffic paths: active sessions by user, active boardings by session, stop-time lookup by stop/time, route-stop lookup by route/sequence, and vehicle freshness by route/time.
- Introduce feed import versioning with blue/green activation so route data can be loaded without blocking readers.
- Cache static GTFS lookup data in memory or Redis: stops, route summaries, stop-route adjacency, and service calendars.
- Build read models for common app screens: station board, route detail, trip detail, nearby departures, active journey, daily bill.

## First Implementation Pass Completed

- Fixed `cmd/loader` so it passes the underlying `*sql.DB` into the GTFS loader.
- Fixed inferred journey distance storage to use kilometers instead of raw meters.
- Made QR code generation safe for short IDs so malformed/test input does not panic the process.

## Suggested Next Pass

1. Add typed domain constants and request validation for session and boarding status.
2. Split journey session SQL scanning into repository helpers.
3. Wrap check-in, board, alight, checkout, and daily bill updates in transactions.
4. Introduce fare-rule database tables and move the hardcoded Delhi fare rules behind a seeded configuration.
5. Replace daily bill date queries with indexed time-range queries.
