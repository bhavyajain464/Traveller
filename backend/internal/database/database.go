package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"indian-transit-backend/internal/config"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Tx struct {
	*sql.Tx
}

// NewFromConfig opens a PostgreSQL/PostGIS connection using either DATABASE_URL
// or the individual DB_* environment variables.
func NewFromConfig(cfg config.DatabaseConfig) (*DB, error) {
	if cfg.URL != "" {
		return NewFromURL(cfg.URL)
	}

	return New(cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
}

func NewFromURL(rawURL string) (*DB, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported database scheme %q (expected postgres or postgresql)", parsed.Scheme)
	}

	params := parsed.Query()
	if params.Get("sslmode") == "" {
		params.Set("sslmode", "disable")
	}
	parsed.RawQuery = params.Encode()

	return NewWithDSN(parsed.String())
}

func NewWithDSN(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func New(host, port, user, password, dbname, sslMode string) (*DB, error) {
	if sslMode == "" {
		sslMode = "disable"
	}

	values := url.Values{}
	values.Set("sslmode", sslMode)
	values.Set("connect_timeout", "10")

	dsn := (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + strings.TrimPrefix(dbname, "/"),
	}).String()

	return NewWithDSN(dsn + "?" + values.Encode())
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.DB.Exec(rebindPostgres(query), args...)
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(rebindPostgres(query), args...)
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(rebindPostgres(query), args...)
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.Tx.Exec(rebindPostgres(query), args...)
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.Tx.Query(rebindPostgres(query), args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.Tx.QueryRow(rebindPostgres(query), args...)
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// SetupConnectionPool configures connection pool parameters.
func (db *DB) SetupConnectionPool(maxOpenConns, maxIdleConns int, connMaxLifetime int) {
	db.DB.SetMaxOpenConns(maxOpenConns)
	db.DB.SetMaxIdleConns(maxIdleConns)
	if connMaxLifetime > 0 {
		db.DB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	}
}

// Rebind converts MySQL-style ? placeholders to PostgreSQL $1, $2, ...
// placeholders. It deliberately ignores ? characters inside single-quoted
// string literals, which covers the SQL used by this codebase.
func Rebind(query string) string {
	return rebindPostgres(query)
}

func rebindPostgres(query string) string {
	var builder strings.Builder
	builder.Grow(len(query) + 8)

	argIndex := 1
	inSingleQuote := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			builder.WriteByte(ch)
			if inSingleQuote && i+1 < len(query) && query[i+1] == '\'' {
				i++
				builder.WriteByte(query[i])
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}
		if ch == '?' && !inSingleQuote {
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(argIndex))
			argIndex++
			continue
		}
		builder.WriteByte(ch)
	}

	return builder.String()
}

// IsUniqueViolation checks if error is a unique constraint violation.
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
