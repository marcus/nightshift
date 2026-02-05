package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/marcus/nightshift/internal/config"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View logs",
	Long: `View nightshift logs.

Displays recent log entries. Use --follow to stream logs in real-time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")
		export, _ := cmd.Flags().GetString("export")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")
		level, _ := cmd.Flags().GetString("level")
		component, _ := cmd.Flags().GetString("component")
		match, _ := cmd.Flags().GetString("match")
		summary, _ := cmd.Flags().GetBool("summary")
		raw, _ := cmd.Flags().GetBool("raw")
		noColor, _ := cmd.Flags().GetBool("no-color")
		pathOverride, _ := cmd.Flags().GetString("path")

		if noColor {
			lipgloss.SetColorProfile(termenv.Ascii)
		}

		logDir := resolveLogDir(pathOverride)

		filter := logFilter{
			level:     strings.ToLower(strings.TrimSpace(level)),
			component: strings.ToLower(strings.TrimSpace(component)),
			match:     strings.ToLower(strings.TrimSpace(match)),
		}
		if filter.level != "" && levelRank(filter.level) == 0 {
			return fmt.Errorf("invalid log level %q (use debug|info|warn|error)", filter.level)
		}
		if sinceStr != "" {
			parsed, err := parseTimeInput(sinceStr, time.Local)
			if err != nil {
				return err
			}
			filter.since = &parsed
		}
		if untilStr != "" {
			parsed, err := parseTimeInput(untilStr, time.Local)
			if err != nil {
				return err
			}
			filter.until = &parsed
		}

		if export != "" {
			return exportLogs(logDir, export, filter, tail)
		}

		if follow {
			if summary {
				return fmt.Errorf("--summary cannot be used with --follow")
			}
			if filter.until != nil {
				return fmt.Errorf("--until cannot be used with --follow")
			}
			return followLogs(logDir, tail, filter, raw)
		}

		return showLogs(logDir, tail, filter, summary, raw)
	},
}

func init() {
	logsCmd.Flags().IntP("tail", "n", 50, "Number of log lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().StringP("export", "e", "", "Export logs to file")
	logsCmd.Flags().String("since", "", "Start time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339)")
	logsCmd.Flags().String("until", "", "End time (YYYY-MM-DD, YYYY-MM-DD HH:MM, or RFC3339)")
	logsCmd.Flags().String("level", "", "Minimum log level (debug|info|warn|error)")
	logsCmd.Flags().String("component", "", "Filter by component substring")
	logsCmd.Flags().String("match", "", "Filter by message substring")
	logsCmd.Flags().Bool("summary", false, "Show summary only")
	logsCmd.Flags().Bool("raw", false, "Show raw log lines without formatting")
	logsCmd.Flags().Bool("no-color", false, "Disable ANSI colors")
	logsCmd.Flags().String("path", "", "Override log directory")
	rootCmd.AddCommand(logsCmd)
}

func defaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "logs")
}

func resolveLogDir(override string) string {
	if override != "" {
		return override
	}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		if cfg.ExpandedLogPath() != "" {
			return cfg.ExpandedLogPath()
		}
	}
	return defaultLogPath()
}

// logEntry represents a parsed JSON log line
type logEntry struct {
	Level     string    `json:"level"`
	Time      time.Time `json:"time"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type logFilter struct {
	since     *time.Time
	until     *time.Time
	level     string
	component string
	match     string
}

type logRecord struct {
	entry  logEntry
	raw    string
	parsed bool
}

type logStats struct {
	files      int
	matched    int
	raw        int
	levels     map[string]int
	components map[string]int
	first      *time.Time
	last       *time.Time
}

type logStyles struct {
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Label      lipgloss.Style
	Muted      lipgloss.Style
	Time       lipgloss.Style
	Component  lipgloss.Style
	LevelDebug lipgloss.Style
	LevelInfo  lipgloss.Style
	LevelWarn  lipgloss.Style
	LevelError lipgloss.Style
}

func newLogStyles() logStyles {
	return logStyles{
		Title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Subtitle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Label:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Time:       lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Component:  lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		LevelDebug: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		LevelInfo:  lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
		LevelWarn:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		LevelError: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
	}
}

func showLogs(logDir string, tail int, filter logFilter, summary bool, raw bool) error {
	entries, stats, err := loadLogEntries(logDir, filter, tail)
	if err != nil {
		return err
	}

	if stats.files == 0 {
		fmt.Println("No log files found.")
		return nil
	}

	if len(entries) == 0 {
		fmt.Println("No log entries matched the filters.")
		return nil
	}

	if raw {
		for _, entry := range entries {
			fmt.Println(entry.raw)
		}
		return nil
	}

	styles := newLogStyles()
	fmt.Print(renderLogHeader(styles, stats, tail, filter, len(entries)))

	if summary {
		return nil
	}

	for _, entry := range entries {
		fmt.Println(renderLogLine(entry, styles))
	}

	return nil
}

