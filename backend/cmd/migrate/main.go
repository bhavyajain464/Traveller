package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"indian-transit-backend/internal/config"
	"indian-transit-backend/internal/database"

	"github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	db, err := database.NewFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		log.Fatalf("failed to resolve migrations directory: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		log.Fatalf("failed to list migrations: %v", err)
	}
	sort.Strings(matches)

	if len(matches) == 0 {
		log.Println("no migration files found")
		return
	}

	for _, migrationFile := range matches {
		log.Printf("running migration: %s", filepath.Base(migrationFile))
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
				if isIgnorableMigrationError(execErr) {
					log.Printf("skipping already-applied statement in %s: %v", filepath.Base(migrationFile), execErr)
					continue
				}
				log.Fatalf("migration %s failed on statement %q: %v", migrationFile, stmt, execErr)
			}
		}
	}

	log.Println("all migrations completed successfully")
}

func resolveMigrationsDir() (string, error) {
	candidates := []string{
		"migrations",
		filepath.Join("backend", "migrations"),
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")))
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not find migrations directory from candidates: %s", strings.Join(candidates, ", "))
}

func isIgnorableMigrationError(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}

	switch pqErr.Code {
	case "42P07", "42701", "42710":
		return true
	default:
		return false
	}
}
