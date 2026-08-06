package deps

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ParseGoMod reads and parses go.mod at the given project path,
// returning all dependencies.
func ParseGoMod(projectPath string) ([]Dependency, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}

	// Collect indirect modules for lookup
	indirect := make(map[string]bool)
	for _, req := range f.Require {
		if req.Indirect {
			indirect[req.Mod.Path] = true
		}
	}

	var deps []Dependency
	for _, req := range f.Require {
		deps = append(deps, Dependency{
			Module:  req.Mod.Path,
			Version: req.Mod.Version,
			Direct:  !indirect[req.Mod.Path],
		})
	}

	return deps, nil
}
