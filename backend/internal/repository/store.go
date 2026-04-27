package repository

import "database/sql"

// DBTX is the minimal query surface shared by *database.DB and *database.Tx.
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}
