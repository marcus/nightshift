// Package setup provides interactive configuration and task-preset selection
// for new nightshift projects.
//
// When a project is initialized, setup helps the user choose which task types
// to enable by offering curated profiles. A [Preset] names one of these
// profiles — [PresetSafe], [PresetBalanced], or [PresetAggressive] — trading
// autonomy for caution.
//
// [PresetTasks] resolves a preset (combined with the available
// [tasks.TaskDefinition]s and detected [RepoSignals]) into the concrete set of
// enabled task types. [DetectRepoSignals] inspects project roots for signals
// such as release notes or architecture-decision records (ADR) that should
// influence which tasks make sense, and [RepoSignals] carries the result.
//
// [tasks]: github.com/marcus/nightshift/internal/tasks
package setup
