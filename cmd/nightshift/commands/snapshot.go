package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/marcus/nightshift/internal/calibrator"
	"github.com/marcus/nightshift/internal/config"
	"github.com/marcus/nightshift/internal/db"
	"github.com/marcus/nightshift/internal/providers"
	"github.com/marcus/nightshift/internal/snapshots"
)

var budgetSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture a usage snapshot",
	Long: `Capture a usage snapshot for budget calibration.

Collects local token counts and (optionally) scrapes the CLI's usage
display via tmux to get the provider's own usage percentage.

  Local data   Reads token counts from the provider's local data files.
               Claude: stats-cache.json   Codex: session JSONL files

  Scraping     Launches the CLI in a tmux session, runs /usage or /status,
               and parses the percentage from the screen output.
               Requires: tmux installed, calibrate_enabled: true in config.

  Inference    If both local tokens and scraped % are available, nightshift
               infers the weekly budget: budget = local_tokens / (scraped% / 100).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		localOnly, _ := cmd.Flags().GetBool("local-only")
		return runBudgetSnapshot(cmd, provider, localOnly)
	},
}

var budgetHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent budget snapshots",
	Long:  `Show recent usage snapshots for budget calibration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		n, _ := cmd.Flags().GetInt("n")
		return runBudgetHistory(provider, n)
	},
}

