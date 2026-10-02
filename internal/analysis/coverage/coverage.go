// Package coverage provides static analysis of metrics instrumentation
// coverage across a Go codebase. It walks .go source files, parses them with
// go/ast, and reports which functions contain instrumentation calls
// (logging, stats, metrics, telemetry) and which do not.
package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultPatterns are the call-site patterns that count as "instrumentation"
// for the purposes of this analyzer. They are matched against the textual
// form of a CallExpr's function expression (e.g. "logging.Component",
// "log.Printf", "logger.Infof", "stats.Record", "metrics.Inc",
// "otel.Tracer", "telemetry.Emit").
var DefaultPatterns = []string{
	"log.",
	"logger.",
	"logging.",
	"zerolog.",
	"zlog.",
	"stats.",
	"metrics.",
	"otel.",
	"telemetry.",
	"trace.",
	"prometheus.",
	"observability.",
	".Info",
	".Infof",
	".Warn",
	".Warnf",
	".Error",
	".Errorf",
	".Debug",
	".Debugf",
	".Trace",
	".Tracef",
	".Record",
	".Emit",
	".Observe",
	".Inc",
}

// Options configures the analyzer.
type Options struct {
	// Root is the directory to analyze (recursively).
	Root string
	// Patterns are the substrings used to identify instrumentation calls.
	// If empty, DefaultPatterns is used.
	Patterns []string
	// Excludes are glob patterns matched against the relative path of each
	// candidate file (e.g. "vendor/**", "**/*_generated.go"). Files matching
	// any glob are skipped.
	Excludes []string
	// IncludeTests, when false, skips *_test.go files.
	IncludeTests bool
}

// FunctionGap describes an uninstrumented function.
type FunctionGap struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// PackageCoverage holds the per-package coverage tally.
type PackageCoverage struct {
	Package             string        `json:"package"`
	Dir                 string        `json:"dir"`
	TotalFuncs          int           `json:"total_funcs"`
	InstrumentedFuncs   int           `json:"instrumented_funcs"`
	Percent             float64       `json:"percent"`
	UninstrumentedFuncs []FunctionGap `json:"uninstrumented_funcs,omitempty"`
}

// OverallCoverage aggregates per-package results.
type OverallCoverage struct {
	Root              string            `json:"root"`
	Patterns          []string          `json:"patterns"`
	Packages          []PackageCoverage `json:"packages"`
	TotalFuncs        int               `json:"total_funcs"`
	InstrumentedFuncs int               `json:"instrumented_funcs"`
	Percent           float64           `json:"percent"`
}

// Analyzer performs metrics instrumentation coverage analysis.
type Analyzer struct {
	opts Options
}

// New creates an analyzer with the supplied options.
func New(opts Options) *Analyzer {
	if len(opts.Patterns) == 0 {
		opts.Patterns = DefaultPatterns
	}
	return &Analyzer{opts: opts}
}

// Analyze walks the configured root and returns the overall coverage report.
func (a *Analyzer) Analyze() (*OverallCoverage, error) {
	root, err := filepath.Abs(a.opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolving root: %w", err)
	}

	pkgs := make(map[string]*PackageCoverage)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			// Skip common directories that don't contain hand-written source.
			if path != root && (name == "vendor" || name == "node_modules" || name == ".git" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !a.opts.IncludeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		if a.isExcluded(rel) {
			return nil
		}
		if isGenerated(path) {
			return nil
		}
		return a.analyzeFile(path, rel, pkgs)
	})
	if err != nil {
		return nil, err
	}

	result := &OverallCoverage{
		Root:     root,
		Patterns: append([]string(nil), a.opts.Patterns...),
		Packages: make([]PackageCoverage, 0, len(pkgs)),
	}

	for _, pc := range pkgs {
		sort.Slice(pc.UninstrumentedFuncs, func(i, j int) bool {
			if pc.UninstrumentedFuncs[i].File == pc.UninstrumentedFuncs[j].File {
				return pc.UninstrumentedFuncs[i].Line < pc.UninstrumentedFuncs[j].Line
			}
			return pc.UninstrumentedFuncs[i].File < pc.UninstrumentedFuncs[j].File
		})
		if pc.TotalFuncs > 0 {
			pc.Percent = float64(pc.InstrumentedFuncs) / float64(pc.TotalFuncs) * 100
		}
		result.Packages = append(result.Packages, *pc)
		result.TotalFuncs += pc.TotalFuncs
		result.InstrumentedFuncs += pc.InstrumentedFuncs
	}

	sort.Slice(result.Packages, func(i, j int) bool {
		return result.Packages[i].Package < result.Packages[j].Package
	})

	if result.TotalFuncs > 0 {
		result.Percent = float64(result.InstrumentedFuncs) / float64(result.TotalFuncs) * 100
	}

	return result, nil
}

func (a *Analyzer) analyzeFile(path, rel string, pkgs map[string]*PackageCoverage) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if hasGeneratedComment(file) {
		return nil
	}

	pkgName := file.Name.Name
	dir := filepath.Dir(rel)
	key := dir + "::" + pkgName
	pc, ok := pkgs[key]
	if !ok {
		pc = &PackageCoverage{Package: pkgName, Dir: dir}
		pkgs[key] = pc
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		pc.TotalFuncs++
		if a.functionInstrumented(fn) {
			pc.InstrumentedFuncs++
		} else {
			pos := fset.Position(fn.Pos())
			pc.UninstrumentedFuncs = append(pc.UninstrumentedFuncs, FunctionGap{
				Name: functionDisplayName(fn),
				File: rel,
				Line: pos.Line,
			})
		}
	}

	return nil
}

func (a *Analyzer) functionInstrumented(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		expr := exprString(call.Fun)
		if a.matchesPattern(expr) {
			found = true
			return false
		}
		return true
	})
	return found
}

func (a *Analyzer) matchesPattern(expr string) bool {
	for _, p := range a.opts.Patterns {
		if p == "" {
			continue
		}
		if strings.Contains(expr, p) {
			return true
		}
	}
	return false
}

func (a *Analyzer) isExcluded(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, pat := range a.opts.Excludes {
		if pat == "" {
			continue
		}
		pat = filepath.ToSlash(pat)
		if matched, err := filepath.Match(pat, rel); err == nil && matched {
			return true
		}
		// Allow "dir/**" style patterns by checking prefix when "**" suffix.
		if strings.HasSuffix(pat, "/**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if strings.HasPrefix(rel, prefix+"/") || rel == prefix {
				return true
			}
		}
		// Plain substring match as fallback for convenience.
		if strings.Contains(rel, pat) {
			return true
		}
	}
	return false
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return exprString(v.Fun) + "()"
	case *ast.IndexExpr:
		return exprString(v.X)
	case *ast.IndexListExpr:
		return exprString(v.X)
	case *ast.ParenExpr:
		return exprString(v.X)
	case *ast.StarExpr:
		return exprString(v.X)
	}
	return ""
}

func functionDisplayName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := exprString(fn.Recv.List[0].Type)
		recv = strings.TrimPrefix(recv, "*")
		return fmt.Sprintf("(%s).%s", recv, fn.Name.Name)
	}
	return fn.Name.Name
}

func hasGeneratedComment(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

func isGenerated(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_generated.go") || strings.HasSuffix(base, ".pb.go")
}
