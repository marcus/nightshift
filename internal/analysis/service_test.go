package analysis

import (
	"strings"
	"testing"
)

func TestCalculateServiceMetricsEmpty(t *testing.T) {
	results := CalculateServiceMetrics(nil)
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}

func TestCalculateServiceMetricsSinglePackage(t *testing.T) {
	pkgs := []PackageInfo{
		{Name: "main", Path: "cmd/app", Files: 3, LOC: 500, ExportedSyms: 5, InternalRefs: 20},
	}

	results := CalculateServiceMetrics(pkgs)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	m := results[0].Metrics
	// Single package with no imports = 0 coupling
	if m.CouplingScore != 0 {
		t.Errorf("expected 0 coupling for isolated package, got %.2f", m.CouplingScore)
	}
	// 500 LOC is 100% of project = size score should be 1.0
	if m.SizeScore != 1.0 {
		t.Errorf("expected size score 1.0 for only package, got %.2f", m.SizeScore)
	}
	// Large, cohesive, isolated package = extract candidate
	if m.Recommendation != "extract" {
		t.Errorf("expected 'extract' recommendation for large cohesive package, got %s", m.Recommendation)
	}
}

func TestCalculateServiceMetricsMultiplePackages(t *testing.T) {
	pkgs := []PackageInfo{
		{
			Name: "api", Path: "internal/api", Files: 10, LOC: 2000,
			Imports:      []string{"internal/db", "internal/auth"},
			ExportedSyms: 30, InternalRefs: 100,
		},
		{
			Name: "db", Path: "internal/db", Files: 5, LOC: 800,
			ImportedBy:   []string{"internal/api"},
			ExportedSyms: 15, InternalRefs: 40,
		},
		{
			Name: "auth", Path: "internal/auth", Files: 3, LOC: 300,
			ImportedBy:   []string{"internal/api"},
			ExportedSyms: 8, InternalRefs: 25,
		},
		{
			Name: "util", Path: "internal/util", Files: 1, LOC: 50,
			ExportedSyms: 3, InternalRefs: 0,
		},
	}

	results := CalculateServiceMetrics(pkgs)
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// api package has 2 efferent deps = some coupling
	for _, r := range results {
		if r.Package.Path == "internal/api" {
			if r.Metrics.EfferentCoupling != 2 {
				t.Errorf("expected api efferent=2, got %d", r.Metrics.EfferentCoupling)
			}
		}
		if r.Package.Path == "internal/db" {
			if r.Metrics.AfferentCoupling != 1 {
				t.Errorf("expected db afferent=1, got %d", r.Metrics.AfferentCoupling)
			}
		}
	}
}

func TestComputeCouplingScore(t *testing.T) {
	tests := []struct {
		afferent int
		efferent int
		wantZero bool
		wantMax  bool
	}{
		{0, 0, true, false},
		{5, 5, false, true},  // 10 total = 1.0
		{2, 1, false, false}, // 3 total = 0.3
	}

	for _, tt := range tests {
		score := computeCouplingScore(tt.afferent, tt.efferent)
		if tt.wantZero && score != 0 {
			t.Errorf("coupling(%d,%d) = %.2f, want 0", tt.afferent, tt.efferent, score)
		}
		if tt.wantMax && score != 1.0 {
			t.Errorf("coupling(%d,%d) = %.2f, want 1.0", tt.afferent, tt.efferent, score)
		}
		if score < 0 || score > 1 {
			t.Errorf("coupling(%d,%d) = %.2f, out of [0,1] range", tt.afferent, tt.efferent, score)
		}
	}
}

func TestComputeCohesionScore(t *testing.T) {
	// No exports = high cohesion (1.0)
	pkg1 := PackageInfo{ExportedSyms: 0, InternalRefs: 10}
	if s := computeCohesionScore(pkg1); s != 1.0 {
		t.Errorf("no exports should give cohesion 1.0, got %.2f", s)
	}

	// Many refs per export = high cohesion
	pkg2 := PackageInfo{ExportedSyms: 5, InternalRefs: 50}
	s2 := computeCohesionScore(pkg2)
	if s2 < 0.9 {
		t.Errorf("10 refs/export should give high cohesion, got %.2f", s2)
	}

	// Few refs per export = low cohesion
	pkg3 := PackageInfo{ExportedSyms: 20, InternalRefs: 5}
	s3 := computeCohesionScore(pkg3)
	if s3 > 0.1 {
		t.Errorf("0.25 refs/export should give low cohesion, got %.2f", s3)
	}
}

func TestComputeSizeScore(t *testing.T) {
	// 0 total LOC
	if s := computeSizeScore(100, 0); s != 0 {
		t.Errorf("size with 0 total should be 0, got %.2f", s)
	}

	// Package is 20% of project = score 1.0
	if s := computeSizeScore(200, 1000); s != 1.0 {
		t.Errorf("20%% of project should be 1.0, got %.2f", s)
	}

	// Package is 10% of project = score 0.5
	s := computeSizeScore(100, 1000)
	if s < 0.49 || s > 0.51 {
		t.Errorf("10%% of project should be ~0.5, got %.2f", s)
	}
}

func TestComputeChurnCorrelation(t *testing.T) {
	// No co-changes
	pkg1 := PackageInfo{}
	if s := computeChurnCorrelation(pkg1, 5); s != 0 {
		t.Errorf("no co-changes should be 0, got %.2f", s)
	}

	// Single package in project
	if s := computeChurnCorrelation(pkg1, 1); s != 0 {
		t.Errorf("single package should be 0, got %.2f", s)
	}

	// Co-changes with all other packages
	pkg2 := PackageInfo{CoChanges: []string{"a", "b", "c", "d"}}
	if s := computeChurnCorrelation(pkg2, 5); s != 1.0 {
		t.Errorf("co-change with all should be 1.0, got %.2f", s)
	}
}

