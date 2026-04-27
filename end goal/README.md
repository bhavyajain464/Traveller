# SBB Journey Planner — System Design

A production-grade system design for the SBB (Swiss Federal Railways) multi-modal journey planning platform. Covers architecture, algorithms, data models, real-time pipelines, and API contracts for a service handling 500M journey queries per day across Switzerland's rail, bus, tram, and boat network.

---

## Overview

| Metric | Value |
|---|---|
| Daily active users | 10M |
| Journey queries / day | 500M |
| Peak throughput | 100K req/s |
| Stops in graph | 300K |
| Scheduled trips / day | 1.2M |
| Timetable data / year | ~5TB |
| P50 query latency | 50ms (RAPTOR) |
| P99 query latency | <500ms |
| Availability target | 99.99% |

---

## Architecture

The system is organised into six horizontal layers:

```
┌─────────────────────────────────────────────────────┐
│  Client Layer    iOS · Android · Web / PWA           │
├─────────────────────────────────────────────────────┤
│  Edge Layer      CDN · API Gateway · Load Balancer   │
├─────────────────────────────────────────────────────┤
│  Service Layer   Journey Planner · Real-time Engine  │
│                  Timetable · Ticketing · Notifs       │
├─────────────────────────────────────────────────────┤
│  Event Bus       Apache Kafka                        │
├─────────────────────────────────────────────────────┤
│  Data Layer      Neo4j · Redis · PostgreSQL · S3     │
├─────────────────────────────────────────────────────┤
│  External        SBB Rail Control · Partners · Pay   │
└─────────────────────────────────────────────────────┘
```

### Client layer

| Client | Stack | Key capabilities |
|---|---|---|
| iOS | Swift / SwiftUI / Core Data | Offline tickets, APNs, lock screen widget |
| Android | Kotlin / Jetpack Compose / Room | Offline tickets, FCM, Glance widget |
| Web / PWA | React / TypeScript / Service Workers | Installable, offline tickets, Web Push |

### Edge layer

- **CDN** — Cloudflare/Fastly with Swiss PoPs. Serves static assets, map tiles, and stationboard responses (30s TTL).
- **API Gateway + Load Balancer** — JWT validation, SwissPass OAuth 2.0, rate limiting (100 req/min per IP), SSL termination, circuit breaker per downstream service, OpenTelemetry tracing.

### Service layer

| Service | Responsibility | Key tech |
|---|---|---|
| Journey Planner | RAPTOR routing on in-memory graph | Go / Rust, Redis |
| Real-time Engine | Ingest and propagate delay events | Apache Flink, Kafka |
| Timetable Service | GTFS import, versioning, snapshot publish | Python, S3 |
| Ticketing Service | Fare calc, payment, signed QR generation | Java, PostgreSQL |
| Notification Service | Push / SMS / WebSocket fan-out | Node.js, APNs, FCM |

### Data layer

| Store | Purpose | Notes |
|---|---|---|
| Neo4j | Stop→Route→Trip graph | Nightly export to RAPTOR binary |
| Redis | Delay deltas, route cache, Pub/Sub | Sentinel HA, <30s failover |
| PostgreSQL | Users, tickets, subscriptions | Partitioned by date |
| S3 | Versioned timetable snapshots | Glacier lifecycle after 14 days |

### External systems

- **SBB Rail Control** — SIRI-ET / SIRI-VM + GTFS-RT feeds every 30s (trains), 60s (buses)
- **Partner Agencies** — PostAuto, ZVV, VBZ each expose independent GTFS-RT endpoints
- **Payment Gateway** — Datatrans (PCI DSS Level 1), TWINT, Apple Pay, Google Pay

---

## Routing Algorithm: RAPTOR

Journey planning uses the **RAPTOR** (Round-based Public Transit Optimized Router) algorithm.

```
Round k = 0:  Initialise — T[source]=depart, T[all others]=∞
Round k = 1:  Find all stops reachable with 0 transfers
Round k = 2:  Find all stops reachable with 1 transfer
...
Round k = n:  Converge when no stop improves
Output:       Pareto-optimal set (time, transfers, cost)
```

**Complexity:** `O(k × R × T)` time, `O(k × |stops|)` space. Typical `k = 3–5`. Swiss network P50: ~50ms.

**Multi-modal support:** Each route is tagged with a transport mode (`train`, `bus`, `tram`, `boat`, `cable_car`). RAPTOR naturally discovers inter-modal journeys by traversing footpath edges between stops of different modes.

**Real-time overlay:** At query time, delay deltas are fetched from Redis and applied on top of the static timetable snapshot loaded in process memory.

---

## Data Model

### Core types

```typescript
type Stop = {
  id: string          // "8503000" — UIC stop code
  name: string        // "Zürich HB"
  lat: number
  lon: number
  modes: Mode[]       // [train, tram, bus]
  platforms: string[]
  timezone: string
}

type Route = {
  id: string          // "IC1"
  mode: Mode
  stops: Stop[]       // ordered stop sequence
  agency: string      // "SBB" | "PostAuto"
  trips: Trip[]       // scheduled runs of this route
}

type Trip = {
  id: string
  routeId: string
  serviceDate: Date
  stopTimes: StopTime[]
  headsign: string
  cancelled: boolean  // real-time flag, overlaid from Redis
}

type Footpath = {
  fromStop: string
  toStop: string
  duration: number    // seconds
  distance: number    // metres
  indoor: boolean     // true = in-station walk
}
```

