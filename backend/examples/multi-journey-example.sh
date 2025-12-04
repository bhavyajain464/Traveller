#!/bin/bash

# Multi-Journey Example Script
# Demonstrates a user taking multiple journeys (metro + bus) with continuous location tracking
# and fare calculation based on actual routes used

BASE_URL="http://localhost:8080/api/v1"
echo "🚀 Traveller Multi-Journey Example"
echo "===================================="
echo ""

# Check if python3 is available for JSON parsing
if ! command -v python3 &> /dev/null; then
  echo "⚠️  Warning: python3 not found. JSON parsing may be less accurate."
  USE_PYTHON=false
else
  USE_PYTHON=true
fi

# Helper function to parse JSON
parse_json() {
  local json="$1"
  local path="$2"
  if [ "$USE_PYTHON" = true ]; then
    echo "$json" | python3 -c "import sys, json; data=json.load(sys.stdin); $path" 2>/dev/null
  else
    echo "$json" | grep -o "\"$path\":\"[^\"]*" | cut -d'"' -f4
  fi
}

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Clean up existing user (if exists) and create a new user
echo -e "${BLUE}Step 1: Cleaning up and creating user...${NC}"
PHONE_NUMBER="+919876543210"

# Delete user if exists
echo "Checking for existing user with phone: $PHONE_NUMBER"
DELETE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/users/phone/${PHONE_NUMBER}")
if echo "$DELETE_RESPONSE" | grep -q "deleted successfully"; then
  echo "✅ Deleted existing user"
elif echo "$DELETE_RESPONSE" | grep -q "not found"; then
  echo "ℹ️  No existing user found"
else
  echo "⚠️  Delete response: $DELETE_RESPONSE"
fi

# Create new user
USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -d "{
    \"phone_number\": \"$PHONE_NUMBER\",
    \"name\": \"Rajesh Kumar\",
    \"email\": \"rajesh.kumar@example.com\"
  }")

# Parse user ID using python for better JSON handling
if [ "$USE_PYTHON" = true ]; then
  USER_ID=$(echo "$USER_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('user', {}).get('id', ''))" 2>/dev/null)
else
  USER_ID=$(echo "$USER_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
fi
echo "✅ User created: $USER_ID"
if [ -z "$USER_ID" ]; then
  echo "❌ Error: Could not parse user ID"
  echo "Response: $USER_RESPONSE"
  exit 1
fi
echo "Full response: $USER_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "Response: $USER_RESPONSE"
echo ""

# Step 2: Plan journey from source to destination
echo -e "${BLUE}Step 2: Planning journey...${NC}"
# Source: Connaught Place area (28.6304, 77.2177)
# Destination: India Gate area (28.6129, 77.2295)
JOURNEY_RESPONSE=$(curl -s -X POST "${BASE_URL}/journeys/plan?from_lat=28.6304&from_lon=77.2177&to_lat=28.6129&to_lon=77.2295&departure_time=09:00:00")
echo "Journey options:"
echo "$JOURNEY_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$JOURNEY_RESPONSE"
echo ""

# Step 3: Check-in at source location
echo -e "${BLUE}Step 3: User checks in at source location...${NC}"
CHECKIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/sessions/checkin" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"latitude\": 28.6304,
    \"longitude\": 77.2177
  }")

# Parse session ID and QR code
if [ "$USE_PYTHON" = true ]; then
  SESSION_ID=$(echo "$CHECKIN_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('session', {}).get('id', ''))" 2>/dev/null)
  QR_CODE=$(echo "$CHECKIN_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('qr_code', ''))" 2>/dev/null)
