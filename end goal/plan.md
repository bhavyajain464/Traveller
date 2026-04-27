# SBB Journey Planner — Implementation Plan

This document covers the phased implementation roadmap, technical decisions, infrastructure requirements, open questions, and risk considerations for the SBB journey planner platform.

---

## Implementation phases

### Phase 1 — Core routing (weeks 1–8)

The goal of this phase is a working journey planner that can answer static timetable queries with no real-time data.

**Deliverables:**
- GTFS timetable ingestion pipeline (SBB + PostAuto feeds)
- Timetable data normalisation and validation
- RAPTOR routing engine (in-process, single-modal)
- Basic REST API: `GET /v3/journey`, `GET /v3/stationboard`, `GET /v3/locations`
- Footpath database for major interchange stations
- Journey Planner service with horizontal scaling

**Exit criteria:**
- Returns correct Pareto-optimal journeys for 99.9% of Zürich–Bern test cases
- P50 query latency < 100ms on Swiss network graph
- API handles 1K req/s under load test

---

### Phase 2 — Multi-modal + real-time (weeks 9–16)

Extend routing to all transport modes and overlay live delay data.

**Deliverables:**
- Multi-modal RAPTOR (train + bus + tram + boat + cable car)
- Partner agency GTFS-RT adapters (ZVV, VBZ, PostAuto)
- Kafka topic schema and producer setup
- Apache Flink stream processor for delay ingestion
- Redis cluster with delay delta writes
- Real-time overlay at Journey Planner query time
- `GET /v3/connections/{id}/realtime` endpoint

**Exit criteria:**
- Real-time delay overlay functional for SBB trains
- Delay ingestion lag < 5s end-to-end
- Multi-modal journeys tested across 20 representative Swiss routes

---

### Phase 3 — Ticketing + SwissPass (weeks 17–24)

Add purchase flow, QR code generation, and discount enforcement.

**Deliverables:**
- Datatrans payment gateway integration
- TWINT, Apple Pay, Google Pay support
- Cryptographically signed QR code ticket generation
- Ticket state machine (reserved → purchased → used → expired)
- SwissPass / GA / Halbtax discount eligibility engine
- `POST /v3/tickets`, `GET /v3/tickets/{id}` endpoints
- PostgreSQL ticket schema with audit trail

**Exit criteria:**
- End-to-end purchase and QR scan tested with Datatrans sandbox
- SwissPass discount correctly applied for test accounts
- Ticket QR validated offline (no network required)

---

### Phase 4 — Notifications + client apps (weeks 25–32)

Delay push notifications and native mobile/web clients.

**Deliverables:**
- Notification service with APNs, FCM, WebSocket
- Subscription management API
- Deduplication and quiet-hours enforcement
- iOS app (SwiftUI, Core Data, offline tickets, widget)
- Android app (Compose, Room, offline tickets, Glance widget)
- Web / PWA (React, Service Workers, Web Push)

**Exit criteria:**
- Push notification delivered to device < 3s from delay event
- Offline ticket display functional without network
- Home screen widget showing next departure

---

### Phase 5 — Scale + hardening (weeks 33–40)

Production-grade reliability, observability, and performance.

**Deliverables:**
- Multi-AZ deployment with active-active failover
- CDN integration for stationboard and static assets
- Full OpenTelemetry tracing across all services
- Chaos engineering runbook (zone failure, Redis failover, Kafka lag)
- GDPR tooling (right to erasure, data export)
- Load testing at 100K req/s
- Runbook for timetable rollback

**Exit criteria:**
- 99.99% availability demonstrated over 30-day soak test
- P99 journey query < 500ms at 100K req/s
- Full observability stack operational (traces, metrics, logs)

---

## Technical decisions

### Why RAPTOR over Dijkstra / A*

Traditional graph search algorithms (Dijkstra, A*) work on spatial graphs where edge weights are static distances or travel times. Transit routing has a fundamentally different structure: the cost of an edge depends on *when* you arrive at the source node — you must wait for the next scheduled departure. RAPTOR is designed specifically for this time-expanded graph structure. Its round-based approach also naturally produces a Pareto front (time vs transfers) rather than a single optimal path.

### Why in-process graph rather than a graph database

At 50ms P50, RAPTOR needs the full timetable graph in RAM on the same machine as the query. Any network hop to an external store (even Neo4j) adds 1–5ms of overhead per graph traversal, compounding to 100s of ms over a full RAPTOR run. Neo4j is used for the canonical store and admin queries, but the RAPTOR binary format (exported nightly) is optimised for sequential memory access patterns and loaded directly into process memory.

### Why Kafka rather than direct service calls

Real-time delay events arrive every 30 seconds from multiple upstream systems and must fan out to multiple consumers: the stream processor, the cache invalidator, the notification service, and the CDN purge worker. A direct call graph would create tight coupling and make it impossible to add new consumers without modifying producers. Kafka gives us durable, replayable, schema-versioned events that any service can consume independently at its own pace.

### Why Redis for delay deltas rather than querying PostgreSQL

Delay data is read at every journey query — up to 100K times per second at peak. The data is small (a few bytes per trip ID), highly volatile, and has no relational structure. Redis at 100K ops/s with sub-millisecond latency is the right tool. PostgreSQL is reserved for durable, relational data (tickets, users) where ACID guarantees and complex queries matter.

### Why signed QR codes for offline ticket validation

Train inspectors need to validate tickets without network connectivity (in tunnels, remote areas). The QR code embeds the ticket payload and an ECDSA signature using a SBB-controlled private key. The inspector app bundles the corresponding public key and can verify the signature offline. The payload includes an expiry timestamp and a trip ID, so the inspector can confirm validity without a network call.

