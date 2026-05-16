package slo

import (
	"strings"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/reporting"
)

func mkTask(project, status, outputType, outputRef string, tokens int, dur time.Duration) reporting.TaskResult {
	return reporting.TaskResult{
		Project:    project,
		TaskType:   "lint-fix",
		Title:      "test",
		Status:     status,
		OutputType: outputType,
		OutputRef:  outputRef,
		TokensUsed: tokens,
		Duration:   dur,
	}
}

func mkRun(start time.Time, tasks ...reporting.TaskResult) *reporting.RunResults {
	end := start.Add(30 * time.Minute)
	used := 0
	for _, t := range tasks {
		used += t.TokensUsed
	}
	return &reporting.RunResults{
		Date:       start,
		StartTime:  start,
		EndTime:    end,
		UsedBudget: used,
		Tasks:      append([]reporting.TaskResult(nil), tasks...),
	}
}

func TestSuggest_EmptyHistory(t *testing.T) {
	out := Suggest(nil, Options{})
	if len(out) != 0 {
		t.Fatalf("expected no candidates from nil runs, got %d", len(out))
	}
	out = Suggest([]*reporting.RunResults{}, Options{})
	if len(out) != 0 {
		t.Fatalf("expected no candidates from empty runs, got %d", len(out))
	}
}

func TestSuggest_SmallSample_LowConfidence(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := []*reporting.RunResults{
		mkRun(now.Add(-120*time.Hour),
			mkTask("/proj/a", "completed", "PR", "https://example/1", 1000, 2*time.Minute),
			mkTask("/proj/a", "completed", "", "", 500, time.Minute),
		),
		mkRun(now.Add(-96*time.Hour),
			mkTask("/proj/a", "completed", "PR", "https://example/2", 800, time.Minute),
			mkTask("/proj/a", "failed", "", "", 100, 30*time.Second),
		),
		mkRun(now.Add(-72*time.Hour),
			mkTask("/proj/a", "completed", "PR", "https://example/3", 1200, 90*time.Second),
			mkTask("/proj/a", "completed", "", "", 700, 75*time.Second),
		),
		mkRun(now.Add(-48*time.Hour),
			mkTask("/proj/a", "completed", "PR", "https://example/4", 1100, 80*time.Second),
		),
		mkRun(now.Add(-24*time.Hour),
			mkTask("/proj/a", "completed", "PR", "https://example/5", 900, 70*time.Second),
		),
	}

	got := Suggest(runs, Options{Now: now, Window: 30 * 24 * time.Hour})
	if len(got) == 0 {
		t.Fatalf("expected at least one candidate from 5 runs, got none")
	}
	for _, c := range got {
		if c.Confidence != ConfidenceLow {
			t.Errorf("candidate %q: confidence=%s want low (n<10)", c.Name, c.Confidence)
		}
	}

	if !hasCandidate(got, "task-success-rate") {
		t.Errorf("expected task-success-rate candidate")
	}
	if !hasCandidate(got, "task-completion-latency") {
		t.Errorf("expected task-completion-latency candidate")
	}
	if !hasCandidate(got, "pr-throughput") {
		t.Errorf("expected pr-throughput candidate")
	}
	if !hasCandidate(got, "token-budget-per-run") {
		t.Errorf("expected token-budget-per-run candidate")
	}
}

func TestSuggest_LargeSample_HighConfidence(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := make([]*reporting.RunResults, 0, 60)
	for i := 0; i < 60; i++ {
		start := now.Add(-time.Duration(i) * 24 * time.Hour)
		runs = append(runs, mkRun(start,
			mkTask("/proj/a", "completed", "PR", "https://example/p", 1500, 2*time.Minute),
			mkTask("/proj/a", "completed", "", "", 500, time.Minute),
		))
	}

	got := Suggest(runs, Options{Now: now, Window: 0})
	if len(got) == 0 {
		t.Fatal("expected candidates with 60 runs")
	}
	for _, c := range got {
		if c.Name == "project-availability" {
			continue
		}
		if c.Confidence != ConfidenceHigh {
			t.Errorf("candidate %q: confidence=%s want high (n>=50)", c.Name, c.Confidence)
		}
	}
}

func TestSuggest_AllFailedRuns(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := make([]*reporting.RunResults, 0, 6)
	for i := 0; i < 6; i++ {
		start := now.Add(-time.Duration(i) * 24 * time.Hour)
		runs = append(runs, mkRun(start,
			mkTask("/proj/a", "failed", "", "", 100, 30*time.Second),
		))
	}

	got := Suggest(runs, Options{Now: now})
	// Success rate should still emit but be floored at 90%
	for _, c := range got {
		if c.Name == "task-success-rate" {
			if !strings.Contains(c.Target, "90%") {
				t.Errorf("all-failed: success-rate target should floor at 90%%, got %q", c.Target)
			}
			return
		}
	}
	t.Errorf("expected a task-success-rate candidate even with all failed runs")
}

