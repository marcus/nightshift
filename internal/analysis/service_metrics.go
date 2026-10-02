package analysis

import (
	"fmt"
	"math"
)

// ServiceMetrics holds computed metrics for a single package's service boundary potential.
type ServiceMetrics struct {
	// CouplingScore measures afferent/efferent coupling ratio (0=isolated, 1=highly coupled)
	CouplingScore float64 `json:"coupling_score"`
	// CohesionScore measures internal vs external references (0=low cohesion, 1=high cohesion)
	CohesionScore float64 `json:"cohesion_score"`
	// SizeScore measures relative LOC size (0=small, 1=very large relative to project)
	SizeScore float64 `json:"size_score"`
	// ChurnCorrelation measures co-change frequency with other packages (0=independent, 1=always co-changed)
	ChurnCorrelation float64 `json:"churn_correlation"`
	// ExtractWorthiness is a composite score (0=keep as-is, 1=strong extraction candidate)
	ExtractWorthiness float64 `json:"extract_worthiness"`
	// AfferentCoupling is the number of packages that depend on this package
	AfferentCoupling int `json:"afferent_coupling"`
	// EfferentCoupling is the number of packages this package depends on
	EfferentCoupling int `json:"efferent_coupling"`
	// Recommendation is the suggested action: "extract", "merge", or "keep"
	Recommendation string `json:"recommendation"`
}

// PackageAnalysis pairs a package with its computed metrics.
type PackageAnalysis struct {
	Package PackageInfo    `json:"package"`
	Metrics ServiceMetrics `json:"metrics"`
}

// CalculateServiceMetrics computes service boundary metrics for each package.
func CalculateServiceMetrics(pkgs []PackageInfo) []PackageAnalysis {
	if len(pkgs) == 0 {
		return nil
	}

	// Compute total LOC across all packages for relative size
	totalLOC := 0
	for _, pkg := range pkgs {
		totalLOC += pkg.LOC
	}

	results := make([]PackageAnalysis, len(pkgs))
	for i, pkg := range pkgs {
		m := computeMetrics(pkg, totalLOC, len(pkgs))
		results[i] = PackageAnalysis{
			Package: pkg,
			Metrics: m,
		}
	}

	return results
}

func computeMetrics(pkg PackageInfo, totalLOC, totalPkgs int) ServiceMetrics {
	afferent := len(pkg.ImportedBy)
	efferent := len(pkg.Imports)

	coupling := computeCouplingScore(afferent, efferent)
	cohesion := computeCohesionScore(pkg)
	size := computeSizeScore(pkg.LOC, totalLOC)
	churn := computeChurnCorrelation(pkg, totalPkgs)
	extract := computeExtractWorthiness(coupling, cohesion, size, churn)
	rec := recommendAction(extract, coupling, cohesion, size, pkg)

	return ServiceMetrics{
		CouplingScore:     coupling,
		CohesionScore:     cohesion,
		SizeScore:         size,
		ChurnCorrelation:  churn,
		ExtractWorthiness: extract,
		AfferentCoupling:  afferent,
		EfferentCoupling:  efferent,
		Recommendation:    rec,
	}
}

// computeCouplingScore calculates instability: Ce / (Ca + Ce).
// High instability (close to 1) = depends on many, depended on by few.
// Low instability (close to 0) = stable, depended on by many.
// We return coupling as the ratio of total connections relative to a reasonable max.
func computeCouplingScore(afferent, efferent int) float64 {
	total := afferent + efferent
	if total == 0 {
		return 0
	}
	// Normalize: assume > 10 total connections is highly coupled
	score := float64(total) / 10.0
	return math.Min(score, 1.0)
}

// computeCohesionScore estimates internal cohesion.
// High exported symbols with few internal references suggests low cohesion (grab bag).
// Many internal references relative to exports suggests high cohesion (tightly related).
func computeCohesionScore(pkg PackageInfo) float64 {
	if pkg.ExportedSyms == 0 {
		return 1.0 // No exports = internally cohesive by default
	}
	// Ratio of internal references to exported symbols
	ratio := float64(pkg.InternalRefs) / float64(pkg.ExportedSyms)
	// Normalize: ratio of 5+ internal refs per export = high cohesion
	score := math.Min(ratio/5.0, 1.0)
	return score
}

// computeSizeScore returns how large this package is relative to the project.
func computeSizeScore(pkgLOC, totalLOC int) float64 {
	if totalLOC == 0 {
		return 0
	}
	// A package with > 20% of total LOC is very large
	ratio := float64(pkgLOC) / float64(totalLOC)
	score := ratio / 0.20
	return math.Min(score, 1.0)
}

// computeChurnCorrelation measures how frequently this package co-changes with others.
func computeChurnCorrelation(pkg PackageInfo, totalPkgs int) float64 {
	if totalPkgs <= 1 {
		return 0
	}
	// Fraction of other packages this one co-changes with
	ratio := float64(len(pkg.CoChanges)) / float64(totalPkgs-1)
	return math.Min(ratio, 1.0)
}

// computeExtractWorthiness combines metrics into a single score.
// Higher = stronger candidate for extraction into its own service.
func computeExtractWorthiness(coupling, cohesion, size, churn float64) float64 {
	// Weights: size and cohesion matter most for extraction decisions
	// High cohesion + large size + low coupling + low churn = good extraction candidate
	// We want: high cohesion (good), large size (indicates complexity), low coupling (clean boundary), low churn (stable)
	score := (cohesion*0.30 + size*0.30 + (1-coupling)*0.20 + (1-churn)*0.20)
	return math.Round(score*1000) / 1000
}

// recommendAction suggests what to do with the package.
func recommendAction(extract, coupling, cohesion, size float64, pkg PackageInfo) string {
	// Small packages with low cohesion: merge candidates
	if size < 0.1 && pkg.Files <= 2 && pkg.LOC < 100 {
		return "merge"
	}

	// High extract-worthiness: extraction candidates
	if extract >= 0.6 && cohesion >= 0.4 && size >= 0.2 {
		return "extract"
	}

	return "keep"
}

// String returns a human-readable summary of service metrics.
func (sm *ServiceMetrics) String() string {
	return fmt.Sprintf(
		"Coupling: %.2f | Cohesion: %.2f | Size: %.2f | Churn: %.2f | Extract: %.3f | Rec: %s",
		sm.CouplingScore,
		sm.CohesionScore,
		sm.SizeScore,
		sm.ChurnCorrelation,
		sm.ExtractWorthiness,
		sm.Recommendation,
	)
}