var budgetCalibrateCmd = &cobra.Command{
	Use:   "calibrate",
	Short: "Show calibration status",
	Long:  `Show inferred budget calibration status for providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		return runBudgetCalibrate(provider)
	},
}

func init() {
	budgetSnapshotCmd.Flags().StringP("provider", "p", "", "Provider to snapshot (claude, codex)")
	budgetSnapshotCmd.Flags().Bool("local-only", false, "Skip tmux scraping and store local-only snapshot")

	budgetHistoryCmd.Flags().StringP("provider", "p", "", "Provider to show history for (claude, codex)")
	budgetHistoryCmd.Flags().IntP("n", "n", 20, "Number of snapshots to show")

	budgetCalibrateCmd.Flags().StringP("provider", "p", "", "Provider to calibrate (claude, codex)")

	budgetCmd.AddCommand(budgetSnapshotCmd)
	budgetCmd.AddCommand(budgetHistoryCmd)
	budgetCmd.AddCommand(budgetCalibrateCmd)
}

func runBudgetSnapshot(cmd *cobra.Command, filterProvider string, localOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer func() { _ = database.Close() }()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled in config.")
		return nil
	}

	// Determine scraper availability and reason if disabled
	scraper := snapshots.UsageScraper(nil)
	scrapeDisabledReason := ""
	if localOnly {
		scrapeDisabledReason = "--local-only flag set"
	} else if !cfg.Budget.CalibrateEnabled {
		scrapeDisabledReason = "calibrate_enabled is false in config"
	} else if strings.EqualFold(cfg.Budget.BillingMode, "api") {
		scrapeDisabledReason = "billing_mode is 'api' (scraping only works with subscription)"
	} else {
		scraper = tmuxScraper{}
	}

	collector := snapshots.NewCollector(
		database,
		providers.NewClaudeWithPath(cfg.ExpandedProviderPath("claude")),
		providers.NewCodexWithPath(cfg.ExpandedProviderPath("codex")),
		providers.NewCopilotWithPath(cfg.ExpandedProviderPath("copilot")),
		scraper,
		weekStartDayFromConfig(cfg),
	)

	fmt.Println("Budget Snapshot")
	fmt.Println("===============")

	ctx := cmd.Context()
	for _, provName := range providerList {
		fmt.Println()
		fmt.Printf("[%s]\n", provName)

		snapshot, err := collector.TakeSnapshot(ctx, provName)
		if err != nil {
			fmt.Printf("  FAILED:       %v\n", err)
			printSnapshotHints(provName, cfg, err)
			continue
		}

		// Local data
		dataSource := providerDataSource(provName)
		dataPath := cfg.ExpandedProviderPath(provName)
		if dataPath != "" {
			dataSource = fmt.Sprintf("%s (%s)", dataSource, dataPath)
		}
		fmt.Printf("  Local weekly: %s tokens\n", formatTokens64(snapshot.LocalTokens))
		fmt.Printf("  Local daily:  %s tokens\n", formatTokens64(snapshot.LocalDaily))
		fmt.Printf("  Data source:  %s\n", dataSource)

		// Scraping status
		if scraper == nil {
			fmt.Printf("  Scraping:     disabled -- %s\n", scrapeDisabledReason)
		} else if snapshot.ScrapeErr != nil {
			fmt.Printf("  Scraping:     FAILED -- %v\n", snapshot.ScrapeErr)
		} else if snapshot.ScrapedPct != nil {
			cmd := "/usage"
			if provName == "codex" {
				cmd = "/status"
			}
			fmt.Printf("  Scraped:      %.1f%% used (via tmux %s)\n", *snapshot.ScrapedPct, cmd)
		} else {
			fmt.Printf("  Scraping:     no data returned\n")
		}

		// Inferred budget
		if snapshot.InferredBudget != nil {
			fmt.Printf("  Budget est:   %s tokens/week\n", formatTokens64(*snapshot.InferredBudget))
		}

		// Reset times
		resetLine := formatResetLine(snapshot.SessionResetTime, snapshot.WeeklyResetTime)
		if resetLine != "" {
			fmt.Printf("  Resets:       %s\n", resetLine)
		}

		// Result
		if snapshot.ScrapedPct != nil {
			fmt.Printf("  Saved:        snapshot #%d\n", snapshot.ID)
		} else {
			fmt.Printf("  Saved:        snapshot #%d (local-only, no scraped %%)\n", snapshot.ID)
		}

		// Diagnostic hints
		printSnapshotHints(provName, cfg, nil)
		printSnapshotDataHints(snapshot, scraper != nil)
	}

	fmt.Println()
	return nil
}

// providerDataSource returns a human description of the local data source.
func providerDataSource(provider string) string {
	switch provider {
	case "claude":
		return "stats-cache.json"
	case "codex":
		return "session JSONL files"
	default:
		return "local files"
	}
}

// printSnapshotHints prints contextual hints for common issues.
func printSnapshotHints(provider string, cfg *config.Config, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "provider is nil"):
		fmt.Printf("  Hint: The %s provider object could not be created.\n", provider)
		fmt.Printf("        Check that providers.%s.enabled is true in config.\n", provider)
	case strings.Contains(errMsg, "token too long"):
		fmt.Printf("  Hint: A session file has lines exceeding the read buffer.\n")
		fmt.Printf("        This is a bug -- please report it.\n")
	case strings.Contains(errMsg, "no such file"):
		path := cfg.ExpandedProviderPath(provider)
		fmt.Printf("  Hint: Data path not found: %s\n", path)
		fmt.Printf("        Verify providers.%s.data_path in config.\n", provider)
	}
}

// printSnapshotDataHints prints hints about zero-data or missing scrape conditions.
func printSnapshotDataHints(snap snapshots.Snapshot, scraperEnabled bool) {
	hints := []string{}

	if snap.LocalTokens == 0 && snap.LocalDaily == 0 {
		switch snap.Provider {
		case "claude":
			hints = append(hints,
				"Local tokens are 0. Claude Code writes usage to stats-cache.json",
				"after each session. Use Claude Code to generate initial data.")
		case "codex":
			// Codex token data comes from session JSONL files.
			// If no tokens and no scrape, suggest running Codex.
			if snap.ScrapedPct == nil {
				hints = append(hints,
					"No local tokens or scraped data for Codex.",
					"Use Codex CLI to generate session data, and enable tmux scraping for budget %.")
			}
		}
	}

	if scraperEnabled && snap.ScrapeErr != nil {
		errMsg := snap.ScrapeErr.Error()
		switch {
		case strings.Contains(errMsg, "tmux not found"):
			hints = append(hints, "tmux is not installed. Install it to enable scraping.")
		case strings.Contains(errMsg, "context deadline") || strings.Contains(errMsg, "timeout"):
			cmd := "claude"
			if snap.Provider == "codex" {
				cmd = "codex"
			}
			hints = append(hints,
				fmt.Sprintf("Scraping timed out. Ensure '%s' starts within ~10s.", cmd),
				"A trust/update prompt may be blocking. Run the CLI manually first.")
		default:
			hints = append(hints,
				fmt.Sprintf("Scrape error: %v", snap.ScrapeErr))
		}
	}

	// Budget inference needs both scraped_pct and local tokens.
	if snap.ScrapedPct != nil && snap.LocalTokens == 0 {
		hints = append(hints,
			"Scraped a usage %, but local tokens are 0, so budget cannot be inferred.",
			"Use the CLI to generate local token data first.")
	}

	for i, h := range hints {
		if i == 0 {
			fmt.Printf("  Note: %s\n", h)
		} else {
			fmt.Printf("        %s\n", h)
		}
	}
}

func runBudgetHistory(filterProvider string, n int) error {
	if n <= 0 {
		return fmt.Errorf("n must be positive")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer func() { _ = database.Close() }()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	collector := snapshots.NewCollector(database, nil, nil, nil, nil, weekStartDayFromConfig(cfg))

	for _, provider := range providerList {
		history, err := collector.GetLatest(provider, n)
		if err != nil {
			fmt.Printf("%s: error: %v\n\n", provider, err)
			continue
		}
		if len(history) == 0 {
			fmt.Printf("[%s]\n  No snapshots yet.\n\n", provider)
			continue
		}

		fmt.Printf("[%s]\n", provider)
		printSnapshotTable(history)
		fmt.Println()
	}

	return nil
}

func runBudgetCalibrate(filterProvider string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.Open(cfg.ExpandedDBPath())
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer func() { _ = database.Close() }()

	providerList, err := resolveProviderList(cfg, filterProvider)
	if err != nil {
		return err
	}

	if len(providerList) == 0 {
		fmt.Println("No providers enabled.")
		return nil
	}

	cal := calibrator.New(database, cfg)
	for _, provider := range providerList {
		result, err := cal.Calibrate(provider)
		if err != nil {
			fmt.Printf("%s: error: %v\n\n", provider, err)
			continue
		}

		fmt.Printf("[%s]\n", provider)
		fmt.Printf("  Source:      %s\n", result.Source)
		fmt.Printf("  Budget:      %s tokens\n", formatTokens64(result.InferredBudget))
		fmt.Printf("  Confidence:  %s\n", result.Confidence)
		fmt.Printf("  Samples:     %d\n", result.SampleCount)
		if result.Variance > 0 {
			fmt.Printf("  Variance:    %.0f\n", result.Variance)
		}
		fmt.Println()
	}

	return nil
}

func printSnapshotTable(history []snapshots.Snapshot) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "Time\tLocal\tDaily\tPct\tInferred\tResets")
	for _, snapshot := range history {
		pct := "-"
		if snapshot.ScrapedPct != nil {
			pct = fmt.Sprintf("%.1f%%", *snapshot.ScrapedPct)
		}
		inferred := "-"
		if snapshot.InferredBudget != nil {
			inferred = formatTokens64(*snapshot.InferredBudget)
		}
		resets := formatResetLine(snapshot.SessionResetTime, snapshot.WeeklyResetTime)
		if resets == "" {
			resets = "-"
		}
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			snapshot.Timestamp.Format("Jan 02 15:04"),
			formatTokens64(snapshot.LocalTokens),
			formatTokens64(snapshot.LocalDaily),
			pct,
			inferred,
			resets,
		)
	}
	_ = writer.Flush()
}
