# Traveller Examples

This directory contains example scripts demonstrating various features of the Traveller backend.

## Multi-Journey Location Tracking Example

### Overview

Demonstrates a complete multi-modal journey (Metro + Bus) with:
- User creation
- Check-in/Check-out flow
- QR code validation
- Route boarding tracking
- Continuous GPS location matching
- Fare calculation based on actual routes used

### Files

- `multi-journey-example.sh` - Complete example script
- `LOCATION_TRACKING_EXAMPLE.md` - Detailed documentation

### Prerequisites

1. **Server must be running**:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

2. **Database must be set up** with Delhi GTFS data loaded

3. **Required tools**:
   - `curl`
   - `python3` (for JSON formatting, optional)

### Running the Example

```bash
cd backend/examples
./multi-journey-example.sh
```

### What It Demonstrates

1. **User Creation**: Creates a new user account
2. **Journey Planning**: Plans route from source to destination
3. **Check-In**: User checks in at source location
4. **Metro Boarding**: User boards Metro, QR validated, location matched
5. **Metro Travel**: Continuous GPS tracking while on Metro
6. **Metro Alighting**: User alights, fare calculated for Metro segment
7. **Bus Boarding**: User boards Bus, QR validated, location matched
8. **Bus Travel**: Continuous GPS tracking while on Bus
9. **Bus Alighting**: User alights, fare calculated for Bus segment
10. **Check-Out**: User checks out, total fare calculated
11. **View Boardings**: Shows all route boardings for the session
12. **View Bill**: Shows daily bill with all journeys

### Location Matching

The system matches:
- **User's GPS location** (from mobile device)
- **Vehicle's GPS location** (from transport vehicle)

If locations match (within threshold), the system confirms user is on the vehicle.

### Expected Output

```
🚀 Traveller Multi-Journey Example
====================================

Step 1: Creating user...
✅ User created: <user-id>

Step 2: Planning journey...
[Journey options displayed]

Step 3: User checks in at source location...
✅ Check-in successful
Session ID: <session-id>
QR Code: <qr-code>

Step 4: User boards Metro (Yellow Line)...
✅ Boarding recorded: <boarding-id>

Step 5: User travels on Metro...
✅ User is on the Metro

Step 6: User alights from Metro...
✅ Alighted from Metro
Metro fare: ₹15.00
Metro distance: 2.5km

Step 7: User boards Bus...
✅ Boarding recorded: <boarding-id>

Step 8: User travels on Bus...
✅ User is on the Bus

Step 9: User alights from Bus...
✅ Alighted from Bus
Bus fare: ₹12.00
Bus distance: 1.2km

Step 10: User checks out at destination...
✅ Check-out successful
Total fare: ₹27.00
Total distance: 3.7km
Routes used: ["YELLOW_LINE", "BUS_123"]

Step 11: Viewing all route boardings...
[All boardings displayed]

Step 12: Viewing daily bill...
[Daily bill displayed]

✅ Multi-journey example completed!
```

### Manual Testing

You can also test individual endpoints manually:

#### 1. Create User
```bash
curl -X POST "http://localhost:8080/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543210",
    "name": "Rajesh Kumar",
    "email": "rajesh@example.com"
  }'
```

#### 2. Check-In
```bash
curl -X POST "http://localhost:8080/api/v1/sessions/checkin" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "<user-id>",
    "latitude": 28.6304,
    "longitude": 77.2177
  }'
```

#### 3. Board Route (with location matching)
```bash
curl -X POST "http://localhost:8080/api/v1/sessions/validate-qr?latitude=28.6304&longitude=77.2177" \
  -H "Content-Type: application/json" \
  -d '{
    "qr_code": "<qr-code>",
    "route_id": "<route-id>"
  }'
```

#### 4. Alight Route
```bash
curl -X POST "http://localhost:8080/api/v1/boardings/alight" \
  -H "Content-Type: application/json" \
  -d '{
    "boarding_id": "<boarding-id>",
    "latitude": 28.6250,
    "longitude": 77.2200
  }'
```

#### 5. Check-Out
```bash
curl -X POST "http://localhost:8080/api/v1/sessions/checkout" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "<session-id>",
    "qr_code": "<qr-code>",
    "latitude": 28.6129,
    "longitude": 77.2295
  }'
```

### Troubleshooting

1. **404 errors**: Make sure server is running and has been restarted after adding user endpoints
2. **No routes found**: Ensure Delhi GTFS data is loaded
3. **Database errors**: Check database connection and migrations
4. **JSON parsing errors**: Install `python3` for better output formatting

### Next Steps

- Integrate with real-time vehicle location APIs
- Add automatic boarding detection
- Implement fraud detection
- Add push notifications for fare updates

