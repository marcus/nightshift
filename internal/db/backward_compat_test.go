package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBackwardCompat_DBDirPermissions verifies that the database directory
// is created with 0700 (rwx------) permissions for security in v0.3.1.
// Old databases should continue to work, new ones get stricter permissions.
func TestBackwardCompat_DBDirPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Check directory permissions are restrictive (0700)
	dirPath := filepath.Dir(dbPath)
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("stat db dir: %v", err)
	}

	mode := info.Mode().Perm()
	expected := os.FileMode(0700)

	if mode != expected {
		t.Errorf("DB directory permissions = %o, want %o", mode, expected)
	}
}

// TestBackwardCompat_OldDatabaseStillWorks verifies that if a user had
// a database from v0.3.0 with 0755 permissions, it still works in v0.3.1.
func TestBackwardCompat_OldDatabaseStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	dbDirPath := filepath.Join(tmpDir, "olddb")

	// Create directory with old permissions (0755)
	if err := os.MkdirAll(dbDirPath, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dbDirPath, "nightshift.db")

	// Open should succeed and run migrations
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open old database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Verify schema was applied
	tables := []string{"projects", "task_history", "assigned_tasks", "run_history", "snapshots"}
	for _, table := range tables {
		if !tableExists(t, database.SQL(), table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}
}

// TestBackwardCompat_MigrationIdempotency verifies that running migrations
// multiple times (as would happen if db.Open is called multiple times)
// is safe and doesn't corrupt the database.
func TestBackwardCompat_MigrationIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nightshift.db")

	// First open - runs all migrations
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}

	version1, err := CurrentVersion(db1.SQL())
	if err != nil {
		t.Fatalf("get version 1: %v", err)
	}

	if err := db1.Close(); err != nil {
		t.Fatalf("close db1: %v", err)
	}

	// Second open - should not re-apply migrations
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer func() { _ = db2.Close() }()

	version2, err := CurrentVersion(db2.SQL())
	if err != nil {
		t.Fatalf("get version 2: %v", err)
	}

	if version1 != version2 {
		t.Errorf("schema version changed after idempotent open: %d -> %d", version1, version2)
	}

	// Both should match current version
	expectedVersion := len(migrations)
	if version2 != expectedVersion {
		t.Errorf("final schema version = %d, want %d", version2, expectedVersion)
	}
}

// TestBackwardCompat_ProviderColumnAdded verifies that the provider column
// added in migration 003 exists for new databases.
func TestBackwardCompat_ProviderColumnAdded(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if !columnExists(t, database.SQL(), "run_history", "provider") {
		t.Fatal("provider column missing from run_history")
	}
}

// TestBackwardCompat_ResetTimeColumnsAdded verifies that reset time columns
// added in migration 002 exist for new databases.
func TestBackwardCompat_ResetTimeColumnsAdded(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if !columnExists(t, database.SQL(), "snapshots", "session_reset_time") {
		t.Fatal("session_reset_time column missing from snapshots")
	}
	if !columnExists(t, database.SQL(), "snapshots", "weekly_reset_time") {
		t.Fatal("weekly_reset_time column missing from snapshots")
	}
}

// TestBackwardCompat_PathExpansion verifies that ~ is correctly expanded
// in database paths, preserving old behavior.
func TestBackwardCompat_PathExpansion(t *testing.T) {
	// Test tilde expansion
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test.db", filepath.Join(home, "test.db")},
		{"/absolute/test.db", "/absolute/test.db"},
		{"relative/test.db", "relative/test.db"},
	}

	for _, tc := range tests {
		result := expandPath(tc.input)
		if result != tc.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestBackwardCompat_EmptyPathUsesDefault verifies that an empty path
// correctly falls back to the default path.
func TestBackwardCompat_EmptyPathUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	database, err := Open("")
	if err != nil {
		t.Fatalf("Open with empty path: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Should have used default path
	expectedPath := filepath.Join(tmpDir, ".local", "share", "nightshift", "nightshift.db")
	if database.path != expectedPath {
		t.Errorf("database path = %q, want %q", database.path, expectedPath)
	}
}

// TestBackwardCompat_PragmasApplied verifies that database pragmas
// (journal_mode=WAL, busy_timeout, etc.) are still applied correctly.
func TestBackwardCompat_PragmasApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Check journal mode
	var journalMode string
	if err := database.SQL().QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	// Check foreign keys enabled
	var fkEnabled int
	if err := database.SQL().QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkEnabled)
	}
}
