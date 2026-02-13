package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/nightshift/internal/budget"
	"github.com/marcus/nightshift/internal/orchestrator"
	"github.com/marcus/nightshift/internal/tasks"
)

// runStyles holds lipgloss styles for colored run output, matching the
// color palette used by preview_output.go.
type runStyles struct {
	Title   lipgloss.Style
	Phase   lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Muted   lipgloss.Style
	Warn    lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
	Accent  lipgloss.Style
}

func newRunStyles() runStyles {
	return runStyles{
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Phase:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		Label:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Value:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Error:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
		Success: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		Accent:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
	}
}

// asyncSpinner renders a braille spinner on the current line using \r.
type asyncSpinner struct {
	mu      sync.Mutex
	label   string
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

var spinnerFrames = []string{"\u280b", "\u2819", "\u2839", "\u2838", "\u283c", "\u2834", "\u2826", "\u2827", "\u2807", "\u280f"}

func newAsyncSpinner() *asyncSpinner {
	return &asyncSpinner{}
}

func (s *asyncSpinner) start(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.label = label
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	go s.run()
}

func (s *asyncSpinner) run() {
	defer close(s.doneCh)
	idx := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			// Clear the spinner line; protect label read
			s.mu.Lock()
			clearLen := len(s.label) + 4
			s.mu.Unlock()
			fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
			return
		case <-ticker.C:
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()
			frame := spinnerFrames[idx%len(spinnerFrames)]
			fmt.Printf("\r  %s %s", frame, label)
			idx++
		}
	}
}

func (s *asyncSpinner) stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()
	close(s.stopCh)
	<-s.doneCh
}

// liveRenderer handles orchestrator events and renders colored output.
// Events are emitted synchronously from a single goroutine, so no mutex
// is needed. The spinner has its own synchronization.
type liveRenderer struct {
	styles  runStyles
	spinner *asyncSpinner
}

func newLiveRenderer() *liveRenderer {
	return &liveRenderer{
		styles:  newRunStyles(),
		spinner: newAsyncSpinner(),
	}
}

// cleanup stops the spinner if still running. Safe to call multiple times.
func (r *liveRenderer) cleanup() {
	r.spinner.stop()
}

// HandleEvent processes an orchestrator event and renders it to the terminal.
// Called synchronously from the orchestrator goroutine — no concurrent access.
func (r *liveRenderer) HandleEvent(e orchestrator.Event) {
	switch e.Type {
	case orchestrator.EventTaskStart:
		fmt.Printf("\n%s %s\n", r.styles.Accent.Render(">>>"), r.styles.Title.Render(e.TaskTitle))

	case orchestrator.EventPhaseStart:
		r.spinner.stop()
		label := phaseLabel(e.Phase)
		fmt.Printf("  %s ", r.styles.Phase.Render(label))
		r.spinner.start(label)

	case orchestrator.EventPhaseEnd:
		r.spinner.stop()
		label := phaseLabel(e.Phase)
		elapsed := e.Duration.Round(time.Millisecond)
		fmt.Printf("  %s %s\n", r.styles.Phase.Render(label), r.styles.Muted.Render(fmt.Sprintf("(%s)", elapsed)))

	case orchestrator.EventIterationStart:
		if e.Iteration > 1 {
			fmt.Printf("  %s\n", r.styles.Warn.Render(fmt.Sprintf("Iteration %d/%d", e.Iteration, e.MaxIter)))
		} else {
			fmt.Printf("  %s\n", r.styles.Label.Render(fmt.Sprintf("Iteration %d/%d", e.Iteration, e.MaxIter)))
		}

	case orchestrator.EventTaskEnd:
		r.spinner.stop()
		elapsed := e.Duration.Round(time.Second)
		switch e.Status {
		case orchestrator.StatusCompleted:
			fmt.Printf("  %s %s\n", r.styles.Success.Render("COMPLETED"), r.styles.Muted.Render(fmt.Sprintf("(%s)", elapsed)))
		case orchestrator.StatusFailed:
			msg := "FAILED"
			if e.Error != "" {
				msg = fmt.Sprintf("FAILED: %s", e.Error)
			}
			fmt.Printf("  %s %s\n", r.styles.Error.Render(msg), r.styles.Muted.Render(fmt.Sprintf("(%s)", elapsed)))
		case orchestrator.StatusAbandoned:
			msg := "ABANDONED"
			if e.Error != "" {
				msg = fmt.Sprintf("ABANDONED: %s", e.Error)
			}
			fmt.Printf("  %s %s\n", r.styles.Warn.Render(msg), r.styles.Muted.Render(fmt.Sprintf("(%s)", elapsed)))
		default:
			fmt.Printf("  %s %s\n", r.styles.Label.Render(string(e.Status)), r.styles.Muted.Render(fmt.Sprintf("(%s)", elapsed)))
		}

	case orchestrator.EventLog:
		// Only surface warn/error to terminal
		switch e.Level {
		case "warn":
			fmt.Printf("  %s %s\n", r.styles.Warn.Render("WARN"), e.Message)
		case "error":
			fmt.Printf("  %s %s\n", r.styles.Error.Render("ERROR"), e.Message)
		}
	}
}

func phaseLabel(phase orchestrator.TaskStatus) string {
	switch phase {
	case orchestrator.StatusPlanning:
		return "PLANNING"
	case orchestrator.StatusExecuting:
		return "IMPLEMENTING"
	case orchestrator.StatusReviewing:
		return "REVIEWING"
	default:
		return string(phase)
	}
}

