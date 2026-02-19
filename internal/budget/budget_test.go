package budget

import (
	"testing"
	"time"

	"github.com/marcus/nightshift/internal/config"
)

// mockClaudeProvider implements ClaudeUsageProvider for testing.
type mockClaudeProvider struct {
	usedPercent float64
	err         error
	source      string
}

func (m *mockClaudeProvider) Name() string { return "claude" }
func (m *mockClaudeProvider) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	return m.usedPercent, m.err
}
func (m *mockClaudeProvider) LastUsedPercentSource() string {
	return m.source
}

// mockCodexProvider implements CodexUsageProvider for testing.
type mockCodexProvider struct {
	usedPercent float64
	resetTime   time.Time
	err         error
}

func (m *mockCodexProvider) Name() string { return "codex" }
func (m *mockCodexProvider) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	return m.usedPercent, m.err
}
func (m *mockCodexProvider) GetResetTime(mode string) (time.Time, error) {
	return m.resetTime, m.err
}

// mockCopilotProvider implements CopilotUsageProvider for testing.
type mockCopilotProvider struct {
	usedPercent float64
	resetTime   time.Time
	err         error
}

func (m *mockCopilotProvider) Name() string { return "copilot" }
func (m *mockCopilotProvider) GetUsedPercent(mode string, monthlyLimit int64) (float64, error) {
	return m.usedPercent, m.err
}
func (m *mockCopilotProvider) GetResetTime(mode string) (time.Time, error) {
	return m.resetTime, m.err
}

type mockBudgetSource struct {
	estimate BudgetEstimate
	err      error
}

func (m *mockBudgetSource) GetBudget(provider string) (BudgetEstimate, error) {
	return m.estimate, m.err
}

type mockTrendAnalyzer struct {
	predicted int64
	err       error
}

func (m *mockTrendAnalyzer) PredictDaytimeUsage(provider string, now time.Time, weeklyBudget int64) (int64, error) {
	return m.predicted, m.err
}

func TestCalculateAllowance_DailyMode(t *testing.T) {
	tests := []struct {
		name           string
		weeklyBudget   int
		maxPercent     int
		reservePercent int
		usedPercent    float64
		wantAllowance  int64
	}{
		{
			name:           "fresh day no usage",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 5,
			usedPercent:    0,
			// daily=100000, available=100000, allowance=10000, reserve=5000, final=5000
			wantAllowance: 5000,
		},
		{
			name:           "50% used today",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 5,
			usedPercent:    50,
			// daily=100000, available=50000, allowance=5000, reserve=5000, final=0
			wantAllowance: 0,
		},
		{
			name:           "20% used today",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 5,
			usedPercent:    20,
			// daily=100000, available=80000, allowance=8000, reserve=5000, final=3000
			wantAllowance: 3000,
		},
		{
			name:           "no reserve",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 0,
			usedPercent:    0,
			// daily=100000, available=100000, allowance=10000, reserve=0, final=10000
			wantAllowance: 10000,
		},
		{
			name:           "high max percent",
			weeklyBudget:   700000,
			maxPercent:     50,
			reservePercent: 0,
			usedPercent:    0,
			// daily=100000, available=100000, allowance=50000, reserve=0, final=50000
			wantAllowance: 50000,
		},
		{
			name:           "fully used day",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 5,
			usedPercent:    100,
			// daily=100000, available=0, allowance=0, reserve=5000, final=0 (capped at 0)
			wantAllowance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:           "daily",
					WeeklyTokens:   tt.weeklyBudget,
					MaxPercent:     tt.maxPercent,
					ReservePercent: tt.reservePercent,
				},
			}

			claude := &mockClaudeProvider{usedPercent: tt.usedPercent}
			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, claude, nil, copilot)

			result, err := mgr.CalculateAllowance("claude")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Allowance != tt.wantAllowance {
				t.Errorf("allowance = %d, want %d", result.Allowance, tt.wantAllowance)
			}
			if result.Mode != "daily" {
				t.Errorf("mode = %s, want daily", result.Mode)
			}
		})
	}
}