---

## Infrastructure requirements

### Compute

| Service | Instance type | Count (prod) | Scaling |
|---|---|---|---|
| Journey Planner | 16-core / 32GB RAM | 20 | Horizontal, stateless |
| Real-time Engine | 8-core / 16GB RAM | 6 | Flink auto-scaling |
| Timetable Service | 4-core / 8GB RAM | 3 | Fixed (low volume) |
| Ticketing Service | 8-core / 16GB RAM | 8 | Horizontal, stateless |
| Notification Service | 4-core / 8GB RAM | 6 | Horizontal, stateless |

### Storage

| Store | Size (prod) | Replication |
|---|---|---|
| Neo4j | 200GB | 1 primary + 2 replicas |
| Redis | 50GB RAM | 3-node cluster + Sentinel |
| PostgreSQL | 2TB | 1 primary + 1 standby + 1 read replica |
| S3 | 10TB | Standard (11 9s durability) |

### Kafka

- 6 brokers, 3 availability zones
- Topics: `rt.delays`, `rt.cancellations`, `rt.additions`, `timetable.updates`, `ticket.events`, `notif.triggers`
- Retention: rt topics 1h, timetable 7 days, ticket.events 7 years

### CDN

- Cloudflare or Fastly with Swiss PoP (Zürich)
- Stationboard cache: 30s TTL, purge via Kafka consumer
- Static assets: immutable caching (content-hashed filenames)
- Map tiles: 24h TTL

---

## Operational runbooks

### Timetable rollback

If a bad timetable causes incorrect routing results:

1. Identify the last known-good snapshot version in S3
2. Update the `TIMETABLE_VERSION` environment variable to the previous date
3. Trigger a rolling restart of all Journey Planner instances
4. Each instance loads the previous snapshot at startup
5. Monitor query accuracy metrics for 15 minutes before declaring resolved

### Redis failover

If the primary Redis node fails:

1. Sentinel promotes a replica automatically within 30s
2. Journey Planner instances reconnect via Sentinel endpoint (no config change needed)
3. Delay overlay is briefly unavailable during failover — queries fall back to static timetable
4. No data loss: delay deltas are ephemeral and will be re-populated from Kafka within 60s

### Kafka consumer lag alert

If the real-time stream processor falls behind:

1. Alert fires when `rt.delays` consumer lag > 500 messages
2. Check Flink job manager for task failures
3. If Flink is healthy: scale up task managers (auto-scaling should have already triggered)
4. If Flink is unhealthy: restart the job from the latest committed offset
5. Lag clears within 2–3 minutes of Flink recovery; no manual intervention needed for routine lag spikes

---

## Open questions

### Footpath data coverage

Currently only major interchange stations have detailed footpath data (walking times between platforms). For smaller stations, a 2-minute default transfer time is used. This causes incorrect results on tight connections at unmapped stations.

**Options:**
- Survey all 300K stops manually (not feasible)
- Derive footpath times from OpenStreetMap routing (automated, ~80% accuracy)
- Use actual passenger boarding data to infer realistic transfer times

**Decision needed by:** Phase 1 exit

---

### Timetable update frequency

Annual timetable changes (Fahrplanwechsel) are handled by the nightly snapshot. But SBB publishes ad-hoc service changes (planned engineering works, diversions) with less than 24 hours' notice.

**Current approach:** Journey Planner only sees ad-hoc changes at the next 03:00 reload, meaning a planned diversion published at 18:00 won't appear in routing results until the following morning.

**Proposed solution:** Trigger an out-of-cycle snapshot reload when the Timetable Service publishes a `timetable.updates` Kafka event. Requires Journey Planner to hot-swap the in-process graph without dropping in-flight queries (a two-phase pointer swap).

**Decision needed by:** Phase 2 exit

---

### Pareto front UI presentation

The journey planning API returns a Pareto set of options (fastest journey vs fewest transfers vs cheapest). How should the mobile app default-sort these for users who don't specify a preference?

**Options:**
- Sort by departure time (simplest, used by current SBB app)
- Sort by a weighted score (e.g. 60% duration + 30% transfers + 10% cost)
- Personalise based on user's historical journey choices (ML)

**Decision needed by:** Phase 4

---

### Partner agency data quality

ZVV and VBZ GTFS-RT feeds are sometimes delayed by 2–5 minutes relative to SBB's own feed, and occasionally publish conflicting delay values for the same connecting service.

**Proposed approach:** Weight delay values by source freshness. If two sources disagree on a trip's delay, prefer the more recently updated value. Publish a data quality dashboard for each agency feed.

**Decision needed by:** Phase 2

---

## Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SBB rail control API change breaks GTFS-RT ingestion | Medium | High | Versioned adapter layer; automated contract tests against feed |
| RAPTOR graph too large for 32GB RAM after 2030 network expansion | Low | High | Monitor graph growth; prepare tiered graph partitioning design |
| Datatrans PCI audit fails | Low | High | Engage PCI QSA early in Phase 3; no card data touches our servers |
| Redis cluster split-brain during network partition | Low | Medium | Sentinel quorum requires 2/3 nodes; single-zone outage handled gracefully |
| Push notification delivery rate drops below 95% | Medium | Medium | Monitor APNs/FCM delivery receipts; fallback to in-app banner on next open |
| GDPR erasure request for ticket audit trail | Medium | Medium | Separate PII from ticket validity data; erase PII, retain anonymised audit record |

---

## Related documents

- [`README.md`](./README.md) — Architecture overview and API reference