func TestSuggest_NoPRs(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := make([]*reporting.RunResults, 0, 8)
	for i := 0; i < 8; i++ {
		start := now.Add(-time.Duration(i) * 24 * time.Hour)
		runs = append(runs, mkRun(start,
			mkTask("/proj/a", "completed", "Report", "/tmp/x", 500, 90*time.Second),
		))
	}

	got := Suggest(runs, Options{Now: now})
	if hasCandidate(got, "pr-throughput") {
		t.Errorf("expected no pr-throughput candidate when no PRs were created")
	}
}

func TestSuggest_MissingTokenData(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := make([]*reporting.RunResults, 0, 8)
	for i := 0; i < 8; i++ {
		start := now.Add(-time.Duration(i) * 24 * time.Hour)
		r := mkRun(start,
			mkTask("/proj/a", "completed", "", "", 0, time.Minute),
		)
		r.UsedBudget = 0
		runs = append(runs, r)
	}

	got := Suggest(runs, Options{Now: now})
	if hasCandidate(got, "token-budget-per-run") {
		t.Errorf("expected no token-budget candidate when token data is missing")
	}
}

func TestSuggest_ProjectFilter(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := []*reporting.RunResults{
		mkRun(now.Add(-48*time.Hour),
			mkTask("/code/alpha", "completed", "PR", "https://x/1", 500, time.Minute),
		),
		mkRun(now.Add(-24*time.Hour),
			mkTask("/code/beta", "completed", "PR", "https://x/2", 500, time.Minute),
		),
		mkRun(now.Add(-12*time.Hour),
			mkTask("/code/alpha", "completed", "PR", "https://x/3", 500, time.Minute),
		),
	}

	got := Suggest(runs, Options{Now: now, Project: "alpha"})
	for _, c := range got {
		if c.Name == "project-availability" {
			t.Errorf("per-project availability should not be emitted when filtering to a single project: %+v", c)
		}
	}
}

func TestSuggest_MinConfidenceFilter(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := []*reporting.RunResults{
		mkRun(now.Add(-72*time.Hour),
			mkTask("/p/a", "completed", "PR", "https://x/1", 1000, time.Minute),
		),
		mkRun(now.Add(-48*time.Hour),
			mkTask("/p/a", "completed", "PR", "https://x/2", 1000, time.Minute),
		),
		mkRun(now.Add(-24*time.Hour),
			mkTask("/p/a", "completed", "PR", "https://x/3", 1000, time.Minute),
		),
	}
	got := Suggest(runs, Options{Now: now, MinConfidence: ConfidenceMedium})
	if len(got) != 0 {
		t.Errorf("expected zero candidates after filtering to medium+, got %d", len(got))
	}
}

func TestSuggest_WindowFilter(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	runs := []*reporting.RunResults{
		mkRun(now.Add(-90*24*time.Hour),
			mkTask("/p/a", "completed", "PR", "https://x/1", 1000, time.Minute),
		),
		mkRun(now.Add(-3*24*time.Hour),
			mkTask("/p/a", "completed", "PR", "https://x/2", 1000, time.Minute),
		),
	}
	got := Suggest(runs, Options{Now: now, Window: 7 * 24 * time.Hour})
	// Only 1 run in window → not enough for any candidate (all gates require >=3-5)
	if len(got) != 0 {
		t.Errorf("expected zero candidates with single in-window run, got %d", len(got))
	}
}

func TestPercentile(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	got := percentile(values, 90)
	if got < 89 || got > 96 {
		t.Errorf("p90: got %.2f, want ~90-96", got)
	}
	if v := percentile(nil, 50); v != 0 {
		t.Errorf("p50 of empty: got %.2f want 0", v)
	}
	if v := percentile([]float64{42}, 50); v != 42 {
		t.Errorf("p50 single: got %.2f want 42", v)
	}
}

func TestRoundUpTokens(t *testing.T) {
	cases := []struct {
		in, want int64
	}{
		{0, 0},
		{50, 100},
		{510, 600},
		{1_200, 1_500},
		{12_000, 12_000},
		{12_345, 13_000},
		{120_001, 125_000},
	}
	for _, c := range cases {
		if got := roundUpTokens(c.in); got != c.want {
			t.Errorf("roundUpTokens(%d)=%d want %d", c.in, got, c.want)
		}
	}
}

func hasCandidate(cs []Candidate, name string) bool {
	for _, c := range cs {
		if c.Name == name {
			return true
		}
	}
	return false
}
