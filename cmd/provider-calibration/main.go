package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type codexTokenUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
}

type sessionMetrics struct {
	Provider       string `json:"provider"`
	File           string `json:"file"`
	CWD            string `json:"cwd,omitempty"`
	Originator     string `json:"originator,omitempty"`
	TokensPrimary  int64  `json:"tokens_primary"`
	TokensAlt      int64  `json:"tokens_alt"`
	UserTurns      int64  `json:"user_turns"`
	AssistantTurns int64  `json:"assistant_turns"`
}

type summaryStats struct {
	Count  int   `json:"count"`
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	Mean   int64 `json:"mean"`
	Median int64 `json:"median"`
	P75    int64 `json:"p75"`
	P90    int64 `json:"p90"`
}

type providerSummary struct {
	Provider              string         `json:"provider"`
	Sessions              int            `json:"sessions"`
	Originators           map[string]int `json:"originators,omitempty"`
	TokensPrimary         summaryStats   `json:"tokens_primary"`
	TokensAlt             summaryStats   `json:"tokens_alt"`
	UserTurns             summaryStats   `json:"user_turns"`
	AssistantTurns        summaryStats   `json:"assistant_turns"`
	PrimaryPerUserTurn    summaryStats   `json:"primary_per_user_turn"`
	AltPerUserTurn        summaryStats   `json:"alt_per_user_turn"`
	PrimaryPerSessionNote string         `json:"primary_per_session_note,omitempty"`
	AltPerSessionNote     string         `json:"alt_per_session_note,omitempty"`
	Warnings              []string       `json:"warnings,omitempty"`
	SampleFiles           []string       `json:"sample_files,omitempty"`
}

type ratioSummary struct {
	CodexPrimaryToClaudePrimaryPerSession float64 `json:"codex_primary_to_claude_primary_per_session"`
	CodexPrimaryToClaudeAltPerSession     float64 `json:"codex_primary_to_claude_alt_per_session"`
	CodexPrimaryToClaudePrimaryPerTurn    float64 `json:"codex_primary_to_claude_primary_per_turn"`
	CodexPrimaryToClaudeAltPerTurn        float64 `json:"codex_primary_to_claude_alt_per_turn"`
	SuggestedMultiplier                   float64 `json:"suggested_multiplier"`
	SuggestedMetric                       string  `json:"suggested_metric"`
}

type report struct {
	RepoFilter       string          `json:"repo_filter,omitempty"`
	CodexOriginator  string          `json:"codex_originator,omitempty"`
	MinUserTurns     int             `json:"min_user_turns"`
	Codex            providerSummary `json:"codex"`
	Claude           providerSummary `json:"claude"`
	Ratios           ratioSummary    `json:"ratios"`
	MethodologyNotes []string        `json:"methodology_notes"`
	Warnings         []string        `json:"warnings,omitempty"`
}

func main() {
	var (
		codexSessions   = flag.String("codex-sessions", filepath.Join(userHomeDir(), ".codex", "sessions"), "Path to Codex sessions directory")
		claudeProjects  = flag.String("claude-projects", filepath.Join(userHomeDir(), ".claude", "projects"), "Path to Claude projects directory")
		repo            = flag.String("repo", "", "Filter sessions to this repo path (exact cwd match after clean)")
		codexOriginator = flag.String("codex-originator", "", "Optional Codex session originator filter (e.g. codex_cli_rs)")
		minUserTurns    = flag.Int("min-user-turns", 1, "Minimum user turns per session to include")
		asJSON          = flag.Bool("json", false, "Output JSON")
		verbose         = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	repoFilter := strings.TrimSpace(*repo)
	if repoFilter != "" {
		repoFilter = filepath.Clean(expandPath(repoFilter))
	}

	codexMetrics, codexOriginators, codexErr := collectCodex(*codexSessions, repoFilter, *codexOriginator, *minUserTurns)
	if codexErr != nil {
		fatalf("collect codex metrics: %v", codexErr)
	}
	claudeMetrics, claudeErr := collectClaude(*claudeProjects, repoFilter, *minUserTurns)
	if claudeErr != nil {
		fatalf("collect claude metrics: %v", claudeErr)
	}

	r := report{
		RepoFilter:      repoFilter,
		CodexOriginator: strings.TrimSpace(*codexOriginator),
		MinUserTurns:    *minUserTurns,
		Codex:           summarizeProvider("codex", codexMetrics, codexOriginators),
		Claude:          summarizeProvider("claude", claudeMetrics, nil),
		MethodologyNotes: []string{
			"Codex primary tokens are billable: non-cached input + output + reasoning output.",
			"Codex alt tokens are raw totals: input + cached input + output + reasoning output.",
			"Claude primary tokens are input + output from message usage.",
			"Claude alt tokens include cache fields: input + output + cache_read_input + cache_creation_input.",
			"Suggested multiplier uses per-user-turn medians: codex primary / claude alt.",
		},
	}
	if len(codexMetrics) < 10 || len(claudeMetrics) < 10 {
		r.Warnings = append(r.Warnings, "Low sample count; collect more sessions before changing budgets.")
	}
	if repoFilter == "" {
		r.Warnings = append(r.Warnings, "No repo filter set; cross-repo behavior may distort ratios.")
	}
	if strings.TrimSpace(*codexOriginator) == "" {
		r.Warnings = append(r.Warnings, "No codex originator filter set; desktop and CLI sessions may be mixed.")
	}

	r.Ratios = computeRatios(r.Codex, r.Claude)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			fatalf("encode json: %v", err)
		}
		return
	}

	printReport(r, *verbose)
}

