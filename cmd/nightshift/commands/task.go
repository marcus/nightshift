package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/marcus/nightshift/internal/logging"
	"github.com/marcus/nightshift/internal/orchestrator"
	"github.com/marcus/nightshift/internal/tasks"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage and run tasks",
	Long:  `List available tasks, show their prompts, and run them against a provider.`,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tasks with budget info",
	Long: `List all available nightshift tasks with their category, cost tier,
and estimated token range.

Use --category to filter by category, --cost to filter by cost tier.
Use --json to output as JSON for scripting.`,
	RunE: runTaskList,
}

var taskShowCmd = &cobra.Command{
	Use:   "show <task-type>",
	Short: "Show task details and prompt",
	Long: `Show a task's metadata and the planning prompt that would be sent to the LLM.

Use --prompt-only to output just the raw prompt text (useful for piping).
Use --json for structured output.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskShow,
}

var taskRunCmd = &cobra.Command{
	Use:   "run <task-type> --provider <claude|codex>",
	Short: "Run a task immediately",
	Long: `Execute a task immediately against a specific provider.

The --provider flag is required. Use --project to set the working directory.
Use --dry-run to see what would happen without executing.`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskRun,
}

func init() {
	taskListCmd.Flags().String("category", "", "Filter by category (pr, analysis, options, safe, map, emergency)")
	taskListCmd.Flags().String("cost", "", "Filter by cost tier (low, medium, high, veryhigh)")
	taskListCmd.Flags().Bool("json", false, "Output as JSON")

	taskShowCmd.Flags().Bool("prompt-only", false, "Output only the raw prompt text")
	taskShowCmd.Flags().Bool("json", false, "Output as JSON")
	taskShowCmd.Flags().StringP("project", "p", "", "Project directory (used in prompt context)")

	taskRunCmd.Flags().String("provider", "", "Provider to run against (claude, codex)")
	taskRunCmd.Flags().StringP("project", "p", "", "Project directory to run in")
	taskRunCmd.Flags().Bool("dry-run", false, "Show prompt without executing")
	taskRunCmd.Flags().Duration("timeout", 30*time.Minute, "Execution timeout")
	_ = taskRunCmd.MarkFlagRequired("provider")

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskRunCmd)
	rootCmd.AddCommand(taskCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	categoryFilter, _ := cmd.Flags().GetString("category")
	costFilter, _ := cmd.Flags().GetString("cost")
	asJSON, _ := cmd.Flags().GetBool("json")

	defs := tasks.AllDefinitionsSorted()

	if categoryFilter != "" {
		cat, err := parseCategoryFilter(categoryFilter)
		if err != nil {
			return err
		}
		defs = filterByCategory(defs, cat)
	}
	if costFilter != "" {
		tier, err := parseCostFilter(costFilter)
		if err != nil {
			return err
		}
		defs = filterByCost(defs, tier)
	}

	if len(defs) == 0 {
		fmt.Println("No tasks match the given filters.")
		return nil
	}

	if asJSON {
		return printTaskListJSON(defs)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tNAME\tCATEGORY\tCOST\tTOKENS\tRISK")
	for _, d := range defs {
		min, max := d.EstimatedTokens()
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s-%s\t%s\n",
			d.Type,
			d.Name,
			categoryShort(d.Category),
			costShort(d.CostTier),
			formatK(min),
			formatK(max),
			d.RiskLevel,
		)
	}
	_ = w.Flush()
	fmt.Printf("\n%d task(s)\n", len(defs))
	return nil
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	taskType := tasks.TaskType(args[0])
	promptOnly, _ := cmd.Flags().GetBool("prompt-only")
	asJSON, _ := cmd.Flags().GetBool("json")
	projectPath, _ := cmd.Flags().GetString("project")

	def, err := tasks.GetDefinition(taskType)
	if err != nil {
		return fmt.Errorf("unknown task: %s\nRun 'nightshift task list' to see available tasks", taskType)
	}

	// Build the planning prompt
	taskInstance := taskInstanceFromDef(def, projectPath)
	orch := orchestrator.New()
	prompt := orch.PlanPrompt(taskInstance)

	if promptOnly {
		fmt.Print(prompt)
		return nil
	}

	if asJSON {
		return printTaskShowJSON(def, prompt)
	}

	min, max := def.EstimatedTokens()
	fmt.Printf("Task:        %s\n", def.Name)
	fmt.Printf("Type:        %s\n", def.Type)
	fmt.Printf("Category:    %s\n", def.Category)
	fmt.Printf("Cost:        %s\n", def.CostTier)
	fmt.Printf("Tokens:      %s - %s\n", formatK(min), formatK(max))
	fmt.Printf("Risk:        %s\n", def.RiskLevel)
	fmt.Printf("Description: %s\n", def.Description)
	fmt.Println()
	fmt.Println("--- Planning Prompt ---")
	fmt.Println(prompt)

	return nil
}

func runTaskRun(cmd *cobra.Command, args []string) error {
	taskType := tasks.TaskType(args[0])
	provider, _ := cmd.Flags().GetString("provider")
	projectPath, _ := cmd.Flags().GetString("project")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	def, err := tasks.GetDefinition(taskType)
	if err != nil {
		return fmt.Errorf("unknown task: %s\nRun 'nightshift task list' to see available tasks", taskType)
	}

	// Resolve project path
	if projectPath == "" {
		var wdErr error
		projectPath, wdErr = os.Getwd()
		if wdErr != nil {
			return fmt.Errorf("get working directory: %w", wdErr)
		}
	}

	// Build the task
	taskInstance := taskInstanceFromDef(def, projectPath)

	// Create orchestrator with the selected agent
	cfg, err := loadConfig(projectPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	agent, err := agentByName(cfg, provider)
	if err != nil {
		return err
	}

	orch := orchestrator.New(
		orchestrator.WithAgent(agent),
		orchestrator.WithConfig(orchestrator.Config{
			MaxIterations: 3,
			AgentTimeout:  timeout,
		}),
		orchestrator.WithLogger(logging.Component("task-run")),
	)

	prompt := orch.PlanPrompt(taskInstance)

	fmt.Printf("Task:     %s (%s)\n", def.Name, def.Type)
	fmt.Printf("Provider: %s\n", provider)
	fmt.Printf("Project:  %s\n", projectPath)
	fmt.Printf("Timeout:  %s\n", timeout)

	min, max := def.EstimatedTokens()
	fmt.Printf("Est:      %s-%s tokens\n", formatK(min), formatK(max))

	if dryRun {
		fmt.Println("\n[dry-run] Would send this prompt:")
		fmt.Println()
		fmt.Println(prompt)
		return nil
	}

	fmt.Println()
	fmt.Println("Running...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		fmt.Println("\ninterrupt received, stopping...")
		cancel()
	}()

	result, err := orch.RunTask(ctx, taskInstance, projectPath)
	if err != nil {
		return fmt.Errorf("task failed: %w", err)
	}

	fmt.Println()
	switch result.Status {
	case orchestrator.StatusCompleted:
		fmt.Printf("COMPLETED in %d iteration(s) (%s)\n", result.Iterations, result.Duration.Round(time.Second))
	case orchestrator.StatusAbandoned:
		fmt.Printf("ABANDONED after %d iteration(s): %s\n", result.Iterations, result.Error)
	default:
		fmt.Printf("FAILED: %s\n", result.Error)
	}

	if result.Output != "" {
		fmt.Println()
		fmt.Println("--- Output ---")
		fmt.Println(result.Output)
	}

	return nil
}

// taskInstanceFromDef creates a tasks.Task from a TaskDefinition for prompt building.
func taskInstanceFromDef(def tasks.TaskDefinition, projectPath string) *tasks.Task {
	id := string(def.Type)
	if projectPath != "" {
		id = fmt.Sprintf("%s:%s", def.Type, projectPath)
	}
	return &tasks.Task{
		ID:          id,
		Title:       def.Name,
		Description: def.Description,
		Priority:    0,
		Type:        def.Type,
	}
}

// --- Filters ---

func parseCategoryFilter(s string) (tasks.TaskCategory, error) {
	switch strings.ToLower(s) {
	case "pr":
		return tasks.CategoryPR, nil
	case "analysis":
		return tasks.CategoryAnalysis, nil
	case "options":
		return tasks.CategoryOptions, nil
	case "safe":
		return tasks.CategorySafe, nil
	case "map":
		return tasks.CategoryMap, nil
	case "emergency":
		return tasks.CategoryEmergency, nil
	default:
		return 0, fmt.Errorf("unknown category: %s (valid: pr, analysis, options, safe, map, emergency)", s)
	}
}

func parseCostFilter(s string) (tasks.CostTier, error) {
	switch strings.ToLower(s) {
	case "low":
		return tasks.CostLow, nil
	case "medium":
		return tasks.CostMedium, nil
	case "high":
		return tasks.CostHigh, nil
	case "veryhigh":
		return tasks.CostVeryHigh, nil
	default:
		return 0, fmt.Errorf("unknown cost tier: %s (valid: low, medium, high, veryhigh)", s)
	}
}

func filterByCategory(defs []tasks.TaskDefinition, cat tasks.TaskCategory) []tasks.TaskDefinition {
	var out []tasks.TaskDefinition
	for _, d := range defs {
		if d.Category == cat {
			out = append(out, d)
		}
	}
	return out
}

func filterByCost(defs []tasks.TaskDefinition, tier tasks.CostTier) []tasks.TaskDefinition {
	var out []tasks.TaskDefinition
	for _, d := range defs {
		if d.CostTier == tier {
			out = append(out, d)
		}
	}
	return out
}

// --- Formatters ---

func categoryShort(c tasks.TaskCategory) string {
	switch c {
	case tasks.CategoryPR:
		return "PR"
	case tasks.CategoryAnalysis:
		return "Analysis"
	case tasks.CategoryOptions:
		return "Options"
	case tasks.CategorySafe:
		return "Safe"
	case tasks.CategoryMap:
		return "Map"
	case tasks.CategoryEmergency:
		return "Emergency"
	default:
		return "?"
	}
}

func costShort(c tasks.CostTier) string {
	switch c {
	case tasks.CostLow:
		return "Low"
	case tasks.CostMedium:
		return "Med"
	case tasks.CostHigh:
		return "High"
	case tasks.CostVeryHigh:
		return "VHigh"
	default:
		return "?"
	}
}

func formatK(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.0fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%dk", tokens/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}

// --- JSON output ---

type taskListEntry struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Cost        string `json:"cost"`
	MinTokens   int    `json:"min_tokens"`
	MaxTokens   int    `json:"max_tokens"`
	Risk        string `json:"risk"`
}

func printTaskListJSON(defs []tasks.TaskDefinition) error {
	entries := make([]taskListEntry, len(defs))
	for i, d := range defs {
		min, max := d.EstimatedTokens()
		entries[i] = taskListEntry{
			Type:        string(d.Type),
			Name:        d.Name,
			Category:    categoryShort(d.Category),
			Description: d.Description,
			Cost:        costShort(d.CostTier),
			MinTokens:   min,
			MaxTokens:   max,
			Risk:        d.RiskLevel.String(),
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

type taskShowEntry struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Cost        string `json:"cost"`
	MinTokens   int    `json:"min_tokens"`
	MaxTokens   int    `json:"max_tokens"`
	Risk        string `json:"risk"`
	Prompt      string `json:"prompt"`
}

func printTaskShowJSON(def tasks.TaskDefinition, prompt string) error {
	min, max := def.EstimatedTokens()
	entry := taskShowEntry{
		Type:        string(def.Type),
		Name:        def.Name,
		Category:    categoryShort(def.Category),
		Description: def.Description,
		Cost:        costShort(def.CostTier),
		MinTokens:   min,
		MaxTokens:   max,
		Risk:        def.RiskLevel.String(),
		Prompt:      prompt,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entry)
}
