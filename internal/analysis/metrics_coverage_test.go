package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGoFile is a test helper that writes a Go source file and returns its path.
func writeGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScannerNoGoFiles(t *testing.T) {
	dir := t.TempDir()
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestScannerNoExportedFunctions(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "internal.go", `package foo

func doStuff() {}
func helper() int { return 1 }
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages (no exported funcs), got %d", len(pkgs))
	}
}

func TestScannerUninstrumentedFunctions(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "api.go", `package api

func GetUser(id string) (string, error) {
	return "user-" + id, nil
}

func ListItems() []string {
	return nil
}

func DeleteItem(id string) error {
	return nil
}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if pkg.TotalExported != 3 {
		t.Errorf("expected 3 exported functions, got %d", pkg.TotalExported)
	}
	if pkg.InstrumentedCount != 0 {
		t.Errorf("expected 0 instrumented functions, got %d", pkg.InstrumentedCount)
	}
	if pkg.CoveragePct != 0 {
		t.Errorf("expected 0%% coverage, got %.1f%%", pkg.CoveragePct)
	}
	if len(pkg.Gaps) != 3 {
		t.Errorf("expected 3 gaps, got %d", len(pkg.Gaps))
	}
}

func TestScannerPrometheusInstrumented(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "handler.go", `package handler

import "github.com/prometheus/client_golang/prometheus"

var reqCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "requests_total",
})

func HandleRequest(w string, r string) {
	reqCounter.Inc()
}

func HealthCheck() string {
	return "ok"
}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if pkg.TotalExported != 2 {
		t.Errorf("expected 2 exported functions, got %d", pkg.TotalExported)
	}
	if pkg.InstrumentedCount != 1 {
		t.Errorf("expected 1 instrumented function, got %d", pkg.InstrumentedCount)
	}
	if pkg.CoveragePct != 50 {
		t.Errorf("expected 50%% coverage, got %.1f%%", pkg.CoveragePct)
	}

	// Check metric types detected
	foundPrometheus := false
	for _, mt := range pkg.DetectedMetricTypes {
		if mt == "prometheus" {
			foundPrometheus = true
		}
	}
	if !foundPrometheus {
		t.Errorf("expected prometheus in detected metric types, got %v", pkg.DetectedMetricTypes)
	}
}

func TestScannerExpvarInstrumented(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "stats.go", `package stats

import "expvar"

var hits = expvar.NewInt("hits")

func RecordHit() {
	hits.Add(1)
}

func GetStats() string {
	return "stats"
}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if pkg.InstrumentedCount != 1 {
		t.Errorf("expected 1 instrumented function, got %d", pkg.InstrumentedCount)
	}
}

func TestScannerHTTPHandlerGap(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "server.go", `package server

import "net/http"

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if len(pkg.Gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(pkg.Gaps))
	}
	if !strings.Contains(pkg.Gaps[0], "HTTP handler") {
		t.Errorf("expected gap to be marked as HTTP handler, got: %s", pkg.Gaps[0])
	}
}

func TestScannerErrorReturnGap(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "service.go", `package service

func Process(data string) error {
	return nil
}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if len(pkg.Gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(pkg.Gaps))
	}
	if !strings.Contains(pkg.Gaps[0], "has error return") {
		t.Errorf("expected gap to note error return, got: %s", pkg.Gaps[0])
	}
}

func TestScannerSkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "api.go", `package api

func GetUser() string { return "user" }
`)
	writeGoFile(t, dir, "api_test.go", `package api

func TestGetUser() {}
func BenchmarkGetUser() {}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].TotalExported != 1 {
		t.Errorf("expected 1 exported function (test funcs excluded), got %d", pkgs[0].TotalExported)
	}
}

func TestScannerCustomMetricKeywords(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "app.go", `package app

func RecordMetrics(name string) {
	recordCounter(name)
}

func recordCounter(name string) {}
`)
	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	// RecordMetrics should be detected as instrumented due to "metric" keyword
	pkg := pkgs[0]
	if pkg.InstrumentedCount != 1 {
		t.Errorf("expected 1 instrumented function (custom keyword), got %d", pkg.InstrumentedCount)
	}
}

func TestComputeSummary(t *testing.T) {
	packages := []PackageCoverage{
		{Path: "pkg/a", TotalExported: 10, InstrumentedCount: 8, CoveragePct: 80, Gaps: []string{"Gap1", "Gap2"}},
		{Path: "pkg/b", TotalExported: 5, InstrumentedCount: 1, CoveragePct: 20, Gaps: []string{"Gap3", "Gap4", "Gap5", "Gap6"}},
	}

	summary := ComputeSummary(packages)

	if summary.TotalPackages != 2 {
		t.Errorf("expected 2 packages, got %d", summary.TotalPackages)
	}
	if summary.TotalExported != 15 {
		t.Errorf("expected 15 total exported, got %d", summary.TotalExported)
	}
	if summary.TotalInstrumented != 9 {
		t.Errorf("expected 9 instrumented, got %d", summary.TotalInstrumented)
	}
	if summary.GapCount != 6 {
		t.Errorf("expected 6 gaps, got %d", summary.GapCount)
	}

	expectedPct := 60.0
	if summary.OverallCoveragePct != expectedPct {
		t.Errorf("expected %.1f%% coverage, got %.1f%%", expectedPct, summary.OverallCoveragePct)
	}
	if summary.RiskLevel != "medium" {
		t.Errorf("expected medium risk, got %s", summary.RiskLevel)
	}
}

func TestComputeSummaryEmpty(t *testing.T) {
	summary := ComputeSummary(nil)
	if summary.TotalPackages != 0 {
		t.Errorf("expected 0 packages, got %d", summary.TotalPackages)
	}
	if summary.OverallCoveragePct != 0 {
		t.Errorf("expected 0%% coverage, got %.1f%%", summary.OverallCoveragePct)
	}
	if summary.RiskLevel != "critical" {
		t.Errorf("expected critical risk for empty, got %s", summary.RiskLevel)
	}
}

func TestAssessCoverageRisk(t *testing.T) {
	tests := []struct {
		pct      float64
		gaps     int
		expected string
	}{
		{90, 2, "low"},
		{85, 10, "medium"},
		{60, 5, "medium"},
		{30, 20, "high"},
		{10, 50, "critical"},
		{0, 0, "critical"},
	}

	for _, tt := range tests {
		got := assessCoverageRisk(tt.pct, tt.gaps)
		if got != tt.expected {
			t.Errorf("assessCoverageRisk(%.0f, %d) = %s, want %s", tt.pct, tt.gaps, got, tt.expected)
		}
	}
}

func TestReportGeneration(t *testing.T) {
	packages := []PackageCoverage{
		{Path: "cmd/app", TotalExported: 5, InstrumentedCount: 3, CoveragePct: 60, Gaps: []string{"Main", "Run"}},
	}

	gen := NewMetricsCoverageReportGenerator()
	report := gen.GenerateCoverageReport("test-project", packages)

	if report.Component != "test-project" {
		t.Errorf("expected component 'test-project', got %s", report.Component)
	}
	if report.Summary == nil {
		t.Fatal("summary should not be nil")
	}
	if report.Timestamp.IsZero() {
		t.Errorf("timestamp should not be zero")
	}
	if report.ReportedAt == "" {
		t.Errorf("reported_at should not be empty")
	}
	if len(report.Recommendations) == 0 {
		t.Errorf("expected recommendations to be generated")
	}
}

func TestRenderCoverageMarkdown(t *testing.T) {
	packages := []PackageCoverage{
		{
			Path:                "internal/api",
			TotalExported:       10,
			InstrumentedCount:   6,
			CoveragePct:         60,
			DetectedMetricTypes: []string{"prometheus"},
			Gaps:                []string{"GetUser (has error return)", "DeleteUser (HTTP handler)"},
		},
	}

	gen := NewMetricsCoverageReportGenerator()
	report := gen.GenerateCoverageReport("my-service", packages)
	markdown := gen.RenderCoverageMarkdown(report)

	checks := []struct {
		label    string
		expected string
	}{
		{"title", "# Metrics Coverage Analysis"},
		{"summary section", "## Coverage Summary"},
		{"package breakdown", "## Package Breakdown"},
		{"gaps section", "## Instrumentation Gaps"},
		{"recommendations", "## Recommendations"},
		{"risk level", "Risk Level"},
		{"package name", "internal/api"},
		{"gap entry", "GetUser"},
		{"coverage levels", "Understanding Coverage Levels"},
	}

	for _, c := range checks {
		if !strings.Contains(markdown, c.expected) {
			t.Errorf("markdown should contain %s (%q)", c.label, c.expected)
		}
	}
}

func TestRenderCoverageMarkdownNoGaps(t *testing.T) {
	packages := []PackageCoverage{
		{
			Path:                "pkg/util",
			TotalExported:       3,
			InstrumentedCount:   3,
			CoveragePct:         100,
			DetectedMetricTypes: []string{"expvar"},
		},
	}

	gen := NewMetricsCoverageReportGenerator()
	report := gen.GenerateCoverageReport("util-lib", packages)
	markdown := gen.RenderCoverageMarkdown(report)

	if strings.Contains(markdown, "## Instrumentation Gaps") {
		t.Errorf("markdown should not contain gaps section when there are no gaps")
	}
}

func TestRecommendationsForZeroCoveragePackages(t *testing.T) {
	packages := []PackageCoverage{
		{Path: "pkg/a", TotalExported: 5, InstrumentedCount: 0, CoveragePct: 0, Gaps: []string{"A", "B", "C", "D", "E"}},
		{Path: "pkg/b", TotalExported: 3, InstrumentedCount: 3, CoveragePct: 100},
	}

	gen := NewMetricsCoverageReportGenerator()
	report := gen.GenerateCoverageReport("mixed", packages)

	foundZeroCovRec := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "zero metrics coverage") {
			foundZeroCovRec = true
			break
		}
	}
	if !foundZeroCovRec {
		t.Errorf("expected recommendation about zero-coverage packages")
	}
}

func TestRecommendationsForHTTPHandlerGaps(t *testing.T) {
	packages := []PackageCoverage{
		{Path: "api", TotalExported: 2, InstrumentedCount: 0, CoveragePct: 0,
			Gaps: []string{"ServeIndex (HTTP handler)", "HandleLogin (HTTP handler)"}},
	}

	gen := NewMetricsCoverageReportGenerator()
	report := gen.GenerateCoverageReport("web-app", packages)

	foundHandlerRec := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "HTTP handler") {
			foundHandlerRec = true
			break
		}
	}
	if !foundHandlerRec {
		t.Errorf("expected recommendation about HTTP handler gaps")
	}
}

func TestScannerMultiplePackages(t *testing.T) {
	dir := t.TempDir()

	// Create sub-packages
	pkgA := filepath.Join(dir, "pkga")
	pkgB := filepath.Join(dir, "pkgb")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}

	writeGoFile(t, pkgA, "a.go", `package pkga

func DoA() string { return "a" }
`)
	writeGoFile(t, pkgB, "b.go", `package pkgb

import "expvar"

var counter = expvar.NewInt("counter")

func DoB() string {
	counter.Add(1)
	return "b"
}
`)

	scanner := NewMetricsCoverageScanner(dir)
	pkgs, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	// Find each package
	var foundA, foundB bool
	for _, pkg := range pkgs {
		if strings.Contains(pkg.Path, "pkga") {
			foundA = true
			if pkg.InstrumentedCount != 0 {
				t.Errorf("pkga should have 0 instrumented, got %d", pkg.InstrumentedCount)
			}
		}
		if strings.Contains(pkg.Path, "pkgb") {
			foundB = true
			if pkg.InstrumentedCount != 1 {
				t.Errorf("pkgb should have 1 instrumented, got %d", pkg.InstrumentedCount)
			}
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected both pkga and pkgb, found A=%v B=%v", foundA, foundB)
	}
}
