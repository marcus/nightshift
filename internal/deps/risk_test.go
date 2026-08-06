package deps

import "testing"

func TestScoreDependency(t *testing.T) {
	findings := []Finding{
		{Module: "github.com/foo/bar", Risk: RiskLow, Category: CategoryLicense},
		{Module: "github.com/foo/bar", Risk: RiskHigh, Category: CategoryVulnerability},
		{Module: "github.com/foo/bar", Risk: RiskMedium, Category: CategoryMaintenance},
		{Module: "github.com/baz/qux", Risk: RiskLow, Category: CategoryLicense},
	}

	if got := ScoreDependency(findings, "github.com/foo/bar"); got != RiskHigh {
		t.Errorf("expected high, got %s", got)
	}
	if got := ScoreDependency(findings, "github.com/baz/qux"); got != RiskLow {
		t.Errorf("expected low, got %s", got)
	}
	if got := ScoreDependency(findings, "github.com/unknown"); got != RiskNone {
		t.Errorf("expected none, got %s", got)
	}
}

func TestSummarizeResults(t *testing.T) {
	findings := []Finding{
		{Risk: RiskCritical},
		{Risk: RiskHigh},
		{Risk: RiskHigh},
		{Risk: RiskMedium},
		{Risk: RiskLow},
		{Risk: RiskLow},
		{Risk: RiskLow},
	}

	summary := SummarizeResults(findings)
	if summary[RiskCritical] != 1 {
		t.Errorf("expected 1 critical, got %d", summary[RiskCritical])
	}
	if summary[RiskHigh] != 2 {
		t.Errorf("expected 2 high, got %d", summary[RiskHigh])
	}
	if summary[RiskMedium] != 1 {
		t.Errorf("expected 1 medium, got %d", summary[RiskMedium])
	}
	if summary[RiskLow] != 3 {
		t.Errorf("expected 3 low, got %d", summary[RiskLow])
	}
}

func TestSortFindings(t *testing.T) {
	findings := []Finding{
		{Module: "z-module", Risk: RiskLow},
		{Module: "a-module", Risk: RiskHigh},
		{Module: "m-module", Risk: RiskCritical},
		{Module: "b-module", Risk: RiskHigh},
	}

	SortFindings(findings)

	if findings[0].Risk != RiskCritical {
		t.Errorf("first should be critical, got %s", findings[0].Risk)
	}
	if findings[1].Module != "a-module" || findings[1].Risk != RiskHigh {
		t.Errorf("second should be a-module/high, got %s/%s", findings[1].Module, findings[1].Risk)
	}
	if findings[2].Module != "b-module" {
		t.Errorf("third should be b-module, got %s", findings[2].Module)
	}
	if findings[3].Risk != RiskLow {
		t.Errorf("last should be low, got %s", findings[3].Risk)
	}
}
