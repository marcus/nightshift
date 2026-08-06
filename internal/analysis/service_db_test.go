package analysis

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE service_advisor_results (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			component       TEXT NOT NULL,
			timestamp       DATETIME NOT NULL,
			packages        TEXT NOT NULL,
			metrics         TEXT NOT NULL,
			recommendations TEXT NOT NULL,
			report_path     TEXT
		);
		CREATE INDEX idx_service_advisor_component_time ON service_advisor_results(component, timestamp DESC);
	`)
	if err != nil {
		t.Fatalf("creating table: %v", err)
	}

	return db
}

func TestServiceAdvisorResultStoreAndLoad(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	result := &ServiceAdvisorResult{
		Component: "test-project",
		Timestamp: time.Now().Truncate(time.Second),
		Packages: []PackageAnalysis{
			{
				Package: PackageInfo{Name: "api", Path: "internal/api", Files: 5, LOC: 500},
				Metrics: ServiceMetrics{CouplingScore: 0.3, CohesionScore: 0.8, Recommendation: "extract"},
			},
		},
		Metrics: []ServiceMetrics{
			{CouplingScore: 0.3, CohesionScore: 0.8, Recommendation: "extract"},
		},
		Recommendations: []string{"Extract internal/api"},
	}

	// Store
	if err := result.Store(db); err != nil {
		t.Fatalf("storing result: %v", err)
	}
	if result.ID == 0 {
		t.Errorf("expected non-zero ID after store")
	}

	// Load latest
	loaded, err := LoadLatestServiceResult(db, "test-project")
	if err != nil {
		t.Fatalf("loading latest: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected result, got nil")
	}
	if loaded.Component != "test-project" {
		t.Errorf("expected component 'test-project', got %s", loaded.Component)
	}
	if len(loaded.Packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(loaded.Packages))
	}
	if loaded.Packages[0].Package.Path != "internal/api" {
		t.Errorf("expected package path 'internal/api', got %s", loaded.Packages[0].Package.Path)
	}
	if len(loaded.Recommendations) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(loaded.Recommendations))
	}
}

func TestServiceAdvisorResultStoreNilDB(t *testing.T) {
	result := &ServiceAdvisorResult{}
	if err := result.Store(nil); err == nil {
		t.Errorf("expected error for nil db")
	}
}

func TestLoadLatestServiceResultNilDB(t *testing.T) {
	_, err := LoadLatestServiceResult(nil, "test")
	if err == nil {
		t.Errorf("expected error for nil db")
	}
}

func TestLoadLatestServiceResultNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	result, err := LoadLatestServiceResult(db, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nonexistent component")
	}
}

func TestLoadAllServiceResults(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Store two results
	for i := 0; i < 2; i++ {
		result := &ServiceAdvisorResult{
			Component:       "test-project",
			Timestamp:       time.Now().Add(time.Duration(i) * time.Hour),
			Packages:        []PackageAnalysis{},
			Metrics:         []ServiceMetrics{},
			Recommendations: []string{"rec"},
		}
		if err := result.Store(db); err != nil {
			t.Fatalf("storing result %d: %v", i, err)
		}
	}

	// Load all
	results, err := LoadAllServiceResults(db, "test-project", time.Time{})
	if err != nil {
		t.Fatalf("loading all: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Load with since filter
	future := time.Now().Add(24 * time.Hour)
	results2, err := LoadAllServiceResults(db, "test-project", future)
	if err != nil {
		t.Fatalf("loading with since: %v", err)
	}
	if len(results2) != 0 {
		t.Errorf("expected 0 results with future since, got %d", len(results2))
	}
}

func TestLoadAllServiceResultsNilDB(t *testing.T) {
	_, err := LoadAllServiceResults(nil, "test", time.Time{})
	if err == nil {
		t.Errorf("expected error for nil db")
	}
}
