# Fare Calculation & Journey Tracking Documentation

## How Fare is Calculated

### Current Implementation

The fare calculation happens **automatically when a user checks out**. Here's the process:

#### Step 1: Journey Path Detection

When user checks out, the system:

1. **Gets check-in location** (stored when user checked in)
2. **Gets check-out location** (from check-out request)
3. **Uses Route Planner** to find the best route between these two points
4. **Extracts journey details**:
   - Which routes were used
   - Distance traveled
   - Number of transfers
   - Route types (AC, Express, Ordinary)

```go
// From journey_session_service.go - CheckOut function
journeyReq := models.JourneyRequest{
    FromLat: session.CheckInLat,      // Check-in location
    FromLon: session.CheckInLon,
    ToLat:   req.Latitude,            // Check-out location
    ToLon:   req.Longitude,
    DepartureTime: &session.CheckInTime,
}

options, err := s.routePlanner.PlanJourney(journeyReq)
```

#### Step 2: Fare Calculation

The fare is calculated using **distance-based pricing**:

**Formula**:
```
Fare = (Distance × FarePerKm × RouteTypeMultiplier) + TransferFees
```

**Components**:

1. **Base Fare**: ₹5.00 (minimum charge)
2. **Fare Per Kilometer**: ₹2.00/km
3. **Route Type Multipliers**:
   - AC Bus (Vayu Vajra): 1.5x
   - Express Bus: 1.2x
   - Ordinary Bus: 1.0x
4. **Transfer Fees**: ₹2.00 per transfer

**Example Calculation**:

```
Journey: 5 km on AC bus + 1 transfer + 3 km on Express bus

Leg 1 (AC Bus):
  Distance: 5 km
  Fare: 5 km × ₹2/km × 1.5 = ₹15.00

Leg 2 (Express Bus):
  Distance: 3 km
  Fare: 3 km × ₹2/km × 1.2 = ₹7.20

Transfer Fee: ₹2.00

Total: ₹15.00 + ₹7.20 + ₹2.00 = ₹24.20
Minimum: ₹5.00 (already exceeded)
Final Fare: ₹24.20
```

#### Step 3: Distance Calculation

Distance is calculated using **PostGIS geospatial functions**:

```sql
SELECT ST_Distance(
    (SELECT location FROM stops WHERE stop_id = $1)::geography,
    (SELECT location FROM stops WHERE stop_id = $2)::geography
) / 1000.0 as distance_km
```

This gives **accurate distance in kilometers** between stops using geographic coordinates.

### Code Flow

```go
// 1. Check-out triggers journey planning
options, err := s.routePlanner.PlanJourney(journeyReq)

// 2. Route planner calculates fare for best option
fare := rp.fareService.CalculateFareForJourney(bestOption, rules)

// 3. Fare calculation breaks down:
//    - For each leg: distance × fare_per_km × multiplier
//    - Add transfer fees
//    - Ensure minimum fare
```

---

## How We Know the User's Journey

### ✅ NEW APPROACH: **Actual Route Tracking**

The system now **tracks actual routes** the user takes, not just infers them. Here's how:

#### Step 1: Check-In (Start of Journey)

When user checks in:
- **Location captured**: GPS coordinates (lat/lon)
- **Nearest stop detected**: Using PostGIS geospatial query
- **Session created**: Active journey session with QR code
- **Stored**: `check_in_lat`, `check_in_lon`, `check_in_stop_id`, `check_in_time`

```go
// User clicks "Check In" button
session := &models.JourneySession{
    CheckInLat: 12.9716,      // User's GPS location
    CheckInLon: 77.5946,
    CheckInStopID: "3280",     // Nearest stop found
    CheckInTime: time.Now(),
    Status: "active"
}
```

#### Step 2: During Journey (Route Boarding Tracking) ⭐ NEW

