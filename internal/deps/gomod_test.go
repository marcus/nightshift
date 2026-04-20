package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantDeps    int
		wantDirect  int
		wantErr     bool
	}{
		{
			name: "simple module with direct and indirect deps",
			content: `module example.com/myproject

go 1.21

require (
	github.com/foo/bar v1.2.3
	github.com/baz/qux v0.1.0 // indirect
)
`,
			wantDeps:   2,
			wantDirect: 1,
		},
		{
			name: "no dependencies",
			content: `module example.com/empty

go 1.21
`,
			wantDeps:   0,
			wantDirect: 0,
		},
		{
			name:    "invalid go.mod",
			content: `this is not valid`,
			wantErr: true,
		},
		{
			name: "multiple direct deps",
			content: `module example.com/multi

go 1.22

require (
	github.com/a/b v1.0.0
	github.com/c/d v0.9.0
	github.com/e/f v1.2.3
	github.com/g/h v0.5.0 // indirect
)
`,
			wantDeps:   4,
			wantDirect: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			deps, err := ParseGoMod(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(deps) != tt.wantDeps {
				t.Errorf("got %d deps, want %d", len(deps), tt.wantDeps)
			}

			directCount := 0
			for _, d := range deps {
				if d.Direct {
					directCount++
				}
			}
			if directCount != tt.wantDirect {
				t.Errorf("got %d direct deps, want %d", directCount, tt.wantDirect)
			}
		})
	}
}

func TestParseGoMod_MissingFile(t *testing.T) {
	_, err := ParseGoMod(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing go.mod")
	}
}
