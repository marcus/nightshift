package analysis

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

// ServiceReport represents a service boundary analysis report.
type ServiceReport struct {
	Timestamp       time.Time         `json:"timestamp"`
	Component       string            `json:"component"`
	Packages        []PackageAnalysis `json:"packages"`
	Recommendations []string          `json:"recommendations"`
	ReportedAt      string            `json:"reported_at"`
	TotalPackages   int               `json:"total_packages"`
	TotalLOC        int               `json:"total_loc"`
}

// ServiceReportGenerator creates formatted service boundary reports.
type ServiceReportGenerator struct{}

// NewServiceReportGenerator creates a new report generator.
func NewServiceReportGenerator() *ServiceReportGenerator {
	return &ServiceReportGenerator{}
}

// Generate creates a service report from package analysis results.
func (rg *ServiceReportGenerator) Generate(component string, analyses []PackageAnalysis) *ServiceReport {
	// Sort by extract-worthiness descending
	sorted := make([]PackageAnalysis, len(analyses))
	copy(sorted, analyses)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Metrics.ExtractWorthiness > sorted[j].Metrics.ExtractWorthiness
	})

	totalLOC := 0
	for _, a := range sorted {
		totalLOC += a.Package.LOC
	}

	report := &ServiceReport{
		Timestamp:       time.Now(),
		Component:       component,
		Packages:        sorted,
		ReportedAt:      time.Now().Format("2006-01-02 15:04:05"),
		TotalPackages:   len(sorted),
		TotalLOC:        totalLOC,
		Recommendations: rg.generateRecommendations(sorted),
	}

	return report
}

func (rg *ServiceReportGenerator) generateRecommendations(analyses []PackageAnalysis) []string {
	var recs []string

	var extractCount, mergeCount int
	for _, a := range analyses {
		switch a.Metrics.Recommendation {
		case "extract":
			extractCount++
		case "merge":
			mergeCount++
		}
	}

	if extractCount > 0 {
		recs = append(recs, fmt.Sprintf("Found %d package(s) that are strong candidates for service extraction.", extractCount))
		for _, a := range analyses {
			if a.Metrics.Recommendation == "extract" {
				recs = append(recs, fmt.Sprintf("  - %s (%d LOC, %.0f%% cohesion, extract score: %.3f)",
					a.Package.Path, a.Package.LOC, a.Metrics.CohesionScore*100, a.Metrics.ExtractWorthiness))
			}
		}
	}

	if mergeCount > 0 {
		recs = append(recs, fmt.Sprintf("Found %d package(s) that may benefit from merging into a neighbor.", mergeCount))
		for _, a := range analyses {
			if a.Metrics.Recommendation == "merge" {
				target := "a related package"
				if len(a.Package.Imports) > 0 {
					target = a.Package.Imports[0]
				}
				recs = append(recs, fmt.Sprintf("  - %s (%d LOC, %d files) could merge into %s",
					a.Package.Path, a.Package.LOC, a.Package.Files, target))
			}
		}
	}

	if extractCount == 0 && mergeCount == 0 {
		recs = append(recs, "Package structure looks healthy. No strong extraction or merge candidates found.")
	}

	// Look for high coupling packages
	for _, a := range analyses {
		if a.Metrics.CouplingScore > 0.7 {
			recs = append(recs, fmt.Sprintf("High coupling: %s has %d afferent + %d efferent dependencies. Consider interface boundaries.",
				a.Package.Path, a.Metrics.AfferentCoupling, a.Metrics.EfferentCoupling))
		}
	}

	// Look for high churn correlation
	for _, a := range analyses {
		if a.Metrics.ChurnCorrelation > 0.5 {
			recs = append(recs, fmt.Sprintf("High change coupling: %s frequently co-changes with %d other packages. This suggests hidden dependencies.",
				a.Package.Path, len(a.Package.CoChanges)))
		}
	}

	return recs
}

// RenderMarkdown generates a markdown representation of the service report.
func (rg *ServiceReportGenerator) RenderMarkdown(report *ServiceReport) string {
	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "# Service Boundary Analysis - %s\n\n", report.Component)
	fmt.Fprintf(&buf, "*Generated: %s*\n\n", report.ReportedAt)

	// Summary
	buf.WriteString("## Summary\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	fmt.Fprintf(&buf, "| Total Packages | %d |\n", report.TotalPackages)
	fmt.Fprintf(&buf, "| Total LOC | %d |\n\n", report.TotalLOC)

	// Package details table
	if len(report.Packages) > 0 {
		buf.WriteString("## Packages by Extract-Worthiness\n\n")
		buf.WriteString("| Package | LOC | Files | Coupling | Cohesion | Size | Churn | Score | Rec |\n")
		buf.WriteString("|---------|-----|-------|----------|----------|------|-------|-------|-----|\n")

		for _, a := range report.Packages {
			fmt.Fprintf(&buf, "| %s | %d | %d | %.2f | %.2f | %.2f | %.2f | %.3f | **%s** |\n",
				a.Package.Path,
				a.Package.LOC,
				a.Package.Files,
				a.Metrics.CouplingScore,
				a.Metrics.CohesionScore,
				a.Metrics.SizeScore,
				a.Metrics.ChurnCorrelation,
				a.Metrics.ExtractWorthiness,
				a.Metrics.Recommendation,
			)
		}
		buf.WriteString("\n")
	}

	// Import graph
	hasImports := false
	for _, a := range report.Packages {
		if len(a.Package.Imports) > 0 || len(a.Package.ImportedBy) > 0 {
			hasImports = true
			break
		}
	}
	if hasImports {
		buf.WriteString("## Import Graph\n\n")
		for _, a := range report.Packages {
			if len(a.Package.Imports) == 0 && len(a.Package.ImportedBy) == 0 {
				continue
			}
			fmt.Fprintf(&buf, "### %s\n", a.Package.Path)
			if len(a.Package.Imports) > 0 {
				fmt.Fprintf(&buf, "- **Depends on**: %s\n", joinPaths(a.Package.Imports))
			}
			if len(a.Package.ImportedBy) > 0 {
				fmt.Fprintf(&buf, "- **Used by**: %s\n", joinPaths(a.Package.ImportedBy))
			}
			buf.WriteString("\n")
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		buf.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			fmt.Fprintf(&buf, "- %s\n", rec)
		}
		buf.WriteString("\n")
	}

	// Scoring explanation
	buf.WriteString("## Scoring Guide\n\n")
	buf.WriteString("- **Coupling**: Total import connections relative to max (0=isolated, 1=highly coupled)\n")
	buf.WriteString("- **Cohesion**: Internal references vs exports (0=low cohesion, 1=high cohesion)\n")
	buf.WriteString("- **Size**: LOC relative to project (0=tiny, 1=very large)\n")
	buf.WriteString("- **Churn**: Co-change frequency with other packages (0=independent, 1=always co-changed)\n")
	buf.WriteString("- **Score**: Composite extract-worthiness (higher = stronger extraction candidate)\n")
	buf.WriteString("- **Rec**: extract (good service candidate), merge (too small), keep (fine as-is)\n")

	return buf.String()
}

func joinPaths(paths []string) string {
	var buf bytes.Buffer
	for i, p := range paths {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "`%s`", p)
	}
	return buf.String()
}
