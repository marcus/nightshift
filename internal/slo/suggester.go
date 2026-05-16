// Package slo derives SLO/SLA candidate suggestions from nightshift run history.
//
// The suggester is read-only: it consumes existing reporting.RunResults and
// produces a slice of Candidate values with rationale and a confidence label.
// It does not persist suggestions or change any nightshift configuration.
package slo

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/reporting"
)

// Confidence describes how much sample data the suggestion is built on.
type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

// Category groups candidates for output rendering.
type Category string

const (
	CategoryReliability  Category = "reliability"
	CategoryLatency      Category = "latency"
	CategoryThroughput   Category = "throughput"
	CategoryCost         Category = "cost"
	CategoryAvailability Category = "availability"
)

// Candidate is a single SLO/SLA recommendation.
type Candidate struct {
	Name       string     `json:"name" yaml:"name"`
	Category   Category   `json:"category" yaml:"category"`
	Metric     string     `json:"metric" yaml:"metric"`
	Target     string     `json:"target" yaml:"target"`
	Window     string     `json:"window" yaml:"window"`
	Rationale  string     `json:"rationale" yaml:"rationale"`
	Confidence Confidence `json:"confidence" yaml:"confidence"`
	SampleSize int        `json:"sample_size" yaml:"sample_size"`
	Project    string     `json:"project,omitempty" yaml:"project,omitempty"`
}

// Options configures a Suggester run.
type Options struct {
	// Window is the lookback period applied to RunResults.StartTime.
	// Zero means "use everything".
	Window time.Duration

	// Project filters to a single project (path basename match) when non-empty.
	Project string

	// Now overrides the current time (for tests). Zero means time.Now.
	Now time.Time

	// MinConfidence filters candidates below this level. Empty means low+.
	MinConfidence Confidence
}

// Suggest produces SLO/SLA candidates from the supplied run history.
//
// runs may be in any order; the suggester sorts a working copy by start time.
func Suggest(runs []*reporting.RunResults, opts Options) []Candidate {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	filtered := filterRuns(runs, opts, now)
	if len(filtered) == 0 {
		return nil
	}

	var candidates []Candidate
	candidates = appendIfPresent(candidates, successRateCandidate(filtered, opts))
	candidates = appendIfPresent(candidates, taskLatencyCandidate(filtered, opts))
	candidates = appendIfPresent(candidates, prThroughputCandidate(filtered, opts))
	candidates = appendIfPresent(candidates, tokenBudgetCandidate(filtered, opts))

	if opts.Project == "" {
		candidates = append(candidates, perProjectAvailabilityCandidates(filtered, opts)...)
	}

	if opts.MinConfidence != "" {
		candidates = filterByConfidence(candidates, opts.MinConfidence)
	}

	return candidates
}

func appendIfPresent(out []Candidate, c *Candidate) []Candidate {
	if c == nil {
		return out
	}
	return append(out, *c)
}

func filterRuns(runs []*reporting.RunResults, opts Options, now time.Time) []*reporting.RunResults {
	out := make([]*reporting.RunResults, 0, len(runs))
	var cutoff time.Time
	if opts.Window > 0 {
		cutoff = now.Add(-opts.Window)
	}

	for _, r := range runs {
		if r == nil {
			continue
		}
		if !cutoff.IsZero() && r.StartTime.Before(cutoff) {
			continue
		}
		if opts.Project != "" && !runTouchesProject(r, opts.Project) {
			continue
		}
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartTime.Before(out[j].StartTime)
	})
	return out
}

func runTouchesProject(r *reporting.RunResults, project string) bool {
	want := strings.ToLower(project)
	for _, task := range r.Tasks {
		if task.Project == "" {
			continue
		}
		name := strings.ToLower(filepath.Base(task.Project))
		full := strings.ToLower(task.Project)
		if name == want || full == want {
			return true
		}
	}
	return false
}

func windowLabel(opts Options) string {
	if opts.Window <= 0 {
		return "all-time"
	}
	days := int(math.Round(opts.Window.Hours() / 24))
	if days <= 1 {
		return "last 24h"
	}
	return fmt.Sprintf("rolling %dd", days)
}