// displayPreflightColored renders the preflight summary with colors.
func displayPreflightColored(plan *preflightPlan) {
	s := newRunStyles()
	hr := strings.Repeat("\u2500", 40)

	fmt.Println()
	fmt.Println(s.Title.Render("Preflight Summary"))
	fmt.Println(s.Muted.Render(hr))

	// Show branch info
	if plan.branch != "" {
		fmt.Printf("  %s %s\n",
			s.Label.Render("Branch:"),
			s.Value.Render(plan.branch))
	}

	// Show provider info from first project that has one
	for _, pp := range plan.projects {
		if pp.provider != nil {
			fmt.Printf("  %s %s %s\n",
				s.Label.Render("Provider:"),
				s.Value.Render(pp.provider.name),
				s.Muted.Render(fmt.Sprintf("(%.1f%% used, %s)", pp.provider.allowance.UsedPercent, pp.provider.allowance.Mode)))
			fmt.Printf("  %s %s\n",
				s.Label.Render("Budget:"),
				s.Value.Render(fmt.Sprintf("%d tokens remaining", pp.provider.allowance.Allowance)))
			break
		}
	}

	// Count active projects (those with tasks)
	active := 0
	for _, pp := range plan.projects {
		if len(pp.tasks) > 0 {
			active++
		}
	}
	fmt.Printf("\n  %s\n", s.Phase.Render(fmt.Sprintf("Projects (%d of %d)", active, len(plan.projects))))

	idx := 0
	for _, pp := range plan.projects {
		if pp.skipReason != "" || len(pp.tasks) == 0 {
			continue
		}
		idx++
		fmt.Printf("  %s %s\n", s.Accent.Render(fmt.Sprintf("%d.", idx)), s.Value.Render(filepath.Base(pp.path)))
		for _, st := range pp.tasks {
			minTok, maxTok := st.Definition.EstimatedTokens()
			fmt.Printf("     %s %s %s\n",
				s.Accent.Render("\u25cf"),
				s.Value.Render(st.Definition.Name),
				s.Muted.Render(fmt.Sprintf("(score=%.1f, cost=%s, ~%dk-%dk tokens)", st.Score, st.Definition.CostTier, minTok/1000, maxTok/1000)))
		}
	}

	// Skipped projects
	var skipped []preflightProject
	for _, pp := range plan.projects {
		if pp.skipReason != "" {
			skipped = append(skipped, pp)
		}
	}
	if len(skipped) > 0 {
		fmt.Printf("\n  %s\n", s.Warn.Render("Skipped:"))
		for _, pp := range skipped {
			fmt.Printf("    %s %s: %s\n", s.Warn.Render("\u25cf"), s.Label.Render(filepath.Base(pp.path)), s.Muted.Render(pp.skipReason))
		}
	}

	// Warnings
	if plan.ignoreBudget {
		fmt.Printf("\n  %s\n", s.Warn.Render("Warnings:"))
		fmt.Printf("    %s %s\n", s.Warn.Render("\u25cf"), s.Warn.Render("--ignore-budget is set: budget limits bypassed"))
	}

	fmt.Println(s.Muted.Render(hr))
	fmt.Println()
}

// displayRunSummaryColored renders the final run summary with colors.
func displayRunSummaryColored(duration time.Duration, tasksRun, tasksCompleted, tasksFailed int, skipReasons []string) {
	s := newRunStyles()
	hr := strings.Repeat("\u2500", 40)

	fmt.Println()
	fmt.Println(s.Muted.Render(hr))
	fmt.Println(s.Title.Render("Run Complete"))
	fmt.Printf("  %s %s\n", s.Label.Render("Duration:"), s.Value.Render(duration.Round(time.Second).String()))

	statusStyle := s.Success
	if tasksFailed > 0 && tasksCompleted == 0 {
		statusStyle = s.Error
	} else if tasksFailed > 0 {
		statusStyle = s.Warn
	}
	fmt.Printf("  %s %s\n", s.Label.Render("Tasks:"),
		statusStyle.Render(fmt.Sprintf("%d run, %d completed, %d failed", tasksRun, tasksCompleted, tasksFailed)))

	if tasksRun == 0 && len(skipReasons) > 0 {
		fmt.Printf("\n  %s\n", s.Warn.Render("Nothing ran because:"))
		for _, reason := range skipReasons {
			fmt.Printf("    %s %s\n", s.Warn.Render("\u25cf"), s.Muted.Render(reason))
		}
	}
	fmt.Println()
}

// displayProjectHeaderColored renders the per-project header with colors.
func displayProjectHeaderColored(projectPath, providerName string, allowance *budget.AllowanceResult, taskCount int, scoredTasks []tasks.ScoredTask) {
	s := newRunStyles()
	hr := strings.Repeat("\u2500", 40)

	fmt.Println()
	fmt.Println(s.Muted.Render(hr))
	fmt.Printf("  %s %s\n", s.Title.Render("Project:"), s.Value.Render(projectPath))
	fmt.Printf("  %s %s\n", s.Label.Render("Provider:"), s.Value.Render(providerName))
	if allowance != nil {
		fmt.Printf("  %s %s\n", s.Label.Render("Budget:"),
			s.Value.Render(fmt.Sprintf("%d tokens available (%.1f%% used, mode=%s)", allowance.Allowance, allowance.UsedPercent, allowance.Mode)))
	}

	fmt.Printf("\n  %s\n", s.Phase.Render(fmt.Sprintf("Selected %d task(s):", taskCount)))
	for i, st := range scoredTasks {
		minTok, maxTok := st.Definition.EstimatedTokens()
		fmt.Printf("    %s %s %s\n",
			s.Accent.Render(fmt.Sprintf("%d.", i+1)),
			s.Value.Render(st.Definition.Name),
			s.Muted.Render(fmt.Sprintf("(score=%.1f, cost=%s, tokens=%d-%d)", st.Score, st.Definition.CostTier, minTok, maxTok)))
	}
}
