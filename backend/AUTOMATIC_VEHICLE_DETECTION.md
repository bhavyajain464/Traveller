# Automatic Vehicle Detection

The Traveller backend now automatically detects which transport vehicle a user is on based on their live GPS location, without requiring them to specify the route or vehicle type.

## How It Works

### 1. Mocked Vehicle Locations

The system maintains a mock database of vehicle locations:
- **Metro trains**: Positioned along metro lines
- **Buses**: Positioned at bus stops and along routes
- **Real-time updates**: Vehicle positions update every 10 seconds (simulated movement)

### 2. Location Matching Algorithm

When a user's location is received:
1. **Search nearby vehicles**: Find all vehicles within 100 meters of user's location
2. **Calculate confidence**: Based on distance (closer = higher confidence)
3. **Select best match**: Vehicle with highest confidence (>0.5 threshold)
4. **Detect transport mode**: Automatically identify Metro (route_type=1) or Bus (route_type=3)

### 3. Automatic Boarding

When user location matches a vehicle:
- **Route ID detected**: Automatically identified from vehicle location
- **Stop detected**: Nearest stop found using geospatial query
- **Boarding recorded**: System automatically records boarding without user input

## API Endpoint

### Auto-Detect and Board

```bash
POST /api/v1/boardings/auto-board
```

**Request:**
```json
{
  "session_id": "session-uuid",
  "qr_code": "TRANSIT-xxx",  // Alternative to session_id
  "latitude": 28.6304,
  "longitude": 77.2177
}
```

**Response:**
```json
{
  "boarding": {
    "id": "boarding-uuid",
    "session_id": "session-uuid",
    "route_id": "YELLOW_LINE",
    "boarding_stop_id": "stop-123",
    "boarding_time": "2024-01-15T09:00:00Z",
    "boarding_lat": 28.6304,
    "boarding_lon": 77.2177,
    "distance": 0,
    "fare": 0
  },
  "vehicle_match": {
    "vehicle_location": {
      "vehicle_id": "vehicle-YELLOW_LINE-0",
      "route_id": "YELLOW_LINE",
      "latitude": 28.6304,
      "longitude": 77.2177,
      "timestamp": "2024-01-15T09:00:00Z"
    },
    "route_id": "YELLOW_LINE",
    "route_name": "Yellow Line",
    "route_type": 1,
    "agency_id": "DMRC",
    "distance": 5.2,
    "confidence": 0.95
  },
  "message": "Vehicle detected and boarding recorded automatically",
  "detected_route": "YELLOW_LINE",
  "detected_mode": "Metro",
  "confidence": 0.95,
  "distance_meters": 5.2
}
```

## Mock Vehicle System

### Initialization

On server startup:
1. Loads active routes from database
2. Creates mock vehicles positioned at stops along routes
3. Starts background goroutine to simulate vehicle movement

### Vehicle Movement Simulation

- Updates every 10 seconds
- Small random movements (~100m)
- Random bearing and speed (20-60 km/h)
- Maintains realistic positions along routes

### Adding Mock Vehicles

For testing, you can add vehicles at specific locations:

```go
vehicleLocationService.AddMockVehicle("YELLOW_LINE", 28.6304, 77.2177)
```

## Example Usage

### Before (Manual)
```bash
# User had to specify route_id
POST /api/v1/boardings/board
{
  "session_id": "xxx",
  "route_id": "YELLOW_LINE",  // Required!
  "latitude": 28.6304,
  "longitude": 77.2177
}
```

### After (Automatic)
```bash
# System automatically detects route
POST /api/v1/boardings/auto-board
{
  "session_id": "xxx",
  "latitude": 28.6304,
  "longitude": 77.2177
  // No route_id needed!
}
```

## Benefits

1. **User Experience**: No need to manually select route or vehicle type
2. **Accuracy**: System confirms user is actually on the vehicle
3. **Fraud Prevention**: Location matching prevents false boardings
4. **Seamless**: Works automatically in background

## Future Enhancements

- Real-time vehicle location APIs (GTFS-RT)
- Machine learning for better matching
- Historical movement patterns
- Multi-vehicle scenarios (user between two vehicles)
- Automatic alighting detection

## Configuration

### Detection Thresholds

- **Search radius**: 100 meters (configurable)
- **Confidence threshold**: 0.5 (50% confidence required)
- **Update frequency**: 10 seconds

### Route Types

- `1` = Metro/Rail
- `3` = Bus
- `2` = Rail (intercity)

## Troubleshooting

**No vehicle detected:**
- User may be too far from any vehicle (>100m)
- No vehicles active on nearby routes
- Check mock vehicle initialization

**Low confidence:**
- Multiple vehicles nearby
- User between vehicles
- Increase search radius or adjust threshold