func collectCodex(root, repoFilter, originatorFilter string, minUserTurns int) ([]sessionMetrics, map[string]int, error) {
	var sessions []sessionMetrics
	originators := map[string]int{}
	originatorFilter = strings.TrimSpace(originatorFilter)

	err := filepath.WalkDir(expandPath(root), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

		originator := ""
		cwd := ""
		userTurns := int64(0)
		assistantTurns := int64(0)
		var first, latest *codexTokenUsage
		eventCount := 0

		for scanner.Scan() {
			line := scanner.Bytes()
			var base struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(line, &base); err != nil {
				continue
			}

			switch base.Type {
			case "session_meta":
				var p struct {
					Originator string `json:"originator"`
					CWD        string `json:"cwd"`
				}
				if err := json.Unmarshal(base.Payload, &p); err == nil {
					originator = strings.TrimSpace(p.Originator)
					cwd = normalizePath(p.CWD)
				}
			case "event_msg":
				var p struct {
					Type string `json:"type"`
					Info *struct {
						Total *codexTokenUsage `json:"total_token_usage"`
					} `json:"info"`
				}
				if err := json.Unmarshal(base.Payload, &p); err != nil {
					continue
				}
				if p.Type == "token_count" && p.Info != nil && p.Info.Total != nil {
					u := *p.Info.Total
					if first == nil {
						first = &u
					}
					latest = &u
					eventCount++
				}
				if p.Type == "user_message" {
					userTurns++
				}
			case "response_item":
				var p struct {
					Type string `json:"type"`
					Role string `json:"role"`
				}
				if err := json.Unmarshal(base.Payload, &p); err != nil {
					continue
				}
				if p.Type == "message" && p.Role == "assistant" {
					assistantTurns++
				}
			}
		}
		if scanErr := scanner.Err(); scanErr != nil && scanErr != io.EOF {
			return nil
		}

		if latest == nil {
			return nil
		}
		if originatorFilter != "" && originator != originatorFilter {
			return nil
		}
		if repoFilter != "" && cwd != repoFilter {
			return nil
		}
		if int(userTurns) < minUserTurns {
			return nil
		}

		src := *latest
		if eventCount > 1 && first != nil {
			delta := codexTokenUsage{
				InputTokens:           latest.InputTokens - first.InputTokens,
				CachedInputTokens:     latest.CachedInputTokens - first.CachedInputTokens,
				OutputTokens:          latest.OutputTokens - first.OutputTokens,
				ReasoningOutputTokens: latest.ReasoningOutputTokens - first.ReasoningOutputTokens,
			}
			if delta.InputTokens >= 0 && delta.CachedInputTokens >= 0 && delta.OutputTokens >= 0 && delta.ReasoningOutputTokens >= 0 {
				src = delta
			}
		}

		input := nonNegative(src.InputTokens)
		cached := nonNegative(src.CachedInputTokens)
		output := nonNegative(src.OutputTokens)
		reasoning := nonNegative(src.ReasoningOutputTokens)

		primary := nonNegative(input-cached) + output + reasoning
		alt := input + cached + output + reasoning
		if primary <= 0 {
			return nil
		}

		originators[originator]++
		sessions = append(sessions, sessionMetrics{
			Provider:       "codex",
			File:           path,
			CWD:            cwd,
			Originator:     originator,
			TokensPrimary:  primary,
			TokensAlt:      alt,
			UserTurns:      userTurns,
			AssistantTurns: assistantTurns,
		})
		return nil
	})

	return sessions, originators, err
}

