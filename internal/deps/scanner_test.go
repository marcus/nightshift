package deps

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

type testLogger struct{ t *testing.T }

func (l *testLogger) Infof(format string, args ...any)  { l.t.Logf("INFO: "+format, args...) }
func (l *testLogger) Warnf(format string, args ...any)  { l.t.Logf("WARN: "+format, args...) }
func (l *testLogger) Errorf(format string, args ...any) { l.t.Logf("ERROR: "+format, args...) }

func TestScanProjectWithMinimalGoMod(t *testing.T) {
	dir := t.TempDir()
	gomod := `module example.com/testmod

go 1.21

require github.com/rs/zerolog v1.34.0
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}

	db := setupTestDB(t)
	store := NewStore(db)
	if err := store.InitTables(); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(store, "", &testLogger{t})

	result, _ := scanner.ScanProject(context.Background(), dir)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Project != filepath.Base(dir) {
		t.Errorf("project = %q, want %q", result.Project, filepath.Base(dir))
	}
	if len(result.Deps) != 1 {
		t.Errorf("got %d deps, want 1", len(result.Deps))
	}

	saved, err := store.LatestScan(result.Project)
	if err != nil {
		t.Fatalf("LatestScan error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved scan")
	}
	if saved.Project != result.Project {
		t.Errorf("saved project = %q, want %q", saved.Project, result.Project)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	if err := store.InitTables(); err != nil {
		t.Fatal(err)
	}

	result := &ScanResult{
		Project: "test-project",
		Deps: []Dependency{
			{Module: "github.com/foo/bar", Version: "v1.0.0"},
		},
		Findings: []Finding{
			{
				Module:    "github.com/foo/bar",
				Version:   "v1.0.0",
				RiskLevel: RiskHigh,
				Category:  CategoryVulnerability,
				Detail:    "CVE-2024-1234",
				CVEID:     "CVE-2024-1234",
				CVSSScore: 7.5,
			},
		},
	}

	id, err := store.SaveScan(result)
	if err != nil {
		t.Fatalf("SaveScan error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive scan id, got %d", id)
	}

	loaded, err := store.LatestScan("test-project")
	if err != nil {
		t.Fatalf("LatestScan error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded scan")
	}
	if len(loaded.Findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(loaded.Findings))
	}
	if loaded.Findings[0].CVEID != "CVE-2024-1234" {
		t.Errorf("CVEID = %q, want CVE-2024-1234", loaded.Findings[0].CVEID)
	}
}

func TestAllLatestScans(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	if err := store.InitTables(); err != nil {
		t.Fatal(err)
	}

	for _, project := range []string{"proj-a", "proj-b"} {
		result := &ScanResult{
			Project:  project,
			Deps:     []Dependency{{Module: "github.com/x/y", Version: "v1.0.0"}},
			Findings: []Finding{{Module: "github.com/x/y", Version: "v1.0.0", RiskLevel: RiskLow, Category: CategoryLicense, Detail: "MIT"}},
		}
		if _, err := store.SaveScan(result); err != nil {
			t.Fatal(err)
		}
	}

	scans, err := store.AllLatestScans()
	if err != nil {
		t.Fatalf("AllLatestScans error: %v", err)
	}
	if len(scans) != 2 {
		t.Fatalf("got %d scans, want 2", len(scans))
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatal(err)
	}
	return db
}