func TestComputeExtractWorthiness(t *testing.T) {
	// Perfect extraction candidate: high cohesion, large, low coupling, low churn
	score := computeExtractWorthiness(0, 1.0, 1.0, 0)
	if score < 0.9 {
		t.Errorf("ideal candidate should score >0.9, got %.3f", score)
	}

	// Poor candidate: high coupling, low cohesion, small, high churn
	score2 := computeExtractWorthiness(1.0, 0, 0, 1.0)
	if score2 > 0.1 {
		t.Errorf("poor candidate should score <0.1, got %.3f", score2)
	}
}

func TestRecommendAction(t *testing.T) {
	// Small package = merge
	rec := recommendAction(0.3, 0.1, 0.1, 0.05, PackageInfo{Files: 1, LOC: 50})
	if rec != "merge" {
		t.Errorf("small package should get 'merge', got %s", rec)
	}

	// High extract score with good metrics = extract
	rec2 := recommendAction(0.7, 0.2, 0.8, 0.3, PackageInfo{Files: 10, LOC: 1000})
	if rec2 != "extract" {
		t.Errorf("large cohesive package should get 'extract', got %s", rec2)
	}

	// Default = keep
	rec3 := recommendAction(0.4, 0.5, 0.3, 0.15, PackageInfo{Files: 5, LOC: 300})
	if rec3 != "keep" {
		t.Errorf("average package should get 'keep', got %s", rec3)
	}
}

func TestServiceMetricsString(t *testing.T) {
	m := ServiceMetrics{
		CouplingScore:     0.3,
		CohesionScore:     0.8,
		SizeScore:         0.5,
		ChurnCorrelation:  0.1,
		ExtractWorthiness: 0.650,
		Recommendation:    "extract",
	}

	str := m.String()
	if !strings.Contains(str, "Coupling: 0.30") {
		t.Errorf("string should contain coupling, got %s", str)
	}
	if !strings.Contains(str, "Rec: extract") {
		t.Errorf("string should contain recommendation, got %s", str)
	}
}

func TestServiceReportGenerate(t *testing.T) {
	analyses := []PackageAnalysis{
		{
			Package: PackageInfo{Name: "api", Path: "internal/api", Files: 5, LOC: 500},
			Metrics: ServiceMetrics{ExtractWorthiness: 0.7, Recommendation: "extract"},
		},
		{
			Package: PackageInfo{Name: "util", Path: "internal/util", Files: 1, LOC: 30},
			Metrics: ServiceMetrics{ExtractWorthiness: 0.2, Recommendation: "merge"},
		},
	}

	gen := NewServiceReportGenerator()
	report := gen.Generate("test-project", analyses)

	if report.Component != "test-project" {
		t.Errorf("expected component 'test-project', got %s", report.Component)
	}
	if report.TotalPackages != 2 {
		t.Errorf("expected 2 packages, got %d", report.TotalPackages)
	}
	if report.TotalLOC != 530 {
		t.Errorf("expected 530 total LOC, got %d", report.TotalLOC)
	}
	// Should be sorted by extract-worthiness descending
	if report.Packages[0].Package.Path != "internal/api" {
		t.Errorf("expected api first (higher score), got %s", report.Packages[0].Package.Path)
	}
	if len(report.Recommendations) == 0 {
		t.Errorf("expected recommendations to be generated")
	}
}

func TestServiceReportRenderMarkdown(t *testing.T) {
	analyses := []PackageAnalysis{
		{
			Package: PackageInfo{
				Name: "api", Path: "internal/api", Files: 5, LOC: 500,
				Imports: []string{"internal/db"}, ImportedBy: []string{"cmd/app"},
			},
			Metrics: ServiceMetrics{
				CouplingScore: 0.3, CohesionScore: 0.8, SizeScore: 0.5,
				ExtractWorthiness: 0.7, Recommendation: "extract",
			},
		},
	}

	gen := NewServiceReportGenerator()
	report := gen.Generate("test-project", analyses)
	md := gen.RenderMarkdown(report)

	if !strings.Contains(md, "# Service Boundary Analysis") {
		t.Errorf("markdown should contain title")
	}
	if !strings.Contains(md, "## Packages by Extract-Worthiness") {
		t.Errorf("markdown should contain packages section")
	}
	if !strings.Contains(md, "## Import Graph") {
		t.Errorf("markdown should contain import graph")
	}
	if !strings.Contains(md, "## Recommendations") {
		t.Errorf("markdown should contain recommendations")
	}
	if !strings.Contains(md, "internal/api") {
		t.Errorf("markdown should contain package path")
	}
}

func TestServiceReportNoExtractOrMerge(t *testing.T) {
	analyses := []PackageAnalysis{
		{
			Package: PackageInfo{Name: "pkg", Path: "internal/pkg", Files: 3, LOC: 200},
			Metrics: ServiceMetrics{ExtractWorthiness: 0.4, Recommendation: "keep"},
		},
	}

	gen := NewServiceReportGenerator()
	report := gen.Generate("test", analyses)

	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "healthy") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected healthy message when no extract/merge candidates, got %v", report.Recommendations)
	}
}

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "b")
	if len(s) != 2 {
		t.Errorf("appendUnique should not duplicate, got %v", s)
	}
	s = appendUnique(s, "c")
	if len(s) != 3 {
		t.Errorf("appendUnique should add new, got %v", s)
	}
}
