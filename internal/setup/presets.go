package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/marcus/nightshift/internal/tasks"
)

type Preset string

const (
	PresetBalanced   Preset = "balanced"
	PresetSafe       Preset = "safe"
	PresetAggressive Preset = "aggressive"
)

type RepoSignals struct {
	HasRelease bool
	HasADR     bool
}

// DetectRepoSignals inspects project roots for release and ADR signals.
func DetectRepoSignals(projects []string) RepoSignals {
	signals := RepoSignals{}
	for _, project := range projects {
		if project == "" {
			continue
		}
		if hasAny(project, []string{
			"CHANGELOG.md",
			filepath.Join(".github", "workflows", "release.yml"),
			filepath.Join(".github", "workflows", "release.yaml"),
		}) {
			signals.HasRelease = true
		}
		if hasAny(project, []string{
			filepath.Join("docs", "adr"),
			filepath.Join("docs", "ADR"),
			"adr",
			"ADR",
		}) {
			signals.HasADR = true
		}
	}
	return signals
}

func PresetTasks(preset Preset, defs []tasks.TaskDefinition, signals RepoSignals) map[tasks.TaskType]bool {
	selected := make(map[tasks.TaskType]bool)
	for _, def := range defs {
		if !presetAllowsTask(preset, def, signals) {
			continue
		}
		selected[def.Type] = true
	}
	return selected
}

func presetAllowsTask(preset Preset, def tasks.TaskDefinition, signals RepoSignals) bool {
	if def.Category != tasks.CategoryPR && def.Category != tasks.CategoryAnalysis {
		return false
	}

	switch preset {
	case PresetSafe:
		if def.RiskLevel != tasks.RiskLow {
			return false
		}
		if def.CostTier > tasks.CostMedium {
			return false
		}
	case PresetAggressive:
		// Aggressive includes everything in Balanced plus higher-risk PR work.
		if def.RiskLevel == tasks.RiskHigh {
			return false
		}
	default:
		if def.RiskLevel > tasks.RiskMedium {
			return false
		}
		if def.CostTier > tasks.CostHigh {
			return false
		}
		if isHeavyPR(def.Type) {
			return false
		}
	}

	if isReleaseTask(def.Type) && !signals.HasRelease {
		return false
	}
	if def.Type == tasks.TaskADRDraft && !signals.HasADR {
		return false
	}

	return true
}

func isReleaseTask(t tasks.TaskType) bool {
	switch t {
	case tasks.TaskChangelogSynth, tasks.TaskReleaseNotes:
		return true
	default:
		return false
	}
}

func isHeavyPR(t tasks.TaskType) bool {
	switch t {
	case tasks.TaskBugFinder, tasks.TaskAutoDRY:
		return true
	default:
		return false
	}
}

func hasAny(root string, relPaths []string) bool {
	for _, rel := range relPaths {
		path := filepath.Join(root, rel)
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() || strings.HasSuffix(rel, ".md") || strings.HasSuffix(rel, ".yml") || strings.HasSuffix(rel, ".yaml") {
				return true
			}
		}
	}
	return false
}
