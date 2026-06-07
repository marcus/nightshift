// Package analysis provides code ownership and bus-factor analysis tools.
//
// It extracts commit history from git repositories, computes ownership
// concentration metrics (Herfindahl index, Gini coefficient, bus factor),
// and generates markdown reports with actionable recommendations.
//
// # Metrics
//
//   - Bus Factor: Minimum contributors whose departure would halve project capacity
//   - Herfindahl Index: Ownership concentration (0=diverse, 1=monopoly)
//   - Gini Coefficient: Contribution inequality (0=equal, 1=unequal)
//   - Top N Percent: Cumulative ownership by top contributors
//
// # Persistence
//
// Analysis results can be stored in and loaded from SQLite via the db.go
// functions (Store, LoadLatest, LoadAll).
package analysis
