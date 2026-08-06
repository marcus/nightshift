package providers_test

import (
	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/snapshots"
)

// Compile-time interface satisfaction assertions for cross-package contracts.
// These live in a _test.go file to avoid circular imports between providers,
// budget, and snapshots packages.

// Claude implements budget and snapshot interfaces.
var (
	_ budget.ClaudeUsageProvider       = (*providers.Claude)(nil)
	_ budget.UsedPercentSourceProvider = (*providers.Claude)(nil)
	_ snapshots.ClaudeUsage            = (*providers.Claude)(nil)
)

// Codex implements budget and snapshot interfaces.
var (
	_ budget.CodexUsageProvider = (*providers.Codex)(nil)
	_ snapshots.CodexUsage      = (*providers.Codex)(nil)
)

// Copilot implements budget and snapshot interfaces.
var (
	_ budget.CopilotUsageProvider = (*providers.Copilot)(nil)
	_ snapshots.CopilotUsage      = (*providers.Copilot)(nil)
)
