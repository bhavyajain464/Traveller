# Journey Planning API - Quick Reference

## Endpoint

**POST** `/api/v1/journeys/plan`

## Description

Finds the best journey options from a source location (lat/long) to a destination location (lat/long) using Delhi Metro and Bus transit.

## Parameters

### Required Query Parameters

- `from_lat` - Source latitude (decimal)
- `from_lon` - Source longitude (decimal)
- `to_lat` - Destination latitude (decimal)
- `to_lon` - Destination longitude (decimal)

### Optional Query Parameters

- `departure_time` - Desired departure time (RFC3339 format or HH:MM:SS)
- `arrival_time` - Desired arrival time (RFC3339 format or HH:MM:SS)

## Example cURL Commands

### Basic Journey Planning

```bash
curl -X POST 'http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910'
```

### With Departure Time (RFC3339 format)

```bash
curl -X POST 'http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910&departure_time=2024-12-04T08:00:00Z'
```

### With Departure Time (Simple format)

```bash
curl -X POST 'http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910&departure_time=08:00:00'
```

### Pretty Print JSON Response

```bash
curl -X POST 'http://localhost:8080/api/v1/journeys/plan?from_lat=28.6139&from_lon=77.2090&to_lat=28.5355&to_lon=77.3910' | python3 -m json.tool
```

## Response Format

### Success Response (200 OK)

```json
{
  "options": [
    {
      "duration": 45,
      "transfers": 1,
      "walking_time": 10,
      "departure_time": "2024-12-04T08:00:00Z",
      "arrival_time": "2024-12-04T08:45:00Z",
      "fare": 25.5,
      "legs": [
        {
          "mode": "walking",
          "from_stop_id": "",
          "from_stop_name": "Origin",
          "to_stop_id": "49",
          "to_stop_name": "New Delhi Metro Station",
          "departure_time": "2024-12-04T08:00:00Z",
          "arrival_time": "2024-12-04T08:10:00Z",
          "duration": 10,
          "stop_count": 0
        },
        {
          "mode": "metro",
          "route_id": "4",
          "route_name": "YELLOW_Huda City Centre to Qutab Minar",
          "from_stop_id": "49",
          "from_stop_name": "New Delhi",
          "to_stop_id": "50",
          "to_stop_name": "Rajiv Chowk",
          "departure_time": "2024-12-04T08:10:00Z",
          "arrival_time": "2024-12-04T08:35:00Z",
          "duration": 25,
          "stop_count": 5
        }
      ]
    }
  ],
  "count": 1
}
```

### Error Response (404 Not Found)

```json
{
  "error": "no journey options found"
}
```

### Error Response (400 Bad Request)

```json
{
  "error": "from_lat, from_lon, to_lat, and to_lon are required"
}
```

## Response Fields

### JourneyOption

- `duration` - Total journey time in minutes
- `transfers` - Number of transfers required
- `walking_time` - Total walking time in minutes
- `departure_time` - Journey start time (ISO 8601)
- `arrival_time` - Journey end time (ISO 8601)
- `fare` - Total fare in INR (optional)
- `legs` - Array of journey legs

### JourneyLeg

- `mode` - Transport mode: "walking", "metro", "bus"
- `route_id` - Route identifier (for transit legs)
- `route_name` - Route name (for transit legs)
- `from_stop_id` - Starting stop ID
- `from_stop_name` - Starting stop name
- `to_stop_id` - Ending stop ID
- `to_stop_name` - Ending stop name
- `departure_time` - Leg start time
- `arrival_time` - Leg end time
- `duration` - Leg duration in minutes
- `stop_count` - Number of stops traveled

## Example: Delhi Metro Stations

### Popular Delhi Locations

**Connaught Place (CP)**
- Lat: 28.6304, Lon: 77.2177

**New Delhi Railway Station**
- Lat: 28.642944, Lon: 77.222351

**Indira Gandhi International Airport**
- Lat: 28.5562, Lon: 77.1000

**Qutub Minar**
- Lat: 28.5245, Lon: 77.1855

### Example: CP to Airport

```bash
curl -X POST 'http://localhost:8080/api/v1/journeys/plan?from_lat=28.6304&from_lon=77.2177&to_lat=28.5562&to_lon=77.1000&departure_time=08:00:00' | python3 -m json.tool
```

## Notes

- The API searches for nearby stops within a reasonable walking distance
- Journey options are sorted by duration (fastest first)
- Supports multi-modal journeys (metro + bus + walking)
- Fare calculation includes Delhi Metro and Bus fare rules
- If no journey options are found, try different coordinates or check if stops exist nearby

## Troubleshooting

If you get "no journey options found":

1. **Check if stops exist nearby:**
   ```bash
   curl 'http://localhost:8080/api/v1/stops/nearby?lat=YOUR_LAT&lon=YOUR_LON&radius=1000&limit=5'
   ```

2. **Try with actual metro station coordinates** (see examples above)

3. **Check server logs** for route planner errors

4. **Verify data is loaded:**
   ```bash
   curl 'http://localhost:8080/health'
   ```

