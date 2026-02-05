package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection and path.
type DB struct {
	sql  *sql.DB
	path string
}

// DefaultPath returns the default database path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "nightshift.db")
}

// Open opens or creates the database, applies pragmas, and runs migrations.
func Open(dbPath string) (*DB, error) {
	if dbPath == "" {
		dbPath = DefaultPath()
	}

	resolved := expandPath(dbPath)
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return nil, fmt.Errorf("creating db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", resolved)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := applyPragmas(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	if err := Migrate(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	if err := importLegacyState(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return &DB{sql: sqlDB, path: resolved}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

// SQL returns the raw *sql.DB for advanced usage.
func (d *DB) SQL() *sql.DB {
	if d == nil {
		return nil
	}
	return d.sql
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("setting pragma %q: %w", pragma, err)
		}
	}
	return nil
}

func expandPath(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}