func TestCalculateAllowance_WeeklyMode(t *testing.T) {
	// Fix time to Tuesday for predictable remainingDays (5 days until Sunday)
	fixedTime := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC) // Tuesday

	tests := []struct {
		name           string
		weeklyBudget   int
		maxPercent     int
		reservePercent int
		usedPercent    float64
		aggressive     bool
		remainingDays  int
		wantAllowance  int64
		wantMultiplier float64
	}{
		{
			name:           "fresh week",
			weeklyBudget:   700000,
			maxPercent:     10,
			reservePercent: 5,
			usedPercent:    0,
			aggressive:     false,
			remainingDays:  5,
			// remaining=700000, perDay=140000, allowance=14000, reserve=35000, final=0 (negative becomes 0)
			wantAllowance:  0,
			wantMultiplier: 1.0,
		},
		{
			name:           "mid-week 30% used",
			weeklyBudget:   700000,
			maxPercent:     20,
			reservePercent: 0,
			usedPercent:    30,
			aggressive:     false,
			remainingDays:  5,
			// remaining=490000, perDay=98000, allowance=19600, reserve=0, final=19599 (rounding)
			wantAllowance:  19599,
			wantMultiplier: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:                "weekly",
					WeeklyTokens:        tt.weeklyBudget,
					MaxPercent:          tt.maxPercent,
					ReservePercent:      tt.reservePercent,
					AggressiveEndOfWeek: tt.aggressive,
				},
			}

			claude := &mockClaudeProvider{usedPercent: tt.usedPercent}
			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, claude, nil, copilot)
			mgr.nowFunc = func() time.Time { return fixedTime }

			result, err := mgr.CalculateAllowance("claude")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Allowance != tt.wantAllowance {
				t.Errorf("allowance = %d, want %d", result.Allowance, tt.wantAllowance)
			}
			if result.Mode != "weekly" {
				t.Errorf("mode = %s, want weekly", result.Mode)
			}
			if result.Multiplier != tt.wantMultiplier {
				t.Errorf("multiplier = %f, want %f", result.Multiplier, tt.wantMultiplier)
			}
		})
	}
}

func TestAggressiveEndOfWeek(t *testing.T) {
	tests := []struct {
		name           string
		dayOfWeek      time.Weekday
		aggressive     bool
		wantMultiplier float64
	}{
		{
			name:           "friday not aggressive",
			dayOfWeek:      time.Friday,
			aggressive:     false,
			wantMultiplier: 1.0,
		},
		{
			name:           "friday aggressive (2 days left)",
			dayOfWeek:      time.Friday,
			aggressive:     true,
			wantMultiplier: 1.0, // 3-2=1, remaining=2 so multiplier=1
		},
		{
			name:           "saturday aggressive (1 day left)",
			dayOfWeek:      time.Saturday,
			aggressive:     true,
			wantMultiplier: 2.0, // 3-1=2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set time to the specified day
			var fixedTime time.Time
			switch tt.dayOfWeek {
			case time.Friday:
				fixedTime = time.Date(2024, 1, 19, 12, 0, 0, 0, time.UTC)
			case time.Saturday:
				fixedTime = time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)
			}

			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:                "weekly",
					WeeklyTokens:        700000,
					MaxPercent:          10,
					ReservePercent:      0,
					AggressiveEndOfWeek: tt.aggressive,
				},
			}

			claude := &mockClaudeProvider{usedPercent: 0}
			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, claude, nil, copilot)
			mgr.nowFunc = func() time.Time { return fixedTime }

			result, err := mgr.CalculateAllowance("claude")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Multiplier != tt.wantMultiplier {
				t.Errorf("multiplier = %f, want %f", result.Multiplier, tt.wantMultiplier)
			}
		})
	}
}

func TestCalculateAllowance_PredictedUsage(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:           "daily",
			WeeklyTokens:   700000,
			MaxPercent:     10,
			ReservePercent: 0,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 0}
	mgr := NewManager(cfg, claude, nil, nil, WithTrendAnalyzer(&mockTrendAnalyzer{predicted: 2000}))

	result, err := mgr.CalculateAllowance("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowance != 8000 {
		t.Fatalf("allowance = %d, want %d", result.Allowance, 8000)
	}
	if result.AllowanceNoDaytime != 10000 {
		t.Fatalf("allowance no daytime = %d, want %d", result.AllowanceNoDaytime, 10000)
	}
	if result.PredictedUsage != 2000 {
		t.Fatalf("predicted usage = %d, want %d", result.PredictedUsage, 2000)
	}
}

