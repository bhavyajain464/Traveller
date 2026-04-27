package services

import (
	"math"
	"testing"
	"time"

	"indian-transit-backend/internal/models"
)

func TestAddAccessAndEgressLegsSkipsEmptyOptions(t *testing.T) {
	adapter := &InMemoryJourneyPlannerAdapter{}
	departureTime := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)

	options := adapter.addAccessAndEgressLegs(
		models.JourneyRequest{},
		models.Stop{ID: "from", Name: "From"},
		models.Stop{ID: "to", Name: "To"},
		departureTime,
		[]models.JourneyOption{{}},
	)

	if len(options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(options))
	}
	if options[0].Duration != math.MaxInt {
		t.Fatalf("expected empty-leg option to be marked invalid, got duration %d", options[0].Duration)
	}
}