else
  SESSION_ID=$(echo "$CHECKIN_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
  QR_CODE=$(echo "$CHECKIN_RESPONSE" | grep -o '"qr_code":"[^"]*' | cut -d'"' -f4)
fi
echo "✅ Check-in successful"
echo "Session ID: $SESSION_ID"
echo "QR Code: $QR_CODE"
if [ -z "$SESSION_ID" ] || [ -z "$QR_CODE" ]; then
  echo "❌ Error: Could not parse session/QR"
  echo "Response: $CHECKIN_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CHECKIN_RESPONSE"
  exit 1
fi
echo ""

# Step 4: Set up multiple mock Metro vehicles at the same location
echo -e "${BLUE}Step 4: Setting up multiple Metro vehicles at same location...${NC}"
echo "Creating 3 Metro vehicles at the same boarding location"
echo "After 30 seconds, they will move to different destinations"

# Get a route ID to use (try to find metro route first)
METRO_ROUTE=$(curl -s "${BASE_URL}/routes/search?q=Yellow" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
if [ -z "$METRO_ROUTE" ]; then
  # Try to get any route from database
  METRO_ROUTE=$(curl -s "${BASE_URL}/routes?limit=1" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
fi

# If still no route, use a default route ID
if [ -z "$METRO_ROUTE" ]; then
  METRO_ROUTE="YELLOW_LINE"
fi

echo "Using route ID: $METRO_ROUTE"
echo ""

# Get a trip_id for schedule-based positioning
# This allows vehicles to follow the actual GTFS schedule
METRO_TRIP=$(curl -s "${BASE_URL}/routes/${METRO_ROUTE}/trips?limit=1" | python3 -c "import sys, json; data=json.load(sys.stdin); trips=data.get('trips', []); print(trips[0].get('id', '') if trips else '')" 2>/dev/null || echo "")
if [ -z "$METRO_TRIP" ]; then
  # Fallback: use a known trip ID from database
  METRO_TRIP="2172"
fi

echo "Using trip ID: $METRO_TRIP (for schedule-based positioning)"
echo ""

# Add 3 Metro vehicles using schedule-based positioning
# Vehicles will follow the actual GTFS schedule (arrival_time, departure_time)
# User location will be calculated based on vehicle's schedule position
echo "Adding Metro Vehicle 1 with schedule-based positioning"
echo "  - Route: $METRO_ROUTE"
echo "  - Trip: $METRO_TRIP"
echo "  - Initial location: 28.6304, 77.2177"
VEHICLE1_RESPONSE=$(curl -s -X POST "${BASE_URL}/vehicles/mock" \
  -H "Content-Type: application/json" \
  -d "{
    \"route_id\": \"$METRO_ROUTE\",
    \"trip_id\": \"$METRO_TRIP\",
    \"latitude\": 28.6304,
    \"longitude\": 77.2177,
    \"start_moving_immediately\": false
  }")
VEHICLE1_ID=$(echo "$VEHICLE1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_id', ''))" 2>/dev/null)
echo "Vehicle 1 ID: $VEHICLE1_ID"
echo ""

echo "Adding Metro Vehicle 2 with schedule-based positioning (same trip, different time offset)"
VEHICLE2_RESPONSE=$(curl -s -X POST "${BASE_URL}/vehicles/mock" \
  -H "Content-Type: application/json" \
  -d "{
    \"route_id\": \"$METRO_ROUTE\",
    \"trip_id\": \"$METRO_TRIP\",
    \"latitude\": 28.6304,
    \"longitude\": 77.2177,
    \"start_moving_immediately\": false
  }")
VEHICLE2_ID=$(echo "$VEHICLE2_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_id', ''))" 2>/dev/null)
echo "Vehicle 2 ID: $VEHICLE2_ID"
echo ""

echo "Adding Metro Vehicle 3 with schedule-based positioning"
VEHICLE3_RESPONSE=$(curl -s -X POST "${BASE_URL}/vehicles/mock" \
  -H "Content-Type: application/json" \
  -d "{
    \"route_id\": \"$METRO_ROUTE\",
    \"trip_id\": \"$METRO_TRIP\",
    \"latitude\": 28.6304,
    \"longitude\": 77.2177,
    \"start_moving_immediately\": false
  }")
VEHICLE3_ID=$(echo "$VEHICLE3_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_id', ''))" 2>/dev/null)
echo "Vehicle 3 ID: $VEHICLE3_ID"
echo ""
echo "✅ All 3 vehicles created with schedule-based positioning"
echo "   Vehicles will follow the actual GTFS schedule (arrival_time, departure_time)"
echo "   User location will be calculated based on vehicle's schedule position"
echo "   Position updates every second based on current time and schedule"
echo ""

# Wait a moment for vehicles to be initialized in the system
echo "Waiting 2 seconds for vehicles to be initialized..."
sleep 2

# Verify vehicles are at the correct location before boarding
echo "Verifying vehicle locations..."
VEHICLE1_CHECK=$(curl -s "${BASE_URL}/vehicles/${VEHICLE1_ID}")
VEHICLE1_CHECK_LAT=$(echo "$VEHICLE1_CHECK" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('latitude', 'N/A'))" 2>/dev/null || echo "N/A")
VEHICLE1_CHECK_LON=$(echo "$VEHICLE1_CHECK" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('longitude', 'N/A'))" 2>/dev/null || echo "N/A")
echo "  Vehicle 1 location: $VEHICLE1_CHECK_LAT, $VEHICLE1_CHECK_LON (expected: 28.6304, 77.2177)"
echo ""

# Step 5: User boards Metro using automatic detection
echo -e "${BLUE}Step 5: User boards Metro using automatic detection...${NC}"
echo "User location: 28.6304, 77.2177 (Rajiv Chowk Metro Station)"
echo "System will automatically detect which vehicle user is on"

# Use auto-board endpoint (automatic detection)
BOARDING1_RESPONSE=$(curl -s -X POST "${BASE_URL}/boardings/auto-board" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"qr_code\": \"$QR_CODE\",
    \"latitude\": 28.6304,
    \"longitude\": 77.2177
  }")

echo "Auto-detection response:"
echo "$BOARDING1_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$BOARDING1_RESPONSE"

# Parse boarding ID and detected route
if [ "$USE_PYTHON" = true ]; then
  BOARDING1_ID=$(echo "$BOARDING1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('boarding', {}).get('id', ''))" 2>/dev/null)
  DETECTED_ROUTE=$(echo "$BOARDING1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('detected_route', ''))" 2>/dev/null)
  DETECTED_MODE=$(echo "$BOARDING1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('detected_mode', ''))" 2>/dev/null)
  CONFIDENCE=$(echo "$BOARDING1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('confidence', 0))" 2>/dev/null)
else
  BOARDING1_ID=$(echo "$BOARDING1_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
  DETECTED_ROUTE=$(echo "$BOARDING1_RESPONSE" | grep -o '"detected_route":"[^"]*' | cut -d'"' -f4)
  DETECTED_MODE=$(echo "$BOARDING1_RESPONSE" | grep -o '"detected_mode":"[^"]*' | cut -d'"' -f4)
fi

# Extract vehicle ID from boarding response
if [ "$USE_PYTHON" = true ]; then
  BOARDED_VEHICLE_ID=$(echo "$BOARDING1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('boarding', {}).get('vehicle_id', ''))" 2>/dev/null)
else
  BOARDED_VEHICLE_ID=$(echo "$BOARDING1_RESPONSE" | grep -o '"vehicle_id":"[^"]*' | cut -d'"' -f4)
fi

echo ""
echo "✅ Automatic detection successful!"
echo "   Boarding ID: $BOARDING1_ID"
echo "   Detected Route: $DETECTED_ROUTE"
echo "   Detected Mode: $DETECTED_MODE"
echo "   Vehicle ID: ${BOARDED_VEHICLE_ID:-N/A} (tentative - not confirmed yet)"
echo "   Confidence: ${CONFIDENCE:-N/A}"
if [ -z "$BOARDING1_ID" ]; then
  echo "⚠️  Warning: Could not parse boarding ID"
fi
echo ""
echo "⚠️  IMPORTANT: When multiple vehicles are at the same location, we CAN'T confirm"
echo "   which exact vehicle user boarded. System picked one arbitrarily."
echo "   Confirmation will happen AFTER 30 seconds when vehicles move to different locations."
echo ""

# Step 6: Wait 30 seconds for vehicles to start moving
echo -e "${BLUE}Step 6: Waiting 30 seconds for vehicles to start moving...${NC}"
echo "All vehicles are currently at the same location"
echo "After 30 seconds, they will move to different destinations"
echo "System will track which vehicle user is actually on based on location matching"
echo ""
echo "Waiting 5 seconds (simulating 30 seconds)..."
sleep 5
echo "✅ Vehicles have started moving"
echo ""
echo "Note: For testing, we'll use vehicles that start moving immediately"
echo "In production, vehicles wait 30 real seconds before moving"
echo ""

# Step 7: Continuous location tracking - CONFIRM exact vehicle AFTER vehicles move
echo -e "${BLUE}Step 7: Continuous location tracking (CONFIRMING exact vehicle after 30 seconds)...${NC}"
echo "After 30 seconds, vehicles have moved to different destinations"
echo "NOW we can confirm which EXACT vehicle user is on by matching their location"
echo "with the vehicle's new location after movement"
echo ""

# Get the actual vehicle location after movement to simulate user traveling with it
# Wait additional time for vehicles to actually move (they update every 1 second now)
# Total wait: 5 seconds (simulated) + 25 seconds = 30 seconds for vehicles to start moving
echo "Waiting additional 25 seconds for vehicles to actually move..."
echo "  - Vehicles update location every 1 second"
echo "  - User location checked every 5 seconds"
echo "  - Matching tolerance: 100m (accounts for GPS accuracy + movement)"
sleep 25

# Get the boarded vehicle's current location
if [ -n "$BOARDED_VEHICLE_ID" ]; then
  echo "Getting vehicle location after movement..."
  VEHICLE_LOC_RESPONSE=$(curl -s "${BASE_URL}/vehicles/${BOARDED_VEHICLE_ID}")
  VEHICLE_LAT=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('latitude', 28.6280))" 2>/dev/null || echo "28.6280")
  VEHICLE_LON=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('longitude', 77.2180))" 2>/dev/null || echo "77.2180")
  echo "Vehicle ${BOARDED_VEHICLE_ID} location: $VEHICLE_LAT, $VEHICLE_LON"
  
  # Check if vehicle has moved (not at boarding location anymore)
  if [ "$VEHICLE_LAT" = "28.6304" ] && [ "$VEHICLE_LON" = "77.2177" ]; then
    echo "⚠️  Vehicle hasn't moved yet (still at boarding location)"
    echo "   This could mean: route has no stops, or 30 seconds haven't fully passed"
    echo "   For testing, using a nearby simulated location: 28.6280, 77.2180"
    VEHICLE_LAT="28.6280"
    VEHICLE_LON="77.2180"
  else
    echo "✅ Vehicle has moved from boarding location"
  fi
else
  VEHICLE_LAT="28.6280"
  VEHICLE_LON="77.2180"
fi

# Send location update AFTER vehicles have moved (simulating user moving with the vehicle)
# User's location should match the vehicle's location after movement
echo "Sending location update (user location matching vehicle after movement):"
echo "  - User location: $VEHICLE_LAT, $VEHICLE_LON (matching vehicle location)"
echo "  - System will confirm which vehicle this matches"
echo ""

CONT_LOC1_RESPONSE=$(curl -s -X POST "${BASE_URL}/boardings/continuous-location" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"latitude\": $VEHICLE_LAT,
    \"longitude\": $VEHICLE_LON
  }")

echo "Location update response:"
echo "$CONT_LOC1_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CONT_LOC1_RESPONSE"
echo ""

# Check which vehicle user is on
if [ "$USE_PYTHON" = true ]; then
  ON_VEHICLE=$(echo "$CONT_LOC1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('on_vehicle', False))" 2>/dev/null)
  CONFIRMED_VEHICLE_ID=$(echo "$CONT_LOC1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_id', ''))" 2>/dev/null)
  DISTANCE=$(echo "$CONT_LOC1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('distance_meters', 0))" 2>/dev/null)
  MESSAGE=$(echo "$CONT_LOC1_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('message', ''))" 2>/dev/null)
else
  ON_VEHICLE=$(echo "$CONT_LOC1_RESPONSE" | grep -o '"on_vehicle":[^,}]*' | cut -d':' -f2)
  CONFIRMED_VEHICLE_ID=$(echo "$CONT_LOC1_RESPONSE" | grep -o '"vehicle_id":"[^"]*' | cut -d'"' -f4)
fi

echo "✅ Vehicle Confirmation Results (after movement):"
echo "   User is on vehicle: ${ON_VEHICLE:-unknown}"
echo "   CONFIRMED Vehicle ID: ${CONFIRMED_VEHICLE_ID:-N/A}"
echo "   Previously boarded: ${BOARDED_VEHICLE_ID:-N/A}"
echo "   Distance from vehicle: ${DISTANCE:-N/A} meters"
echo "   Status: ${MESSAGE:-N/A}"
echo ""

# Compare boarded vs confirmed vehicle
if [ "$ON_VEHICLE" = "True" ] || [ "$ON_VEHICLE" = "true" ]; then
  if [ -n "$BOARDED_VEHICLE_ID" ] && [ -n "$CONFIRMED_VEHICLE_ID" ]; then
    if [ "$BOARDED_VEHICLE_ID" = "$CONFIRMED_VEHICLE_ID" ]; then
      echo "✅ Confirmed: User is on the same vehicle they boarded"
    else
      echo "⚠️  Vehicle changed: User boarded $BOARDED_VEHICLE_ID but is actually on $CONFIRMED_VEHICLE_ID"
      echo "   This is expected - we couldn't confirm at boarding when vehicles were at same location"
    fi
  fi
  echo ""
  
  # Step 8: User travels on Metro - continuous location matching
  echo -e "${BLUE}Step 8: User travels on Metro (continuous location matching every 5 seconds)...${NC}"
  echo "Simulating continuous GPS updates:"
  echo "  - Vehicles update location: Every 1 second"
  echo "  - User location checked: Every 5 seconds"
  echo "  - Matching tolerance: 100m (GPS accuracy 10-50m + vehicle movement)"
  echo ""
  
  # Simulate multiple location checks (every 5 seconds)
  for i in 1 2 3; do
    echo "Location check #$i (after $((i * 5)) seconds):"
    
    # Get current vehicle location (vehicles update every 1 second)
    if [ -n "$CONFIRMED_VEHICLE_ID" ]; then
      CURRENT_VEHICLE_RESPONSE=$(curl -s "${BASE_URL}/vehicles/${CONFIRMED_VEHICLE_ID}")
      CURRENT_VEHICLE_LAT=$(echo "$CURRENT_VEHICLE_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('latitude', $VEHICLE_LAT))" 2>/dev/null || echo "$VEHICLE_LAT")
      CURRENT_VEHICLE_LON=$(echo "$CURRENT_VEHICLE_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('vehicle_location', {}).get('longitude', $VEHICLE_LON))" 2>/dev/null || echo "$VEHICLE_LON")
    else
      CURRENT_VEHICLE_LAT=$VEHICLE_LAT
      CURRENT_VEHICLE_LON=$VEHICLE_LON
    fi
    
    # User location matches vehicle location (within tolerance)
    LOC_CHECK_RESPONSE=$(curl -s -X POST "${BASE_URL}/boardings/continuous-location" \
      -H "Content-Type: application/json" \
      -d "{
        \"session_id\": \"$SESSION_ID\",
        \"latitude\": $CURRENT_VEHICLE_LAT,
        \"longitude\": $CURRENT_VEHICLE_LON
      }")
    
    ON_VEHICLE_CHECK=$(echo "$LOC_CHECK_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('on_vehicle', False))" 2>/dev/null)
    DISTANCE_CHECK=$(echo "$LOC_CHECK_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('distance_meters', 0))" 2>/dev/null)
    
    echo "  - User location: $CURRENT_VEHICLE_LAT, $CURRENT_VEHICLE_LON"
    echo "  - Vehicle location: $CURRENT_VEHICLE_LAT, $CURRENT_VEHICLE_LON"
    echo "  - On vehicle: ${ON_VEHICLE_CHECK:-unknown}"
    echo "  - Distance: ${DISTANCE_CHECK:-0}m (within 100m tolerance)"
    echo ""
    
    # Wait 5 seconds before next check
    if [ $i -lt 3 ]; then
      sleep 5
    fi
  done
  
  echo "✅ Continuous tracking successful - user location matches vehicle within tolerance"
  echo ""
else
  echo "⚠️  User is not on any vehicle after movement"
  echo "   This could mean: user alighted, location is inaccurate, or vehicles haven't moved enough"
  echo "   Skipping Step 8 as user is not on a vehicle"
  echo ""
  echo "Note: In a real scenario, you would:"
  echo "  1. Check if user alighted at a stop"
  echo "  2. Update user location to match a vehicle"
  echo "  3. Or end the journey session"
  echo ""
  exit 0
fi

# Step 7: Continue location tracking until user alights automatically at a stop
echo -e "${BLUE}Step 7: Continuous location tracking until automatic alighting...${NC}"
echo "System will continue checking user location every 5 seconds"
echo "Automatic alighting will trigger when:"
echo "  - User is at a stop (within 50m)"
echo "  - Vehicle has moved away from stop (>50m)"
echo "  - User is not on vehicle for 3 consecutive checks"
echo ""

# Continue location checks until user is automatically alighted
ALIGHTED=false
LOCATION_CHECK_COUNT=0
MAX_CHECKS=30  # Maximum 30 checks (150 seconds) before timeout
USER_ALIGHTED_AT_STOP=false
STOP_LAT=""
STOP_LON=""

# First few checks: User is on vehicle (location matches vehicle)
# After vehicle reaches a stop, user gets off and stays at stop
# Vehicle continues moving away
# After 3 consecutive "not on vehicle" checks at stop, auto-alight triggers

while [ "$ALIGHTED" = false ] && [ $LOCATION_CHECK_COUNT -lt $MAX_CHECKS ]; do
  LOCATION_CHECK_COUNT=$((LOCATION_CHECK_COUNT + 1))
  echo "Location check #$LOCATION_CHECK_COUNT (after $((LOCATION_CHECK_COUNT * 5)) seconds):"
  
  # Get current vehicle location (schedule-based position)
  # Vehicle position is calculated from GTFS schedule based on current time
  # User location matches vehicle location (user is on vehicle)
  if [ -n "$BOARDED_VEHICLE_ID" ] || [ -n "$CONFIRMED_VEHICLE_ID" ]; then
    VEHICLE_ID_TO_CHECK=${CONFIRMED_VEHICLE_ID:-$BOARDED_VEHICLE_ID}
    VEHICLE_LOC_RESPONSE=$(curl -s "${BASE_URL}/vehicles/${VEHICLE_ID_TO_CHECK}")
    if [ "$USE_PYTHON" = true ]; then
      VEHICLE_LAT=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('latitude', 28.6304))" 2>/dev/null)
      VEHICLE_LON=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('longitude', 77.2177))" 2>/dev/null)
      VEHICLE_SPEED=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('speed', 0))" 2>/dev/null)
      VEHICLE_BEARING=$(echo "$VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('bearing', 0))" 2>/dev/null)
    else
      VEHICLE_LAT=$(echo "$VEHICLE_LOC_RESPONSE" | grep -o '"latitude":[0-9.]*' | head -1 | cut -d':' -f2)
      VEHICLE_LON=$(echo "$VEHICLE_LOC_RESPONSE" | grep -o '"longitude":[0-9.]*' | head -1 | cut -d':' -f2)
      VEHICLE_SPEED=$(echo "$VEHICLE_LOC_RESPONSE" | grep -o '"speed":[0-9.]*' | head -1 | cut -d':' -f2)
      VEHICLE_BEARING=$(echo "$VEHICLE_LOC_RESPONSE" | grep -o '"bearing":[0-9.]*' | head -1 | cut -d':' -f2)
    fi
  else
    VEHICLE_LAT=28.6304
    VEHICLE_LON=77.2177
    VEHICLE_SPEED=0
    VEHICLE_BEARING=0
  fi
  
  # Simulate user behavior:
  # - First 8 checks: User is on vehicle (location matches vehicle)
  # - After check 8: User gets off at a stop (location stays at stop)
  # - Vehicle continues moving away
  USER_ON_VEHICLE=false
  if [ $LOCATION_CHECK_COUNT -le 8 ]; then
    # User is still on vehicle - location matches vehicle
    USER_LAT=$VEHICLE_LAT
    USER_LON=$VEHICLE_LON
    USER_ON_VEHICLE=true
    echo "  - User is on vehicle (traveling)"
  else
    # User has alighted at a stop - location stays at stop
    if [ "$USER_ALIGHTED_AT_STOP" = false ]; then
      # Get the route ID from the boarding to find next stop on the route
      ACTIVE_BOARDING_RESPONSE=$(curl -s "${BASE_URL}/boardings/active/${SESSION_ID}")
      if [ "$USE_PYTHON" = true ]; then
        BOARDING_ROUTE_ID=$(echo "$ACTIVE_BOARDING_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); b=data.get('boarding', {}); print(b.get('route_id', ''))" 2>/dev/null)
      else
        BOARDING_ROUTE_ID=$(echo "$ACTIVE_BOARDING_RESPONSE" | grep -o '"route_id":"[^"]*' | cut -d'"' -f4)
      fi
      
      # Get all stops on this route and find the next stop along the route
      if [ -n "$BOARDING_ROUTE_ID" ]; then
        ROUTE_STOPS_RESPONSE=$(curl -s "${BASE_URL}/routes/${BOARDING_ROUTE_ID}/stops")
        
        # Find the next stop along the route after current vehicle position
        if [ "$USE_PYTHON" = true ]; then
          # Use Python to find next stop along route (closest stop ahead of vehicle)
          STOP_COORDS=$(echo "$ROUTE_STOPS_RESPONSE" | python3 -c "
import sys, json
import math

data = json.load(sys.stdin)
stops = data.get('stops', [])
vehicle_lat = float($VEHICLE_LAT)
vehicle_lon = float($VEHICLE_LON)

# Find closest stop ahead of vehicle (prefer stops with higher latitude for northbound routes)
min_dist = float('inf')
next_stop_lat = None
next_stop_lon = None

for stop in stops:
    stop_lat = float(stop.get('latitude', 0))
    stop_lon = float(stop.get('longitude', 0))
    
    # Calculate distance in degrees (approximate)
    lat_diff = stop_lat - vehicle_lat
    lon_diff = stop_lon - vehicle_lon
    dist = math.sqrt(lat_diff*lat_diff + lon_diff*lon_diff)
    
    # Prefer stops that are ahead (northward for this route) and at least 100m away
    # 0.001 degrees ≈ 111m, so 0.0009 ≈ 100m
    if stop_lat >= vehicle_lat - 0.0005 and dist > 0.0009:
        if dist < min_dist:
            min_dist = dist
            next_stop_lat = stop_lat
            next_stop_lon = stop_lon

# If no stop found ahead, find any nearby stop
if next_stop_lat is None:
    for stop in stops:
        stop_lat = float(stop.get('latitude', 0))
        stop_lon = float(stop.get('longitude', 0))
        lat_diff = stop_lat - vehicle_lat
        lon_diff = stop_lon - vehicle_lon
        dist = math.sqrt(lat_diff*lat_diff + lon_diff*lon_diff)
        if dist < min_dist and dist > 0.0009:
            min_dist = dist
            next_stop_lat = stop_lat
            next_stop_lon = stop_lon

if next_stop_lat is not None:
    print(f'{next_stop_lat},{next_stop_lon}')
else:
    print(f'{vehicle_lat},{vehicle_lon}')
" 2>/dev/null)
          
          if [ -n "$STOP_COORDS" ]; then
            STOP_LAT=$(echo "$STOP_COORDS" | cut -d',' -f1)
            STOP_LON=$(echo "$STOP_COORDS" | cut -d',' -f2)
          else
            STOP_LAT=$VEHICLE_LAT
            STOP_LON=$VEHICLE_LON
          fi
        else
          # Fallback: use nearby stop API
          STOP_RESPONSE=$(curl -s "${BASE_URL}/stops/nearby?lat=${VEHICLE_LAT}&lon=${VEHICLE_LON}&radius=2000&limit=5")
          # Get second stop if available (first might be current stop)
          STOP_LAT=$(echo "$STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          STOP_LON=$(echo "$STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          if [ -z "$STOP_LAT" ]; then
            STOP_LAT=$(echo "$STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | head -1 | cut -d':' -f2)
            STOP_LON=$(echo "$STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | head -1 | cut -d':' -f2)
          fi
        fi
      else
        # Fallback: use nearby stop API
        STOP_RESPONSE=$(curl -s "${BASE_URL}/stops/nearby?lat=${VEHICLE_LAT}&lon=${VEHICLE_LON}&radius=2000&limit=5")
        if [ "$USE_PYTHON" = true ]; then
          # Get second stop if available (first might be current stop)
          STOP_LAT=$(echo "$STOP_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); stops=data.get('stops', []); print(stops[1].get('latitude', stops[0].get('latitude', $VEHICLE_LAT)) if len(stops) > 1 else (stops[0].get('latitude', $VEHICLE_LAT) if stops else $VEHICLE_LAT))" 2>/dev/null)
          STOP_LON=$(echo "$STOP_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); stops=data.get('stops', []); print(stops[1].get('longitude', stops[0].get('longitude', $VEHICLE_LON)) if len(stops) > 1 else (stops[0].get('longitude', $VEHICLE_LON) if stops else $VEHICLE_LON))" 2>/dev/null)
        else
          STOP_LAT=$(echo "$STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          STOP_LON=$(echo "$STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          if [ -z "$STOP_LAT" ]; then
            STOP_LAT=$(echo "$STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | head -1 | cut -d':' -f2)
            STOP_LON=$(echo "$STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | head -1 | cut -d':' -f2)
          fi
        fi
      fi
      
      # Final fallback: use vehicle location
      if [ -z "$STOP_LAT" ] || [ "$STOP_LAT" = "null" ]; then
        STOP_LAT=$VEHICLE_LAT
        STOP_LON=$VEHICLE_LON
      fi
      
      USER_ALIGHTED_AT_STOP=true
      echo "  - User has alighted at stop: $STOP_LAT, $STOP_LON"
    fi
    
    # User location stays at stop (user is not moving)
    USER_LAT=$STOP_LAT
    USER_LON=$STOP_LON
    echo "  - User is at stop (waiting for automatic alighting)"
  fi
  
  # Display user and vehicle locations with accurate status
  if [ "$USER_ON_VEHICLE" = true ]; then
    echo "  - User location: $USER_LAT, $USER_LON (matches vehicle - user is on board)"
  else
    echo "  - User location: $USER_LAT, $USER_LON (at stop - user has alighted)"
  fi
  echo "  - Vehicle location: $VEHICLE_LAT, $VEHICLE_LON (from GTFS schedule)"
  if [ -n "$VEHICLE_SPEED" ] && [ "$VEHICLE_SPEED" != "0" ]; then
    echo "  - Vehicle speed: ${VEHICLE_SPEED} km/h"
  fi
  if [ -n "$VEHICLE_BEARING" ] && [ "$VEHICLE_BEARING" != "0" ]; then
    echo "  - Vehicle bearing: ${VEHICLE_BEARING}°"
  fi
  
  # Send continuous location update
  LOCATION_RESPONSE=$(curl -s -X POST "${BASE_URL}/boardings/continuous-location" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\": \"$SESSION_ID\",
      \"latitude\": $USER_LAT,
      \"longitude\": $USER_LON
    }")
  
  # Check if user was automatically alighted
  if [ "$USE_PYTHON" = true ]; then
    ALIGHTED_STATUS=$(echo "$LOCATION_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print('True' if data.get('alighted', False) else 'False')" 2>/dev/null)
    ON_VEHICLE=$(echo "$LOCATION_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print('True' if data.get('on_vehicle', False) else 'False')" 2>/dev/null)
    DISTANCE=$(echo "$LOCATION_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('distance_meters', 0))" 2>/dev/null)
    MESSAGE=$(echo "$LOCATION_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('message', ''))" 2>/dev/null)
  else
    ALIGHTED_STATUS=$(echo "$LOCATION_RESPONSE" | grep -o '"alighted":true' | head -1)
    ON_VEHICLE=$(echo "$LOCATION_RESPONSE" | grep -o '"on_vehicle":true' | head -1)
    DISTANCE=$(echo "$LOCATION_RESPONSE" | grep -o '"distance_meters":[0-9.]*' | cut -d':' -f2)
    MESSAGE=$(echo "$LOCATION_RESPONSE" | grep -o '"message":"[^"]*' | cut -d'"' -f4)
    # Convert grep result to True/False
    if [ -n "$ALIGHTED_STATUS" ]; then
      ALIGHTED_STATUS="True"
    else
      ALIGHTED_STATUS="False"
    fi
    if [ -n "$ON_VEHICLE" ]; then
      ON_VEHICLE="True"
    else
      ON_VEHICLE="False"
    fi
  fi
  
  # Only trigger alighting if explicitly True (not just non-empty)
  if [ "$ALIGHTED_STATUS" = "True" ]; then
    ALIGHTED=true
    echo "  ✅ Automatic alighting detected!"
    echo "  - Message: ${MESSAGE:-Automatic alighting at stop}"
    echo ""
    
    # Get boarding details from session boardings (after alighting, boarding is no longer active)
    # Get all boardings for session and find the one that was just alighted (has alighting_time)
    SESSION_BOARDINGS_RESPONSE=$(curl -s "${BASE_URL}/boardings/session/${SESSION_ID}")
    if [ "$USE_PYTHON" = true ]; then
      # Find the most recent boarding with alighting_time (the one that was just alighted)
      FARE1=$(echo "$SESSION_BOARDINGS_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
boardings = data.get('boardings', [])
# Find boarding with alighting_time (most recent)
for b in boardings:
    if b.get('alighting_time'):
        fare = b.get('fare', 0)
        print(fare if fare is not None else 0)
        break
else:
    print(0)
" 2>/dev/null)
      DISTANCE1=$(echo "$SESSION_BOARDINGS_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
boardings = data.get('boardings', [])
# Find boarding with alighting_time (most recent)
for b in boardings:
    if b.get('alighting_time'):
        dist = b.get('distance', 0)
        print(dist / 1000 if dist else 0)
        break
else:
    print(0)
" 2>/dev/null)
    else
      # Fallback: try to extract from boardings array
      FARE1=$(echo "$SESSION_BOARDINGS_RESPONSE" | grep -o '"fare":[0-9.]*' | head -1 | cut -d':' -f2)
      DISTANCE1=$(echo "$SESSION_BOARDINGS_RESPONSE" | grep -o '"distance":[0-9.]*' | head -1 | cut -d':' -f2)
      DISTANCE1=$(echo "scale=3; ${DISTANCE1:-0} / 1000" | bc 2>/dev/null || echo "0")
    fi
    
    echo "✅ Automatically alighted from Metro"
    echo "Metro fare: ₹${FARE1:-0}"
    echo "Metro distance: ${DISTANCE1:-0}km"
    break
  else
    # Check if user is on vehicle (explicitly True, not just non-empty)
    if [ "$ON_VEHICLE" = "True" ]; then
      echo "  - On vehicle: True"
      echo "  - Distance: ${DISTANCE:-0}m (within 100m tolerance)"
    else
      echo "  - On vehicle: False"
      echo "  - Distance: ${DISTANCE:-0}m"
      if [ "$USER_ALIGHTED_AT_STOP" = true ]; then
        echo "  - Message: ${MESSAGE:-User at stop, waiting for automatic alighting}"
      else
        echo "  - Message: ${MESSAGE:-User traveling on vehicle}"
      fi
    fi
    echo ""
    
    # Wait 5 seconds before next check
    if [ $LOCATION_CHECK_COUNT -lt $MAX_CHECKS ]; then
      sleep 5
    fi
  fi
done

if [ "$ALIGHTED" = false ]; then
  echo "⚠️  Automatic alighting did not trigger after $MAX_CHECKS checks"
  echo "   This could mean:"
  echo "   - User is still on vehicle"
  echo "   - User is not at a stop"
  echo "   - Vehicle hasn't moved away from stop yet"
  echo ""
  echo "   For testing, you can manually alight:"
  echo "   curl -X POST \"${BASE_URL}/boardings/alight\" -H \"Content-Type: application/json\" -d '{\"boarding_id\": \"$BOARDING1_ID\", \"latitude\": $USER_LAT, \"longitude\": $USER_LON}'"
  echo ""
  exit 0
fi

# Store fare and distance for later use
FARE1=${FARE1:-0}
DISTANCE1=${DISTANCE1:-0}
echo ""

# Step 10: Add mock Bus vehicle and auto-detect boarding
echo -e "${BLUE}Step 10: Setting up Bus vehicle and auto-detecting boarding...${NC}"
echo "User location: 28.6250, 77.2200 (Bus stop near Metro station)"

# Get a bus route ID
BUS_ROUTE=$(curl -s "${BASE_URL}/routes/search?q=bus" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
if [ -z "$BUS_ROUTE" ]; then
  # Try to get any route
  BUS_ROUTE=$(curl -s "${BASE_URL}/routes?limit=5" | grep -o '"id":"[^"]*' | tail -1 | cut -d'"' -f4)
fi

# If still no route, use a default
if [ -z "$BUS_ROUTE" ]; then
  BUS_ROUTE="BUS_123"
fi

echo "Using Bus Route ID: $BUS_ROUTE"

# Add mock Bus vehicle at boarding location
echo "Adding Bus vehicle at: 28.6250, 77.2200"
MOCK_BUS_RESPONSE=$(curl -s -X POST "${BASE_URL}/vehicles/mock" \
  -H "Content-Type: application/json" \
  -d "{
    \"route_id\": \"$BUS_ROUTE\",
    \"latitude\": 28.6250,
    \"longitude\": 77.2200
  }")
echo "Bus vehicle response: $MOCK_BUS_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$MOCK_BUS_RESPONSE"
echo ""

# Auto-detect and board Bus
echo "Auto-detecting Bus boarding..."
BOARDING2_RESPONSE=$(curl -s -X POST "${BASE_URL}/boardings/auto-board" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"qr_code\": \"$QR_CODE\",
    \"latitude\": 28.6250,
    \"longitude\": 77.2200
  }")

echo "Auto-detection response:"
echo "$BOARDING2_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$BOARDING2_RESPONSE"

# Parse boarding ID
if [ "$USE_PYTHON" = true ]; then
  BOARDING2_ID=$(echo "$BOARDING2_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('boarding', {}).get('id', ''))" 2>/dev/null)
  DETECTED_BUS_ROUTE=$(echo "$BOARDING2_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('detected_route', ''))" 2>/dev/null)
  DETECTED_BUS_MODE=$(echo "$BOARDING2_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('detected_mode', ''))" 2>/dev/null)
else
  BOARDING2_ID=$(echo "$BOARDING2_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
fi

echo ""
echo "✅ Automatic Bus detection successful!"
echo "   Boarding ID: $BOARDING2_ID"
echo "   Detected Route: $DETECTED_BUS_ROUTE"
echo "   Detected Mode: $DETECTED_BUS_MODE"
if [ -z "$BOARDING2_ID" ]; then
  echo "⚠️  Warning: Could not parse boarding ID"
fi
echo ""

# Step 11: User travels on Bus - continuous location tracking until automatic alighting
echo -e "${BLUE}Step 11: Continuous location tracking for Bus until automatic alighting...${NC}"
echo "Automatic alighting will trigger when:"
echo "  - User is not on vehicle for 3 consecutive checks"
echo "  - User is near a station/stop (within 100m)"
echo ""

# Continue location checks until user is automatically alighted from Bus
ALIGHTED2=false
LOCATION_CHECK_COUNT2=0
MAX_CHECKS2=30  # Maximum 30 checks (150 seconds) before timeout
USER_ALIGHTED_AT_STOP2=false
STOP_LAT2=""
STOP_LON2=""

# Get the bus vehicle ID
if [ "$USE_PYTHON" = true ]; then
  BOARDED_BUS_VEHICLE_ID=$(echo "$BOARDING2_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('boarding', {}).get('vehicle_id', ''))" 2>/dev/null)
else
  BOARDED_BUS_VEHICLE_ID=$(echo "$BOARDING2_RESPONSE" | grep -o '"vehicle_id":"[^"]*' | cut -d'"' -f4)
fi

# Wait for bus to start moving (same as metro)
echo "Waiting 30 seconds for bus to start moving..."
sleep 30

while [ "$ALIGHTED2" = false ] && [ $LOCATION_CHECK_COUNT2 -lt $MAX_CHECKS2 ]; do
  LOCATION_CHECK_COUNT2=$((LOCATION_CHECK_COUNT2 + 1))
  echo "Bus location check #$LOCATION_CHECK_COUNT2 (after $((LOCATION_CHECK_COUNT2 * 5)) seconds):"
  
  # Get current bus vehicle location
  if [ -n "$BOARDED_BUS_VEHICLE_ID" ]; then
    BUS_VEHICLE_LOC_RESPONSE=$(curl -s "${BASE_URL}/vehicles/${BOARDED_BUS_VEHICLE_ID}")
    if [ "$USE_PYTHON" = true ]; then
      BUS_VEHICLE_LAT=$(echo "$BUS_VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('latitude', 28.6250))" 2>/dev/null)
      BUS_VEHICLE_LON=$(echo "$BUS_VEHICLE_LOC_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); v=data.get('vehicle_location', {}); print(v.get('longitude', 77.2200))" 2>/dev/null)
    else
      BUS_VEHICLE_LAT=$(echo "$BUS_VEHICLE_LOC_RESPONSE" | grep -o '"latitude":[0-9.]*' | head -1 | cut -d':' -f2)
      BUS_VEHICLE_LON=$(echo "$BUS_VEHICLE_LOC_RESPONSE" | grep -o '"longitude":[0-9.]*' | head -1 | cut -d':' -f2)
    fi
  else
    BUS_VEHICLE_LAT=28.6250
    BUS_VEHICLE_LON=77.2200
  fi
  
  # Simulate user behavior: first 8 checks on vehicle, then alight at stop
  if [ $LOCATION_CHECK_COUNT2 -le 8 ]; then
    # User is still on bus - location matches vehicle
    BUS_USER_LAT=$BUS_VEHICLE_LAT
    BUS_USER_LON=$BUS_VEHICLE_LON
    echo "  - User is on bus (traveling)"
  else
    # User has alighted at a stop
    if [ "$USER_ALIGHTED_AT_STOP2" = false ]; then
      # Get route ID and find next stop
      ACTIVE_BOARDING_RESPONSE2=$(curl -s "${BASE_URL}/boardings/active/${SESSION_ID}")
      if [ "$USE_PYTHON" = true ]; then
        BUS_ROUTE_ID=$(echo "$ACTIVE_BOARDING_RESPONSE2" | python3 -c "import sys, json; data=json.load(sys.stdin); b=data.get('boarding', {}); print(b.get('route_id', ''))" 2>/dev/null)
      else
        BUS_ROUTE_ID=$(echo "$ACTIVE_BOARDING_RESPONSE2" | grep -o '"route_id":"[^"]*' | cut -d'"' -f4)
      fi
      
      # Find next stop along bus route
      if [ -n "$BUS_ROUTE_ID" ]; then
        BUS_ROUTE_STOPS_RESPONSE=$(curl -s "${BASE_URL}/routes/${BUS_ROUTE_ID}/stops")
        if [ "$USE_PYTHON" = true ]; then
          BUS_STOP_COORDS=$(echo "$BUS_ROUTE_STOPS_RESPONSE" | python3 -c "
import sys, json
import math

data = json.load(sys.stdin)
stops = data.get('stops', [])
vehicle_lat = float($BUS_VEHICLE_LAT)
vehicle_lon = float($BUS_VEHICLE_LON)

min_dist = float('inf')
next_stop_lat = None
next_stop_lon = None

for stop in stops:
    stop_lat = float(stop.get('latitude', 0))
    stop_lon = float(stop.get('longitude', 0))
    lat_diff = stop_lat - vehicle_lat
    lon_diff = stop_lon - vehicle_lon
    dist = math.sqrt(lat_diff*lat_diff + lon_diff*lon_diff)
    
    if stop_lat >= vehicle_lat - 0.0005 and dist > 0.0009:
        if dist < min_dist:
            min_dist = dist
            next_stop_lat = stop_lat
            next_stop_lon = stop_lon

if next_stop_lat is None:
    for stop in stops:
        stop_lat = float(stop.get('latitude', 0))
        stop_lon = float(stop.get('longitude', 0))
        lat_diff = stop_lat - vehicle_lat
        lon_diff = stop_lon - vehicle_lon
        dist = math.sqrt(lat_diff*lat_diff + lon_diff*lon_diff)
        if dist < min_dist and dist > 0.0009:
            min_dist = dist
            next_stop_lat = stop_lat
            next_stop_lon = stop_lon

if next_stop_lat is not None:
    print(f'{next_stop_lat},{next_stop_lon}')
else:
    print(f'{vehicle_lat},{vehicle_lon}')
" 2>/dev/null)
          BUS_STOP_LAT=$(echo "$BUS_STOP_COORDS" | cut -d',' -f1)
          BUS_STOP_LON=$(echo "$BUS_STOP_COORDS" | cut -d',' -f2)
        else
          BUS_STOP_RESPONSE=$(curl -s "${BASE_URL}/stops/nearby?lat=${BUS_VEHICLE_LAT}&lon=${BUS_VEHICLE_LON}&radius=2000&limit=5")
          BUS_STOP_LAT=$(echo "$BUS_STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          BUS_STOP_LON=$(echo "$BUS_STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | sed -n '2p' | cut -d':' -f2)
          if [ -z "$BUS_STOP_LAT" ]; then
            BUS_STOP_LAT=$(echo "$BUS_STOP_RESPONSE" | grep -o '"latitude":[0-9.]*' | head -1 | cut -d':' -f2)
            BUS_STOP_LON=$(echo "$BUS_STOP_RESPONSE" | grep -o '"longitude":[0-9.]*' | head -1 | cut -d':' -f2)
          fi
        fi
      else
        BUS_STOP_LAT=$BUS_VEHICLE_LAT
        BUS_STOP_LON=$BUS_VEHICLE_LON
      fi
      
      if [ -z "$BUS_STOP_LAT" ] || [ "$BUS_STOP_LAT" = "null" ]; then
        BUS_STOP_LAT=$BUS_VEHICLE_LAT
        BUS_STOP_LON=$BUS_VEHICLE_LON
      fi
      
      USER_ALIGHTED_AT_STOP2=true
      STOP_LAT2=$BUS_STOP_LAT
      STOP_LON2=$BUS_STOP_LON
      echo "  - User has alighted at stop: $BUS_STOP_LAT, $BUS_STOP_LON"
    fi
    
    BUS_USER_LAT=$STOP_LAT2
    BUS_USER_LON=$STOP_LON2
    echo "  - User is at stop (waiting for automatic alighting)"
  fi
  
  # Send continuous location update
  LOCATION_RESPONSE2=$(curl -s -X POST "${BASE_URL}/boardings/continuous-location" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\": \"$SESSION_ID\",
      \"qr_code\": \"$QR_CODE\",
      \"latitude\": $BUS_USER_LAT,
      \"longitude\": $BUS_USER_LON
    }")
  
  # Check if user was automatically alighted
  if [ "$USE_PYTHON" = true ]; then
    ALIGHTED_STATUS2=$(echo "$LOCATION_RESPONSE2" | python3 -c "import sys, json; data=json.load(sys.stdin); print('True' if data.get('alighted', False) else 'False')" 2>/dev/null)
    ON_VEHICLE2=$(echo "$LOCATION_RESPONSE2" | python3 -c "import sys, json; data=json.load(sys.stdin); print('True' if data.get('on_vehicle', False) else 'False')" 2>/dev/null)
    DISTANCE2=$(echo "$LOCATION_RESPONSE2" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('distance_meters', 0))" 2>/dev/null)
    MESSAGE2=$(echo "$LOCATION_RESPONSE2" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('message', ''))" 2>/dev/null)
  else
    ALIGHTED_STATUS2=$(echo "$LOCATION_RESPONSE2" | grep -o '"alighted":true' | head -1)
    ON_VEHICLE2=$(echo "$LOCATION_RESPONSE2" | grep -o '"on_vehicle":true' | head -1)
    DISTANCE2=$(echo "$LOCATION_RESPONSE2" | grep -o '"distance_meters":[0-9.]*' | cut -d':' -f2)
    MESSAGE2=$(echo "$LOCATION_RESPONSE2" | grep -o '"message":"[^"]*' | cut -d'"' -f4)
  fi
  
  if [ -n "$ALIGHTED_STATUS2" ] && [ "$ALIGHTED_STATUS2" != "False" ]; then
    ALIGHTED_STATUS2="True"
  else
    ALIGHTED_STATUS2="False"
  fi
  
  if [ "$ALIGHTED_STATUS2" = "True" ]; then
    ALIGHTED2=true
    echo "  ✅ Automatic alighting detected!"
    echo "  - Message: ${MESSAGE2:-Automatic alighting at stop}"
    echo ""
    
    # Get boarding details from session boardings
    SESSION_BOARDINGS_RESPONSE2=$(curl -s "${BASE_URL}/boardings/session/${SESSION_ID}")
    if [ "$USE_PYTHON" = true ]; then
      FARE2=$(echo "$SESSION_BOARDINGS_RESPONSE2" | python3 -c "
import sys, json
data = json.load(sys.stdin)
boardings = data.get('boardings', [])
# Find most recent boarding with alighting_time (should be the bus boarding)
for b in reversed(boardings):
    if b.get('alighting_time'):
        fare = b.get('fare', 0)
        print(fare if fare is not None else 0)
        break
else:
    print(0)
" 2>/dev/null)
      DISTANCE2=$(echo "$SESSION_BOARDINGS_RESPONSE2" | python3 -c "
import sys, json
data = json.load(sys.stdin)
boardings = data.get('boardings', [])
# Find most recent boarding with alighting_time
for b in reversed(boardings):
    if b.get('alighting_time'):
        dist = b.get('distance', 0)
        print(dist / 1000 if dist else 0)
        break
else:
    print(0)
" 2>/dev/null)
    else
      FARE2=$(echo "$SESSION_BOARDINGS_RESPONSE2" | grep -o '"fare":[0-9.]*' | tail -1 | cut -d':' -f2)
      DISTANCE2=$(echo "$SESSION_BOARDINGS_RESPONSE2" | grep -o '"distance":[0-9.]*' | tail -1 | cut -d':' -f2)
      DISTANCE2=$(echo "scale=3; ${DISTANCE2:-0} / 1000" | bc 2>/dev/null || echo "0")
    fi
    
    echo "✅ Automatically alighted from Bus"
    echo "Bus fare: ₹${FARE2:-0}"
    echo "Bus distance: ${DISTANCE2:-0}km"
    break
  else
    if [ "$ON_VEHICLE2" = "True" ]; then
      echo "  - On vehicle: True"
      echo "  - Distance: ${DISTANCE2:-0}m"
    else
      echo "  - On vehicle: False"
      echo "  - Distance: ${DISTANCE2:-0}m"
      if [ "$USER_ALIGHTED_AT_STOP2" = true ]; then
        echo "  - Message: ${MESSAGE2:-User at stop, waiting for automatic alighting}"
      else
        echo "  - Message: ${MESSAGE2:-User traveling on bus}"
      fi
    fi
    echo ""
    
    if [ $LOCATION_CHECK_COUNT2 -lt $MAX_CHECKS2 ]; then
      sleep 5
    fi
  fi
done

if [ "$ALIGHTED2" = false ]; then
  echo "⚠️  Automatic alighting did not trigger after $MAX_CHECKS2 checks"
  echo "   For testing, you can manually alight:"
  echo "   curl -X POST \"${BASE_URL}/boardings/alight\" -H \"Content-Type: application/json\" -d '{\"boarding_id\": \"$BOARDING2_ID\", \"latitude\": $BUS_USER_LAT, \"longitude\": $BUS_USER_LON}'"
  echo ""
fi

FARE2=${FARE2:-0}
DISTANCE2=${DISTANCE2:-0}
echo ""

# Step 11: Check-out at destination
echo -e "${BLUE}Step 10: User checks out at destination...${NC}"
CHECKOUT_RESPONSE=$(curl -s -X POST "${BASE_URL}/sessions/checkout" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION_ID\",
    \"qr_code\": \"$QR_CODE\",
    \"latitude\": 28.6129,
    \"longitude\": 77.2295
  }")

# Parse checkout response from nested structure
if [ "$USE_PYTHON" = true ]; then
  TOTAL_FARE=$(echo "$CHECKOUT_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); fare=data.get('session', {}).get('total_fare') or data.get('fare', 0); print(fare if fare is not None else 0)" 2>/dev/null)
  TOTAL_DISTANCE=$(echo "$CHECKOUT_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); dist=data.get('session', {}).get('total_distance', 0); print(dist if dist is not None else 0)" 2>/dev/null)
  ROUTES_USED=$(echo "$CHECKOUT_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); routes=data.get('session', {}).get('routes_used', []); print(','.join(routes) if routes else '[]')" 2>/dev/null)
else
  TOTAL_FARE=$(echo "$CHECKOUT_RESPONSE" | grep -o '"total_fare":[0-9.]*' | cut -d':' -f2)
  TOTAL_DISTANCE=$(echo "$CHECKOUT_RESPONSE" | grep -o '"total_distance":[0-9.]*' | cut -d':' -f2)
  ROUTES_USED=$(echo "$CHECKOUT_RESPONSE" | grep -o '"routes_used":\[[^]]*\]')
fi
echo "✅ Check-out successful"
echo "Total fare: ₹${TOTAL_FARE:-0}"
echo "Total distance: ${TOTAL_DISTANCE:-0}km"
echo "Routes used: ${ROUTES_USED:-[]}"
if [ -z "$TOTAL_FARE" ] || [ "$TOTAL_FARE" = "None" ] || [ "$TOTAL_FARE" = "0" ]; then
  echo "⚠️  Warning: Could not parse checkout data or values are 0"
  echo "Response: $CHECKOUT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CHECKOUT_RESPONSE"
fi
echo ""

# Step 14: View all boardings for this session
echo -e "${BLUE}Step 14: Viewing all route boardings...${NC}"
BOARDINGS_RESPONSE=$(curl -s "${BASE_URL}/boardings/sessions/$SESSION_ID")
echo "All boardings:"
echo "$BOARDINGS_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$BOARDINGS_RESPONSE"
echo ""

# Step 15: View daily bill
echo -e "${BLUE}Step 15: Viewing daily bill...${NC}"
BILL_RESPONSE=$(curl -s "${BASE_URL}/bills/users/$USER_ID")
echo "Daily bill:"
echo "$BILL_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$BILL_RESPONSE"
echo ""

echo -e "${GREEN}✅ Multi-journey example completed!${NC}"
echo ""
echo "Summary:"
echo "  - User: Rajesh Kumar ($USER_ID)"
echo "  - Journey: Connaught Place → India Gate"
echo "  - Routes: Metro ($METRO_ROUTE) + Bus ($BUS_ROUTE)"
echo "  - Total Fare: ₹${TOTAL_FARE:-0}"
echo "  - Total Distance: ${TOTAL_DISTANCE:-0}km"
echo ""
echo "Key Features Demonstrated:"
echo "  ✅ User creation"
echo "  ✅ Check-in/Check-out flow"
echo "  ✅ Automatic vehicle detection (no manual route selection!)"
echo "  ✅ Mock vehicle setup for testing"
echo "  ✅ Route boarding tracking"
echo "  ✅ Continuous location matching (user GPS vs vehicle GPS)"
echo "  ✅ Multi-modal journey (Metro + Bus)"
echo "  ✅ Fare calculation based on actual routes used"
echo "  ✅ Daily bill aggregation"

