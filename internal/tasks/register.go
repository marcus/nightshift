// register.go converts custom task configs into TaskDefinitions and registers them.

package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/config"
)

// RegisterCustomTasksFromConfig converts custom task configs into TaskDefinitions
// and registers them. If any registration fails, previously registered tasks from
// this call are rolled back.
func RegisterCustomTasksFromConfig(customs []config.CustomTaskConfig) error {
	var registered []TaskType
	for _, c := range customs {
		cat := parseCategoryString(c.Category)
		cost := parseCostTierString(c.CostTier)
		risk := parseRiskLevelString(c.RiskLevel)

		interval := DefaultIntervalForCategory(cat)
		if c.Interval != "" {
			d, err := time.ParseDuration(c.Interval)
			if err != nil {
				// Rollback
				for _, t := range registered {
					UnregisterCustom(t)
				}
				return fmt.Errorf("custom task %q: invalid interval %q: %w", c.Type, c.Interval, err)
			}
			interval = d
		}

		def := TaskDefinition{
			Type:            TaskType(c.Type),
			Category:        cat,
			Name:            c.Name,
			Description:     c.Description,
			CostTier:        cost,
			RiskLevel:       risk,
			DefaultInterval: interval,
		}

		if err := RegisterCustom(def); err != nil {
			for _, t := range registered {
				UnregisterCustom(t)
			}
			return fmt.Errorf("custom task %q: %w", c.Type, err)
		}
		registered = append(registered, TaskType(c.Type))
	}
	return nil
}

// parseCategoryString maps a config category string to TaskCategory.
// Defaults to CategoryAnalysis if empty or unrecognized.
func parseCategoryString(s string) TaskCategory {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pr":
		return CategoryPR
	case "analysis":
		return CategoryAnalysis
	case "options":
		return CategoryOptions
	case "safe":
		return CategorySafe
	case "map":
		return CategoryMap
	case "emergency":
		return CategoryEmergency
	default:
		return CategoryAnalysis
	}
}

// parseCostTierString maps a config cost tier string to CostTier.
// Defaults to CostMedium if empty or unrecognized.
func parseCostTierString(s string) CostTier {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return CostLow
	case "medium":
		return CostMedium
	case "high":
		return CostHigh
	case "very-high":
		return CostVeryHigh
	default:
		return CostMedium
	}
}

// parseRiskLevelString maps a config risk level string to RiskLevel.
// Defaults to RiskLow if empty or unrecognized.
func parseRiskLevelString(s string) RiskLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return RiskLow
	case "medium":
		return RiskMedium
	case "high":
		return RiskHigh
	default:
		return RiskLow
	}
}
