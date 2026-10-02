package silo

import (
	"testing"
	"time"
)

type fakeSource struct{ commits []Commit }

func (f *fakeSource) Commits() ([]Commit, error) { return f.commits, nil }

func TestSingleAuthorFileIsSilo(t *testing.T) {
	now := time.Now()
	src := &fakeSource{commits: []Commit{
		{Author: "alice", When: now, Files: []string{"a.go"}},
		{Author: "alice", When: now, Files: []string{"a.go"}},
	}}
	res, err := Analyze(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !res[0].IsSilo || res[0].TopAuthor != "alice" {
		t.Fatalf("expected single-author silo, got %+v", res)
	}
	if res[0].Dominance != 1.0 {
		t.Fatalf("expected dominance 1.0, got %v", res[0].Dominance)
	}
}

func TestBalancedFileNotSilo(t *testing.T) {
	now := time.Now()
	src := &fakeSource{commits: []Commit{
		{Author: "a", When: now, Files: []string{"b.go"}},
		{Author: "b", When: now, Files: []string{"b.go"}},
		{Author: "c", When: now, Files: []string{"b.go"}},
		{Author: "d", When: now, Files: []string{"b.go"}},
	}}
	res, _ := Analyze(src, Options{DominanceThreshold: 0.8})
	if res[0].IsSilo {
		t.Fatalf("expected non-silo, got %+v", res[0])
	}
}

func TestThresholdEdge(t *testing.T) {
	now := time.Now()
	// 4 of 5 commits = 0.8 dominance — at threshold
	src := &fakeSource{commits: []Commit{
		{Author: "a", When: now, Files: []string{"f"}},
		{Author: "a", When: now, Files: []string{"f"}},
		{Author: "a", When: now, Files: []string{"f"}},
		{Author: "a", When: now, Files: []string{"f"}},
		{Author: "b", When: now, Files: []string{"f"}},
	}}
	res, _ := Analyze(src, Options{DominanceThreshold: 0.8})
	if !res[0].IsSilo {
		t.Fatalf("expected silo at 0.8 threshold, got %+v", res[0])
	}
	res2, _ := Analyze(src, Options{DominanceThreshold: 0.9})
	if res2[0].IsSilo {
		t.Fatalf("expected not-silo at 0.9 threshold, got %+v", res2[0])
	}
}

func TestPathFilter(t *testing.T) {
	now := time.Now()
	src := &fakeSource{commits: []Commit{
		{Author: "a", When: now, Files: []string{"keep/x.go", "skip/y.go"}},
	}}
	res, _ := Analyze(src, Options{PathFilter: "keep/"})
	if len(res) != 1 || res[0].Path != "keep/x.go" {
		t.Fatalf("path filter failed: %+v", res)
	}
}

func TestGroupByDir(t *testing.T) {
	now := time.Now()
	src := &fakeSource{commits: []Commit{
		{Author: "a", When: now, Files: []string{"pkg/a.go"}},
		{Author: "a", When: now, Files: []string{"pkg/b.go"}},
		{Author: "b", When: now, Files: []string{"pkg/c.go"}},
	}}
	res, _ := Analyze(src, Options{GroupByDir: true})
	if len(res) != 1 || res[0].Path != "pkg" || res[0].Commits != 3 {
		t.Fatalf("group-by-dir failed: %+v", res)
	}
}
