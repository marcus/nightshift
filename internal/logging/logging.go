// Package logging provides structured logging with file rotation for nightshift.
// Supports JSON and text formats with date-based log file naming.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog with nightshift-specific functionality.
type Logger struct {
	zl        zerolog.Logger
	component string
	logDir    string
	file      *os.File
	mu        sync.Mutex
}

// Config holds logging configuration.
type Config struct {
	Level         string // debug, info, warn, error
	Path          string // Log directory path
	Format        string // json, text
	RetentionDays int    // Days to keep logs (default 7)
}

// DefaultConfig returns default logging configuration.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Level:         "info",
		Path:          filepath.Join(home, ".local", "share", "nightshift", "logs"),
		Format:        "json",
		RetentionDays: 7,
	}
}

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// Init initializes the global logger with the given configuration.
func Init(cfg Config) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	logger, err := New(cfg)
	if err != nil {
		return err
	}

	// Close previous global logger if exists
	if globalLogger != nil && globalLogger.file != nil {
		_ = globalLogger.file.Close()
	}

	globalLogger = logger
	return nil
}

// New creates a new Logger instance.
func New(cfg Config) (*Logger, error) {
	// Set defaults
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 7
	}

	// Parse log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create log directory
	if cfg.Path != "" {
		cfg.Path = expandPath(cfg.Path)
		if err := os.MkdirAll(cfg.Path, 0755); err != nil {
			return nil, fmt.Errorf("creating log dir: %w", err)
		}
	}

	logger := &Logger{
		logDir: cfg.Path,
	}

	// Set up output writers
	var writers []io.Writer

	// File output if path specified
	if cfg.Path != "" {
		logFile := logger.currentLogPath()
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		logger.file = f
		writers = append(writers, f)

		// Clean up old logs
		go logger.cleanOldLogs(cfg.RetentionDays)
	}

	// Configure output
	var output io.Writer
	if len(writers) == 0 {
		output = os.Stderr
	} else {
		output = io.MultiWriter(writers...)
	}

	// Apply format
	if cfg.Format == "text" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    true,
		}
	}

	logger.zl = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return logger, nil
}

// currentLogPath returns the log file path for today.
func (l *Logger) currentLogPath() string {
	filename := fmt.Sprintf("nightshift-%s.log", time.Now().Format("2006-01-02"))
	return filepath.Join(l.logDir, filename)
}

// cleanOldLogs removes log files older than retention days.
func (l *Logger) cleanOldLogs(retentionDays int) {
	if l.logDir == "" {
		return
	}

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "nightshift-") || !strings.HasSuffix(name, ".log") {
			continue
		}

		// Parse date from filename: nightshift-YYYY-MM-DD.log
		dateStr := strings.TrimPrefix(name, "nightshift-")
		dateStr = strings.TrimSuffix(dateStr, ".log")

		logDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if logDate.Before(cutoff) {
			_ = os.Remove(filepath.Join(l.logDir, name))
		}
	}
}

// WithComponent returns a new Logger with the component field set.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		zl:        l.zl.With().Str("component", component).Logger(),
		component: component,
		logDir:    l.logDir,
		file:      l.file,
	}
}

// With returns a new Logger with additional context fields.
func (l *Logger) With() zerolog.Context {
	return l.zl.With()
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Info logs an info message.
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.zl.Error().Msg(msg)
}

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(format string, args ...any) {
	l.zl.Debug().Msgf(format, args...)
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...any) {
	l.zl.Info().Msgf(format, args...)
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...any) {
	l.zl.Warn().Msgf(format, args...)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...any) {
	l.zl.Error().Msgf(format, args...)
}

// DebugCtx logs a debug message with context fields.
func (l *Logger) DebugCtx(msg string, fields map[string]any) {
	event := l.zl.Debug()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// InfoCtx logs an info message with context fields.
func (l *Logger) InfoCtx(msg string, fields map[string]any) {
	event := l.zl.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// WarnCtx logs a warning message with context fields.
func (l *Logger) WarnCtx(msg string, fields map[string]any) {
	event := l.zl.Warn()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// ErrorCtx logs an error message with context fields.
func (l *Logger) ErrorCtx(msg string, fields map[string]any) {
	event := l.zl.Error()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// Err logs an error with the error field.
func (l *Logger) Err(err error) *zerolog.Event {
	return l.zl.Error().Err(err)
}

// Event returns a new log event at the specified level.
func (l *Logger) Event(level string) *zerolog.Event {
	switch level {
	case "debug":
		return l.zl.Debug()
	case "warn":
		return l.zl.Warn()
	case "error":
		return l.zl.Error()
	default:
		return l.zl.Info()
	}
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// LogFiles returns a list of log files sorted by date (newest first).
func (l *Logger) LogFiles() ([]string, error) {
	if l.logDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "nightshift-") && strings.HasSuffix(name, ".log") {
			files = append(files, filepath.Join(l.logDir, name))
		}
	}

	// Sort by date descending (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	return files, nil
}

// Global logger functions

// Get returns the global logger.
func Get() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		// Return a default stderr logger if not initialized
		return &Logger{
			zl: zerolog.New(os.Stderr).With().Timestamp().Logger(),
		}
	}
	return globalLogger
}

// Component returns a logger with the specified component.
func Component(name string) *Logger {
	return Get().WithComponent(name)
}

// Debug logs a debug message to the global logger.
func Debug(msg string) {
	Get().Debug(msg)
}

// Info logs an info message to the global logger.
func Info(msg string) {
	Get().Info(msg)
}

// Warn logs a warning message to the global logger.
func Warn(msg string) {
	Get().Warn(msg)
}

// Error(msg string) logs an error message to the global logger.
func Error(msg string) {
	Get().Error(msg)
}

// Helper functions

func parseLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
