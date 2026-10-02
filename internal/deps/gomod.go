package deps

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ParseGoMod reads and parses a go.mod file from the given project path,
// returning all required dependencies.
func ParseGoMod(projectPath string) ([]Dependency, error) {
	gomodPath := filepath.Join(projectPath, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}

	replacements := make(map[string]struct {
		mod     string
		version string
	})
	for _, rep := range f.Replace {
		if rep.New.Path != "" {
			replacements[rep.Old.Path] = struct {
				mod     string
				version string
			}{mod: rep.New.Path, version: rep.New.Version}
		}
	}

	var deps []Dependency
	for _, req := range f.Require {
		mod := req.Mod.Path
		ver := req.Mod.Version
		if rep, ok := replacements[mod]; ok {
			mod = rep.mod
			ver = rep.version
		}
		deps = append(deps, Dependency{
			Module:   mod,
			Version:  ver,
			Indirect: req.Indirect,
		})
	}

	return deps, nil
}