func classifyConfidence(n int) Confidence {
	switch {
	case n >= 50:
		return ConfidenceHigh
	case n >= 10:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

func confidenceRank(c Confidence) int {
	switch c {
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	default:
		return 0
	}
}

func filterByConfidence(in []Candidate, min Confidence) []Candidate {
	want := confidenceRank(min)
	if want == 0 {
		return in
	}
	out := in[:0]
	for _, c := range in {
		if confidenceRank(c.Confidence) >= want {
			out = append(out, c)
		}
	}
	return out
}

// successRateCandidate suggests a reliability SLO based on per-run task
// success rate. Target is the p10 of run success rates (i.e., we should beat
// it 90% of the time), floored at 90%.
func successRateCandidate(runs []*reporting.RunResults, opts Options) *Candidate {
	rates := make([]float64, 0, len(runs))
	totalCompleted, totalFailed := 0, 0

	for _, r := range runs {
		var completed, failed int
		for _, task := range r.Tasks {
			switch task.Status {
			case "completed":
				completed++
			case "failed":
				failed++
			}
		}
		denom := completed + failed
		if denom == 0 {
			continue
		}
		totalCompleted += completed
		totalFailed += failed
		rates = append(rates, float64(completed)/float64(denom)*100)
	}

	if len(rates) < 3 {
		return nil
	}

	overall := float64(totalCompleted) / float64(totalCompleted+totalFailed) * 100
	p10 := percentile(rates, 10)
	target := math.Min(p10, overall)
	if target < 90 {
		target = 90
	}
	target = math.Floor(target)

	return &Candidate{
		Name:     "task-success-rate",
		Category: CategoryReliability,
		Metric:   "completed_tasks / (completed_tasks + failed_tasks)",
		Target:   fmt.Sprintf(">= %.0f%%", target),
		Window:   windowLabel(opts),
		Rationale: fmt.Sprintf(
			"observed success rate is %.1f%% (p10 across %d runs: %.1f%%); target floored at 90%%",
			overall, len(rates), p10,
		),
		Confidence: classifyConfidence(len(rates)),
		SampleSize: len(rates),
	}
}

// taskLatencyCandidate suggests a latency SLO based on p95 task duration.
func taskLatencyCandidate(runs []*reporting.RunResults, opts Options) *Candidate {
	durations := make([]float64, 0, 128)
	for _, r := range runs {
		for _, task := range r.Tasks {
			if task.Status != "completed" {
				continue
			}
			if task.Duration <= 0 {
				continue
			}
			durations = append(durations, task.Duration.Seconds())
		}
	}

	if len(durations) < 5 {
		return nil
	}

	p95 := percentile(durations, 95)
	target := time.Duration(math.Ceil(p95)) * time.Second
	target = roundUpDuration(target)

	return &Candidate{
		Name:     "task-completion-latency",
		Category: CategoryLatency,
		Metric:   "completed task wall-clock duration",
		Target:   fmt.Sprintf("p95 <= %s", formatDurationShort(target)),
		Window:   windowLabel(opts),
		Rationale: fmt.Sprintf(
			"observed p95 across %d completed tasks is %s; target rounded up to next round value",
			len(durations), formatDurationShort(time.Duration(math.Round(p95))*time.Second),
		),
		Confidence: classifyConfidence(len(durations)),
		SampleSize: len(durations),
	}
}

// prThroughputCandidate suggests an SLA on PRs per run.
func prThroughputCandidate(runs []*reporting.RunResults, opts Options) *Candidate {
	if len(runs) < 5 {
		return nil
	}

	counts := make([]float64, 0, len(runs))
	runsWithPR := 0
	for _, r := range runs {
		var prs int
		for _, task := range r.Tasks {
			if strings.EqualFold(task.OutputType, "pr") && task.OutputRef != "" {
				prs++
			}
		}
		counts = append(counts, float64(prs))
		if prs > 0 {
			runsWithPR++
		}
	}

	if runsWithPR == 0 {
		return nil
	}

	median := percentile(counts, 50)
	target := math.Floor(median)
	if target < 1 {
		target = 1
	}

	return &Candidate{
		Name:     "pr-throughput",
		Category: CategoryThroughput,
		Metric:   "PRs created per run",
		Target:   fmt.Sprintf(">= %.0f PR(s) per run", target),
		Window:   windowLabel(opts),
		Rationale: fmt.Sprintf(
			"%d/%d runs produced a PR; median PRs/run = %.1f",
			runsWithPR, len(runs), median,
		),
		Confidence: classifyConfidence(len(runs)),
		SampleSize: len(runs),
	}
}

// tokenBudgetCandidate suggests a cost SLO using p90 tokens-per-run as a ceiling.
func tokenBudgetCandidate(runs []*reporting.RunResults, opts Options) *Candidate {
	usage := make([]float64, 0, len(runs))
	for _, r := range runs {
		tokens := r.UsedBudget
		if tokens == 0 {
			for _, task := range r.Tasks {
				tokens += task.TokensUsed
			}
		}
		if tokens <= 0 {
			continue
		}
		usage = append(usage, float64(tokens))
	}

	if len(usage) < 5 {
		return nil
	}

	p90 := percentile(usage, 90)
	target := roundUpTokens(int64(math.Ceil(p90)))

	return &Candidate{
		Name:     "token-budget-per-run",
		Category: CategoryCost,
		Metric:   "tokens used per run",
		Target:   fmt.Sprintf("<= %s tokens/run (p90)", formatTokens(target)),
		Window:   windowLabel(opts),
		Rationale: fmt.Sprintf(
			"observed p90 across %d runs with token data is %s",
			len(usage), formatTokens(int64(math.Round(p90))),
		),
		Confidence: classifyConfidence(len(usage)),
		SampleSize: len(usage),
	}
}

// perProjectAvailabilityCandidates produces one availability SLO per project
// that has at least 3 runs in the window.
func perProjectAvailabilityCandidates(runs []*reporting.RunResults, opts Options) []Candidate {
	type projectState struct {
		days      map[string]struct{}
		successes map[string]struct{}
		runs      int
	}

	projects := map[string]*projectState{}
	for _, r := range runs {
		seenInRun := map[string]bool{}
		for _, task := range r.Tasks {
			if task.Project == "" {
				continue
			}
			name := filepath.Base(task.Project)
			st, ok := projects[name]
			if !ok {
				st = &projectState{
					days:      map[string]struct{}{},
					successes: map[string]struct{}{},
				}
				projects[name] = st
			}
			day := r.StartTime.UTC().Format("2006-01-02")
			st.days[day] = struct{}{}
			if !seenInRun[name] {
				st.runs++
				seenInRun[name] = true
			}
			if task.Status == "completed" {
				st.successes[day] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(projects))
	for n := range projects {
		names = append(names, n)
	}
	sort.Strings(names)

	var out []Candidate
	for _, name := range names {
		st := projects[name]
		nights := len(st.days)
		if nights < 3 {
			continue
		}
		good := len(st.successes)
		pct := float64(good) / float64(nights) * 100
		target := math.Floor(pct)
		if target < 70 {
			target = 70
		}

		out = append(out, Candidate{
			Name:     "project-availability",
			Category: CategoryAvailability,
			Project:  name,
			Metric:   "% of run-days with >= 1 successful task",
			Target:   fmt.Sprintf(">= %.0f%%", target),
			Window:   windowLabel(opts),
			Rationale: fmt.Sprintf(
				"%d/%d run-days produced at least one successful task for %s",
				good, nights, name,
			),
			Confidence: classifyConfidence(nights),
			SampleSize: nights,
		})
	}
	return out
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := (p / 100) * float64(len(sorted)-1)
	low := int(math.Floor(rank))
	high := int(math.Ceil(rank))
	if low == high {
		return sorted[low]
	}
	weight := rank - float64(low)
	return sorted[low]*(1-weight) + sorted[high]*weight
}

func roundUpDuration(d time.Duration) time.Duration {
	switch {
	case d <= 30*time.Second:
		return ((d + 4*time.Second) / (5 * time.Second)) * (5 * time.Second)
	case d <= 5*time.Minute:
		return ((d + 29*time.Second) / (30 * time.Second)) * (30 * time.Second)
	case d <= time.Hour:
		return ((d + 59*time.Second) / time.Minute) * time.Minute
	default:
		return ((d + 4*time.Minute + 59*time.Second) / (5 * time.Minute)) * (5 * time.Minute)
	}
}

func roundUpTokens(n int64) int64 {
	if n <= 0 {
		return 0
	}
	switch {
	case n < 1_000:
		return ((n + 99) / 100) * 100
	case n < 10_000:
		return ((n + 499) / 500) * 500
	case n < 100_000:
		return ((n + 999) / 1_000) * 1_000
	default:
		return ((n + 4_999) / 5_000) * 5_000
	}
}

func formatTokens(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	str := fmt.Sprintf("%d", n)
	var groups []string
	for len(str) > 3 {
		groups = append([]string{str[len(str)-3:]}, groups...)
		str = str[:len(str)-3]
	}
	groups = append([]string{str}, groups...)
	return sign + strings.Join(groups, ",")
}

func formatDurationShort(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(math.Round(d.Seconds())))
	}
	if d < time.Hour {
		mins := int(d / time.Minute)
		secs := int((d % time.Minute) / time.Second)
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d / time.Hour)
	mins := int((d % time.Hour) / time.Minute)
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}
