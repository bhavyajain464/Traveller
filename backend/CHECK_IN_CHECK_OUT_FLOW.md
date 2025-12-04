# Check-In/Check-Out Flow Documentation

## Overview

This system implements an EasyRide-style check-in/check-out system where users:
1. Check-in to start a journey → Get QR code
2. Use QR code across multiple transport modes
3. Check-out when journey ends
4. Receive daily bill next day
5. Pay once per day for all journeys

## User Flow

### 1. Check-In (Start Journey)

**Endpoint**: `POST /api/v1/sessions/checkin`

**Request**:
```json
{
  "user_id": "user123",
  "latitude": 12.9716,
  "longitude": 77.5946,
  "stop_id": "3280"  // Optional, will be inferred from location
}
```

**Response**:
```json
{
  "session": {
    "id": "session-uuid",
    "user_id": "user123",
    "qr_code": "TRANSIT-1234567890-abc12345-user123",
    "check_in_time": "2024-01-15T08:30:00Z",
    "check_in_stop_id": "3280",
    "status": "active"
  },
  "qr_code": "TRANSIT-1234567890-abc12345-user123",
  "qr_ticket": {
    "code": "TRANSIT-1234567890-abc12345-user123",
    "user_id": "user123",
    "session_id": "session-uuid",
    "check_in_time": "2024-01-15T08:30:00Z",
    "expires_at": "2024-01-16T08:30:00Z",
    "is_valid": true
  },
  "message": "Check-in successful. Show QR code to conductor."
}
```

**What happens**:
- System finds nearest stop to user's location
- Creates a journey session with status "active"
- Generates unique QR code (valid for 24 hours)
- Returns QR code to user

### 2. QR Code Validation (By Conductor)

**Endpoint**: `POST /api/v1/sessions/validate-qr`

**Request**:
```json
{
  "qr_code": "TRANSIT-1234567890-abc12345-user123",
  "route_id": "327H"
}
```

**Response**:
```json
{
  "valid": true,
  "qr_ticket": {
    "code": "TRANSIT-1234567890-abc12345-user123",
    "user_id": "user123",
    "check_in_time": "2024-01-15T08:30:00Z",
    "is_valid": true
  },
  "message": "QR code is valid"
}
```

**What happens**:
- Conductor scans QR code on their device
- System validates QR code is active and not expired
- Returns validation status
- User can board the vehicle

### 3. Check-Out (End Journey)

**Endpoint**: `POST /api/v1/sessions/checkout`

**Request**:
```json
{
  "session_id": "session-uuid",  // or use qr_code
  "qr_code": "TRANSIT-1234567890-abc12345-user123",
  "latitude": 12.9352,
  "longitude": 77.6245,
  "stop_id": "3282"  // Optional
}
```

**Response**:
```json
{
  "session": {
    "id": "session-uuid",
    "user_id": "user123",
    "check_in_time": "2024-01-15T08:30:00Z",
    "check_out_time": "2024-01-15T09:15:00Z",
    "check_in_stop_id": "3280",
    "check_out_stop_id": "3282",
    "status": "completed",
    "routes_used": ["327H", "295A"],
    "total_distance": 12.5,
    "total_fare": 25.00
  },
  "message": "Check-out successful. Journey completed.",
  "fare": 25.00
}
```

**What happens**:
- System finds nearest stop to check-out location
- Calculates journey route and distance
- Calculates fare based on distance and routes used
- Updates session status to "completed"
- Updates daily bill for the user

### 4. Daily Bill Generation

**Automatic Process** (runs daily at midnight):
- System aggregates all completed journeys for previous day
- Creates/updates daily bill for each user
- Calculates total fare, distance, journey count

**Manual Trigger**: `POST /api/v1/bills/generate?date=2024-01-14`

### 5. View Daily Bill

**Endpoint**: `GET /api/v1/bills/users/:user_id?date=2024-01-15`

**Response**:
```json
{
  "id": "bill-uuid",
  "user_id": "user123",
  "bill_date": "2024-01-15",
  "total_journeys": 3,
  "total_distance": 45.2,
  "total_fare": 90.50,
  "status": "pending",
  "journeys": [
    {
      "id": "session-1",
      "check_in_time": "2024-01-15T08:30:00Z",
      "check_out_time": "2024-01-15T09:15:00Z",
      "total_fare": 25.00
    },
    {
      "id": "session-2",
      "check_in_time": "2024-01-15T10:00:00Z",
      "check_out_time": "2024-01-15T10:45:00Z",
      "total_fare": 30.00
    },
    {
      "id": "session-3",
      "check_in_time": "2024-01-15T18:00:00Z",
      "check_out_time": "2024-01-15T18:30:00Z",
      "total_fare": 35.50
    }
  ]
}
```

### 6. Pay Daily Bill

**Endpoint**: `POST /api/v1/bills/:bill_id/pay`

**Request**:
```json
{
  "payment_id": "payment-xyz123",
  "payment_method": "upi"  // or "card", "wallet"
}
```

**Response**:
```json
{
  "message": "Bill marked as paid",
  "bill_id": "bill-uuid",
  "payment_id": "payment-xyz123",
  "payment_method": "upi"
}
```

## Key Features

### Multi-Modal Support
- QR code works across all transport modes (bus, metro, train)
- System tracks which routes were used
- Fare calculation considers all modes

### Daily Aggregation
- All journeys from a day are aggregated into one bill
- User pays once per day
- Bill includes detailed journey breakdown

### QR Code Security
- QR codes expire after 24 hours
- Each QR code is unique and tied to a session
- Validation ensures QR code is active

### Automatic Fare Calculation
- System calculates fare based on:
  - Distance traveled
  - Routes used
  - Route types (AC, Express, Ordinary)
  - Transfers

## Database Schema

### journey_sessions
- Tracks each check-in/check-out session
- Stores QR code, locations, routes used, fare

### daily_bills
- Aggregates all journeys for a user per day
- Tracks payment status
- Links to individual journey sessions

## Implementation Notes

1. **QR Code Format**: `TRANSIT-{timestamp}-{sessionID}-{userID}`
2. **Session Validity**: 24 hours from check-in
3. **Bill Generation**: Runs automatically daily, can be triggered manually
4. **Fare Calculation**: Uses distance-based pricing with route multipliers
5. **Location Matching**: Uses PostGIS for accurate stop detection