func TestDaysUntilWeeklyReset_Claude(t *testing.T) {
	tests := []struct {
		dayOfWeek time.Weekday
		wantDays  int
	}{
		{time.Sunday, 7},
		{time.Monday, 6},
		{time.Tuesday, 5},
		{time.Wednesday, 4},
		{time.Thursday, 3},
		{time.Friday, 2},
		{time.Saturday, 1},
	}

	for _, tt := range tests {
		t.Run(tt.dayOfWeek.String(), func(t *testing.T) {
			// Create a date that falls on the specified day
			// Jan 14, 2024 is a Sunday
			baseDate := time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC)
			fixedTime := baseDate.AddDate(0, 0, int(tt.dayOfWeek))

			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:         "weekly",
					WeeklyTokens: 700000,
				},
			}

			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, nil, nil, copilot)
			mgr.nowFunc = func() time.Time { return fixedTime }

			days, err := mgr.DaysUntilWeeklyReset("claude")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if days != tt.wantDays {
				t.Errorf("days = %d, want %d (day=%s)", days, tt.wantDays, tt.dayOfWeek)
			}
		})
	}
}

func TestDaysUntilWeeklyReset_Codex(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		resetTime time.Time
		wantDays  int
	}{
		{
			name:      "3 days until reset",
			resetTime: now.Add(72 * time.Hour),
			wantDays:  3,
		},
		{
			name:      "6 hours until reset",
			resetTime: now.Add(6 * time.Hour),
			wantDays:  1,
		},
		{
			name:      "past reset time",
			resetTime: now.Add(-1 * time.Hour),
			wantDays:  1,
		},
		{
			name:      "zero reset time",
			resetTime: time.Time{},
			wantDays:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:         "weekly",
					WeeklyTokens: 700000,
				},
			}

			codex := &mockCodexProvider{resetTime: tt.resetTime}
			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, nil, codex, copilot)
			mgr.nowFunc = func() time.Time { return now }

			days, err := mgr.DaysUntilWeeklyReset("codex")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if days != tt.wantDays {
				t.Errorf("days = %d, want %d", days, tt.wantDays)
			}
		})
	}
}

func TestPerProviderBudget(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:         "daily",
			WeeklyTokens: 700000,
			MaxPercent:   10,
			PerProvider: map[string]int{
				"claude": 350000, // Half the default
			},
		},
	}

	claude := &mockClaudeProvider{usedPercent: 0}
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, claude, nil, copilot)

	result, err := mgr.CalculateAllowance("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// daily = 350000/7 = 50000
	// allowance = 50000 * 0.10 = 5000
	// reserve = 50000 * 0.05 = 2500
	// final = 5000 - 2500 = 2500
	expectedDaily := int64(50000)
	if result.BudgetBase != expectedDaily {
		t.Errorf("BudgetBase = %d, want %d", result.BudgetBase, expectedDaily)
	}
}

func TestBudgetSourceOverridesConfig(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:         "weekly",
			WeeklyTokens: 100000,
			MaxPercent:   10,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 0}
	source := &mockBudgetSource{estimate: BudgetEstimate{
		WeeklyTokens: 700000,
		Source:       "calibrated",
		Confidence:   "high",
		SampleCount:  6,
	}}

	mgr := NewManager(cfg, claude, nil, nil, WithBudgetSource(source))
	result, err := mgr.CalculateAllowance("claude")
	if err != nil {
		t.Fatalf("CalculateAllowance error: %v", err)
	}

	if result.BudgetSource != "calibrated" {
		t.Fatalf("BudgetSource = %s", result.BudgetSource)
	}
	if result.BudgetConfidence != "high" {
		t.Fatalf("BudgetConfidence = %s", result.BudgetConfidence)
	}
	if result.BudgetSampleCount != 6 {
		t.Fatalf("BudgetSampleCount = %d", result.BudgetSampleCount)
	}
}

func TestBudgetSourceFallbacksToConfig(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:         "weekly",
			WeeklyTokens: 100000,
			MaxPercent:   10,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 0}
	source := &mockBudgetSource{estimate: BudgetEstimate{
		WeeklyTokens: 0,
	}}

	mgr := NewManager(cfg, claude, nil, nil, WithBudgetSource(source))
	result, err := mgr.CalculateAllowance("claude")
	if err != nil {
		t.Fatalf("CalculateAllowance error: %v", err)
	}

	if result.BudgetSource != "config" {
		t.Fatalf("BudgetSource = %s", result.BudgetSource)
	}
}

func TestCanRun(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:           "daily",
			WeeklyTokens:   700000,
			MaxPercent:     10,
			ReservePercent: 0,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 0}
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, claude, nil, copilot)

	// Available: 10000 tokens (100000 * 10%)
	tests := []struct {
		estimated int64
		canRun    bool
	}{
		{5000, true},
		{10000, true},
		{10001, false},
		{50000, false},
	}

	for _, tt := range tests {
		canRun, err := mgr.CanRun("claude", tt.estimated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if canRun != tt.canRun {
			t.Errorf("CanRun(%d) = %v, want %v", tt.estimated, canRun, tt.canRun)
		}
	}
}

