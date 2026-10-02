package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// MetricPattern defines a pattern that indicates metrics instrumentation.
type MetricPattern struct {
	Package  string   // import path substring to match
	Keywords []string // identifiers/selectors that indicate instrumentation
}

// knownMetricPatterns are recognized instrumentation patterns.
var knownMetricPatterns = []MetricPattern{
	{Package: "prometheus", Keywords: []string{"Counter", "Gauge", "Histogram", "Summary", "NewCounter", "NewGauge", "NewHistogram", "NewSummary", "CounterVec", "GaugeVec", "HistogramVec", "SummaryVec", "Inc", "Add", "Set", "Observe"}},
	{Package: "otel", Keywords: []string{"Meter", "Counter", "Histogram", "UpDownCounter", "NewMeter", "Float64Counter", "Int64Counter", "Float64Histogram", "Int64Histogram"}},
	{Package: "statsd", Keywords: []string{"Count", "Gauge", "Histogram", "Timing", "Incr", "Decr", "Distribution"}},
	{Package: "datadog", Keywords: []string{"Count", "Gauge", "Histogram", "Timing", "Incr", "Distribution"}},
	{Package: "expvar", Keywords: []string{"NewInt", "NewFloat", "NewMap", "NewString", "Int", "Float", "Map", "Add", "Set"}},
}

// structuralKeywords are identifier substrings that indicate metrics instrumentation
// even without a known import package (custom metric types, structured logging counters).
var structuralKeywords = []string{
	"counter", "gauge", "histogram", "metric", "metrics",
	"measure", "instrument", "observe", "record",
	"statsd", "prometheus", "telemetry",
}

// PackageCoverage holds metrics instrumentation coverage for a single Go package.
type PackageCoverage struct {
	Path                string   `json:"path"`
	TotalExported       int      `json:"total_exported"`
	InstrumentedCount   int      `json:"instrumented_count"`
	CoveragePct         float64  `json:"coverage_pct"`
	DetectedMetricTypes []string `json:"detected_metric_types,omitempty"`
	Gaps                []string `json:"gaps,omitempty"`
}

// CoverageSummary holds the overall summary of metrics instrumentation coverage.
type CoverageSummary struct {
	OverallCoveragePct float64 `json:"overall_coverage_pct"`
	TotalPackages      int     `json:"total_packages"`
	TotalExported      int     `json:"total_exported"`
	TotalInstrumented  int     `json:"total_instrumented"`
	GapCount           int     `json:"gap_count"`
	RiskLevel          string  `json:"risk_level"`
}

// MetricsCoverageScanner walks Go source files and detects metrics instrumentation.
type MetricsCoverageScanner struct {
	root string
}

// NewMetricsCoverageScanner creates a scanner rooted at the given directory.
func NewMetricsCoverageScanner(root string) *MetricsCoverageScanner {
	return &MetricsCoverageScanner{root: root}
}

// Scan walks all Go packages under root and computes coverage.
func (s *MetricsCoverageScanner) Scan() ([]PackageCoverage, error) {
	pkgDirs, err := s.findGoPackages()
	if err != nil {
		return nil, err
	}

	var results []PackageCoverage
	for _, dir := range pkgDirs {
		cov, err := s.scanPackage(dir)
		if err != nil {
			continue // skip packages that fail to parse
		}
		if cov.TotalExported == 0 {
			continue // skip packages with no exported functions
		}
		results = append(results, *cov)
	}

	return results, nil
}

// findGoPackages discovers directories containing .go files under root.
func (s *MetricsCoverageScanner) findGoPackages() ([]string, error) {
	seen := make(map[string]bool)
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			base := info.Name()
			if base == "vendor" || base == "testdata" || base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			dir := filepath.Dir(path)
			seen[dir] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	return dirs, nil
}

// scanPackage analyzes a single Go package directory.
func (s *MetricsCoverageScanner) scanPackage(dir string) (*PackageCoverage, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(s.root, dir)
	if relPath == "" || relPath == "." {
		relPath = filepath.Base(dir)
	}

	cov := &PackageCoverage{Path: relPath}
	metricTypes := make(map[string]bool)

	for _, pkg := range pkgs {
		// Collect import paths across all files in the package
		importPaths := make(map[string]bool)
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				importPaths[p] = true
			}
		}

		// Detect which metric frameworks are imported
		detectedPatterns := s.detectImportedPatterns(importPaths)

		for _, file := range pkg.Files {
			s.analyzeFile(file, detectedPatterns, cov, metricTypes)
		}
	}

	for t := range metricTypes {
		cov.DetectedMetricTypes = append(cov.DetectedMetricTypes, t)
	}

	if cov.TotalExported > 0 {
		cov.CoveragePct = float64(cov.InstrumentedCount) * 100.0 / float64(cov.TotalExported)
	}

	return cov, nil
}

