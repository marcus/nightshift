package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantDeps  []Dependency
		wantErr   bool
	}{
		{
			name: "direct and indirect deps",
			content: `module example.com/mymod

go 1.21

require (
	github.com/foo/bar v1.2.3
	github.com/baz/qux v0.1.0 // indirect
)
`,
			wantDeps: []Dependency{
				{Module: "github.com/foo/bar", Version: "v1.2.3", Indirect: false},
				{Module: "github.com/baz/qux", Version: "v0.1.0", Indirect: true},
			},
		},
		{
			name: "replace directive",
			content: `module example.com/mymod

go 1.21

require github.com/old/mod v1.0.0

replace github.com/old/mod => github.com/new/mod v2.0.0
`,
			wantDeps: []Dependency{
				{Module: "github.com/new/mod", Version: "v2.0.0", Indirect: false},
			},
		},
		{
			name:    "empty go.mod",
			content: "module example.com/mymod\n\ngo 1.21\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			deps, err := ParseGoMod(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGoMod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if len(deps) != len(tt.wantDeps) {
				t.Fatalf("got %d deps, want %d", len(deps), len(tt.wantDeps))
			}
			for i, got := range deps {
				want := tt.wantDeps[i]
				if got.Module != want.Module || got.Version != want.Version || got.Indirect != want.Indirect {
					t.Errorf("dep[%d] = %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

func TestParseGoModMissingFile(t *testing.T) {
	_, err := ParseGoMod(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing go.mod")
	}
}
