package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/udit-001/harbor/internal/migrate"
	_ "modernc.org/sqlite"
)

// maxOpenConns caps the connection pool size.
//
// Do NOT set this to 1. With a single connection, any code path that checks
// out a connection and fails to return it (a leaked *sql.Rows whose Close is
// skipped, an uncommitted tx, a goroutine that died mid-query) permanently
// deadlocks the entire database: every subsequent query blocks forever on
// pool acquisition, and busy_timeout cannot help because it governs the
// SQLite file lock, not Go's pool checkout. The result is the dashboard
// booting, the listener accepting connections, and every DB-touching route
// hanging with zero bytes returned.
//
// A small pool (4) gives enough headroom that a single leak degrades
// performance instead of wedging the server, while staying low enough that
// SQLite writer contention is rare (WAL + busy_timeout=5000 serializes
// writers at the file level anyway). This is a mitigation; a leaked
// connection is still a bug to find and fix.
const maxOpenConns = 4

// Store wraps the SQLite database. The *sqlx.DB handle is private so that
// callers cannot bypass the typed query methods with raw SQL — the workspace
// scoping from WorkspaceStore stays enforced (LEARN-12).
type Store struct {
	db *sqlx.DB
}

// nowTimestamp returns the current UTC time as an RFC3339Nano string.
// This is the single source of truth for timestamp formatting across
// all write paths, ensuring ORDER BY works correctly on TEXT columns.
func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// SQL exposes the underlying *sql.DB for migration tooling (goose). It is
// intentionally narrow: only the migrate package needs the raw handle.
func (s *Store) SQL() *sql.DB { return s.db.DB }

// Open opens (or creates) the SQLite database and runs migrations.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.DB.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Run goose migrations
	if err := migrate.Up(db.DB); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	store := &Store{db: db}

	return store, nil
}

// OpenRaw opens a raw *sql.DB without migrations or sqlx wrapping.
// Used by the migrate CLI commands to avoid double-migration.
func OpenRaw(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return nil, fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

var _ sql.DB
