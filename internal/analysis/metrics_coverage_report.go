package analysis

import (
	"bytes"
	"fmt"
	"time"
)

// MetricsCoverageReport represents a complete metrics instrumentation coverage report.
type MetricsCoverageReport struct {
	Timestamp       time.Time         `json:"timestamp"`
	Component       string            `json:"component"`
	Summary         *CoverageSummary  `json:"summary"`
	Packages        []PackageCoverage `json:"packages"`
	Recommendations []string          `json:"recommendations"`
	ReportedAt      string            `json:"reported_at"`
}

// MetricsCoverageReportGenerator creates formatted reports from coverage scan results.
type MetricsCoverageReportGenerator struct{}

// NewMetricsCoverageReportGenerator creates a new report generator.
func NewMetricsCoverageReportGenerator() *MetricsCoverageReportGenerator {
	return &MetricsCoverageReportGenerator{}
}

// GenerateCoverageReport creates a report from scanned packages.
func (g *MetricsCoverageReportGenerator) GenerateCoverageReport(component string, packages []PackageCoverage) *MetricsCoverageReport {
	summary := ComputeSummary(packages)

	return &MetricsCoverageReport{
		Timestamp:       time.Now(),
		Component:       component,
		Summary:         summary,
		Packages:        packages,
		Recommendations: g.generateRecommendations(summary, packages),
		ReportedAt:      time.Now().Format("2006-01-02 15:04:05"),
	}
}

// generateRecommendations creates action items based on coverage.
func (g *MetricsCoverageReportGenerator) generateRecommendations(summary *CoverageSummary, packages []PackageCoverage) []string {
	var recs []string

	switch summary.RiskLevel {
	case "critical":
		recs = append(recs, "CRITICAL: Less than 20% of exported functions have metrics instrumentation. Implement observability as a priority.")
		recs = append(recs, "Start by adding metrics to HTTP handlers and public API entry points.")
	case "high":
		recs = append(recs, "HIGH RISK: Many exported functions lack metrics. Focus on error paths and handler functions.")
		recs = append(recs, "Consider adding Prometheus counters or OpenTelemetry spans to critical paths.")
	case "medium":
		recs = append(recs, "MEDIUM RISK: Moderate metrics coverage. Identify high-traffic paths missing instrumentation.")
		recs = append(recs, "Review gap list and prioritize HTTP handlers and error-returning functions.")
	case "low":
		recs = append(recs, "GOOD: Metrics coverage is healthy across the codebase.")
		recs = append(recs, "Maintain current instrumentation practices and review new code for coverage.")
	}

	// Find packages with zero coverage
	var zeroCov []string
	for _, pkg := range packages {
		if pkg.InstrumentedCount == 0 && pkg.TotalExported > 0 {
			zeroCov = append(zeroCov, pkg.Path)
		}
	}
	if len(zeroCov) > 0 {
		if len(zeroCov) <= 5 {
			recs = append(recs, fmt.Sprintf("Packages with zero metrics coverage: %v", zeroCov))
		} else {
			recs = append(recs, fmt.Sprintf("%d packages have zero metrics coverage. Prioritize those with HTTP handlers.", len(zeroCov)))
		}
	}

	// Count handler gaps
	handlerGaps := 0
	for _, pkg := range packages {
		for _, gap := range pkg.Gaps {
			if contains(gap, "HTTP handler") {
				handlerGaps++
			}
		}
	}
	if handlerGaps > 0 {
		recs = append(recs, fmt.Sprintf("%d HTTP handler(s) lack metrics instrumentation. These are high-priority gaps.", handlerGaps))
	}

	return recs
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
}

// RenderCoverageMarkdown generates a markdown representation of the coverage report.
func (g *MetricsCoverageReportGenerator) RenderCoverageMarkdown(report *MetricsCoverageReport) string {
	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "# Metrics Coverage Analysis - %s\n\n", report.Component)
	fmt.Fprintf(&buf, "*Generated: %s*\n\n", report.ReportedAt)

	// Summary section
	buf.WriteString("## Coverage Summary\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	fmt.Fprintf(&buf, "| Risk Level | **%s** |\n", report.Summary.RiskLevel)
	fmt.Fprintf(&buf, "| Overall Coverage | %.1f%% |\n", report.Summary.OverallCoveragePct)
	fmt.Fprintf(&buf, "| Total Packages | %d |\n", report.Summary.TotalPackages)
	fmt.Fprintf(&buf, "| Exported Functions | %d |\n", report.Summary.TotalExported)
	fmt.Fprintf(&buf, "| Instrumented | %d |\n", report.Summary.TotalInstrumented)
	fmt.Fprintf(&buf, "| Gaps | %d |\n\n", report.Summary.GapCount)

	// Per-package breakdown
	if len(report.Packages) > 0 {
		buf.WriteString("## Package Breakdown\n\n")
		buf.WriteString("| Package | Exported | Instrumented | Coverage | Metric Types |\n")
		buf.WriteString("|---------|----------|-------------|----------|-------------|\n")
		for _, pkg := range report.Packages {
			types := "-"
			if len(pkg.DetectedMetricTypes) > 0 {
				types = fmt.Sprintf("%v", pkg.DetectedMetricTypes)
			}
			fmt.Fprintf(&buf, "| %s | %d | %d | %.1f%% | %s |\n",
				pkg.Path, pkg.TotalExported, pkg.InstrumentedCount, pkg.CoveragePct, types)
		}
		buf.WriteString("\n")
	}

	// Gaps section
	totalGaps := 0
	for _, pkg := range report.Packages {
		totalGaps += len(pkg.Gaps)
	}
	if totalGaps > 0 {
		buf.WriteString("## Instrumentation Gaps\n\n")
		for _, pkg := range report.Packages {
			if len(pkg.Gaps) == 0 {
				continue
			}
			fmt.Fprintf(&buf, "### %s\n\n", pkg.Path)
			for _, gap := range pkg.Gaps {
				fmt.Fprintf(&buf, "- `%s`\n", gap)
			}
			buf.WriteString("\n")
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		buf.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			isHighPriority := len(rec) > 0 && (rec[0] == 'G' || rec[0] == 'H' || rec[0] == 'C' || rec[0] == 'M') &&
				(bytes.HasPrefix([]byte(rec), []byte("GOOD")) ||
					bytes.HasPrefix([]byte(rec), []byte("HIGH")) ||
					bytes.HasPrefix([]byte(rec), []byte("CRITICAL")) ||
					bytes.HasPrefix([]byte(rec), []byte("MEDIUM")))

			if isHighPriority {
				fmt.Fprintf(&buf, "**%s**\n\n", rec)
			} else {
				fmt.Fprintf(&buf, "- %s\n", rec)
			}
		}
		buf.WriteString("\n")
	}

	// Coverage explanation
	buf.WriteString("## Understanding Coverage Levels\n\n")
	buf.WriteString("- **Low risk** (>=80%): Well-instrumented codebase with few gaps.\n")
	buf.WriteString("- **Medium risk** (50-80%): Moderate coverage, key paths may lack metrics.\n")
	buf.WriteString("- **High risk** (20-50%): Significant gaps in observability.\n")
	buf.WriteString("- **Critical** (<20%): Minimal metrics instrumentation. Incidents will lack data.\n")

	return buf.String()
}
