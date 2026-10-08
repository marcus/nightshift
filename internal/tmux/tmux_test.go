package tmux

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	outputs [][]byte
	calls   int
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	if len(f.outputs) == 0 {
		return []byte(""), nil
	}
	if f.calls >= len(f.outputs) {
		return f.outputs[len(f.outputs)-1], nil
	}
	out := f.outputs[f.calls]
	f.calls++
	return out, nil
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"sgr color", "hello \x1b[31mred\x1b[0m world", "hello red world"},
		{"cursor movement", "abc\x1b[2Adef", "abcdef"},
		{"osc sequence", "text\x1b]0;title\x07more", "textmore"},
		{"no ansi", "plain text", "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Fatalf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWaitForPattern(t *testing.T) {
	runner := &fakeRunner{
		outputs: [][]byte{
			[]byte("no match"),
			[]byte("still no"),
			[]byte("usage 42%"),
		},
	}
	session := NewSession("test", WithRunner(runner))

	ctx := context.Background()
	out, err := session.WaitForPattern(ctx, regexp.MustCompile(`42%`), 50*time.Millisecond, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForPattern error: %v", err)
	}
	if !strings.Contains(out, "42%") {
		t.Fatalf("WaitForPattern output = %q", out)
	}
}

func TestParseClaudeWeeklyPct(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    float64
		wantErr bool
	}{
		{
			name: "same line format",
			output: `
Current week (all models)   45%
Current week (Sonnet only)  12%
`,
			want: 45,
		},
		{
			name: "multiline real output",
			output: `
Current session
██████████████████████████████████████████  0% used
Resets 8:59pm (America/Los_Angeles)

Current week (all models)
██████████████████████░░░░░░░░░░░░░░░░░░░  59% used
Resets Feb 8 at 9:59am (America/Los_Angeles)

Current week (Sonnet only)
█░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  5% used
Resets Feb 8 at 10:59am (America/Los_Angeles)
`,
			want: 59,
		},
		{
			name:   "multiline with ansi",
			output: "\x1b[1mCurrent week (all models)\x1b[0m\n\x1b[34m████\x1b[0m 72% used",
			want:   72,
		},
		{
			name: "decimal percent",
			output: `
Current week (all models)
████ 72.5% used
`,
			want: 72.5,
		},
		{
			name:    "no match",
			output:  "nothing relevant here",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct, err := parseClaudeWeeklyPct(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got pct=%v", pct)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseClaudeWeeklyPct error: %v", err)
			}
			if pct != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, pct)
			}
		})
	}
}

func TestCountNonEmptyLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"blank lines only", "\n\n\n", 0},
		{"single line", "hello", 1},
		{"mixed", "hello\n\n  \nworld\n", 2},
		{"shell prompt", "$ claude\n", 1},
		{"tui rendered", "Welcome\nWhat can I help with?\n>\ninput area\nstatus bar\nfooter\n", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countNonEmptyLines(tt.input)
			if got != tt.want {
				t.Fatalf("countNonEmptyLines = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCtxSleep(t *testing.T) {
	// Normal sleep completes
	ctx := context.Background()
	start := time.Now()
	err := ctxSleep(ctx, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("ctxSleep error: %v", err)
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("ctxSleep returned too early")
	}

	// Cancelled context returns immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = ctxSleep(ctx, 10*time.Second)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestWaitForSubstantialContent(t *testing.T) {
	// Simulate: first 2 polls return sparse content, third returns TUI
	runner := &fakeRunner{
		outputs: [][]byte{
			// new-session
			[]byte(""),
			// first capture: shell prompt only
			[]byte("$ claude\n"),
			// second capture: still loading
			[]byte("$ claude\nLoading...\n"),
			// third capture: TUI rendered (>5 non-empty lines)
			[]byte("Welcome to Claude Code\nVersion 1.0\nProject: nightshift\n\nWhat can I help with?\n>\nReady\n"),
		},
	}
	session := NewSession("test", WithRunner(runner))

	ctx := context.Background()
	output, err := waitForSubstantialContent(ctx, session, 5*time.Second)
	if err != nil {
		t.Fatalf("waitForSubstantialContent error: %v", err)
	}
	if !strings.Contains(output, "Welcome") {
		t.Fatalf("expected TUI output, got: %q", output)
	}
}

func TestWaitForSubstantialContentTimeout(t *testing.T) {
	// Always returns sparse content -> should timeout
	runner := &fakeRunner{
		outputs: [][]byte{
			// new-session
			[]byte(""),
			// capture always returns sparse
			[]byte("$ claude\n"),
		},
	}
	session := NewSession("test", WithRunner(runner))

	ctx := context.Background()
	_, err := waitForSubstantialContent(ctx, session, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout in error, got: %v", err)
	}
}

func TestParseClaudeResetTimes(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantSession string
		wantWeekly  string
	}{
		{
			name: "real claude output",
			output: `
Current session
██████████████████████████████████████████  0% used
Resets 8:59pm (America/Los_Angeles)

Current week (all models)
██████████████████████░░░░░░░░░░░░░░░░░░░  59% used
Resets Feb 8 at 9:59am (America/Los_Angeles)

Current week (Sonnet only)
█░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  5% used
Resets Feb 8 at 10:59am (America/Los_Angeles)
`,
			wantSession: "8:59pm (America/Los_Angeles)",
			wantWeekly:  "Feb 8 at 9:59am (America/Los_Angeles)",
		},
		{
			name: "simple time format",
			output: `
Current session
██ 0% used
Resets 9pm (America/Los_Angeles)

Current week (all models)
██ 42% used
Resets Feb 8 at 10am (America/Los_Angeles)
`,
			wantSession: "9pm (America/Los_Angeles)",
			wantWeekly:  "Feb 8 at 10am (America/Los_Angeles)",
		},
		{
			name:        "no reset times",
			output:      "nothing relevant here",
			wantSession: "",
			wantWeekly:  "",
		},
		{
			name: "session only",
			output: `
Current session
██ 0% used
Resets 9pm (America/Los_Angeles)
`,
			wantSession: "9pm (America/Los_Angeles)",
			wantWeekly:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, weekly := parseClaudeResetTimes(tt.output)
			if session != tt.wantSession {
				t.Errorf("session reset = %q, want %q", session, tt.wantSession)
			}
			if weekly != tt.wantWeekly {
				t.Errorf("weekly reset = %q, want %q", weekly, tt.wantWeekly)
			}
		})
	}
}

