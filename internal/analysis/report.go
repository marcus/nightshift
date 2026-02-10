package analysis

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

// Report represents a bus-factor analysis report for a codebase or component.
type Report struct {
	Timestamp     time.Time          `json:"timestamp"`
	Component     string             `json:"component"` // "overall", "dir/path", etc
	Metrics       *OwnershipMetrics  `json:"metrics"`
	Contributors  []CommitAuthor     `json:"contributors"`
	Recommendations []string         `json:"recommendations"`
	ReportedAt    string             `json:"reported_at"`
}

// ReportGenerator creates formatted reports from analysis results.
type ReportGenerator struct{}

// NewReportGenerator creates a new report generator.
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// Generate creates a report from authors and metrics.
func (rg *ReportGenerator) Generate(component string, authors []CommitAuthor, metrics *OwnershipMetrics) *Report {
	// Sort authors by commits descending
	sorted := make([]CommitAuthor, len(authors))
	copy(sorted, authors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Commits > sorted[j].Commits
	})

	report := &Report{
		Timestamp:      time.Now(),
		Component:      component,
		Metrics:        metrics,
		Contributors:   sorted,
		ReportedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Recommendations: rg.generateRecommendations(metrics, sorted),
	}

	return report
}

// generateRecommendations creates action items based on metrics.
func (rg *ReportGenerator) generateRecommendations(metrics *OwnershipMetrics, authors []CommitAuthor) []string {
	var recs []string

	switch metrics.RiskLevel {
	case "critical":
		recs = append(recs, "CRITICAL: Knowledge concentrated with 1-2 people. Implement immediate knowledge transfer plan.")
		if len(authors) > 0 {
			recs = append(recs, fmt.Sprintf("Pair %s with team members on code reviews and architecture decisions.", authors[0].Name))
		}

	case "high":
		recs = append(recs, "HIGH RISK: Consider pairing sessions and documentation to distribute knowledge.")
		recs = append(recs, fmt.Sprintf("Target at least %d active contributors to reduce bus factor.", metrics.BusFactor+1))

	case "medium":
		recs = append(recs, "MEDIUM RISK: Healthy but could improve contributor diversity.")
		recs = append(recs, "Encourage junior developers to contribute to this area.")

	case "low":
		recs = append(recs, "GOOD: Knowledge is well-distributed across contributors.")
		recs = append(recs, "Maintain current contributor diversity practices.")
	}

	// Additional recommendations based on metrics
	if metrics.Top1Percent > 0.7 {
		recs = append(recs, fmt.Sprintf("The top contributor (%s) owns %.1f%% of commits. Code review from others is critical.", authors[0].Name, metrics.Top1Percent))
	}

	if metrics.BusFactor <= 2 {
		recs = append(recs, fmt.Sprintf("Bus factor of %d is low. Any single absence is risky.", metrics.BusFactor))
	}

	return recs
}

// RenderMarkdown generates a markdown representation of the report.
func (rg *ReportGenerator) RenderMarkdown(report *Report) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString(fmt.Sprintf("# Bus Factor Analysis - %s\n\n", report.Component))
	buf.WriteString(fmt.Sprintf("*Generated: %s*\n\n", report.ReportedAt))

	// Metrics section
	buf.WriteString("## Ownership Metrics\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	fmt.Fprintf(&buf, "| Risk Level | **%s** |\n", report.Metrics.RiskLevel)
	fmt.Fprintf(&buf, "| Bus Factor | %d contributors |\n", report.Metrics.BusFactor)
	fmt.Fprintf(&buf, "| Total Contributors | %d |\n", report.Metrics.TotalContributors)
	fmt.Fprintf(&buf, "| Top 1 Contributor | %.1f%% |\n", report.Metrics.Top1Percent)
	fmt.Fprintf(&buf, "| Top 3 Contributors | %.1f%% |\n", report.Metrics.Top3Percent)
	fmt.Fprintf(&buf, "| Top 5 Contributors | %.1f%% |\n", report.Metrics.Top5Percent)
	fmt.Fprintf(&buf, "| Herfindahl Index | %.3f (0=diverse, 1=concentrated) |\n", report.Metrics.HerfindahlIndex)
	fmt.Fprintf(&buf, "| Gini Coefficient | %.3f (0=equal, 1=unequal) |\n\n", report.Metrics.GiniCoefficient)

	// Contributors section
	if len(report.Contributors) > 0 {
		buf.WriteString("## Top Contributors\n\n")
		total := 0
		for _, author := range report.Contributors {
			total += author.Commits
		}

		for i, author := range report.Contributors {
			if i >= 10 { // Show top 10
				fmt.Fprintf(&buf, "... and %d more contributors\n", len(report.Contributors)-10)
				break
			}
			pct := float64(author.Commits) * 100 / float64(total)
			bar := rg.progressBar(int(pct / 5))
			fmt.Fprintf(&buf, "%d. **%s** <%s> - %d commits (%.1f%%) %s\n", i+1, author.Name, author.Email, author.Commits, pct, bar)
		}
		buf.WriteString("\n")
	}

	// Recommendations section
	if len(report.Recommendations) > 0 {
		buf.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			if len(rec) > 0 && (rec[0:4] == "GOOD" || rec[0:4] == "HIGH" || rec[0:4] == "CRIT" || rec[0:4] == "MEDI") {
				// High priority items
				fmt.Fprintf(&buf, "**%s**\n\n", rec)
			} else {
				fmt.Fprintf(&buf, "- %s\n", rec)
			}
		}
		buf.WriteString("\n")
	}

	// Risk explanation
	buf.WriteString("## Understanding the Risk Level\n\n")
	buf.WriteString("- **Low**: Knowledge well-distributed. Multiple people understand each area.\n")
	buf.WriteString("- **Medium**: Healthy but could improve. Some areas have limited coverage.\n")
	buf.WriteString("- **High**: Concerning. Critical areas depend on few people.\n")
	buf.WriteString("- **Critical**: Urgent. Single person dependencies exist.\n\n")

	buf.WriteString("## Metrics Explanation\n\n")
	buf.WriteString("- **Bus Factor**: Minimum number of contributors whose loss would significantly impact the project.\n")
	buf.WriteString("- **Herfindahl Index**: Market concentration index (adapted for contributor concentration).\n")
	buf.WriteString("- **Gini Coefficient**: Income inequality measure, adapted for knowledge distribution.\n")
	buf.WriteString("- **Top N %**: Cumulative ownership percentage of top N contributors.\n")

	return buf.String()
}

// progressBar creates a simple ASCII progress bar.
func (rg *ReportGenerator) progressBar(filled int) string {
	if filled > 20 {
		filled = 20
	}
	var bar bytes.Buffer
	for i := 0; i < 20; i++ {
		if i < filled {
			bar.WriteRune('█')
		} else {
			bar.WriteRune('░')
		}
	}
	return fmt.Sprintf("[%s]", bar.String())
}
