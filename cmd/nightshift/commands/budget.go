package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/trends"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show budget status",
	Long: `Display current budget status and usage.

Shows spending across all providers or a specific provider.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		return runBudget(provider)
	},
}

func init() {
	budgetCmd.Flags().StringP("provider", "p", "", "Show specific provider status (claude, codex)")
	rootCmd.AddCommand(budgetCmd)
}

func runBudget(filterProvider string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer database.Close()

	// Initialize providers
	var claude *providers.Claude
	var codex *providers.Codex

	if cfg.Providers.Claude.Enabled {
		dataPath := cfg.ExpandedProviderPath("claude")
		if dataPath != "" {
			claude = providers.NewClaudeWithPath(dataPath)
		} else {
			claude = providers.NewClaude()
		}
	}

	if cfg.Providers.Codex.Enabled {
		dataPath := cfg.ExpandedProviderPath("codex")
		if dataPath != "" {
			codex = providers.NewCodexWithPath(dataPath)
		} else {
			codex = providers.NewCodex()
		}
	}

	// Create budget manager
	cal := calibrator.New(database, cfg)
	trend := trends.NewAnalyzer(database, cfg.Budget.SnapshotRetentionDays)
	mgr := budget.NewManagerFromProviders(cfg, claude, codex, budget.WithBudgetSource(cal), budget.WithTrendAnalyzer(trend))

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	// Print header
	mode := cfg.Budget.Mode
	if mode == "" {
		mode = config.DefaultBudgetMode
	}
	fmt.Printf("Budget Status (mode: %s)\n", mode)
	fmt.Println("================================")
	fmt.Println()

	// Print status for each provider
	for _, provName := range providerList {
		if err := printProviderBudget(mgr, cfg, provName, codex, cal); err != nil {
			fmt.Printf("%s: error: %v\n\n", provName, err)
			continue
		}
		fmt.Println()
	}

	return nil
}

func printProviderBudget(mgr *budget.Manager, cfg *config.Config, provName string, codex *providers.Codex, source budget.BudgetSource) error {
	result, err := mgr.CalculateAllowance(provName)
	if err != nil {
		return err
	}

	estimate := budget.BudgetEstimate{
		WeeklyTokens: int64(cfg.GetProviderBudget(provName)),
		Source:       "config",
	}
	if source != nil {
		if resolved, err := source.GetBudget(provName); err == nil && resolved.WeeklyTokens > 0 {
			estimate = resolved
			if estimate.Source == "" {
				estimate.Source = "calibrated"
			}
		}
	}
	weeklyBudget := estimate.WeeklyTokens

	// Resolve config values for the equation display
	maxPercent := cfg.Budget.MaxPercent
	if maxPercent <= 0 {
		maxPercent = config.DefaultMaxPercent
	}
	reservePercent := cfg.Budget.ReservePercent
	if reservePercent < 0 {
		reservePercent = config.DefaultReservePercent
	}

	// Provider name header
	fmt.Printf("[%s]\n", provName)

	// Mode-specific display
	if result.Mode == "daily" {
		dailyBudget := weeklyBudget / 7
		usedTokens := int64(float64(dailyBudget) * result.UsedPercent / 100)
		remaining := dailyBudget - usedTokens

		fmt.Printf("  Mode:         %s\n", result.Mode)
		fmt.Printf("  Weekly:       %s tokens%s\n", formatTokens64(weeklyBudget), formatBudgetMeta(estimate))
		fmt.Printf("  Daily:        %s tokens\n", formatTokens64(dailyBudget))

		// Used today with low-data warning
		usedLine := fmt.Sprintf("  Used today:   %s (%.1f%%)", formatTokens64(usedTokens), result.UsedPercent)
		if result.UsedPercent == 0 && (estimate.Confidence == "low" || estimate.Confidence == "medium") {
			usedLine += fmt.Sprintf("  [limited data — %d samples]", estimate.SampleCount)
		}
		fmt.Println(usedLine)

		fmt.Printf("  Remaining:    %s tokens\n", formatTokens64(remaining))
		if result.PredictedUsage > 0 {
			fmt.Printf("  Daytime:      %s tokens reserved\n", formatTokens64(result.PredictedUsage))
		}
		fmt.Printf("  Reserve:      %s tokens\n", formatTokens64(result.ReserveAmount))

		// Nightshift equation: remaining * maxPercent% = preReserve - reserve [- daytime] = available
		preReserve := remaining * int64(maxPercent) / 100
		reserve := dailyBudget * int64(reservePercent) / 100
		if result.PredictedUsage > 0 {
			fmt.Printf("  Nightshift:   %s remaining × %d%% max = %s − %s reserve − %s daytime = %s available\n",
				formatTokens64(remaining), maxPercent, formatTokens64(preReserve),
				formatTokens64(reserve), formatTokens64(result.PredictedUsage),
				formatTokens64(result.Allowance))
		} else {
			fmt.Printf("  Nightshift:   %s remaining × %d%% max = %s − %s reserve = %s available\n",
				formatTokens64(remaining), maxPercent, formatTokens64(preReserve),
				formatTokens64(reserve), formatTokens64(result.Allowance))
		}
	} else {
		// Weekly mode
		usedTokens := int64(float64(weeklyBudget) * result.UsedPercent / 100)
		remaining := weeklyBudget - usedTokens

		fmt.Printf("  Mode:         %s\n", result.Mode)
		fmt.Printf("  Weekly:       %s tokens%s\n", formatTokens64(weeklyBudget), formatBudgetMeta(estimate))

		// Used with low-data warning
		usedLine := fmt.Sprintf("  Used:         %s (%.1f%%)", formatTokens64(usedTokens), result.UsedPercent)
		if result.UsedPercent == 0 && (estimate.Confidence == "low" || estimate.Confidence == "medium") {
			usedLine += fmt.Sprintf("  [limited data — %d samples]", estimate.SampleCount)
		}
		fmt.Println(usedLine)

		fmt.Printf("  Remaining:    %s tokens\n", formatTokens64(remaining))
		fmt.Printf("  Days left:    %d\n", result.RemainingDays)
		if result.PredictedUsage > 0 {
			fmt.Printf("  Daytime:      %s tokens reserved\n", formatTokens64(result.PredictedUsage))
		}

		if result.Multiplier > 1.0 {
			fmt.Printf("  Multiplier:   %.1fx (end-of-week)\n", result.Multiplier)
		}

		fmt.Printf("  Reserve:      %s tokens\n", formatTokens64(result.ReserveAmount))

		// Nightshift equation: remaining/days * maxPercent% = preReserve - reserve [- daytime] = available
		days := result.RemainingDays
		if days <= 0 {
			days = 1
		}
		perDay := remaining / int64(days)
		preReserve := perDay * int64(maxPercent) / 100
		reserve := result.ReserveAmount
		if result.PredictedUsage > 0 {
			fmt.Printf("  Nightshift:   %s remaining × %d%% max = %s − %s reserve − %s daytime = %s available\n",
				formatTokens64(perDay), maxPercent, formatTokens64(preReserve),
				formatTokens64(reserve), formatTokens64(result.PredictedUsage),
				formatTokens64(result.Allowance))
		} else {
			fmt.Printf("  Nightshift:   %s remaining × %d%% max = %s − %s reserve = %s available\n",
				formatTokens64(perDay), maxPercent, formatTokens64(preReserve),
				formatTokens64(reserve), formatTokens64(result.Allowance))
		}
	}

	// Show reset time for Codex
	if provName == "codex" && codex != nil {
		resetTime, err := codex.GetResetTime(result.Mode)
		if err == nil && !resetTime.IsZero() {
			fmt.Printf("  Resets at:    %s\n", formatResetTime(resetTime))
		}
	}

	// Budget used bar
	fmt.Printf("  Budget used:  %s\n", progressBar(result.UsedPercent, 30))

	return nil
}

func formatTokens64(tokens int64) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

func formatBudgetMeta(estimate budget.BudgetEstimate) string {
	if estimate.Source == "" {
		return ""
	}

	parts := []string{estimate.Source}
	if estimate.Confidence != "" {
		parts = append(parts, fmt.Sprintf("%s confidence", estimate.Confidence))
	}
	if estimate.SampleCount > 0 {
		parts = append(parts, fmt.Sprintf("%d samples", estimate.SampleCount))
	}

	return " (" + strings.Join(parts, ", ") + ")"
}

func formatResetTime(t time.Time) string {
	now := time.Now()
	duration := t.Sub(now)

	if duration <= 0 {
		return "now"
	}

	// Show relative time
	if duration < time.Hour {
		return fmt.Sprintf("in %d min (%s)", int(duration.Minutes()), t.Format("15:04"))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("in %dh %dm (%s)", int(duration.Hours()), int(duration.Minutes())%60, t.Format("15:04"))
	}

	days := int(duration.Hours() / 24)
	return fmt.Sprintf("in %d days (%s)", days, t.Format("Jan 2 15:04"))
}

func progressBar(percent float64, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	filled := int(percent * float64(width) / 100)
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "#"
	}
	for i := 0; i < empty; i++ {
		bar += "-"
	}

	return fmt.Sprintf("[%s] %.1f%%", bar, percent)
}
