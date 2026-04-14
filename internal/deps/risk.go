package deps

import (
	"sort"
)

// ScoreDependency returns the highest risk level across all findings for a module.
func ScoreDependency(findings []Finding) RiskLevel {
	max := RiskNone
	for _, f := range findings {
		if f.RiskLevel.Severity() > max.Severity() {
			max = f.RiskLevel
		}
	}
	return max
}

// SummarizeResults counts findings by risk level.
func SummarizeResults(result *ScanResult) (critical, high, medium, low int) {
	for _, f := range result.Findings {
		switch f.RiskLevel {
		case RiskCritical:
			critical++
		case RiskHigh:
			high++
		case RiskMedium:
			medium++
		case RiskLow:
			low++
		}
	}
	return
}

// SortFindings sorts findings by severity descending, then by module name ascending.
func SortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		si := findings[i].RiskLevel.Severity()
		sj := findings[j].RiskLevel.Severity()
		if si != sj {
			return si > sj
		}
		return findings[i].Module < findings[j].Module
	})
}
