package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(host, port, user, password, dbname, sslmode string) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable PostGIS extension
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS postgis")
	if err != nil {
		return nil, fmt.Errorf("failed to enable PostGIS extension: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// IsUniqueViolation checks if error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	if err, ok := err.(*pq.Error); ok {
		return err.Code == "23505"
	}
	return false
}


