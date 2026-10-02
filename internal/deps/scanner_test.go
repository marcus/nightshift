package deps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	gomod := `module example.com/empty

go 1.21
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(ScanOptions{
		ProjectPath: dir,
		Concurrency: 2,
	})

	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalDeps != 0 {
		t.Errorf("expected 0 deps, got %d", result.TotalDeps)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestScanner_MissingGoMod(t *testing.T) {
	scanner := NewScanner(ScanOptions{
		ProjectPath: t.TempDir(),
		Concurrency: 2,
	})

	_, err := scanner.Scan(context.Background())
	if err == nil {
		t.Fatal("expected error for missing go.mod")
	}
}
