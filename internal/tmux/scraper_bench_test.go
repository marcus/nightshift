package tmux

import "testing"

// Representative Claude /usage output for benchmarks.
var claudeUsageOutput = `
Current session
  ████████████████████░░░░░░░░ 0% used
  Resets 9pm (America/Los_Angeles)

Current week (all models)
  ████████████████████░░░░░░░░ 59% used
  Resets Feb 8 at 10am (America/Los_Angeles)
`

// Representative Codex /status output for benchmarks.
var codexStatusOutput = `
5h limit:     [████████████████████░░░░] 100% left (resets 02:50 on 8 Feb)
Weekly limit: [██████░░░░░░░░░░░░░░░░░░] 77% left (resets 20:08 on 9 Feb)
`

func BenchmarkParseClaudeWeeklyPct(b *testing.B) {
	for b.Loop() {
		_, _ = parseClaudeWeeklyPct(claudeUsageOutput)
	}
}

func BenchmarkParseCodexWeeklyPct(b *testing.B) {
	for b.Loop() {
		_, _ = parseCodexWeeklyPct(codexStatusOutput)
	}
}

func BenchmarkParseClaudeResetTimes(b *testing.B) {
	for b.Loop() {
		_, _ = parseClaudeResetTimes(claudeUsageOutput)
	}
}

func BenchmarkParseCodexResetTimes(b *testing.B) {
	for b.Loop() {
		_, _ = parseCodexResetTimes(codexStatusOutput)
	}
}
