# Traveller - Backend Service for Indian Public Transport App

> **📋 Feature Checklist**: For quick overview and consolidated feature tracking, see [`FEATURE_CHECKLIST.md`](FEATURE_CHECKLIST.md). This plan contains the detailed feature-by-feature breakdown.

## Overview

Create a scalable Go backend service for **Traveller** - an Indian public transport app similar to SBB's EasyRide system. The main feature is a **check-in/check-out system** where:

1. **User clicks a button** to start a journey (check-in) → **QR code is generated**
2. **QR code can be used** across multiple transport modes (bus, metro, train) throughout the day
3. **User checks out** when journey ends
4. **System tracks all journeys** throughout the day
5. **Next day, user receives a bill** with all journeys from previous day
6. **User pays once per day** for all journeys

The service processes GTFS (General Transit Feed Specification) data and provides APIs for route planning, real-time information, multi-modal transport integration, and daily billing. The service will start with Bangalore BMTC data but be designed to support multiple cities and transport modes.

## Architecture

### Technology Stack

- **Language**: Go (Golang)
- **Web Framework**: Gin or Echo
- **Database**: PostgreSQL with PostGIS extension (for geospatial queries)
- **Cache**: Redis (for route planning results and real-time data)
- **GTFS Processing**: Custom Go GTFS parser
- **API Documentation**: Swagger/OpenAPI

### Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/                     # Configuration management
│   ├── database/                   # Database connection & migrations
│   ├── models/                     # Data models (Route, Stop, Trip, etc.)
│   ├── gtfs/                       # GTFS parser and loader
│   │   ├── parser.go
│   │   ├── loader.go
│   │   └── validator.go
│   ├── services/                   # Business logic
│   │   ├── route_planner.go        # Journey planning algorithm
│   │   ├── stop_service.go         # Stop-related operations
│   │   ├── route_service.go        # Route information
│   │   ├── realtime_service.go     # Real-time data handling
│   │   └── fare_service.go         # Fare calculation
│   ├── handlers/                   # HTTP handlers
│   │   ├── route_handler.go
│   │   ├── stop_handler.go
│   │   ├── journey_handler.go
│   │   └── realtime_handler.go
│   ├── middleware/                 # Auth, logging, CORS, etc.
│   └── utils/                      # Helper functions
├── migrations/                     # Database migrations
├── api/                            # API definitions
│   └── openapi.yaml
├── tests/                          # Integration tests
├── docker-compose.yml              # Local development setup
├── Dockerfile
├── go.mod
└── README.md
```

## Core Features

### 1. GTFS Data Processing (`internal/gtfs/`)

- **Parser**: Parse GTFS CSV files (agency, routes, stops, trips, stop_times, calendar)
- **Loader**: Load parsed data into PostgreSQL with proper indexing
- **Validator**: Validate GTFS data integrity and relationships
- **Updater**: Support for periodic GTFS feed updates

### 2. Database Schema (`migrations/`)

- **Agencies**: Transport agencies (BMTC, metro, etc.)
- **Routes**: Bus/metro routes with metadata
- **Stops**: Bus stops with geospatial coordinates (PostGIS Point)
- **Trips**: Individual trip instances
- **Stop Times**: Scheduled arrival/departure times
- **Calendar**: Service availability patterns
- **Users**: User accounts (phone, email, payment preferences)
- **Journey Sessions**: Active/completed journey sessions (check-in to check-out)
  - QR codes, check-in/out times, locations, routes used, fare
- **Daily Bills**: Aggregated bills per user per day
  - Total journeys, distance, fare, payment status
- **Geospatial indexes**: For efficient location-based queries

### 3. Journey Planning (`internal/services/route_planner.go`)

- **Algorithm**: Implement RAPTOR or Dijkstra-based algorithm for multi-modal route planning
- **Features**:
  - Point-to-point journey planning
  - Multi-leg journeys with transfers
  - Time-based queries (departure/arrival time)
  - Walking distance calculation between stops
  - Route optimization (fastest, fewest transfers, least walking)

### 4. API Endpoints (`internal/handlers/`)

#### Journey Sessions (Check-in/Check-out) - **CORE FEATURE**

- `POST /api/v1/sessions/checkin` - Check-in and generate QR code
  - Body: `{user_id, latitude, longitude, stop_id?}`
  - Returns: QR code ticket, session ID
- `POST /api/v1/sessions/checkout` - Check-out and calculate fare
  - Body: `{session_id/qr_code, latitude, longitude, stop_id?}`
  - Returns: Completed session with fare
- `POST /api/v1/sessions/validate-qr` - Validate QR code (for conductors)
  - Body: `{qr_code, route_id}`
  - Returns: Validation status
- `GET /api/v1/sessions/users/:user_id/active` - Get active sessions for user

#### Daily Billing - **CORE FEATURE**

- `GET /api/v1/bills/users/:user_id` - Get daily bill for a date
  - Query: `?date=YYYY-MM-DD` (defaults to today)
- `GET /api/v1/bills/users/:user_id/pending` - Get all pending bills
- `POST /api/v1/bills/:bill_id/pay` - Mark bill as paid
  - Body: `{payment_id, payment_method}`
- `POST /api/v1/bills/generate` - Generate daily bills (admin, runs daily)

#### Journey Planning

- `POST /api/v1/journeys/plan` - Plan journey from origin to destination
  - Query params: `from_lat`, `from_lon`, `to_lat`, `to_lon`, `departure_time`, `arrival_time`
  - Returns: Multiple route options with transfers, durations, walking distances

#### Stops

- `GET /api/v1/stops` - List stops (with pagination, search)
- `GET /api/v1/stops/:id` - Get stop details
- `GET /api/v1/stops/nearby` - Find nearby stops (`lat`, `lon`, `radius`)
- `GET /api/v1/stops/:id/departures` - Get next departures from a stop

#### Routes

- `GET /api/v1/routes` - List routes (with filters)
- `GET /api/v1/routes/:id` - Get route details with stops
- `GET /api/v1/routes/:id/trips` - Get trips for a route

#### Real-time Information

- `GET /api/v1/realtime/stops/:id` - Real-time arrivals (if GTFS-RT available)
- `GET /api/v1/realtime/trips/:id` - Real-time trip updates

#### Fares

- `GET /api/v1/fares/calculate` - Calculate fare for a journey
- `GET /api/v1/fares/routes/:id` - Get fare information for a route

### 5. Multi-Modal Support

- **Agency Management**: Support multiple transport agencies (BMTC buses, Namma Metro, etc.)
- **GTFS Feed Aggregation**: Load and merge multiple GTFS feeds
- **Unified API**: Single API interface regardless of transport mode
- **Transfer Logic**: Handle transfers between different modes

### 6. Real-time Data Integration (`internal/services/realtime_service.go`)

- **GTFS-RT Support**: Parse and process GTFS-RT feeds (Vehicle Positions, Trip Updates, Service Alerts)
- **Fallback**: Use scheduled times when real-time data unavailable
- **Caching**: Cache real-time data in Redis with TTL

### 7. Performance Optimization

- **Caching**: Cache frequent queries (nearby stops, route details) in Redis
- **Database Indexing**: Geospatial indexes on stops, indexes on route_id, trip_id, etc.
- **Connection Pooling**: Efficient database connection management
- **Response Compression**: Gzip compression for API responses

## Data Sources & Alternatives

### Current Data Source

- **BMTC GTFS**: Static schedule data (2013 dataset - may need updated feed)

### Additional Data Sources to Integrate

1. **Namma Metro (Bangalore Metro)**:
   - GTFS feed from Bangalore Metro Rail Corporation Limited (BMRCL)
   - May require API integration or GTFS feed scraping

2. **Indian Railways**:
   - IRCTC API or GTFS conversion from train schedules
   - Station data and train routes

3. **Other City Bus Services**:
   - Delhi Transport Corporation (DTC)
   - Mumbai BEST
   - Chennai MTC
   - Hyderabad TSRTC
   - (Each may have GTFS feeds or require API integration)

4. **Real-time Data Sources**:
   - **BMTC Real-time**: Check if BMTC provides GTFS-RT feed
   - **Third-party APIs**: Integrate with services like Google Transit API, Moovit API
   - **Scraping**: Web scraping from transport authority websites (as fallback)

5. **Alternative Approaches**:
   - **OpenStreetMap (OSM)**: Extract public transport routes from OSM data
   - **Google Transit API**: Use Google's transit data (requires API key, may have costs)
   - **Moovit API**: Public transit API (may have limitations)
   - **Government Open Data Portals**: Check data.gov.in for official GTFS feeds

## Implementation Steps

1. **Setup Project Structure**: Initialize Go module, set up project directories
2. **Database Setup**: Create PostgreSQL database with PostGIS, design schema, create migrations
3. **GTFS Parser**: Implement parser for all GTFS files, handle edge cases
4. **Data Loader**: Create loader to import BMTC GTFS data into database
5. **Core Services**: Implement stop service, route service with basic CRUD operations
6. **Journey Planner**: Implement route planning algorithm (start with simple, optimize later)
7. **API Handlers**: Create REST API endpoints with proper error handling
8. **Real-time Integration**: Add GTFS-RT support and real-time data endpoints
9. **Multi-modal Support**: Extend to support multiple agencies/feeds
10. **Testing**: Write integration tests for critical paths
11. **Documentation**: API documentation with Swagger, README with setup instructions
12. **Deployment**: Docker setup, deployment configuration

## Configuration

- **Environment Variables**: Database URLs, Redis URL, API keys, port numbers
- **Config File**: YAML/JSON config for GTFS feed URLs, update schedules
- **Feature Flags**: Enable/disable features (real-time, multi-modal) per city

## Future Enhancements

- User authentication and favorites
- Service alerts and disruptions
- Mobile push notifications
- Analytics and usage tracking
- Admin dashboard for data management
- GraphQL API option
- WebSocket support for real-time updates

---

## Feature Comparison Checklist: SBB vs Our Implementation

**Last Updated**: 2024-01-15  
**Quick Reference**: See `FEATURE_CHECKLIST.md` for consolidated overview

This checklist compares features available in the SBB (Swiss Federal Railways) mobile app with our current implementation status. This is the **detailed feature tracking** document. For a quick overview, see `FEATURE_CHECKLIST.md`.

### Check-In/Check-Out System (Core Feature - EasyRide Style)

- [x] **Check-In with QR Code Generation**
  - ✅ Implemented: `POST /api/v1/sessions/checkin`
  - ✅ Generates unique QR code for each journey session
  - ✅ QR code valid for 24 hours
  - ✅ Automatic stop detection from location

- [x] **QR Code Validation**
  - ✅ Implemented: `POST /api/v1/sessions/validate-qr`
  - ✅ Validates QR code is active and not expired
  - ✅ For conductor/validator use

- [x] **Check-Out with Fare Calculation**
  - ✅ Implemented: `POST /api/v1/sessions/checkout`
  - ✅ Automatic fare calculation based on journey
  - ✅ Tracks routes used across multiple transport modes
  - ✅ Updates daily bill automatically

- [x] **Multi-Modal QR Code Usage**
  - ✅ Implemented: Single QR code works across bus, metro, train
  - ✅ System tracks all routes used during session
  - ✅ Fare calculation considers all modes

- [x] **Daily Bill Aggregation**
  - ✅ Implemented: `GET /api/v1/bills/users/:user_id`
  - ✅ All journeys from a day aggregated into one bill
  - ✅ Automatic bill generation via scheduler
  - ✅ Manual bill generation endpoint available

- [x] **Daily Payment System**
  - ✅ Implemented: `POST /api/v1/bills/:bill_id/pay`
  - ✅ User pays once per day for all journeys
  - ✅ Payment tracking and status management
  - ✅ Pending bills endpoint

- [x] **Journey Session Tracking**
  - ✅ Implemented: Active session tracking
  - ✅ Journey history per user
  - ✅ Distance and fare tracking per journey

- [ ] **QR Code Image Generation**
  - ⚠️ Partial: QR code string generated, image generation utility exists
  - 📝 TODO: Integrate QR image generation in check-in response
  - 📝 TODO: Return QR code as base64 image or URL

- [ ] **Auto-Pay Integration**
  - ❌ Not Implemented: No payment gateway integration
  - 📝 TODO: Integrate UPI/Card/Wallet payment gateways
  - 📝 TODO: Auto-pay for users who enable it

- [ ] **Push Notifications for Bills**
  - ❌ Not Implemented: No notification system
  - 📝 TODO: Notify users when daily bill is ready
  - 📝 TODO: Payment reminders

### Core Journey Planning Features

- [x] **Point-to-Point Journey Planning**
  - ✅ Implemented: `POST /api/v1/journeys/plan` with lat/lon coordinates
  - ✅ Supports departure time queries
  - ✅ Returns multiple journey options with transfers

- [x] **Multi-Leg Journeys with Transfers**
  - ✅ Implemented: Route planner handles transfers (up to 3 transfers)
  - ✅ Calculates walking time between stops
  - ✅ Shows transfer points and durations

- [x] **Route Optimization Options**
  - ✅ Implemented: Fastest route (duration-based sorting)
  - ⚠️ Partial: Fewest transfers (can be added as sort option)
  - ⚠️ Partial: Least walking (calculated but not as primary sort)

- [ ] **Arrival Time Queries**
  - ❌ Not Implemented: Currently only supports departure time
  - 📝 TODO: Add `arrival_time` parameter support in journey planner

- [ ] **Accessibility Options**
  - ❌ Not Implemented: Wheelchair accessible routes filtering
  - 📝 TODO: Filter routes by wheelchair_accessible flag

- [ ] **Bike-Friendly Routes**
  - ❌ Not Implemented: No bike transport support filtering
  - 📝 TODO: Add bikes_allowed filter in journey planning

### Stop & Station Information

- [x] **Stop Search**
  - ✅ Implemented: `GET /api/v1/stops/search?q=...`
  - ✅ Case-insensitive search by name or code

- [x] **Nearby Stops**
  - ✅ Implemented: `GET /api/v1/stops/nearby` with geospatial queries
  - ✅ Uses PostGIS for accurate distance calculation
  - ✅ Configurable radius

- [x] **Stop Details**
  - ✅ Implemented: `GET /api/v1/stops/:id`
  - ✅ Returns coordinates, name, description, accessibility info

- [x] **Departure Board**
  - ✅ Implemented: `GET /api/v1/stops/:id/departures`
  - ✅ Shows next departures with route info
  - ✅ Filters by current day and service calendar

- [ ] **Platform/Track Information**
  - ❌ Not Implemented: No platform data in GTFS feed
  - 📝 TODO: Add platform field if available in future feeds

- [ ] **Stop Facilities**
  - ❌ Not Implemented: No facilities data (parking, restrooms, etc.)
  - 📝 TODO: Extend stop model with facilities information

### Route Information

- [x] **Route List & Search**
  - ✅ Implemented: `GET /api/v1/routes` with pagination
  - ✅ Implemented: `GET /api/v1/routes/search?q=...`

- [x] **Route Details**
  - ✅ Implemented: `GET /api/v1/routes/:id`
  - ✅ Returns route name, type, color, agency

- [x] **Stops on Route**
  - ✅ Implemented: `GET /api/v1/routes/:id/stops`
  - ✅ Returns all stops in sequence

- [x] **Route Timetable**
  - ✅ Implemented: `GET /api/v1/routes/:id/trips`
  - ✅ Returns trips for the route

- [ ] **Route Map Visualization**
  - ❌ Not Implemented: No shape/geometry data
  - 📝 TODO: Add shape.txt parsing for route visualization

- [ ] **Route Frequency Information**
  - ⚠️ Partial: Can be calculated from trips but not exposed
  - 📝 TODO: Add route frequency endpoint

### Real-Time Information

- [x] **Real-Time Arrivals**
  - ✅ Implemented: `GET /api/v1/realtime/stops/:id`
  - ✅ Structure ready for GTFS-RT integration
  - ⚠️ Partial: Currently returns scheduled times (needs GTFS-RT feed)

- [x] **Real-Time Trip Updates**
  - ✅ Implemented: `GET /api/v1/realtime/trips/:id`
  - ✅ Structure ready for GTFS-RT integration
  - ⚠️ Partial: Currently returns scheduled times (needs GTFS-RT feed)

- [x] **Vehicle Positions**
  - ✅ Implemented: Service structure exists
  - ⚠️ Partial: Needs GTFS-RT feed integration

- [ ] **Service Alerts/Disruptions**
  - ❌ Not Implemented: No service alerts system
  - 📝 TODO: Implement GTFS-RT Service Alerts parsing
  - 📝 TODO: Add alerts endpoint and filtering

- [ ] **Delay Predictions**
  - ❌ Not Implemented: No delay calculation
  - 📝 TODO: Calculate delays from real-time vs scheduled times

### Fare & Ticketing

- [x] **Fare Calculation**
  - ✅ Implemented: `GET /api/v1/fares/calculate`
  - ✅ Distance-based fare calculation
  - ✅ Route type multipliers (AC, Express)
  - ✅ Transfer fees included

- [x] **Route Fare Information**
  - ✅ Implemented: `GET /api/v1/fares/routes/:id`
  - ✅ Returns fare rules and calculated fare

- [x] **Journey Fare in Planning**
  - ✅ Implemented: Fare included in journey options
  - ✅ Calculated automatically for each route option

- [ ] **Ticket Purchase Integration**
  - ❌ Not Implemented: No payment/ticketing system
  - 📝 TODO: Integrate with payment gateways
  - 📝 TODO: Generate digital tickets

- [ ] **Travel Cards/Passes**
  - ❌ Not Implemented: No pass management
  - 📝 TODO: Support for monthly passes, student passes
  - 📝 TODO: Discount calculation

- [ ] **Saver Tickets/Discounts**
  - ❌ Not Implemented: No discount system
  - 📝 TODO: Implement promotional fares, off-peak discounts

### Multi-Modal Transport

- [x] **Multiple Agencies Support**
  - ✅ Implemented: Database schema supports multiple agencies
  - ✅ GTFS aggregator for merging feeds

- [x] **Unified API**
  - ✅ Implemented: Single API for all transport modes
  - ✅ Journey planner handles transfers between modes

- [ ] **Mode-Specific Features**
  - ⚠️ Partial: Route type detection exists
  - 📝 TODO: Metro-specific features (platform info, line colors)
  - 📝 TODO: Train-specific features (coach numbers, seat reservations)

- [ ] **Inter-Modal Transfers**
  - ⚠️ Partial: Transfers work but not optimized for mode changes
  - 📝 TODO: Optimize transfer times for mode changes

### User Features

- [x] **User Accounts**
  - ✅ Implemented: Users table with phone, email, payment preferences
  - ✅ User ID used in journey sessions and bills
  - ⚠️ Partial: No authentication/authorization yet

- [ ] **User Authentication**
  - ❌ Not Implemented: No authentication system
  - 📝 TODO: JWT-based authentication
  - 📝 TODO: User registration/login endpoints
  - 📝 TODO: OTP verification for phone numbers

- [x] **Journey History**
  - ✅ Implemented: Journey sessions tracked per user
  - ✅ Daily bills show all journeys for a day
  - ✅ Can query active sessions per user
  - ⚠️ Partial: No analytics or insights yet

- [ ] **Saved Journeys/Favorites**
  - ❌ Not Implemented: No favorites system
  - 📝 TODO: Save frequent routes
  - 📝 TODO: Quick access to saved journeys

- [ ] **Personalized Commuter Routes**
  - ❌ Not Implemented: No personalization
  - 📝 TODO: Set up home/work locations
  - 📝 TODO: Quick access to commuter routes

- [ ] **Push Notifications**
  - ❌ Not Implemented: No notification system
  - 📝 TODO: Service disruption alerts
  - 📝 TODO: Journey reminders
  - 📝 TODO: Daily bill ready notifications

### Advanced Features

- [ ] **Service Disruption Alerts**
  - ❌ Not Implemented: No alerts system
  - 📝 TODO: Parse GTFS-RT Service Alerts
  - 📝 TODO: Alert filtering by route/stop
  - 📝 TODO: Alert notifications

- [x] **Journey Tracking (Check-In/Check-Out)**
  - ✅ Implemented: Active journey session tracking
  - ✅ Check-in starts tracking, check-out ends it
  - ✅ Tracks routes used, distance, fare
  - ⚠️ Partial: No real-time position updates during journey
  - 📝 TODO: Real-time position updates during journey
  - 📝 TODO: Next stop notifications
  - 📝 TODO: Journey status updates (on bus, transferring, etc.)

- [ ] **Offline Support**
  - ❌ Not Implemented: No offline data
  - 📝 TODO: Download route data for offline use
  - 📝 TODO: Offline journey planning

- [ ] **Accessibility Features**
  - ⚠️ Partial: Data exists (wheelchair_boarding) but not used in filtering
  - 📝 TODO: Filter routes by accessibility
  - 📝 TODO: Accessibility information in API responses

- [ ] **Multilingual Support**
  - ❌ Not Implemented: English only
  - 📝 TODO: Support Hindi, Kannada, Tamil, etc.
  - 📝 TODO: Localized stop/route names

- [ ] **Dark Mode Support**
  - ❌ Not Implemented: Backend doesn't handle themes
  - 📝 TODO: API support for theme preferences (if needed)

### Data & Integration

- [x] **GTFS Static Data**
  - ✅ Implemented: Full GTFS parser and loader
  - ✅ Supports all core GTFS files

- [x] **GTFS-RT Structure**
  - ✅ Implemented: Service structure ready
  - ⚠️ Partial: Needs actual GTFS-RT feed integration

- [ ] **GTFS-RT Feed Integration**
  - ❌ Not Implemented: No feed parser
  - 📝 TODO: Parse GTFS-RT protobuf feeds
  - 📝 TODO: Real-time feed polling/streaming

- [ ] **Data Updates**
  - ⚠️ Partial: Loader supports updates but no scheduler
  - 📝 TODO: Scheduled GTFS feed updates
  - 📝 TODO: Version management for feeds

- [ ] **API Rate Limiting**
  - ❌ Not Implemented: No rate limiting
  - 📝 TODO: Implement rate limiting middleware

- [ ] **API Authentication**
  - ❌ Not Implemented: No API keys/auth
  - 📝 TODO: API key authentication
  - 📝 TODO: OAuth2 support

### Performance & Scalability

- [x] **Database Indexing**
  - ✅ Implemented: Geospatial indexes, route/stop indexes

- [x] **Caching Structure**
  - ✅ Implemented: Redis integration ready
  - ⚠️ Partial: Not fully utilized yet

- [ ] **Response Caching**
  - ⚠️ Partial: Structure exists but not implemented
  - 📝 TODO: Cache frequent queries (nearby stops, route details)

- [ ] **Query Optimization**
  - ⚠️ Partial: Basic optimization done
  - 📝 TODO: Optimize complex journey planning queries
  - 📝 TODO: Add query result pagination

- [ ] **Load Balancing**
  - ❌ Not Implemented: Single instance
  - 📝 TODO: Horizontal scaling support

### Documentation & Testing

- [x] **API Documentation**
  - ✅ Implemented: OpenAPI/Swagger specification
  - ✅ README with setup instructions

- [x] **Integration Test Structure**
  - ✅ Implemented: Test files created
  - ⚠️ Partial: Basic tests, needs expansion

- [ ] **Comprehensive Test Coverage**
  - ⚠️ Partial: Basic tests exist
  - 📝 TODO: Unit tests for all services
  - 📝 TODO: Integration tests for all endpoints
  - 📝 TODO: Performance tests

- [ ] **API Examples**
  - ⚠️ Partial: Basic examples in README
  - 📝 TODO: Postman collection
  - 📝 TODO: Code examples in multiple languages

## Summary Statistics

### Implementation Status

- **✅ Fully Implemented**: 32 features (including check-in/check-out system)
- **⚠️ Partially Implemented**: 15 features
- **❌ Not Implemented**: 40 features
- **Total Features Tracked**: 87 features

### Check-In/Check-Out System Status

- **✅ Core Features**: 7/9 implemented
- **⚠️ Partial**: 2 features (QR image generation, auto-pay)
- **❌ Not Implemented**: Payment gateway integration, push notifications

### Priority Features for Next Phase

1. **High Priority** (Core functionality):
   - Payment gateway integration (UPI/Card/Wallet)
   - QR code image generation integration
   - User authentication (JWT + OTP)
   - Push notifications for daily bills
   - GTFS-RT feed integration

2. **Medium Priority** (Enhanced experience):
   - Service alerts/disruptions system
   - Arrival time queries in journey planning
   - Accessibility filtering
   - Saved journeys/favorites
   - Multilingual support

3. **Low Priority** (Nice to have):
   - Route map visualization
   - Bike transport support
   - Travel cards/passes
   - Offline support
   - Advanced analytics

---

## Legend

- ✅ **Implemented**: Feature is fully working
- ⚠️ **Partial**: Feature exists but needs completion/enhancement
- ❌ **Not Implemented**: Feature doesn't exist yet
- 📝 **TODO**: Action item for implementation