func followLogs(logDir string, initialLines int, filter logFilter, raw bool) error {
	entries, stats, err := loadLogEntries(logDir, filter, initialLines)
	if err != nil {
		return err
	}

	if stats.files == 0 {
		fmt.Println("No log files found.")
		return nil
	}

	styles := newLogStyles()
	if raw {
		for _, entry := range entries {
			fmt.Println(entry.raw)
		}
	} else {
		fmt.Print(renderLogHeader(styles, stats, initialLines, filter, len(entries)))
		for _, entry := range entries {
			fmt.Println(renderLogLine(entry, styles))
		}
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	// Watch the log directory
	if err := watcher.Add(logDir); err != nil {
		return fmt.Errorf("watching log dir: %w", err)
	}

	// Track current log file
	currentFile := currentLogFile(logDir)
	var file *os.File
	var reader *bufio.Reader

	if currentFile != "" {
		file, err = os.Open(currentFile)
		if err == nil {
			// Seek to end
			_, _ = file.Seek(0, io.SeekEnd)
			reader = bufio.NewReader(file)
		}
	}

	if !raw {
		fmt.Println("--- Following logs (Ctrl+C to exit) ---")
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Check for new log file (date rollover)
			newFile := currentLogFile(logDir)
			if newFile != currentFile {
				if file != nil {
					_ = file.Close()
				}
				currentFile = newFile
				file, err = os.Open(currentFile)
				if err != nil {
					continue
				}
				reader = bufio.NewReader(file)
			}

			// Read new lines on write
			if event.Op&fsnotify.Write == fsnotify.Write && reader != nil {
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					record := parseLogLine(strings.TrimSuffix(line, "\n"))
					if !matchesFilter(record, filter) {
						continue
					}
					if raw {
						fmt.Println(record.raw)
					} else {
						fmt.Println(renderLogLine(record, styles))
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

func exportLogs(logDir, outFile string, filter logFilter, tail int) error {
	entries, stats, err := loadLogEntries(logDir, filter, tail)
	if err != nil {
		return err
	}
	if stats.files == 0 {
		return fmt.Errorf("no log files found")
	}
	if len(entries) == 0 {
		return fmt.Errorf("no log entries matched the filters")
	}

	out, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() { _ = out.Close() }()

	for _, entry := range entries {
		if _, err := out.WriteString(entry.raw + "\n"); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
	}

	fmt.Printf("Exported %d log lines to %s\n", len(entries), outFile)
	return nil
}

func getLogFiles(logDir string) ([]string, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading log dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "nightshift-") && strings.HasSuffix(name, ".log") {
			files = append(files, filepath.Join(logDir, name))
		}
	}

	// Sort by date ascending (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i] < files[j]
	})

	return files, nil
}

func currentLogFile(logDir string) string {
	filename := fmt.Sprintf("nightshift-%s.log", time.Now().Format("2006-01-02"))
	path := filepath.Join(logDir, filename)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func loadLogEntries(logDir string, filter logFilter, tail int) ([]logRecord, logStats, error) {
	files, err := getLogFiles(logDir)
	if err != nil {
		return nil, logStats{}, err
	}

	stats := logStats{
		files:      len(files),
		levels:     make(map[string]int),
		components: make(map[string]int),
	}

	var entries []logRecord
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			record := parseLogLine(line)
			if !matchesFilter(record, filter) {
				continue
			}
			stats.matched++
			updateLogStats(&stats, record)
			entries = appendWithTail(entries, record, tail)
		}
		_ = f.Close()
	}

	return entries, stats, nil
}

func parseLogLine(line string) logRecord {
	record := logRecord{raw: line}
	var entry logEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Level != "" {
		record.entry = entry
		record.parsed = true
	}
	return record
}

func matchesFilter(record logRecord, filter logFilter) bool {
	if filter.level != "" {
		if !record.parsed {
			return false
		}
		if levelRank(record.entry.Level) < levelRank(filter.level) {
			return false
		}
	}
	if filter.component != "" {
		if !record.parsed {
			return false
		}
		if !strings.Contains(strings.ToLower(record.entry.Component), filter.component) {
			return false
		}
	}
	if filter.match != "" {
		if record.parsed {
			content := strings.ToLower(record.entry.Message + " " + record.entry.Error)
			if !strings.Contains(content, filter.match) {
				return false
			}
		} else if !strings.Contains(strings.ToLower(record.raw), filter.match) {
			return false
		}
	}
	if filter.since != nil || filter.until != nil {
		if !record.parsed {
			return false
		}
		ts := record.entry.Time
		if filter.since != nil && ts.Before(*filter.since) {
			return false
		}
		if filter.until != nil && ts.After(*filter.until) {
			return false
		}
	}
	return true
}

