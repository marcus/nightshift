package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "default config",
			cfg: Config{
				Path:   tmpDir,
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "text format",
			cfg: Config{
				Path:   tmpDir,
				Level:  "debug",
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			cfg: Config{
				Path:  tmpDir,
				Level: "invalid",
			},
			wantErr: true,
		},
		{
			name: "no path (stderr only)",
			cfg: Config{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && logger != nil {
				_ = logger.Close()
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := New(Config{
		Path:   tmpDir,
		Level:  "debug",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test all log methods
	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	logger.Debugf("debug %s", "formatted")
	logger.Infof("info %s", "formatted")
	logger.Warnf("warn %s", "formatted")
	logger.Errorf("error %s", "formatted")

	logger.DebugCtx("debug ctx", map[string]any{"key": "value"})
	logger.InfoCtx("info ctx", map[string]any{"key": "value"})
	logger.WarnCtx("warn ctx", map[string]any{"key": "value"})
	logger.ErrorCtx("error ctx", map[string]any{"key": "value"})

	// Verify log file was created
	logFile := filepath.Join(tmpDir, "nightshift-"+time.Now().Format("2006-01-02")+".log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("log file not created: %s", logFile)
	}
}

func TestWithComponent(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := New(Config{
		Path:   tmpDir,
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	componentLogger := logger.WithComponent("test_component")
	if componentLogger.component != "test_component" {
		t.Errorf("expected component 'test_component', got '%s'", componentLogger.component)
	}

	componentLogger.Info("test message")
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old log files
	oldDates := []string{
		time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
		time.Now().AddDate(0, 0, -8).Format("2006-01-02"),
		time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
	}

	for _, date := range oldDates {
		filename := filepath.Join(tmpDir, "nightshift-"+date+".log")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test log file: %v", err)
		}
	}

	logger, err := New(Config{
		Path:          tmpDir,
		Level:         "info",
		Format:        "json",
		RetentionDays: 7,
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Wait for cleanup goroutine
	time.Sleep(100 * time.Millisecond)

	// Check that old files were deleted
	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "nightshift-") && strings.HasSuffix(name, ".log") {
			dateStr := strings.TrimPrefix(name, "nightshift-")
			dateStr = strings.TrimSuffix(dateStr, ".log")
			logDate, _ := time.Parse("2006-01-02", dateStr)
			cutoff := time.Now().AddDate(0, 0, -7)
			if logDate.Before(cutoff) {
				t.Errorf("old log file should have been deleted: %s", name)
			}
		}
	}
}

func TestLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test log files
	dates := []string{
		time.Now().Format("2006-01-02"),
		time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
	}

	for _, date := range dates {
		filename := filepath.Join(tmpDir, "nightshift-"+date+".log")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test log file: %v", err)
		}
	}

	logger := &Logger{logDir: tmpDir}
	files, err := logger.LogFiles()
	if err != nil {
		t.Fatalf("LogFiles() error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 log files, got %d", len(files))
	}

	// Verify sorted newest first
	if len(files) >= 2 && files[0] < files[1] {
		t.Error("log files not sorted newest first")
	}
}

func TestGlobalLogger(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(Config{
		Path:   tmpDir,
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Test global functions
	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	// Test component helper
	compLogger := Component("test")
	if compLogger.component != "test" {
		t.Errorf("Component() returned wrong component")
	}

	// Get() should return the initialized logger
	logger := Get()
	if logger == nil {
		t.Error("Get() returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("expected default level 'info', got '%s'", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected default format 'json', got '%s'", cfg.Format)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("expected default retention 7, got %d", cfg.RetentionDays)
	}
	if !strings.Contains(cfg.Path, "nightshift/logs") {
		t.Errorf("expected default path to contain 'nightshift/logs', got '%s'", cfg.Path)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level   string
		wantErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"error", false},
		{"DEBUG", false},
		{"INFO", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			_, err := parseLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLevel(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
		})
	}
}
