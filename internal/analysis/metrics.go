package analysis

import (
	"fmt"
	"math"
	"sort"
)

// OwnershipMetrics contains calculated bus-factor metrics.
type OwnershipMetrics struct {
	// HerfindahlIndex measures concentration (0-1 scale, 1 = max concentration)
	HerfindahlIndex float64 `json:"herfindahl_index"`
	// GiniCoefficient measures inequality (0 = equal, 1 = max inequality)
	GiniCoefficient float64 `json:"gini_coefficient"`
	// Top 1, 3, 5 ownership percentages
	Top1Percent float64 `json:"top_1_percent"`
	Top3Percent float64 `json:"top_3_percent"`
	Top5Percent float64 `json:"top_5_percent"`
	// Risk level assessment
	RiskLevel string `json:"risk_level"` // low, medium, high, critical
	// Number of contributors
	TotalContributors int `json:"total_contributors"`
	// Minimum contributors needed to reach 50% of commits
	BusFactor int `json:"bus_factor"`
}

// CalculateMetrics computes ownership concentration metrics from authors.
func CalculateMetrics(authors []CommitAuthor) *OwnershipMetrics {
	if len(authors) == 0 {
		return &OwnershipMetrics{RiskLevel: "unknown", TotalContributors: 0}
	}

	// Sort by commits descending
	sorted := make([]CommitAuthor, len(authors))
	copy(sorted, authors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Commits > sorted[j].Commits
	})

	// Total commits
	totalCommits := 0
	for _, author := range sorted {
		totalCommits += author.Commits
	}

	// Calculate metrics
	hIndex := calculateHerfindahl(sorted, totalCommits)
	gini := calculateGini(sorted, totalCommits)
	top1 := calculateTopN(sorted, totalCommits, 1)
	top3 := calculateTopN(sorted, totalCommits, 3)
	top5 := calculateTopN(sorted, totalCommits, 5)
	busFactor := calculateBusFactor(sorted, totalCommits)
	riskLevel := assessRiskLevel(hIndex, top1, len(authors))

	return &OwnershipMetrics{
		HerfindahlIndex:   hIndex,
		GiniCoefficient:   gini,
		Top1Percent:       top1 * 100,
		Top3Percent:       top3 * 100,
		Top5Percent:       top5 * 100,
		RiskLevel:         riskLevel,
		TotalContributors: len(authors),
		BusFactor:         busFactor,
	}
}

// calculateHerfindahl computes the Herfindahl-Hirschman Index (HHI).
// HHI = sum of (market_share)^2 for each participant
// Ranges from 1/n to 1, where 1 = perfect monopoly
func calculateHerfindahl(authors []CommitAuthor, total int) float64 {
	if total == 0 || len(authors) == 0 {
		return 0
	}

	hhi := 0.0
	for _, author := range authors {
		share := float64(author.Commits) / float64(total)
		hhi += share * share
	}

	// Normalize to 0-1 scale
	n := float64(len(authors))
	minHHI := 1.0 / n
	if minHHI >= 1.0 {
		// Single author case
		return 1.0
	}
	normalized := (hhi - minHHI) / (1.0 - minHHI)
	return math.Min(math.Max(normalized, 0), 1)
}

// calculateGini computes the Gini coefficient for wealth inequality.
// 0 = perfect equality, 1 = perfect inequality
func calculateGini(authors []CommitAuthor, total int) float64 {
	if total == 0 || len(authors) == 0 {
		return 0
	}

	// Calculate cumulative shares
	n := len(authors)
	cumSum := 0.0
	for i := range authors {
		cumSum += float64((i + 1) * authors[i].Commits)
	}

	// Gini formula: (2 * sum) / (n * total) - (n + 1) / n
	gini := (2.0*cumSum)/(float64(n)*float64(total)) - (float64(n)+1.0)/float64(n)
	return math.Min(math.Max(gini, 0), 1)
}

// calculateTopN returns the percentage of commits from the top N contributors.
func calculateTopN(authors []CommitAuthor, total int, n int) float64 {
	if total == 0 || len(authors) == 0 {
		return 0
	}

	if n > len(authors) {
		n = len(authors)
	}

	topCommits := 0
	for i := 0; i < n; i++ {
		topCommits += authors[i].Commits
	}

	return float64(topCommits) / float64(total)
}

// calculateBusFactor returns the minimum number of contributors needed to reach 50% of commits.
func calculateBusFactor(authors []CommitAuthor, total int) int {
	if total == 0 {
		return 0
	}

	threshold := float64(total) * 0.5
	commits := 0

	for i, author := range authors {
		commits += author.Commits
		if float64(commits) >= threshold {
			return i + 1
		}
	}

	return len(authors)
}

// assessRiskLevel determines bus-factor risk based on metrics.
func assessRiskLevel(hIndex float64, top1Percent float64, numContributors int) string {
	// Critical: 1 person has >80% OR total contributors <= 1
	if top1Percent > 0.8 || numContributors <= 1 {
		return "critical"
	}

	// High: 1 person has >50% OR 2 people have >80% OR contributors <= 2
	if top1Percent > 0.5 || numContributors <= 2 {
		return "high"
	}

	// Medium: Herfindahl > 0.3 OR contributors <= 5
	if hIndex > 0.3 || numContributors <= 5 {
		return "medium"
	}

	return "low"
}

// RiskLevelScore returns a numeric score (0-100) for the risk level.
func RiskLevelScore(level string) int {
	switch level {
	case "critical":
		return 75
	case "high":
		return 50
	case "medium":
		return 25
	case "low":
		return 0
	default:
		return -1
	}
}

// String returns a human-readable summary of metrics.
func (om *OwnershipMetrics) String() string {
	if om.TotalContributors == 0 {
		return "No contributors found"
	}

	return fmt.Sprintf(
		"Contributors: %d | Bus Factor: %d | Top 1: %.1f%% | Risk: %s | HHI: %.3f",
		om.TotalContributors,
		om.BusFactor,
		om.Top1Percent,
		om.RiskLevel,
		om.HerfindahlIndex,
	)
}
