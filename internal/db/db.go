package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store wraps a *sql.DB with helpers for clipd's persistence layer.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path and
// applies migrations. It is safe to call multiple times — schema
// statements use IF NOT EXISTS.
func Open(path string) (*Store, error) {
	// modernc.org/sqlite uses the "sqlite" driver name (not "sqlite3").
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	d.SetMaxOpenConns(1) // SQLite + WAL: single writer simplifies things
	if _, err := d.Exec(schemaSQL); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: d}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Raw returns the underlying *sql.DB for advanced use (tests, etc.).
func (s *Store) Raw() *sql.DB { return s.db }
