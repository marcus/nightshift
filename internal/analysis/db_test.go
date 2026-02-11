package analysis

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

func TestBusFactorResultStore(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create the table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	result := &BusFactorResult{
		Component:  "core-api",
		Timestamp:  time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		RiskLevel:  "high",
		ReportPath: "/path/to/report.md",
		Metrics: &OwnershipMetrics{
			HerfindahlIndex: 0.45,
			BusFactor:       2,
			RiskLevel:       "high",
		},
		Contributors: []CommitAuthor{
			{Name: "Alice", Email: "alice@example.com", Commits: 100},
			{Name: "Bob", Email: "bob@example.com", Commits: 50},
		},
	}

	err = result.Store(database.SQL())
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	if result.ID == 0 {
		t.Fatal("expected non-zero ID after Store")
	}
}

func TestBusFactorResultStoreNilDB(t *testing.T) {
	result := &BusFactorResult{
		Component: "test",
		Timestamp: time.Now(),
	}

	err := result.Store(nil)
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestLoadLatest(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create the table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Store two results
	now := time.Now()
	result1 := &BusFactorResult{
		Component: "core-api",
		Timestamp: now.Add(-time.Hour),
		RiskLevel: "low",
		Metrics: &OwnershipMetrics{
			HerfindahlIndex: 0.2,
			BusFactor:       5,
		},
		Contributors: []CommitAuthor{},
	}
	if err := result1.Store(database.SQL()); err != nil {
		t.Fatalf("Store result1: %v", err)
	}

	result2 := &BusFactorResult{
		Component: "core-api",
		Timestamp: now,
		RiskLevel: "high",
		Metrics: &OwnershipMetrics{
			HerfindahlIndex: 0.5,
			BusFactor:       2,
		},
		Contributors: []CommitAuthor{},
	}
	if err := result2.Store(database.SQL()); err != nil {
		t.Fatalf("Store result2: %v", err)
	}

	// Load latest should return result2
	loaded, err := LoadLatest(database.SQL(), "core-api")
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}

	if loaded == nil {
		t.Fatal("expected non-nil result")
	}
	if loaded.RiskLevel != "high" {
		t.Fatalf("expected risk level 'high', got %s", loaded.RiskLevel)
	}
	if loaded.Metrics.BusFactor != 2 {
		t.Fatalf("expected bus factor 2, got %d", loaded.Metrics.BusFactor)
	}
}

func TestLoadLatestNotFound(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create empty table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	loaded, err := LoadLatest(database.SQL(), "nonexistent")
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}

	if loaded != nil {
		t.Fatal("expected nil result for nonexistent component")
	}
}

func TestLoadLatestNilDB(t *testing.T) {
	_, err := LoadLatest(nil, "test")
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestLoadAll(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create the table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Store multiple results for the same component
	now := time.Now()
	for i := 0; i < 3; i++ {
		result := &BusFactorResult{
			Component: "utils",
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			RiskLevel: "medium",
			Metrics: &OwnershipMetrics{
				HerfindahlIndex: 0.3,
				BusFactor:       3,
			},
			Contributors: []CommitAuthor{},
		}
		if err := result.Store(database.SQL()); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	// Load all should return 3 results
	results, err := LoadAll(database.SQL(), "utils", time.Time{})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestLoadAllWithDateFilter(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create the table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Store results at different times
	baseTime := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		result := &BusFactorResult{
			Component: "auth",
			Timestamp: baseTime.Add(-time.Duration(i*24) * time.Hour),
			RiskLevel: "low",
			Metrics: &OwnershipMetrics{
				HerfindahlIndex: 0.2,
				BusFactor:       4,
			},
			Contributors: []CommitAuthor{},
		}
		if err := result.Store(database.SQL()); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	// Load all with a date filter (24 hours back - should exclude i=2,3)
	since := baseTime.Add(-24 * time.Hour)
	results, err := LoadAll(database.SQL(), "auth", since)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// Should include i=0 (baseTime) and i=1 (baseTime-24h) = 2 results
	if len(results) != 2 {
		t.Fatalf("expected 2 results after filtering, got %d", len(results))
	}
}

func TestLoadAllNilDB(t *testing.T) {
	_, err := LoadAll(nil, "test", time.Time{})
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestLoadAllEmpty(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Create empty table
	_, err = database.SQL().Exec(`
		CREATE TABLE IF NOT EXISTS bus_factor_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			component TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metrics TEXT NOT NULL,
			contributors TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			report_path TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	results, err := LoadAll(database.SQL(), "nonexistent", time.Time{})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
