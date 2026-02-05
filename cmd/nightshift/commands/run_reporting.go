package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/reporting"
)

type runReport struct {
	results    *reporting.RunResults
	usedBudget int
}

func newRunReport(start time.Time, startBudget int) *runReport {
	return &runReport{
		results: &reporting.RunResults{
			Date:            start,
			StartTime:       start,
			StartBudget:     startBudget,
			UsedBudget:      0,
			RemainingBudget: 0,
			Tasks:           []reporting.TaskResult{},
		},
	}
}

func (r *runReport) addTask(task reporting.TaskResult) {
	r.results.Tasks = append(r.results.Tasks, task)
	r.usedBudget += task.TokensUsed
}

func (r *runReport) finalize(cfg *config.Config, log *logging.Logger) {
	if r == nil || r.results == nil || cfg == nil {
		return
	}

	r.results.EndTime = time.Now()
	r.results.UsedBudget = r.usedBudget
	r.results.RemainingBudget = r.results.StartBudget - r.usedBudget
	if r.results.RemainingBudget < 0 {
		r.results.RemainingBudget = 0
	}

	logPath := ""
	if cfg.ExpandedLogPath() != "" {
		logPath = filepath.Join(cfg.ExpandedLogPath(), fmt.Sprintf("nightshift-%s.log", r.results.StartTime.Format("2006-01-02")))
	}
	r.results.LogPath = logPath

	if cfg.Reporting.MorningSummary {
		gen := reporting.NewGenerator(cfg)
		summary, err := gen.Generate(r.results)
		if err != nil {
			log.Warnf("summary generate: %v", err)
		} else {
			path := reporting.DefaultSummaryPath(r.results.Date)
			if err := gen.Save(summary, path); err != nil {
				log.Warnf("summary save: %v", err)
			} else {
				log.Infof("summary saved: %s", path)
			}
		}
	}

	reportPath := reporting.DefaultRunReportPath(r.results.EndTime)
	if err := reporting.SaveRunReport(r.results, reportPath, r.results.LogPath); err != nil {
		log.Warnf("run report save: %v", err)
	} else {
		log.Infof("run report saved: %s", reportPath)
	}

	resultsPath := reporting.DefaultRunResultsPath(r.results.EndTime)
	if err := reporting.SaveRunResults(r.results, resultsPath); err != nil {
		log.Warnf("run results save: %v", err)
	} else {
		log.Infof("run results saved: %s", resultsPath)
	}
}

func calculateRunBudgetStart(cfg *config.Config, budgetMgr *budget.Manager, log *logging.Logger) int {
	if cfg == nil || budgetMgr == nil {
		return 0
	}
	total := 0
	if cfg.Providers.Claude.Enabled {
		if allowance, err := budgetMgr.CalculateAllowance("claude"); err == nil {
			total += int(allowance.Allowance)
		} else if log != nil {
			log.Warnf("budget claude: %v", err)
		}
	}
	if cfg.Providers.Codex.Enabled {
		if allowance, err := budgetMgr.CalculateAllowance("codex"); err == nil {
			total += int(allowance.Allowance)
		} else if log != nil {
			log.Warnf("budget codex: %v", err)
		}
	}
	return total
}
