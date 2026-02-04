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

	"github.com/fsnotify/fsnotify"
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

		logDir := defaultLogPath()

		if export != "" {
			return exportLogs(logDir, export)
		}

		if follow {
			return followLogs(logDir, tail)
		}

		return showLogs(logDir, tail)
	},
}

func init() {
	logsCmd.Flags().IntP("tail", "n", 50, "Number of log lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().StringP("export", "e", "", "Export logs to file")
	rootCmd.AddCommand(logsCmd)
}

func defaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nightshift", "logs")
}

// logEntry represents a parsed JSON log line
type logEntry struct {
	Level     string    `json:"level"`
	Time      time.Time `json:"time"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func showLogs(logDir string, n int) error {
	files, err := getLogFiles(logDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No log files found.")
		return nil
	}

	lines := readLastLines(files, n)

	for _, line := range lines {
		printLogLine(line)
	}

	return nil
}

func followLogs(logDir string, initialLines int) error {
	files, err := getLogFiles(logDir)
	if err != nil {
		return err
	}

	// Show initial lines
	if len(files) > 0 && initialLines > 0 {
		lines := readLastLines(files, initialLines)
		for _, line := range lines {
			printLogLine(line)
		}
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

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
			file.Seek(0, io.SeekEnd)
			reader = bufio.NewReader(file)
		}
	}

	fmt.Println("--- Following logs (Ctrl+C to exit) ---")

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
					file.Close()
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
					printLogLine(strings.TrimSuffix(line, "\n"))
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

func exportLogs(logDir, outFile string) error {
	files, err := getLogFiles(logDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no log files found")
	}

	out, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	totalLines := 0

	// Process files in chronological order (oldest first)
	for i := len(files) - 1; i >= 0; i-- {
		f, err := os.Open(files[i])
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			out.WriteString(scanner.Text() + "\n")
			totalLines++
		}
		f.Close()
	}

	fmt.Printf("Exported %d log lines to %s\n", totalLines, outFile)
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

	// Sort by date descending (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
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

func readLastLines(files []string, n int) []string {
	var lines []string

	for _, file := range files {
		if len(lines) >= n {
			break
		}

		fileLines := readFileLines(file)
		remaining := n - len(lines)

		if len(fileLines) <= remaining {
			lines = append(fileLines, lines...)
		} else {
			// Take last 'remaining' lines from this file
			lines = append(fileLines[len(fileLines)-remaining:], lines...)
		}
	}

	return lines
}

func readFileLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func printLogLine(line string) {
	// Try to parse as JSON
	var entry logEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		// Format nicely
		level := formatLogLevel(entry.Level)
		ts := entry.Time.Format("15:04:05")

		if entry.Component != "" {
			fmt.Printf("%s %s [%s] %s", ts, level, entry.Component, entry.Message)
		} else {
			fmt.Printf("%s %s %s", ts, level, entry.Message)
		}

		if entry.Error != "" {
			fmt.Printf(" error=%s", entry.Error)
		}
		fmt.Println()
	} else {
		// Print raw line
		fmt.Println(line)
	}
}

func formatLogLevel(level string) string {
	switch level {
	case "debug":
		return "DBG"
	case "info":
		return "INF"
	case "warn":
		return "WRN"
	case "error":
		return "ERR"
	default:
		return strings.ToUpper(level[:3])
	}
}