When user boards a route (bus/metro/train):
- **QR Code Validation**: Conductor scans QR code → `POST /api/v1/sessions/validate-qr`
- **Boarding Recorded**: System automatically records:
  - Which route user boarded (`route_id`)
  - Boarding stop (`boarding_stop_id`)
  - Boarding time and location
  - Stored in `route_boardings` table

```go
// When conductor validates QR code:
POST /api/v1/sessions/validate-qr
{
  "qr_code": "TRANSIT-123...",
  "route_id": "327H"  // Bus route or metro line
}

// System automatically records boarding:
boarding := RouteBoarding{
    SessionID: session.ID,
    RouteID: "327H",
    BoardingStopID: "3280",
    BoardingTime: time.Now(),
    ...
}
```

**Manual Boarding** (if needed):
```go
POST /api/v1/boardings/board
{
  "qr_code": "TRANSIT-123...",
  "route_id": "327H",
  "latitude": 12.9716,
  "longitude": 77.5946
}
```

**Alighting from Route**:
```go
POST /api/v1/boardings/alight
{
  "boarding_id": "boarding-uuid",
  "latitude": 12.9352,
  "longitude": 77.6245
}
```

#### Step 3: Check-Out (End of Journey)

When user checks out:
- **Location captured**: GPS coordinates (lat/lon)
- **Active boarding checked**: If user is still on a route, auto-alight
- **Fare calculated**: From **actual tracked routes**, not inferred
- **Routes used**: From `route_boardings` table

```go
// User clicks "Check Out" button
// System:
// 1. Checks if user is on active route → auto-alight
// 2. Calculates fare from actual boardings:
totalDistance, totalFare, routesUsed := routeBoardingService.CalculateFareFromBoardings(sessionID)

// Uses ACTUAL routes user took:
session.RoutesUsed = routesUsed  // ["327H", "295A"] - actual routes
session.TotalFare = totalFare    // Calculated from actual distances
```

### What Gets Stored

In `journey_sessions` table:
- `check_in_lat/lon` - Where journey started
- `check_out_lat/lon` - Where journey ended
- `check_in_stop_id` - Nearest stop at start
- `check_out_stop_id` - Nearest stop at end
- `routes_used` - Array of route IDs (inferred from route planner)
- `total_distance` - Total distance in km
- `total_fare` - Calculated fare

### ✅ How Route Tracking Works

**Route Boarding Table** (`route_boardings`):
- Tracks each route segment user takes
- Records boarding and alighting stops
- Calculates distance and fare per segment
- Links to journey session

**Example Journey Tracking**:
```
Session: session-123
├── Boarding 1: Route 327H (Bus)
│   ├── Board: Stop A → 8:30 AM
│   ├── Alight: Stop B → 8:45 AM
│   ├── Distance: 5 km
│   └── Fare: ₹15.00 (AC bus)
│
├── Boarding 2: Route METRO-1 (Metro)
│   ├── Board: Stop C → 8:50 AM
│   ├── Alight: Stop D → 9:05 AM
│   ├── Distance: 8 km
│   └── Fare: ₹16.00 (Metro)
│
└── Boarding 3: Route 295A (Bus)
    ├── Board: Stop E → 9:10 AM
    ├── Alight: Stop F → 9:25 AM
    ├── Distance: 3 km
    └── Fare: ₹7.20 (Express bus)

Total: 16 km, ₹38.20
```

### Fallback: Inferred Journey

