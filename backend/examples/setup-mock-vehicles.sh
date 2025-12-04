#!/bin/bash

# Script to set up mock vehicles at specific locations for testing
# This simulates vehicles at the locations used in the journey example

BASE_URL="http://localhost:8080/api/v1"
echo "🚗 Setting up mock vehicles for testing..."
echo ""

# Note: Since we're using in-memory mock vehicles, we'll need to add them via API
# For now, the VehicleLocationService initializes vehicles automatically on startup
# But we can add specific vehicles if needed

echo "✅ Mock vehicles are initialized automatically when server starts"
echo "   Vehicles are positioned at stops along active routes"
echo ""
echo "To add specific vehicles, you can use the AddMockVehicle method in the service"
echo "or restart the server to reload vehicles from database routes"
echo ""

