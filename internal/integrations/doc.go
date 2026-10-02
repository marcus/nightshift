// Package integrations provides readers for external configuration and task
// sources so nightshift can incorporate context that lives outside its own
// database.
//
// Each reader implements a small, uniform surface (Enabled, Name, and Read)
// and is constructed from the project [config]. The built-in readers are:
//
//   - [ClaudeMDReader] extracts project context from claude.md files.
//   - [AgentsMDReader] extracts agent behavior preferences from agents.md /
//     AGENTS.md files.
//   - GitHubReader surfaces GitHub issues as actionable tasks.
//   - TDReader imports tasks from a td task-management store.
//
// Results from individual readers are combined into an [AggregatedResult]:
// per-reader [Result]s, the union of [TaskItem]s and [Hint]s, a combined
// context string suitable for injection into agent prompts, and any non-fatal
// [ReaderError]s encountered along the way.
//
// [config]: github.com/marcus/nightshift/internal/config
package integrations