func TestSummary(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:           "daily",
			WeeklyTokens:   700000,
			MaxPercent:     10,
			ReservePercent: 5,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 25}
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, claude, nil, copilot)

	summary, err := mgr.Summary("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary == "" {
		t.Error("summary should not be empty")
	}

	// Should contain key info
	if !contains(summary, "claude") {
		t.Error("summary should contain provider name")
	}
	if !contains(summary, "25.0%") {
		t.Error("summary should contain used percent")
	}
}

func TestCalculateAllowance_UsedPercentSource(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:           "daily",
			WeeklyTokens:   700000,
			MaxPercent:     10,
			ReservePercent: 5,
		},
	}

	claude := &mockClaudeProvider{usedPercent: 25, source: "jsonl-fallback"}
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, claude, nil, copilot)

	result, err := mgr.CalculateAllowance("claude")
	if err != nil {
		t.Fatalf("CalculateAllowance error: %v", err)
	}
	if result.UsedPercentSource != "jsonl-fallback" {
		t.Fatalf("UsedPercentSource = %q, want %q", result.UsedPercentSource, "jsonl-fallback")
	}
}

func TestCalculateAllowance_Codex(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:           "daily",
			WeeklyTokens:   700000,
			MaxPercent:     10,
			ReservePercent: 0,
			PerProvider:    map[string]int{"codex": 500000},
		},
	}

	codex := &mockCodexProvider{usedPercent: 24} // 24% used (from scraped data)
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, nil, codex, copilot)

	result, err := mgr.CalculateAllowance("codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// daily = 500000/7 = 71428
	// available = 71428 * (1 - 0.24) = 54285
	// allowance = 54285 * 0.10 = 5428
	if result.Allowance <= 0 {
		t.Fatalf("expected positive allowance for codex, got %d", result.Allowance)
	}
	if result.Mode != "daily" {
		t.Fatalf("mode = %s, want daily", result.Mode)
	}
	if result.UsedPercent != 24 {
		t.Fatalf("used percent = %f, want 24", result.UsedPercent)
	}
}

func TestGetUsedPercent_Errors(t *testing.T) {
	cfg := &config.Config{
		Budget: config.BudgetConfig{
			Mode:         "daily",
			WeeklyTokens: 700000,
		},
	}

	// Test missing claude provider
	copilot := &mockCopilotProvider{usedPercent: 0}
	mgr := NewManager(cfg, nil, nil, copilot)
	_, err := mgr.GetUsedPercent("claude")
	if err == nil {
		t.Error("expected error for missing claude provider")
	}

	// Test missing codex provider
	_, err = mgr.GetUsedPercent("codex")
	if err == nil {
		t.Error("expected error for missing codex provider")
	}

	// Test unknown provider
	_, err = mgr.GetUsedPercent("unknown")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestTracker_BackwardCompat(t *testing.T) {
	tracker := NewTracker(10000) // 100 dollars in cents

	tracker.Record("claude", 1000, 500)
	tracker.Record("codex", 500, 200)

	remaining := tracker.Remaining()
	expected := int64(10000 - 500 - 200)
	if remaining != expected {
		t.Errorf("Remaining() = %d, want %d", remaining, expected)
	}
}

func TestReserveEnforcement(t *testing.T) {
	tests := []struct {
		name           string
		reservePercent int
		maxPercent     int
		usedPercent    float64
		wantPositive   bool
	}{
		{
			name:           "reserve larger than allowance",
			reservePercent: 20,
			maxPercent:     10,
			usedPercent:    0,
			wantPositive:   false, // reserve 20000 > allowance 10000
		},
		{
			name:           "reserve smaller than allowance",
			reservePercent: 2,
			maxPercent:     10,
			usedPercent:    0,
			wantPositive:   true, // reserve 2000 < allowance 10000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Budget: config.BudgetConfig{
					Mode:           "daily",
					WeeklyTokens:   700000,
					MaxPercent:     tt.maxPercent,
					ReservePercent: tt.reservePercent,
				},
			}

			claude := &mockClaudeProvider{usedPercent: tt.usedPercent}
			copilot := &mockCopilotProvider{usedPercent: 0}
			mgr := NewManager(cfg, claude, nil, copilot)

			result, err := mgr.CalculateAllowance("claude")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotPositive := result.Allowance > 0
			if gotPositive != tt.wantPositive {
				t.Errorf("allowance positive = %v, want %v (allowance=%d)", gotPositive, tt.wantPositive, result.Allowance)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