If routes are not tracked (e.g., QR validation doesn't record boarding):
- System falls back to route planner inference
- Calculates optimal route from check-in to check-out
- Less accurate but still works

### Benefits of Route Tracking

✅ **Accurate Fare Calculation**:
- Fare based on actual routes taken
- Different fares for metro vs bus vs train
- Accurate distance per route segment

✅ **Multi-Modal Support**:
- Tracks routes across different transport modes
- Metro, bus, train all tracked separately
- Each mode can have different fare rules

✅ **Journey History**:
- Complete record of user's actual journey
- Which routes, stops, times
- Useful for analytics and dispute resolution

### API Endpoints for Route Tracking

1. **Board Route** (automatic via QR validation):
   ```
   POST /api/v1/sessions/validate-qr
   {
     "qr_code": "...",
     "route_id": "327H"
   }
   ```

2. **Manual Board Route**:
   ```
   POST /api/v1/boardings/board
   {
     "qr_code": "...",
     "route_id": "327H",
     "latitude": 12.9716,
     "longitude": 77.5946
   }
   ```

3. **Alight Route**:
   ```
   POST /api/v1/boardings/alight
   {
     "boarding_id": "...",
     "latitude": 12.9352,
     "longitude": 77.6245
   }
   ```

4. **Get Session Boardings**:
   ```
   GET /api/v1/boardings/sessions/:session_id
   ```

5. **Get Active Boarding**:
   ```
   GET /api/v1/boardings/sessions/:session_id/active
   ```

---

## Example Flow

### Scenario: User travels from Point A to Point B

**Check-In** (8:30 AM):
```
Location: 12.9716, 77.5946 (near Kempegowda Bus Station)
Nearest Stop: "3280" (Palace Ground)
QR Code Generated: TRANSIT-1234567890-abc12345-user123
```

**During Journey**:
```
- User boards Route 327H (validated QR code)
- Transfers to Route 295A
- Uses same QR code for both routes
```

**Check-Out** (9:15 AM):
```
Location: 12.9352, 77.6245 (near destination)
Nearest Stop: "3282" (Pallavi Talkies)

System calculates:
- Best route: 327H → 295A (inferred)
- Distance: 12.5 km
- Fare: ₹25.00
- Routes Used: ["327H", "295A"]
```

**Stored in Database**:
```json
{
  "check_in_time": "2024-01-15T08:30:00Z",
  "check_out_time": "2024-01-15T09:15:00Z",
  "check_in_stop_id": "3280",
  "check_out_stop_id": "3282",
  "routes_used": ["327H", "295A"],
  "total_distance": 12.5,
  "total_fare": 25.00
}
```

---

## Fare Calculation Details

### Fare Rules (Current - BMTC)

```go
BaseFare:            ₹5.00    // Minimum fare
FarePerKm:           ₹2.00    // Per kilometer
TransferFee:         ₹2.00    // Per transfer
ACBusMultiplier:     1.5x     // AC buses cost 50% more
ExpressBusMultiplier: 1.2x    // Express buses cost 20% more
```

### Calculation Logic

```go
// For each leg of journey:
legFare = distance × FarePerKm × RouteTypeMultiplier

// Total fare:
totalFare = sum(all leg fares) + (TransferFee × number of transfers)

// Ensure minimum:
if totalFare < BaseFare {
    totalFare = BaseFare
}
```

### Distance Calculation Methods

1. **Primary**: PostGIS `ST_Distance()` - Accurate geographic distance
2. **Fallback**: Haversine formula - If PostGIS fails
3. **Estimation**: 0.5 km per stop - If both fail

---

## Summary

### Fare Calculation
- ✅ **Automatic**: Calculated on check-out
- ✅ **Distance-based**: Uses PostGIS for accurate distances
- ✅ **Route-aware**: Considers route types (AC/Express)
- ✅ **Transfer-aware**: Adds fees for transfers
- ⚠️ **Inferred**: Based on optimal route, not actual route taken

### Journey Tracking
- ✅ **Check-in/Check-out locations**: GPS coordinates stored
- ✅ **Nearest stops**: Automatically detected
- ✅ **Routes used**: Inferred from route planner
- ⚠️ **Not real-time**: We don't track exact path during journey
- ⚠️ **Assumed route**: Uses optimal route, may differ from actual

### Improvements Needed
1. Track actual routes user boards (via QR scanning on vehicles)
2. Real-time GPS tracking during journey
3. User confirmation of routes taken
4. Integration with vehicle tracking systems

