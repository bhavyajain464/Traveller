package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"
)

func main() {
	cfg := config.Load()

	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	matches, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		log.Fatalf("failed to list migrations: %v", err)
	}
	sort.Strings(matches)

	if len(matches) == 0 {
		log.Println("no migration files found")
		return
	}

	for _, migrationFile := range matches {
		log.Printf("running migration: %s", migrationFile)
		content, readErr := os.ReadFile(migrationFile)
		if readErr != nil {
			log.Fatalf("failed reading %s: %v", migrationFile, readErr)
		}

		lines := strings.Split(string(content), "\n")
		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			filtered = append(filtered, line)
		}

		parts := strings.Split(strings.Join(filtered, "\n"), ";")
		for _, part := range parts {
			stmt := strings.TrimSpace(part)
			if stmt == "" {
				continue
			}
			if _, execErr := db.Exec(stmt); execErr != nil {
				log.Fatalf("migration %s failed on statement %q: %v", migrationFile, stmt, execErr)
			}
		}
	}

	log.Println("all migrations completed successfully")
}
