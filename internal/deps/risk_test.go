package deps

import (
	"testing"
)

func TestScoreDependency(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     RiskLevel
	}{
		{
			name:     "no findings",
			findings: nil,
			want:     RiskNone,
		},
		{
			name: "single critical",
			findings: []Finding{
				{RiskLevel: RiskCritical},
			},
			want: RiskCritical,
		},
		{
			name: "mixed levels returns highest",
			findings: []Finding{
				{RiskLevel: RiskLow},
				{RiskLevel: RiskHigh},
				{RiskLevel: RiskMedium},
			},
			want: RiskHigh,
		},
		{
			name: "all low",
			findings: []Finding{
				{RiskLevel: RiskLow},
				{RiskLevel: RiskLow},
			},
			want: RiskLow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreDependency(tt.findings)
			if got != tt.want {
				t.Errorf("ScoreDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeResults(t *testing.T) {
	result := &ScanResult{
		Findings: []Finding{
			{RiskLevel: RiskCritical},
			{RiskLevel: RiskHigh},
			{RiskLevel: RiskHigh},
			{RiskLevel: RiskMedium},
			{RiskLevel: RiskMedium},
			{RiskLevel: RiskMedium},
			{RiskLevel: RiskLow},
		},
	}

	c, h, m, l := SummarizeResults(result)
	if c != 1 || h != 2 || m != 3 || l != 1 {
		t.Errorf("SummarizeResults() = (%d, %d, %d, %d), want (1, 2, 3, 1)", c, h, m, l)
	}
}

func TestSortFindings(t *testing.T) {
	findings := []Finding{
		{Module: "b-mod", RiskLevel: RiskLow},
		{Module: "a-mod", RiskLevel: RiskCritical},
		{Module: "c-mod", RiskLevel: RiskHigh},
		{Module: "a-mod", RiskLevel: RiskHigh},
	}

	SortFindings(findings)

	expected := []struct {
		module string
		level  RiskLevel
	}{
		{"a-mod", RiskCritical},
		{"a-mod", RiskHigh},
		{"c-mod", RiskHigh},
		{"b-mod", RiskLow},
	}

	for i, f := range findings {
		if f.Module != expected[i].module || f.RiskLevel != expected[i].level {
			t.Errorf("findings[%d] = {%s, %s}, want {%s, %s}",
				i, f.Module, f.RiskLevel, expected[i].module, expected[i].level)
		}
	}
}
