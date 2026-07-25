// Package budget implements token budget calculation and allocation for
// nightshift runs.
//
// Given a provider and its recent usage, the package computes how many tokens
// may be safely spent on the next run. It supports two modes:
//
//   - daily: a fixed per-day allowance.
//   - weekly: a rolling weekly budget with reserve holding and an aggressive
//     end-of-week multiplier so leftover budget is spent down before reset.
//
// The primary result type is [AllowanceResult], which carries the final token
// allowance together with the metadata used to derive it (used percentage and
// its source, predicted remaining daytime usage, reserve amount, mode, and the
// confidence/sample count of the underlying budget estimate).
//
// Usage data is supplied through provider-specific interfaces
// ([ClaudeUsageProvider], [CodexUsageProvider], [CopilotUsageProvider]), all
// extending the common [UsageProvider]. Budget estimates may come from config,
// the API, or the [calibrator] via the [BudgetSource] interface, which returns
// a [BudgetEstimate] annotated with source, confidence, sample count, and
// variance.
//
// [calibrator]: github.com/marcus/nightshift/internal/calibrator
package budget