func collectClaude(root, repoFilter string, minUserTurns int) ([]sessionMetrics, error) {
	var sessions []sessionMetrics

	err := filepath.WalkDir(expandPath(root), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

		userTurns := int64(0)
		assistantTurns := int64(0)
		primary := int64(0)
		alt := int64(0)
		cwd := ""

		for scanner.Scan() {
			line := scanner.Bytes()
			var entry struct {
				Type    string `json:"type"`
				CWD     string `json:"cwd"`
				Message *struct {
					Role  string `json:"role"`
					Usage *struct {
						InputTokens              int64 `json:"input_tokens"`
						OutputTokens             int64 `json:"output_tokens"`
						CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
						CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}

			if cwd == "" && entry.CWD != "" {
				cwd = normalizePath(entry.CWD)
			}
			if entry.Type == "user" {
				userTurns++
			}
			if entry.Message != nil && entry.Message.Role == "assistant" {
				assistantTurns++
			}
			if entry.Message == nil || entry.Message.Usage == nil {
				continue
			}

			u := entry.Message.Usage
			primary += u.InputTokens + u.OutputTokens
			alt += u.InputTokens + u.OutputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		}
		if scanErr := scanner.Err(); scanErr != nil && scanErr != io.EOF {
			return nil
		}

		if primary <= 0 {
			return nil
		}
		if repoFilter != "" && cwd != repoFilter {
			return nil
		}
		if int(userTurns) < minUserTurns {
			return nil
		}

		sessions = append(sessions, sessionMetrics{
			Provider:       "claude",
			File:           path,
			CWD:            cwd,
			TokensPrimary:  primary,
			TokensAlt:      alt,
			UserTurns:      userTurns,
			AssistantTurns: assistantTurns,
		})
		return nil
	})

	return sessions, err
}

func summarizeProvider(provider string, sessions []sessionMetrics, originators map[string]int) providerSummary {
	s := providerSummary{
		Provider:    provider,
		Sessions:    len(sessions),
		Originators: originators,
	}
	if len(sessions) == 0 {
		return s
	}

	primaryVals := make([]int64, 0, len(sessions))
	altVals := make([]int64, 0, len(sessions))
	userTurns := make([]int64, 0, len(sessions))
	assistantTurns := make([]int64, 0, len(sessions))
	primaryPerTurn := make([]int64, 0, len(sessions))
	altPerTurn := make([]int64, 0, len(sessions))

	for i, m := range sessions {
		primaryVals = append(primaryVals, m.TokensPrimary)
		altVals = append(altVals, m.TokensAlt)
		userTurns = append(userTurns, m.UserTurns)
		assistantTurns = append(assistantTurns, m.AssistantTurns)
		if m.UserTurns > 0 {
			primaryPerTurn = append(primaryPerTurn, m.TokensPrimary/m.UserTurns)
			altPerTurn = append(altPerTurn, m.TokensAlt/m.UserTurns)
		}
		if i < 10 {
			s.SampleFiles = append(s.SampleFiles, m.File)
		}
	}

	s.TokensPrimary = calcStats(primaryVals)
	s.TokensAlt = calcStats(altVals)
	s.UserTurns = calcStats(userTurns)
	s.AssistantTurns = calcStats(assistantTurns)
	s.PrimaryPerUserTurn = calcStats(primaryPerTurn)
	s.AltPerUserTurn = calcStats(altPerTurn)

	if provider == "codex" {
		s.PrimaryPerSessionNote = "primary = billable (non-cached input + output + reasoning output)"
		s.AltPerSessionNote = "alt = raw total (input + cached input + output + reasoning output)"
	}
	if provider == "claude" {
		s.PrimaryPerSessionNote = "primary = input + output"
		s.AltPerSessionNote = "alt = input + output + cache_read_input + cache_creation_input"
	}

	if s.Sessions < 10 {
		s.Warnings = append(s.Warnings, "Low sample count; treat ratios as directional only.")
	}

	return s
}

func computeRatios(codex, claude providerSummary) ratioSummary {
	r := ratioSummary{
		SuggestedMetric: "codex primary per-user-turn / claude alt per-user-turn",
	}

	r.CodexPrimaryToClaudePrimaryPerSession = safeRatio(codex.TokensPrimary.Median, claude.TokensPrimary.Median)
	r.CodexPrimaryToClaudeAltPerSession = safeRatio(codex.TokensPrimary.Median, claude.TokensAlt.Median)
	r.CodexPrimaryToClaudePrimaryPerTurn = safeRatio(codex.PrimaryPerUserTurn.Median, claude.PrimaryPerUserTurn.Median)
	r.CodexPrimaryToClaudeAltPerTurn = safeRatio(codex.PrimaryPerUserTurn.Median, claude.AltPerUserTurn.Median)

	// Current default suggestion favors cache-inclusive Claude accounting for subscription calibration.
	r.SuggestedMultiplier = r.CodexPrimaryToClaudeAltPerTurn
	return r
}

func printReport(r report, verbose bool) {
	fmt.Println("Provider Calibration Report")
	fmt.Println("===========================")
	if r.RepoFilter != "" {
		fmt.Printf("Repo filter: %s\n", r.RepoFilter)
	} else {
		fmt.Println("Repo filter: (none)")
	}
	if r.CodexOriginator != "" {
		fmt.Printf("Codex originator filter: %s\n", r.CodexOriginator)
	} else {
		fmt.Println("Codex originator filter: (none)")
	}
	fmt.Printf("Minimum user turns: %d\n\n", r.MinUserTurns)

	printProviderSummary(r.Codex)
	fmt.Println()
	printProviderSummary(r.Claude)
	fmt.Println()

	fmt.Println("Ratios")
	fmt.Println("------")
	fmt.Printf("codex_primary / claude_primary (per session median): %.2fx\n", r.Ratios.CodexPrimaryToClaudePrimaryPerSession)
	fmt.Printf("codex_primary / claude_alt     (per session median): %.2fx\n", r.Ratios.CodexPrimaryToClaudeAltPerSession)
	fmt.Printf("codex_primary / claude_primary (per user-turn median): %.2fx\n", r.Ratios.CodexPrimaryToClaudePrimaryPerTurn)
	fmt.Printf("codex_primary / claude_alt     (per user-turn median): %.2fx\n", r.Ratios.CodexPrimaryToClaudeAltPerTurn)
	fmt.Printf("suggested multiplier (%s): %.2fx\n", r.Ratios.SuggestedMetric, r.Ratios.SuggestedMultiplier)

	if len(r.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings")
		fmt.Println("--------")
		for _, w := range r.Warnings {
			fmt.Printf("- %s\n", w)
		}
	}

	if verbose {
		fmt.Println()
		fmt.Println("Methodology")
		fmt.Println("-----------")
		for _, n := range r.MethodologyNotes {
			fmt.Printf("- %s\n", n)
		}
	}
}

func printProviderSummary(s providerSummary) {
	fmt.Printf("[%s]\n", s.Provider)
	fmt.Printf("sessions: %d\n", s.Sessions)
	if len(s.Originators) > 0 {
		keys := make([]string, 0, len(s.Originators))
		for k := range s.Originators {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%d", k, s.Originators[k]))
		}
		fmt.Printf("originators: %s\n", strings.Join(parts, ", "))
	}
	fmt.Printf("tokens/session (primary): median=%s mean=%s p90=%s\n", formatInt(s.TokensPrimary.Median), formatInt(s.TokensPrimary.Mean), formatInt(s.TokensPrimary.P90))
	fmt.Printf("tokens/session (alt):     median=%s mean=%s p90=%s\n", formatInt(s.TokensAlt.Median), formatInt(s.TokensAlt.Mean), formatInt(s.TokensAlt.P90))
	fmt.Printf("user turns/session:       median=%s mean=%s p90=%s\n", formatInt(s.UserTurns.Median), formatInt(s.UserTurns.Mean), formatInt(s.UserTurns.P90))
	fmt.Printf("tokens/user-turn (primary): median=%s mean=%s p90=%s\n", formatInt(s.PrimaryPerUserTurn.Median), formatInt(s.PrimaryPerUserTurn.Mean), formatInt(s.PrimaryPerUserTurn.P90))
	fmt.Printf("tokens/user-turn (alt):     median=%s mean=%s p90=%s\n", formatInt(s.AltPerUserTurn.Median), formatInt(s.AltPerUserTurn.Mean), formatInt(s.AltPerUserTurn.P90))
	if s.PrimaryPerSessionNote != "" {
		fmt.Printf("note primary: %s\n", s.PrimaryPerSessionNote)
	}
	if s.AltPerSessionNote != "" {
		fmt.Printf("note alt:     %s\n", s.AltPerSessionNote)
	}
	for _, w := range s.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func calcStats(vals []int64) summaryStats {
	if len(vals) == 0 {
		return summaryStats{}
	}
	sorted := append([]int64(nil), vals...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var sum int64
	for _, v := range sorted {
		sum += v
	}

	return summaryStats{
		Count:  len(sorted),
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Mean:   sum / int64(len(sorted)),
		Median: percentile(sorted, 0.50),
		P75:    percentile(sorted, 0.75),
		P90:    percentile(sorted, 0.90),
	}
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func safeRatio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func nonNegative(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(expandPath(path))
}

func expandPath(path string) string {
	if path == "~" {
		return userHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(userHomeDir(), path[2:])
	}
	return path
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func formatInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
