package deps

import "sort"

// riskWeight maps risk levels to numeric scores for sorting.
var riskWeight = map[RiskLevel]int{
	RiskCritical: 4,
	RiskHigh:     3,
	RiskMedium:   2,
	RiskLow:      1,
	RiskNone:     0,
}

// ScoreDependency returns the highest risk level across all findings for a module.
func ScoreDependency(findings []Finding, module string) RiskLevel {
	highest := RiskNone
	for _, f := range findings {
		if f.Module == module && riskWeight[f.Risk] > riskWeight[highest] {
			highest = f.Risk
		}
	}
	return highest
}

// SummarizeResults counts findings by risk level.
func SummarizeResults(findings []Finding) map[RiskLevel]int {
	summary := make(map[RiskLevel]int)
	for _, f := range findings {
		summary[f.Risk]++
	}
	return summary
}

// SortFindings orders findings by severity (critical first), then by module name.
func SortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		wi := riskWeight[findings[i].Risk]
		wj := riskWeight[findings[j].Risk]
		if wi != wj {
			return wi > wj
		}
		return findings[i].Module < findings[j].Module
	})
}