func TestParseCodexResetTimes(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantSession string
		wantWeekly  string
	}{
		{
			name: "real codex output",
			output: `
5h limit:          [████████████████████   ] 71% left (resets 20:15)
Weekly limit:      [████████████████████   ] 77% left (resets 20:08 on 9 Feb)
`,
			wantSession: "20:15",
			wantWeekly:  "20:08 on 9 Feb",
		},
		{
			name: "different times",
			output: `
5h limit: 29% used (resets 01:18)
Weekly limit: 23% used (resets 01:18 on 5 Feb)
`,
			wantSession: "01:18",
			wantWeekly:  "01:18 on 5 Feb",
		},
		{
			name:        "no reset times",
			output:      "nothing relevant here",
			wantSession: "",
			wantWeekly:  "",
		},
		{
			name: "weekly only",
			output: `
Weekly limit: 23% used (resets 01:18 on 5 Feb)
`,
			wantSession: "",
			wantWeekly:  "01:18 on 5 Feb",
		},
		{
			name: "new format with date on 5h line",
			output: `
  5h limit:             [████████████████████] 100% left (resets 02:50 on 8 Feb)
  Weekly limit:         [███░░░░░░░░░░░░░░░░░] 13% left (resets 20:08 on 9 Feb)
`,
			wantSession: "02:50 on 8 Feb",
			wantWeekly:  "20:08 on 9 Feb",
		},
		{
			name: "new format session only",
			output: `
  5h limit:             [████████████████████] 100% left (resets 02:50 on 8 Feb)
`,
			wantSession: "02:50 on 8 Feb",
			wantWeekly:  "",
		},
		{
			name: "fallback weekly when prefix differs",
			output: `some header
(resets 14:30 on 3 Mar)
`,
			wantSession: "",
			wantWeekly:  "14:30 on 3 Mar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, weekly := parseCodexResetTimes(tt.output)
			if session != tt.wantSession {
				t.Errorf("session reset = %q, want %q", session, tt.wantSession)
			}
			if weekly != tt.wantWeekly {
				t.Errorf("weekly reset = %q, want %q", weekly, tt.wantWeekly)
			}
		})
	}
}

func TestParseCodexWeeklyPct(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    float64
		wantErr bool
	}{
		{
			name: "left format converts to used",
			output: `
5h limit:          [████████████████████   ] 71% left (resets 20:15)
Weekly limit:      [████████████████████   ] 77% left (resets 20:08 on 9 Feb)
`,
			want: 23, // 100 - 77 = 23% used
		},
		{
			name: "used format stays as used",
			output: `
5h limit: 29% used
Weekly limit: 23% used
`,
			want: 23,
		},
		{
			name: "no qualifier defaults to raw value",
			output: `
Weekly limit: 30%
`,
			want: 30,
		},
		{
			name: "decimal left converts to used",
			output: `
Weekly limit: 77.5% left
`,
			want: 22.5,
		},
		{
			name: "decimal used stays as used",
			output: `
Weekly limit: 23.5% used
`,
			want: 23.5,
		},
		{
			name:    "no match",
			output:  "nothing relevant here",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct, err := parseCodexWeeklyPct(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got pct=%v", pct)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCodexWeeklyPct error: %v", err)
			}
			if pct != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, pct)
			}
		})
	}
}
