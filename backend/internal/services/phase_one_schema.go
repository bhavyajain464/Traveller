package services

import (
	"database/sql"
	"fmt"
	"strings"

	"indian-transit-backend/internal/database"
)

var requiredPhaseOneTables = []string{
	"journey_segments",
	"journey_events",
	"fare_transactions",
}

// EnsurePhaseOneJourneySchema fails fast when the Phase 1 journey tables are
// missing so startup points to migrations instead of later write failures.
func EnsurePhaseOneJourneySchema(db *database.DB) error {
	missing := make([]string, 0, len(requiredPhaseOneTables))

	for _, tableName := range requiredPhaseOneTables {
		var resolved sql.NullString
		if err := db.QueryRow(`SELECT to_regclass(?)`, "public."+tableName).Scan(&resolved); err != nil {
			return fmt.Errorf("check table %s: %w", tableName, err)
		}
		if !resolved.Valid || resolved.String == "" {
			missing = append(missing, tableName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"missing required Phase 1 tables: %s; run the backend migrations (for example: `cd backend && go run cmd/migrate/main.go`)",
			strings.Join(missing, ", "),
		)
	}

	return nil
}
