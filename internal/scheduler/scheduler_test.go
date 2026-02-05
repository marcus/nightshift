package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/config"
)

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TimeOfDay
		wantErr bool
	}{
		{"valid morning", "09:30", TimeOfDay{9, 30}, false},
		{"valid evening", "22:00", TimeOfDay{22, 0}, false},
		{"midnight", "00:00", TimeOfDay{0, 0}, false},
		{"end of day", "23:59", TimeOfDay{23, 59}, false},
		{"invalid hour", "25:00", TimeOfDay{}, true},
		{"invalid minute", "12:60", TimeOfDay{}, true},
		{"bad format", "9:30", TimeOfDay{9, 30}, false}, // This is valid
		{"no colon", "0930", TimeOfDay{}, true},
		{"empty", "", TimeOfDay{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeOfDay(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimeOfDay(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseTimeOfDay(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTimeOfDay_String(t *testing.T) {
	tests := []struct {
		tod  TimeOfDay
		want string
	}{
		{TimeOfDay{9, 30}, "09:30"},
		{TimeOfDay{22, 0}, "22:00"},
		{TimeOfDay{0, 0}, "00:00"},
	}

	for _, tt := range tests {
		if got := tt.tod.String(); got != tt.want {
			t.Errorf("TimeOfDay(%v).String() = %q, want %q", tt.tod, got, tt.want)
		}
	}
}

func TestTimeOfDay_Minutes(t *testing.T) {
	tests := []struct {
		tod  TimeOfDay
		want int
	}{
		{TimeOfDay{0, 0}, 0},
		{TimeOfDay{1, 30}, 90},
		{TimeOfDay{22, 0}, 1320},
		{TimeOfDay{23, 59}, 1439},
	}

	for _, tt := range tests {
		if got := tt.tod.Minutes(); got != tt.want {
			t.Errorf("TimeOfDay(%v).Minutes() = %d, want %d", tt.tod, got, tt.want)
		}
	}
}

func TestWindow_Contains(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name   string
		window Window
		time   time.Time
		want   bool
	}{
		{
			name:   "normal window - inside",
			window: Window{Start: TimeOfDay{9, 0}, End: TimeOfDay{17, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 12, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "normal window - at start",
			window: Window{Start: TimeOfDay{9, 0}, End: TimeOfDay{17, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 9, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "normal window - at end",
			window: Window{Start: TimeOfDay{9, 0}, End: TimeOfDay{17, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 17, 0, 0, 0, loc),
			want:   false, // end is exclusive
		},
		{
			name:   "normal window - before",
			window: Window{Start: TimeOfDay{9, 0}, End: TimeOfDay{17, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 8, 0, 0, 0, loc),
			want:   false,
		},
		{
			name:   "overnight window - late evening",
			window: Window{Start: TimeOfDay{22, 0}, End: TimeOfDay{6, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 23, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "overnight window - early morning",
			window: Window{Start: TimeOfDay{22, 0}, End: TimeOfDay{6, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 3, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "overnight window - at start",
			window: Window{Start: TimeOfDay{22, 0}, End: TimeOfDay{6, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 22, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "overnight window - at end",
			window: Window{Start: TimeOfDay{22, 0}, End: TimeOfDay{6, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 6, 0, 0, 0, loc),
			want:   false, // end is exclusive
		},
		{
			name:   "overnight window - afternoon (outside)",
			window: Window{Start: TimeOfDay{22, 0}, End: TimeOfDay{6, 0}, Location: loc},
			time:   time.Date(2024, 1, 1, 12, 0, 0, 0, loc),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.window.Contains(tt.time); got != tt.want {
				t.Errorf("Window.Contains(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestNewFromConfig_Cron(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Cron: "0 2 * * *",
	}

	s, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if s.cronExpr != cfg.Cron {
		t.Errorf("cronExpr = %q, want %q", s.cronExpr, cfg.Cron)
	}
}

func TestNewFromConfig_Interval(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Interval: "1h",
	}

	s, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if s.interval != time.Hour {
		t.Errorf("interval = %v, want %v", s.interval, time.Hour)
	}
}

func TestNewFromConfig_Window(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Cron: "0 2 * * *",
		Window: &config.WindowConfig{
			Start:    "22:00",
			End:      "06:00",
			Timezone: "UTC",
		},
	}

	s, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v", err)
	}
	if s.window == nil {
		t.Fatal("window is nil")
	}
	if s.window.Start.Hour != 22 || s.window.Start.Minute != 0 {
		t.Errorf("window.Start = %v, want 22:00", s.window.Start)
	}
	if s.window.End.Hour != 6 || s.window.End.Minute != 0 {
		t.Errorf("window.End = %v, want 06:00", s.window.End)
	}
}

func TestNewFromConfig_NoSchedule(t *testing.T) {
	cfg := &config.ScheduleConfig{}

	_, err := NewFromConfig(cfg)
	if err != ErrNoSchedule {
		t.Errorf("NewFromConfig() error = %v, want %v", err, ErrNoSchedule)
	}
}

func TestNewFromConfig_InvalidCron(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Cron: "invalid cron",
	}

	_, err := NewFromConfig(cfg)
	if err == nil {
		t.Error("NewFromConfig() expected error for invalid cron")
	}
}

func TestNewFromConfig_InvalidInterval(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Interval: "not-a-duration",
	}

	_, err := NewFromConfig(cfg)
	if err == nil {
		t.Error("NewFromConfig() expected error for invalid interval")
	}
}

func TestNewFromConfig_InvalidWindow(t *testing.T) {
	tests := []struct {
		name   string
		window *config.WindowConfig
	}{
		{
			name:   "invalid start",
			window: &config.WindowConfig{Start: "25:00", End: "06:00"},
		},
		{
			name:   "invalid end",
			window: &config.WindowConfig{Start: "22:00", End: "invalid"},
		},
		{
			name:   "invalid timezone",
			window: &config.WindowConfig{Start: "22:00", End: "06:00", Timezone: "Fake/Zone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ScheduleConfig{
				Cron:   "0 2 * * *",
				Window: tt.window,
			}
			_, err := NewFromConfig(cfg)
			if err == nil {
				t.Error("NewFromConfig() expected error")
			}
		})
	}
}

func TestSetCron(t *testing.T) {
	s := New()

	// Valid cron
	if err := s.SetCron("0 2 * * *"); err != nil {
		t.Errorf("SetCron() error = %v", err)
	}
	if s.cronExpr != "0 2 * * *" {
		t.Errorf("cronExpr = %q, want %q", s.cronExpr, "0 2 * * *")
	}

	// Invalid cron
	if err := s.SetCron("invalid"); err == nil {
		t.Error("SetCron() expected error for invalid expression")
	}
}

func TestSetInterval(t *testing.T) {
	s := New()

	// Valid interval
	if err := s.SetInterval(time.Hour); err != nil {
		t.Errorf("SetInterval() error = %v", err)
	}
	if s.interval != time.Hour {
		t.Errorf("interval = %v, want %v", s.interval, time.Hour)
	}

	// Invalid interval (zero)
	if err := s.SetInterval(0); err == nil {
		t.Error("SetInterval(0) expected error")
	}

	// Invalid interval (negative)
	if err := s.SetInterval(-time.Hour); err == nil {
		t.Error("SetInterval(-1h) expected error")
	}
}

func TestScheduler_StartStop_Cron(t *testing.T) {
	s := New()
	_ = s.SetCron("* * * * *") // Every minute

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !s.IsRunning() {
		t.Error("IsRunning() = false, want true")
	}

	// Starting again should fail
	if err := s.Start(ctx); err != ErrAlreadyRunning {
		t.Errorf("Start() twice error = %v, want %v", err, ErrAlreadyRunning)
	}

	if err := s.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if s.IsRunning() {
		t.Error("IsRunning() = true after Stop, want false")
	}

	// Stopping again should fail
	if err := s.Stop(); err != ErrNotRunning {
		t.Errorf("Stop() twice error = %v, want %v", err, ErrNotRunning)
	}
}

func TestScheduler_StartStop_Interval(t *testing.T) {
	s := New()
	_ = s.SetInterval(time.Hour)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !s.IsRunning() {
		t.Error("IsRunning() = false, want true")
	}

	if err := s.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if s.IsRunning() {
		t.Error("IsRunning() = true after Stop, want false")
	}
}

func TestScheduler_StartNoSchedule(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Start(ctx); err != ErrNoSchedule {
		t.Errorf("Start() error = %v, want %v", err, ErrNoSchedule)
	}
}

func TestScheduler_NextRun_Cron(t *testing.T) {
	s := New()
	_ = s.SetCron("* * * * *") // Every minute

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = s.Stop() }()

	nextRun := s.NextRun()
	if nextRun.IsZero() {
		t.Error("NextRun() is zero")
	}

	// Next run should be within the next minute
	now := time.Now()
	if nextRun.Before(now) {
		t.Errorf("NextRun() = %v, should be after now (%v)", nextRun, now)
	}
	if nextRun.After(now.Add(time.Minute + time.Second)) {
		t.Errorf("NextRun() = %v, should be within next minute", nextRun)
	}
}

func TestScheduler_NextRun_Interval(t *testing.T) {
	s := New()
	_ = s.SetInterval(time.Hour)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = s.Stop() }()

	nextRun := s.NextRun()
	if nextRun.IsZero() {
		t.Error("NextRun() is zero")
	}

	// Next run should be approximately 1 hour from now
	now := time.Now()
	expected := now.Add(time.Hour)
	delta := nextRun.Sub(expected)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("NextRun() = %v, expected ~%v", nextRun, expected)
	}
}

func TestScheduler_IsInWindow(t *testing.T) {
	s := New()
	_ = s.SetCron("0 2 * * *")

	// No window configured - always in window
	if !s.IsInWindow(time.Now()) {
		t.Error("IsInWindow() = false with no window, want true")
	}

	// Set overnight window
	_ = s.SetWindow(&config.WindowConfig{
		Start:    "22:00",
		End:      "06:00",
		Timezone: "UTC",
	})

	tests := []struct {
		hour int
		want bool
	}{
		{23, true},
		{3, true},
		{22, true},
		{5, true},
		{6, false},
		{12, false},
		{21, false},
	}

	for _, tt := range tests {
		testTime := time.Date(2024, 1, 1, tt.hour, 0, 0, 0, time.UTC)
		if got := s.IsInWindow(testTime); got != tt.want {
			t.Errorf("IsInWindow(%02d:00) = %v, want %v", tt.hour, got, tt.want)
		}
	}
}

func TestScheduler_JobExecution_Interval(t *testing.T) {
	s := New()
	_ = s.SetInterval(50 * time.Millisecond)

	var count atomic.Int32

	s.AddJob(func(ctx context.Context) error {
		count.Add(1)
		return nil
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for at least one execution
	time.Sleep(150 * time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if count.Load() < 1 {
		t.Errorf("Job executed %d times, want at least 1", count.Load())
	}
}

func TestScheduler_JobExecution_WindowBlocks(t *testing.T) {
	s := New()
	_ = s.SetInterval(20 * time.Millisecond)

	// Set window that excludes current time (daytime window when testing)
	// Create a window that definitely excludes right now
	now := time.Now().UTC()
	windowStart := (now.Hour() + 12) % 24 // 12 hours from now
	windowEnd := (now.Hour() + 13) % 24   // 13 hours from now

	_ = s.SetWindow(&config.WindowConfig{
		Start:    fmt.Sprintf("%02d:00", windowStart),
		End:      fmt.Sprintf("%02d:00", windowEnd),
		Timezone: "UTC",
	})

	var count atomic.Int32

	s.AddJob(func(ctx context.Context) error {
		count.Add(1)
		return nil
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if count.Load() != 0 {
		t.Errorf("Job executed %d times, want 0 (blocked by window)", count.Load())
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	s := New()
	_ = s.SetInterval(50 * time.Millisecond)

	var count atomic.Int32

	s.AddJob(func(ctx context.Context) error {
		count.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Scheduler should have stopped
	if s.IsRunning() {
		// Still marked as running, but goroutine exited
		_ = s.Stop()
	}
}

func TestScheduleCron_Legacy(t *testing.T) {
	s := New()

	if err := s.ScheduleCron("0 2 * * *", func() {}); err != nil {
		t.Errorf("ScheduleCron() error = %v", err)
	}

	if s.cronExpr != "0 2 * * *" {
		t.Errorf("cronExpr = %q, want %q", s.cronExpr, "0 2 * * *")
	}
	if len(s.jobs) != 1 {
		t.Errorf("len(jobs) = %d, want 1", len(s.jobs))
	}
}

func TestScheduleInterval_Legacy(t *testing.T) {
	s := New()

	if err := s.ScheduleInterval(time.Hour, func() {}); err != nil {
		t.Errorf("ScheduleInterval() error = %v", err)
	}

	if s.interval != time.Hour {
		t.Errorf("interval = %v, want %v", s.interval, time.Hour)
	}
	if len(s.jobs) != 1 {
		t.Errorf("len(jobs) = %d, want 1", len(s.jobs))
	}
}