### Graph edge types

| Edge | From → To | Weight |
|---|---|---|
| Board | Stop → Trip | Departure time |
| Alight | Trip → Stop | Arrival time |
| In-vehicle | Trip stop → Trip stop | Travel time |
| Transfer | Stop → Stop (same station) | Minimum transfer buffer |
| Footpath | Stop → Stop (nearby) | Walking time |

### Timetable versioning

Timetable snapshots are published nightly to S3, versioned by date (`2024-01-15`). Each Journey Planner instance loads the current snapshot into process memory at 03:00 and serves all queries from it, overlaying real-time delay deltas from Redis. This means a bad timetable can be rolled back by deploying the previous snapshot version.

---

## Real-time Pipeline

```
SBB Rail Control
     │  SIRI-ET / GTFS-RT (every 30s)
     ▼
GTFS-RT Adapter  ──────────────────────── normalise to SIRI
     │
     ▼
Kafka topics                rt.delays · rt.cancellations · rt.additions
     │
     ├──────────────────────────────────┐
     ▼                                  ▼
Flink Stream Processor         Timetable Service
  write delay deltas               invalidate caches
     │
     ▼
Redis Cache             delay delta per trip_id (TTL = until trip ends)
     │
     ├─────────────────────────────────┐
     ▼                                 ▼
Journey Planner              Notification Service
  overlay at query time          APNs · FCM · WebSocket
                                      │
                                      ▼
                                 Mobile Clients
```

**SLA targets:**

| Metric | Target |
|---|---|
| Delay ingestion lag | < 5s |
| Push notification delivery | < 3s |
| Redis write throughput | 100K ops/s |
| Journey re-query on delay | < 200ms |

---

## API Reference

### Journey planning

```
GET /v3/journey
  ?from=8503000          # origin stop (UIC code)
  &to=8507000            # destination stop
  &time=2024-01-15T14:30 # departure time (ISO 8601)
  &mode=departure        # or "arrival"
  &results=5             # number of options
  &transportations[]=train
  &transportations[]=bus
```

**Response:**

```json
{
  "connections": [
    {
      "duration": "00:56:00",
      "transfers": 0,
      "from": {
        "station": { "id": "8503000", "name": "Zürich HB" },
        "departure": "2024-01-15T14:32:00+01:00",
        "platform": "7"
      },
      "to": {
        "station": { "id": "8507000", "name": "Bern" },
        "arrival": "2024-01-15T15:28:00+01:00"
      },
      "sections": [
        {
          "journey": { "name": "IC 1", "operator": "SBB" },
          "realtime": { "delay": 0, "cancelled": false }
        }
      ]
    }
  ]
}
```

### All endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/v3/journey` | Plan a journey |
| GET | `/v3/stationboard` | Departures / arrivals at a stop |
| GET | `/v3/locations` | Search stops or addresses |
| GET | `/v3/connections/{id}/realtime` | Live status of a saved journey |
| POST | `/v3/tickets` | Purchase a ticket |
| GET | `/v3/tickets/{id}` | Retrieve QR code payload |
| POST | `/v3/subscriptions` | Subscribe to delay notifications |
| DELETE | `/v3/subscriptions/{id}` | Unsubscribe |

### Caching strategy

| Layer | What | TTL | Invalidation |
|---|---|---|---|
| CDN | Stationboard responses | 30s | Kafka delay event |
| Redis L1 | Popular pre-computed routes | 5min | Timetable update |
| Redis L2 | Delay deltas per trip | Until trip ends | GTFS-RT push |
| In-process | RAPTOR graph in RAM | Daily | 03:00 nightly reload |

---

## Non-functional requirements

| Requirement | Target | Mechanism |
|---|---|---|
| P99 journey query latency | < 500ms | RAPTOR in-process graph, Redis cache |
| Availability | 99.99% | Multi-AZ, circuit breakers, graceful degradation |
| Real-time update lag | < 5s | Kafka + Flink streaming pipeline |
| Peak throughput | 100K req/s | Horizontal scaling of stateless services |
| GDPR compliance | Mandatory | Data minimisation, right to erasure, DPA in CH |
| Multi-language | DE / FR / IT / EN | i18n at API layer, locale param |
| Offline support | Tickets + saved routes | Core Data / Room, signed QR stored locally |

---

## Transport modes supported

- `train` — IC, IR, RE, S-Bahn
- `bus` — PostAuto regional, city buses
- `tram` — Zürich, Bern, Basel, Geneva trams
- `boat` — Lake boats (Zürichsee, Bodensee, etc.)
- `cable_car` — Mountain funiculars and aerial tramways
- `walk` — Footpath connections between stops

---

## Related documents

- [`plan.md`](./plan.md) — Implementation roadmap, phases, and open questions
