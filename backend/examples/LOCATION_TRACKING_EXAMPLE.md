# Multi-Journey Location Tracking Example

This example demonstrates how the Traveller system tracks a user's journey across multiple transport modes (Metro + Bus) using continuous GPS location matching.

## Overview

The system tracks:
1. **User's GPS location** - Continuous updates from user's mobile device
2. **Vehicle's GPS location** - Real-time location from transport vehicle (bus/metro)
3. **Location Matching** - Validates that user is actually on the vehicle
4. **Route Boarding** - Records which route user boarded
5. **Fare Calculation** - Charges based on actual routes used and distance traveled

## Flow Diagram

```
User Journey Flow:
┌─────────────────┐
│ 1. Check-In     │ → User starts journey, gets QR code
│    (Source)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 2. Board Metro  │ → QR validated, boarding recorded
│    (GPS Match)  │ → User location ≈ Vehicle location
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 3. Travel Metro │ → Continuous GPS tracking
│    (On Vehicle)  │ → User location matches vehicle location
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 4. Alight Metro │ → Fare calculated for Metro segment
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 5. Board Bus    │ → QR validated, boarding recorded
│    (GPS Match)  │ → User location ≈ Vehicle location
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 6. Travel Bus   │ → Continuous GPS tracking
│    (On Vehicle)  │ → User location matches vehicle location
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 7. Alight Bus   │ → Fare calculated for Bus segment
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 8. Check-Out   │ → Total fare calculated
│    (Destination)│ → Daily bill updated
└─────────────────┘
```

## Location Matching Logic

### How It Works

1. **User's Location**: Continuously tracked via mobile GPS
2. **Vehicle Location**: Received from vehicle's GPS tracker (via GTFS-RT or API)
3. **Matching Algorithm**:
   - Calculate distance between user location and vehicle location
   - If distance < threshold (e.g., 100 meters), user is considered "on vehicle"
   - This prevents fare fraud and ensures accurate tracking

### Example Matching

```javascript
// User boarding Metro
User Location:    { lat: 28.6304, lon: 77.2177 }  // Rajiv Chowk Station
Vehicle Location: { lat: 28.6304, lon: 77.2177 }  // Metro train at platform
Distance:         0 meters
Match: ✅ User is on Metro

// User traveling on Metro
User Location:    { lat: 28.6280, lon: 77.2180 }  // Moving with train
Vehicle Location: { lat: 28.6280, lon: 77.2180 }  // Metro train location
Distance:         0 meters
Match: ✅ User is still on Metro

// User boarding Bus
User Location:    { lat: 28.6250, lon: 77.2200 }  // Bus stop
Vehicle Location: { lat: 28.6250, lon: 77.2200 }  // Bus at stop
Distance:         0 meters
Match: ✅ User is on Bus
```

## API Endpoints Used

### 1. Create User
```bash
POST /api/v1/users
{
  "phone_number": "+919876543210",
  "name": "Rajesh Kumar",
  "email": "rajesh.kumar@example.com"
}
```

### 2. Check-In
```bash
POST /api/v1/sessions/checkin
{
  "user_id": "user-uuid",
  "latitude": 28.6304,
  "longitude": 77.2177
}
```

### 3. Validate QR & Record Boarding
```bash
POST /api/v1/sessions/validate-qr?latitude=28.6304&longitude=77.2177
{
  "qr_code": "TRANSIT-xxx",
  "route_id": "YELLOW_LINE"
}
```

### 4. Record Boarding (Alternative)
```bash
POST /api/v1/boardings/board
{
  "session_id": "session-uuid",
  "qr_code": "TRANSIT-xxx",
  "route_id": "YELLOW_LINE",
  "latitude": 28.6304,
  "longitude": 77.2177
}
```

### 5. Record Alighting
```bash
POST /api/v1/boardings/alight
{
  "boarding_id": "boarding-uuid",
  "latitude": 28.6250,
  "longitude": 77.2200
}
```

### 6. Check-Out
```bash
POST /api/v1/sessions/checkout
{
  "session_id": "session-uuid",
  "qr_code": "TRANSIT-xxx",
  "latitude": 28.6129,
  "longitude": 77.2295
}
```

## Running the Example

```bash
cd backend/examples
./multi-journey-example.sh
```

## Expected Output

The script will:
1. ✅ Create a user account
2. ✅ Plan journey from source to destination
3. ✅ Check-in at source location
4. ✅ Board Metro (with location matching)
5. ✅ Travel on Metro (continuous tracking)
6. ✅ Alight from Metro (fare calculated)
7. ✅ Board Bus (with location matching)
8. ✅ Travel on Bus (continuous tracking)
9. ✅ Alight from Bus (fare calculated)
10. ✅ Check-out at destination (total fare calculated)
11. ✅ View all route boardings
12. ✅ View daily bill

## Fare Calculation

Fares are calculated based on:
- **Distance traveled** on each route segment
- **Route type** (Metro vs Bus) - different fare rules
- **Agency** (DMRC vs DIMTS/DTC) - different pricing
- **Transfer fees** (if applicable)

Example:
- Metro segment: 2.5 km → ₹15 (DMRC pricing)
- Bus segment: 1.2 km → ₹12 (DIMTS pricing)
- **Total fare**: ₹27

## Location Matching Thresholds

The system uses different thresholds for different scenarios:

- **Boarding validation**: 100 meters (user must be near vehicle)
- **Continuous tracking**: 200 meters (allows for GPS drift)
- **Alighting validation**: 100 meters (user must be near stop)

These thresholds can be configured per transport mode.

## Security Features

1. **QR Code Expiry**: QR codes expire after a set time
2. **Location Validation**: User location must match vehicle location
3. **Active Boarding Check**: Prevents multiple simultaneous boardings
4. **Session Validation**: Ensures check-in before boarding
5. **Distance Validation**: Prevents impossible journeys

## Future Enhancements

- Real-time vehicle location via GTFS-RT
- Automatic boarding detection (no QR scan needed)
- Predictive fare calculation
- Route optimization based on real-time data
- Fraud detection using ML