func updateLogStats(stats *logStats, record logRecord) {
	if record.parsed {
		level := strings.ToLower(record.entry.Level)
		stats.levels[level]++
		if record.entry.Component != "" {
			stats.components[record.entry.Component]++
		}
		ts := record.entry.Time
		if stats.first == nil || ts.Before(*stats.first) {
			stats.first = &ts
		}
		if stats.last == nil || ts.After(*stats.last) {
			stats.last = &ts
		}
		return
	}
	stats.raw++
}

func appendWithTail(entries []logRecord, record logRecord, tail int) []logRecord {
	if tail <= 0 {
		return append(entries, record)
	}
	if len(entries) < tail {
		return append(entries, record)
	}
	copy(entries, entries[1:])
	entries[len(entries)-1] = record
	return entries
}

func renderLogHeader(styles logStyles, stats logStats, tail int, filter logFilter, displayed int) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Nightshift Logs"))
	b.WriteString("\n")

	rangeLabel := "time unknown"
	if stats.first != nil && stats.last != nil {
		rangeLabel = fmt.Sprintf("%s → %s", formatTimeShort(*stats.first), formatTimeShort(*stats.last))
	}

	summaryParts := []string{
		fmt.Sprintf("Files: %d", stats.files),
		fmt.Sprintf("Matches: %d", stats.matched),
		fmt.Sprintf("Showing: %d", displayed),
	}
	if tail > 0 && displayed < stats.matched {
		summaryParts[2] = fmt.Sprintf("Showing: last %d", displayed)
	}

	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("%s | %s", rangeLabel, strings.Join(summaryParts, " | "))))
	b.WriteString("\n")

	if len(stats.levels) > 0 || stats.raw > 0 {
		var levels []string
		for _, key := range []string{"debug", "info", "warn", "error"} {
			if count := stats.levels[key]; count > 0 {
				levels = append(levels, fmt.Sprintf("%s %d", strings.ToUpper(key), count))
			}
		}
		if stats.raw > 0 {
			levels = append(levels, fmt.Sprintf("RAW %d", stats.raw))
		}
		if len(levels) > 0 {
			b.WriteString(styles.Label.Render("Levels: "))
			b.WriteString(styles.Muted.Render(strings.Join(levels, " · ")))
			b.WriteString("\n")
		}
	}

	if len(stats.components) > 0 {
		var components []string
		for name, count := range stats.components {
			components = append(components, fmt.Sprintf("%s (%d)", name, count))
		}
		sort.Strings(components)
		b.WriteString(styles.Label.Render("Components: "))
		b.WriteString(styles.Muted.Render(strings.Join(components, ", ")))
		b.WriteString("\n")
	}

	var filters []string
	if filter.level != "" {
		filters = append(filters, fmt.Sprintf("level>=%s", filter.level))
	}
	if filter.component != "" {
		filters = append(filters, fmt.Sprintf("component~=%s", filter.component))
	}
	if filter.match != "" {
		filters = append(filters, fmt.Sprintf("match~=%s", filter.match))
	}
	if filter.since != nil {
		filters = append(filters, fmt.Sprintf("since=%s", formatTimeShort(*filter.since)))
	}
	if filter.until != nil {
		filters = append(filters, fmt.Sprintf("until=%s", formatTimeShort(*filter.until)))
	}
	if len(filters) > 0 {
		b.WriteString(styles.Muted.Render("Filters: " + strings.Join(filters, ", ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return b.String()
}

func renderLogLine(record logRecord, styles logStyles) string {
	if !record.parsed {
		return styles.Muted.Render(record.raw)
	}

	ts := styles.Time.Render(record.entry.Time.Format("15:04:05"))
	levelTag := formatLogLevel(record.entry.Level)
	level := styleLogLevel(styles, record.entry.Level).Render(levelTag)

	component := record.entry.Component
	componentLabel := ""
	if component != "" {
		componentLabel = styles.Component.Render("[" + component + "]")
	}

	message := record.entry.Message
	if record.entry.Error != "" {
		message = message + " error=" + record.entry.Error
	}

	if componentLabel != "" {
		return fmt.Sprintf("%s %s %s %s", ts, level, componentLabel, message)
	}
	return fmt.Sprintf("%s %s %s", ts, level, message)
}

func styleLogLevel(styles logStyles, level string) lipgloss.Style {
	switch strings.ToLower(level) {
	case "debug":
		return styles.LevelDebug
	case "info":
		return styles.LevelInfo
	case "warn":
		return styles.LevelWarn
	case "error":
		return styles.LevelError
	default:
		return styles.LevelInfo
	}
}

func levelRank(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 1
	case "info":
		return 2
	case "warn":
		return 3
	case "error":
		return 4
	default:
		return 0
	}
}

func formatLogLevel(level string) string {
	lower := strings.ToLower(level)
	switch lower {
	case "debug":
		return "DBG"
	case "info":
		return "INF"
	case "warn":
		return "WRN"
	case "error":
		return "ERR"
	}
	if lower == "" {
		return "UNK"
	}
	if len(lower) <= 3 {
		return strings.ToUpper(lower)
	}
	return strings.ToUpper(lower[:3])
}
