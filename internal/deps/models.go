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

// Category classifies the type of risk finding.
type Category string

const (
	CategoryVulnerability Category = "vulnerability"
	CategoryMaintenance   Category = "maintenance"
	CategoryLicense       Category = "license"
)

// Dependency represents a single Go module dependency.
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Direct  bool   `json:"direct"`
}

// Finding represents a single risk finding for a dependency.
type Finding struct {
	Module      string    `json:"module"`
	Version     string    `json:"version"`
	Category    Category  `json:"category"`
	Risk        RiskLevel `json:"risk"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Reference   string    `json:"reference,omitempty"`
}

// ScanResult holds the complete results of a dependency scan.
type ScanResult struct {
	Project   string    `json:"project"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration"`

	TotalDeps  int `json:"total_deps"`
	DirectDeps int `json:"direct_deps"`

	Findings []Finding         `json:"findings"`
	Summary  map[RiskLevel]int `json:"summary"`
}
