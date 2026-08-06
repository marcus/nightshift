// Package projects handles multi-project discovery, resolution, and budget
// allocation for nightshift.
//
// A nightshift invocation can target more than one repository. This package
// discovers candidate projects ([DiscoverProjectsInDir], [IsProjectPath]),
// expands glob patterns while honouring excludes ([ExpandGlobPatterns]), and
// merges per-project configuration over the global [config.Config]
// ([MergeProjectConfig]).
//
// A resolved [Project] carries its absolute path, priority, merged config, and
// a normalized budget weight. Selection helpers choose what to work on next
// based on priority and staleness against the [state] store: [SelectNext],
// [SortByPriority], [FilterProcessedToday], and [FilterNotProcessedSince].
//
// Budget is distributed across the selected projects by [AllocateBudget],
// which returns a [BudgetAllocation] per project describing its token share
// and percentage of the total.
//
// [config]: github.com/marcus/nightshift/internal/config
// [state]: github.com/marcus/nightshift/internal/state
package projects
