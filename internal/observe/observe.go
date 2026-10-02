// Package observe provides lightweight, thread-safe metrics collection
// for nightshift orchestration runs. Counters, gauges, and duration
// histograms are accumulated in memory and flushed as a single structured
// JSON log entry via zerolog at the end of each run.
package observe

import (
	"sync"
	"time"

	"github.com/marcus/nightshift/internal/logging"
)

// Collector accumulates metrics during a run and flushes them as a
// single structured log entry. All methods are safe for concurrent use.
// A nil *Collector is valid and all operations are no-ops, so callers
// can skip nil checks.
type Collector struct {
	mu        sync.Mutex
	counters  map[string]int64
	gauges    map[string]float64
	durations map[string][]time.Duration
}

// New returns a new Collector ready for use.
func New() *Collector {
	return &Collector{
		counters:  make(map[string]int64),
		gauges:    make(map[string]float64),
		durations: make(map[string][]time.Duration),
	}
}

// Counter increments a named counter by delta.
func (c *Collector) Counter(name string, delta int64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.counters[name] += delta
	c.mu.Unlock()
}

// Gauge sets a named gauge to the given value.
func (c *Collector) Gauge(name string, value float64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.gauges[name] = value
	c.mu.Unlock()
}

// Duration records a duration sample for the named histogram.
func (c *Collector) Duration(name string, d time.Duration) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.durations[name] = append(c.durations[name], d)
	c.mu.Unlock()
}

// durationStats summarises a slice of durations.
type durationStats struct {
	Count int     `json:"count"`
	MinMs float64 `json:"min_ms"`
	MaxMs float64 `json:"max_ms"`
	AvgMs float64 `json:"avg_ms"`
	SumMs float64 `json:"sum_ms"`
}

func computeStats(ds []time.Duration) durationStats {
	if len(ds) == 0 {
		return durationStats{}
	}
	min := ds[0]
	max := ds[0]
	var sum time.Duration
	for _, d := range ds {
		sum += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	return durationStats{
		Count: len(ds),
		MinMs: float64(min.Milliseconds()),
		MaxMs: float64(max.Milliseconds()),
		AvgMs: float64(sum.Milliseconds()) / float64(len(ds)),
		SumMs: float64(sum.Milliseconds()),
	}
}

// Snapshot returns a copy of all collected metrics as a plain map.
func (c *Collector) Snapshot() map[string]any {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string]any, len(c.counters)+len(c.gauges)+len(c.durations))
	for k, v := range c.counters {
		out[k] = v
	}
	for k, v := range c.gauges {
		out[k] = v
	}
	for k, ds := range c.durations {
		out[k] = computeStats(ds)
	}
	return out
}

// Flush emits all collected metrics as a single structured zerolog info
// entry and resets the collector. Subsequent calls will emit empty data
// until new metrics are recorded.
func (c *Collector) Flush(logger *logging.Logger) {
	if c == nil || logger == nil {
		return
	}
	snap := c.Snapshot()
	if len(snap) == 0 {
		return
	}

	logger.InfoCtx("run metrics", snap)

	// Reset
	c.mu.Lock()
	c.counters = make(map[string]int64)
	c.gauges = make(map[string]float64)
	c.durations = make(map[string][]time.Duration)
	c.mu.Unlock()
}
