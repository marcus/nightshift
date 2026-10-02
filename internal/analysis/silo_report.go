package analysis

import (
	"bytes"
	"fmt"
	"time"
)

// SiloReportGenerator creates formatted silo reports.
type SiloReportGenerator struct{}

// NewSiloReportGenerator creates a new silo report generator.
func NewSiloReportGenerator() *SiloReportGenerator {
	return &SiloReportGenerator{}
}

// Generate creates a SiloReport from silo entries.
func (sg *SiloReportGenerator) Generate(repoPath string, depth int, entries []SiloEntry) *SiloReport {
	report := &SiloReport{
		Timestamp:  time.Now(),
		RepoPath:   repoPath,
		Depth:      depth,
		Entries:    entries,
		TotalDirs:  len(entries),
		ReportedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, e := range entries {
		switch e.RiskLevel {
		case "critical":
			report.CriticalCount++
		case "high":
			report.HighCount++
		}
	}

	report.Recommendations = sg.generateRecommendations(entries)
	return report
}

// generateRecommendations creates action items based on silo analysis.
func (sg *SiloReportGenerator) generateRecommendations(entries []SiloEntry) []string {
	var recs []string

	var criticalDirs, highDirs []string
	for _, e := range entries {
		switch e.RiskLevel {
		case "critical":
			criticalDirs = append(criticalDirs, e.Directory)
		case "high":
			highDirs = append(highDirs, e.Directory)
		}
	}

	if len(criticalDirs) > 0 {
		recs = append(recs, fmt.Sprintf("CRITICAL: %d directories have single-person knowledge silos. Prioritize knowledge transfer immediately.", len(criticalDirs)))
		for _, dir := range criticalDirs {
			for _, e := range entries {
				if e.Directory == dir && len(e.TopContributors) > 0 {
					recs = append(recs, fmt.Sprintf("  - %s: dominated by %s (%d commits). Schedule pairing sessions.", dir, e.TopContributors[0].Name, e.TopContributors[0].Commits))
					break
				}
			}
		}
	}

	if len(highDirs) > 0 {
		recs = append(recs, fmt.Sprintf("HIGH RISK: %d directories have limited contributor diversity. Encourage cross-team contributions.", len(highDirs)))
	}

	if len(criticalDirs) == 0 && len(highDirs) == 0 {
		recs = append(recs, "GOOD: No critical knowledge silos detected. Maintain current collaboration practices.")
	}

	if len(entries) > 0 {
		recs = append(recs, "Consider rotating code review assignments to spread knowledge across more team members.")
		recs = append(recs, "Document architectural decisions in high-risk directories to reduce person-dependent knowledge.")
	}

	return recs
}

// RenderMarkdown generates a markdown representation of the silo report.
func (sg *SiloReportGenerator) RenderMarkdown(report *SiloReport) string {
	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "# Knowledge Silo Analysis\n\n")
	fmt.Fprintf(&buf, "*Generated: %s*\n\n", report.ReportedAt)

	// Summary
	buf.WriteString("## Summary\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	fmt.Fprintf(&buf, "| Directories Analyzed | %d |\n", report.TotalDirs)
	fmt.Fprintf(&buf, "| Critical Silos | %d |\n", report.CriticalCount)
	fmt.Fprintf(&buf, "| High Risk Silos | %d |\n", report.HighCount)
	fmt.Fprintf(&buf, "| Analysis Depth | %d |\n\n", report.Depth)

	// Silo table
	if len(report.Entries) > 0 {
		buf.WriteString("## Directory Silo Risk\n\n")
		buf.WriteString("| Directory | Top Contributor | Silo Score | Contributors | Commits | Risk |\n")
		buf.WriteString("|-----------|----------------|------------|--------------|---------|------|\n")

		for _, entry := range report.Entries {
			topName := "-"
			topPct := 0.0
			if len(entry.TopContributors) > 0 {
				topName = entry.TopContributors[0].Name
				if entry.TotalCommits > 0 {
					topPct = float64(entry.TopContributors[0].Commits) * 100 / float64(entry.TotalCommits)
				}
			}

			fmt.Fprintf(&buf, "| %s | %s (%.0f%%) | %.2f | %d | %d | **%s** |\n",
				entry.Directory,
				topName,
				topPct,
				entry.SiloScore,
				entry.ContributorCount,
				entry.TotalCommits,
				entry.RiskLevel,
			)
		}
		buf.WriteString("\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		buf.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			if len(rec) > 0 && (rec[0] == 'C' || rec[0] == 'H' || rec[0] == 'G') &&
				(bytes.HasPrefix([]byte(rec), []byte("CRITICAL")) ||
					bytes.HasPrefix([]byte(rec), []byte("HIGH")) ||
					bytes.HasPrefix([]byte(rec), []byte("GOOD"))) {
				fmt.Fprintf(&buf, "**%s**\n\n", rec)
			} else {
				fmt.Fprintf(&buf, "- %s\n", rec)
			}
		}
		buf.WriteString("\n")
	}

	// Risk explanation
	buf.WriteString("## Understanding Silo Risk\n\n")
	buf.WriteString("- **Critical**: Single person owns >80% of commits, or only one contributor.\n")
	buf.WriteString("- **High**: Top contributor owns >60%, or only two contributors.\n")
	buf.WriteString("- **Medium**: Top contributor owns >40%, or three or fewer contributors.\n")
	buf.WriteString("- **Low**: Knowledge is well-distributed across multiple contributors.\n")

	return buf.String()
}
