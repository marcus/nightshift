package deps

import "time"

// RiskLevel represents the severity of a dependency risk finding.
type RiskLevel string

const (
	RiskCritical RiskLevel = "critical"
	RiskHigh     RiskLevel = "high"
	RiskMedium   RiskLevel = "medium"
	RiskLow      RiskLevel = "low"
	RiskNone     RiskLevel = "none"
)

// Severity returns a numeric severity for sorting (higher = more severe).
func (r RiskLevel) Severity() int {
	switch r {
	case RiskCritical:
		return 4
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	case RiskLow:
		return 1
	default:
		return 0
	}
}

// Category classifies what type of risk a finding represents.
type Category string

const (
	CategoryVulnerability Category = "vulnerability"
	CategoryMaintenance   Category = "maintenance"
	CategoryLicense       Category = "license"
)

// Dependency represents a single Go module dependency.
type Dependency struct {
	Module   string `json:"module"`
	Version  string `json:"version"`
	Indirect bool   `json:"indirect"`
}

// Finding represents a single risk finding for a dependency.
type Finding struct {
	Module    string    `json:"module"`
	Version   string    `json:"version"`
	RiskLevel RiskLevel `json:"risk_level"`
	Category  Category  `json:"category"`
	Detail    string    `json:"detail"`
	CVEID     string    `json:"cve_id,omitempty"`
	CVSSScore float64   `json:"cvss_score,omitempty"`
}

// ScanResult holds the complete results of scanning a project's dependencies.
type ScanResult struct {
	ID        int64        `json:"id,omitempty"`
	Project   string       `json:"project"`
	ScannedAt time.Time    `json:"scanned_at"`
	Deps      []Dependency `json:"deps"`
	Findings  []Finding    `json:"findings"`
}
