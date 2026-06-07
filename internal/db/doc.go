// Package db provides SQLite-backed storage for nightshift state and snapshots.
//
// It manages the database lifecycle (open, migrate, close) and exposes methods
// for the persistence layer used by other internal packages.
//
// # Database Location
//
// The default path is ~/.local/share/nightshift/nightshift.db. Override with
// the config key db.path or the --db flag.
//
// # Migrations
//
// Schema changes are tracked in the migrations.go file. Each migration is
// applied once and recorded in a schema_version table. The database is
// automatically migrated on Open.
//
// # Legacy Import
//
// On first migration, the package imports state from the legacy state.json
// file if it exists (from nightshift versions prior to SQLite).
package db
