package tests

import (
	"testing"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/services"
)

func TestStopService(t *testing.T) {
	// This is a placeholder test structure
	// In a real scenario, you would set up a test database
	t.Skip("Integration test requires test database setup")
}

func TestRouteService(t *testing.T) {
	t.Skip("Integration test requires test database setup")
}

func TestRoutePlanner(t *testing.T) {
	t.Skip("Integration test requires test database setup")
}

// Example test setup helper
func setupTestDB(t *testing.T) *database.DB {
	// Connect to test database
	db, err := database.New(
		"localhost",
		"5432",
		"postgres",
		"postgres",
		"transit_test",
		"disable",
	)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	return db
}

func TestStopService_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	stopService := services.NewStopService(db)
	
	// Test with a known stop ID from Delhi Metro data (New Delhi station)
	stop, err := stopService.GetByID("49")
	if err != nil {
		t.Fatalf("Failed to get stop: %v", err)
	}

	if stop == nil {
		t.Fatal("Stop should not be nil")
	}

	if stop.ID != "49" {
		t.Errorf("Expected stop ID 49, got %s", stop.ID)
	}
}

func TestStopService_FindNearby(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	stopService := services.NewStopService(db)
	
	// Test with Delhi coordinates (Connaught Place area)
	stops, err := stopService.FindNearby(28.6304, 77.2177, 1000, 10)
	if err != nil {
		t.Fatalf("Failed to find nearby stops: %v", err)
	}

	if len(stops) == 0 {
		t.Fatal("Should find at least one nearby stop")
	}
}