// detectImportedPatterns returns the metric patterns whose packages are imported.
func (s *MetricsCoverageScanner) detectImportedPatterns(imports map[string]bool) []MetricPattern {
	var matched []MetricPattern
	for _, pattern := range knownMetricPatterns {
		for imp := range imports {
			if strings.Contains(imp, pattern.Package) {
				matched = append(matched, pattern)
				break
			}
		}
	}
	return matched
}

// analyzeFile walks a parsed Go file and checks each exported function for instrumentation.
func (s *MetricsCoverageScanner) analyzeFile(file *ast.File, patterns []MetricPattern, cov *PackageCoverage, metricTypes map[string]bool) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}

		cov.TotalExported++

		isHandler := isHTTPHandler(fn)
		hasInstrumentation := s.functionHasInstrumentation(fn, patterns, metricTypes)

		if hasInstrumentation {
			cov.InstrumentedCount++
		} else {
			gap := fn.Name.Name
			if isHandler {
				gap += " (HTTP handler)"
			}
			if hasErrorReturn(fn) {
				gap += " (has error return)"
			}
			cov.Gaps = append(cov.Gaps, gap)
		}
	}
}

// functionHasInstrumentation checks whether a function body contains metric calls.
func (s *MetricsCoverageScanner) functionHasInstrumentation(fn *ast.FuncDecl, patterns []MetricPattern, metricTypes map[string]bool) bool {
	if fn.Body == nil {
		return false
	}

	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}

		switch node := n.(type) {
		case *ast.CallExpr:
			if s.isMetricCall(node, patterns, metricTypes) {
				found = true
				return false
			}
		case *ast.Ident:
			if s.isStructuralMetricIdent(node.Name) {
				found = true
				return false
			}
		}
		return true
	})

	return found
}

// isMetricCall checks if a call expression matches known metric patterns.
func (s *MetricsCoverageScanner) isMetricCall(call *ast.CallExpr, patterns []MetricPattern, metricTypes map[string]bool) bool {
	name := callExprName(call)
	if name == "" {
		return false
	}

	// Check against imported metric patterns
	for _, pattern := range patterns {
		for _, kw := range pattern.Keywords {
			if strings.Contains(name, kw) {
				metricTypes[pattern.Package] = true
				return true
			}
		}
	}

	// Check structural keywords in call names
	lower := strings.ToLower(name)
	for _, kw := range structuralKeywords {
		if strings.Contains(lower, kw) {
			metricTypes["custom"] = true
			return true
		}
	}

	return false
}

// isStructuralMetricIdent checks if an identifier name suggests metric usage.
func (s *MetricsCoverageScanner) isStructuralMetricIdent(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range structuralKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// callExprName extracts a readable name from a call expression.
func callExprName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			return ident.Name + "." + fn.Sel.Name
		}
		return fn.Sel.Name
	}
	return ""
}

// isHTTPHandler checks if a function signature matches http.Handler/HandlerFunc patterns.
func isHTTPHandler(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	params := fn.Type.Params.List
	if len(params) < 2 {
		return false
	}

	// Check for (http.ResponseWriter, *http.Request) pattern
	hasWriter := false
	hasRequest := false
	for _, param := range params {
		typeName := typeString(param.Type)
		if strings.Contains(typeName, "ResponseWriter") {
			hasWriter = true
		}
		if strings.Contains(typeName, "Request") {
			hasRequest = true
		}
	}
	return hasWriter && hasRequest
}

// hasErrorReturn checks if the function returns an error type.
func hasErrorReturn(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		if typeString(field.Type) == "error" {
			return true
		}
	}
	return false
}

// typeString returns a simplified string representation of a type expression.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.StarExpr:
		return typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	}
	return ""
}

// ComputeSummary computes an overall CoverageSummary from per-package results.
func ComputeSummary(packages []PackageCoverage) *CoverageSummary {
	summary := &CoverageSummary{
		TotalPackages: len(packages),
	}

	for _, pkg := range packages {
		summary.TotalExported += pkg.TotalExported
		summary.TotalInstrumented += pkg.InstrumentedCount
		summary.GapCount += len(pkg.Gaps)
	}

	if summary.TotalExported > 0 {
		summary.OverallCoveragePct = float64(summary.TotalInstrumented) * 100.0 / float64(summary.TotalExported)
	}

	summary.RiskLevel = assessCoverageRisk(summary.OverallCoveragePct, summary.GapCount)
	return summary
}

// assessCoverageRisk determines the risk level based on coverage percentage and gap count.
func assessCoverageRisk(coveragePct float64, gapCount int) string {
	switch {
	case coveragePct >= 80 && gapCount <= 5:
		return "low"
	case coveragePct >= 80:
		return "medium"
	case coveragePct >= 50:
		return "medium"
	case coveragePct >= 20:
		return "high"
	default:
		return "critical"
	}
}
