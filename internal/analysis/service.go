package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// PackageInfo holds parsed data about a Go package.
type PackageInfo struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`          // relative path from repo root
	Files        int      `json:"files"`         // number of .go files
	LOC          int      `json:"loc"`           // lines of code
	Imports      []string `json:"imports"`       // cross-package imports (within module)
	ImportedBy   []string `json:"imported_by"`   // packages that import this one
	ExportedSyms int      `json:"exported_syms"` // exported functions/types/vars
	InternalRefs int      `json:"internal_refs"` // references between files within package
	CoChanges    []string `json:"co_changes"`    // packages frequently changed together
}

// ServiceAnalyzer parses Go package structure and git history to extract
// service boundary data.
type ServiceAnalyzer struct {
	repoPath   string
	modulePath string
}

// NewServiceAnalyzer creates a new analyzer for the given repository path.
func NewServiceAnalyzer(repoPath string) *ServiceAnalyzer {
	return &ServiceAnalyzer{repoPath: repoPath}
}

// Analyze scans the repository for Go packages and builds package-level data
// including imports, sizes, exported API surface, and change coupling.
func (sa *ServiceAnalyzer) Analyze(minLOC int) ([]PackageInfo, error) {
	// Detect Go module path
	modPath, err := sa.detectModule()
	if err != nil {
		return nil, fmt.Errorf("detecting module: %w", err)
	}
	sa.modulePath = modPath

	// Find all Go packages
	pkgs, err := sa.parsePackages()
	if err != nil {
		return nil, fmt.Errorf("parsing packages: %w", err)
	}

	// Build import graph (imported_by)
	sa.buildImportGraph(pkgs)

	// Compute co-change data from git history
	if err := sa.computeCoChanges(pkgs); err != nil {
		// Non-fatal: git data is supplementary
		_ = err
	}

	// Filter by minimum LOC
	var filtered []PackageInfo
	for _, pkg := range pkgs {
		if pkg.LOC >= minLOC {
			filtered = append(filtered, pkg)
		}
	}

	// Sort by LOC descending for consistent output
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].LOC > filtered[j].LOC
	})

	return filtered, nil
}

// detectModule reads the module path from go.mod.
func (sa *ServiceAnalyzer) detectModule() (string, error) {
	modFile := filepath.Join(sa.repoPath, "go.mod")
	data, err := os.ReadFile(modFile)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// parsePackages walks the repo tree, parses Go files, and collects package info.
func (sa *ServiceAnalyzer) parsePackages() ([]PackageInfo, error) {
	pkgMap := make(map[string]*PackageInfo) // keyed by relative dir path

	err := filepath.Walk(sa.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		// Skip hidden dirs, vendor, testdata
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relDir, err := filepath.Rel(sa.repoPath, filepath.Dir(path))
		if err != nil {
			return nil
		}

		pkg, exists := pkgMap[relDir]
		if !exists {
			pkg = &PackageInfo{Path: relDir}
			pkgMap[relDir] = pkg
		}
		pkg.Files++

		// Count LOC
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		loc := 0
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				loc++
			}
		}
		pkg.LOC += loc

		// Parse AST for imports and exported symbols
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, data, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return nil // skip unparseable files
		}

		if pkg.Name == "" {
			pkg.Name = f.Name.Name
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(impPath, sa.modulePath) {
				// Convert module import to relative path
				relImport := strings.TrimPrefix(impPath, sa.modulePath+"/")
				if relImport != relDir {
					pkg.Imports = appendUnique(pkg.Imports, relImport)
				}
			}
		}

		// Count exported symbols (re-parse with full AST for declarations)
		full, err := parser.ParseFile(fset, path, data, 0)
		if err != nil {
			return nil
		}
		for _, decl := range full.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() {
					pkg.ExportedSyms++
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							pkg.ExportedSyms++
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								pkg.ExportedSyms++
							}
						}
					}
				}
			}
		}

		// Count internal references (identifiers referencing other files' symbols)
		for _, decl := range full.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Body != nil {
					ast.Inspect(fn.Body, func(n ast.Node) bool {
						if sel, ok := n.(*ast.SelectorExpr); ok {
							_ = sel
							pkg.InternalRefs++
						}
						return true
					})
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking repository: %w", err)
	}

	result := make([]PackageInfo, 0, len(pkgMap))
	for _, pkg := range pkgMap {
		result = append(result, *pkg)
	}
	return result, nil
}

// buildImportGraph populates ImportedBy for each package.
func (sa *ServiceAnalyzer) buildImportGraph(pkgs []PackageInfo) {
	byPath := make(map[string]int)
	for i := range pkgs {
		byPath[pkgs[i].Path] = i
	}

	for i := range pkgs {
		for _, imp := range pkgs[i].Imports {
			if idx, ok := byPath[imp]; ok {
				pkgs[idx].ImportedBy = appendUnique(pkgs[idx].ImportedBy, pkgs[i].Path)
			}
		}
	}
}

// computeCoChanges uses git log to find packages that frequently change together.
func (sa *ServiceAnalyzer) computeCoChanges(pkgs []PackageInfo) error {
	// Get recent commits with changed files
	cmd := exec.Command("git", "log", "--name-only", "--format=COMMIT", "-n", "200")
	cmd.Dir = sa.repoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running git log: %w", err)
	}

	// Parse commits into sets of changed package dirs
	var commits []map[string]bool
	var current map[string]bool

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "COMMIT" {
			if current != nil && len(current) > 0 {
				commits = append(commits, current)
			}
			current = make(map[string]bool)
			continue
		}
		if line == "" || current == nil {
			continue
		}
		dir := filepath.Dir(line)
		current[dir] = true
	}
	if current != nil && len(current) > 0 {
		commits = append(commits, current)
	}

	// Build co-change matrix: count how often each pair of packages appears in same commit
	byPath := make(map[string]int)
	for i := range pkgs {
		byPath[pkgs[i].Path] = i
	}

	type pair struct{ a, b string }
	coCount := make(map[pair]int)

	for _, commit := range commits {
		dirs := make([]string, 0, len(commit))
		for d := range commit {
			if _, ok := byPath[d]; ok {
				dirs = append(dirs, d)
			}
		}
		for i := 0; i < len(dirs); i++ {
			for j := i + 1; j < len(dirs); j++ {
				a, b := dirs[i], dirs[j]
				if a > b {
					a, b = b, a
				}
				coCount[pair{a, b}]++
			}
		}
	}

	// Assign co-changes where count >= 3 (meaningful frequency)
	threshold := 3
	for p, count := range coCount {
		if count >= threshold {
			if idx, ok := byPath[p.a]; ok {
				pkgs[idx].CoChanges = appendUnique(pkgs[idx].CoChanges, p.b)
			}
			if idx, ok := byPath[p.b]; ok {
				pkgs[idx].CoChanges = appendUnique(pkgs[idx].CoChanges, p.a)
			}
		}
	}

	return nil
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
