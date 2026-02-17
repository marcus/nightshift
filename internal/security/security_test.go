package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != ModeReadOnly {
		t.Errorf("expected mode %s, got %s", ModeReadOnly, cfg.Mode)
	}

	if cfg.EnableWrites {
		t.Error("expected EnableWrites to be false by default")
	}

	if cfg.MaxBudgetPercent != 75 {
		t.Errorf("expected MaxBudgetPercent 75, got %d", cfg.MaxBudgetPercent)
	}

	if cfg.AllowGitPush {
		t.Error("expected AllowGitPush to be false by default")
	}
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Mode:         ModeReadOnly,
		AuditLogPath: filepath.Join(tmpDir, "audit"),
		FirstRunFile: filepath.Join(tmpDir, ".first_run"),
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = m.Close() }()

	if m.credentials == nil {
		t.Error("expected credentials manager to be initialized")
	}

	if m.audit == nil {
		t.Error("expected audit logger to be initialized")
	}
}

func TestIsFirstRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		AuditLogPath: filepath.Join(tmpDir, "audit"),
		FirstRunFile: filepath.Join(tmpDir, ".first_run"),
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = m.Close() }()

	// First run should return true
	if !m.IsFirstRun() {
		t.Error("expected IsFirstRun to return true")
	}

	// Mark complete
	if err := m.MarkFirstRunComplete(); err != nil {
		t.Fatalf("MarkFirstRunComplete failed: %v", err)
	}

	// Now should return false
	if m.IsFirstRun() {
		t.Error("expected IsFirstRun to return false after marking complete")
	}
}

func TestValidateWriteAccess(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		mode         SafetyMode
		enableWrites bool
		firstRun     bool
		wantErr      bool
	}{
		{
			name:         "read-only mode without writes",
			mode:         ModeReadOnly,
			enableWrites: false,
			firstRun:     false,
			wantErr:      true,
		},
		{
			name:         "read-only mode with writes enabled",
			mode:         ModeReadOnly,
			enableWrites: true,
			firstRun:     false,
			wantErr:      false,
		},
		{
			name:         "normal mode without writes flag",
			mode:         ModeNormal,
			enableWrites: false,
			firstRun:     false,
			wantErr:      false, // Normal mode allows writes without explicit flag
		},
		{
			name:         "normal mode with writes",
			mode:         ModeNormal,
			enableWrites: true,
			firstRun:     false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstRunFile := filepath.Join(tmpDir, tt.name+".first_run")
			cfg := Config{
				Mode:         tt.mode,
				EnableWrites: tt.enableWrites,
				AuditLogPath: filepath.Join(tmpDir, tt.name+"_audit"),
				FirstRunFile: firstRunFile,
			}

			// Create first run file if not first run
			if !tt.firstRun {
				_ = os.MkdirAll(filepath.Dir(firstRunFile), 0755)
				f, _ := os.Create(firstRunFile)
				_ = f.Close()
			}

			m, err := NewManager(cfg)
			if err != nil {
				t.Fatalf("NewManager failed: %v", err)
			}
			defer func() { _ = m.Close() }()

			err = m.ValidateWriteAccess()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWriteAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateGitPush(t *testing.T) {
	tmpDir := t.TempDir()

	// Test push not allowed
	cfg := Config{
		AllowGitPush: false,
		AuditLogPath: filepath.Join(tmpDir, "audit1"),
		FirstRunFile: filepath.Join(tmpDir, ".first_run1"),
	}

	m1, _ := NewManager(cfg)
	defer func() { _ = m1.Close() }()

	if err := m1.ValidateGitPush(); err == nil {
		t.Error("expected error when push not allowed")
	}

	// Test push allowed
	cfg.AllowGitPush = true
	cfg.AuditLogPath = filepath.Join(tmpDir, "audit2")
	cfg.FirstRunFile = filepath.Join(tmpDir, ".first_run2")

	m2, _ := NewManager(cfg)
	defer func() { _ = m2.Close() }()

	if err := m2.ValidateGitPush(); err != nil {
		t.Errorf("expected no error when push allowed, got: %v", err)
	}
}

func TestValidateBudgetSpend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		MaxBudgetPercent: 10,
		AuditLogPath:     filepath.Join(tmpDir, "audit"),
		FirstRunFile:     filepath.Join(tmpDir, ".first_run"),
	}

	m, _ := NewManager(cfg)
	defer func() { _ = m.Close() }()

	// Under budget
	if err := m.ValidateBudgetSpend(5); err != nil {
		t.Errorf("expected no error at 5%%, got: %v", err)
	}

	// At budget
	if err := m.ValidateBudgetSpend(10); err != nil {
		t.Errorf("expected no error at 10%%, got: %v", err)
	}

	// Over budget
	if err := m.ValidateBudgetSpend(15); err == nil {
		t.Error("expected error at 15%")
	}
}

func TestValidateProjectPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "home directory is blocked",
			path:    home,
			wantErr: true,
		},
		{
			name:    "root is blocked",
			path:    "/",
			wantErr: true,
		},
		{
			name:    "tmp is blocked",
			path:    "/tmp",
			wantErr: true,
		},
		{
			name:    "etc is blocked",
			path:    "/etc",
			wantErr: true,
		},
		{
			name:    "project subdirectory is allowed",
			path:    filepath.Join(home, "Sites", "my-project"),
			wantErr: false,
		},
		{
			name:    "deep project path is allowed",
			path:    filepath.Join(home, "code", "org", "repo"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSetMode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Mode:         ModeReadOnly,
		AuditLogPath: filepath.Join(tmpDir, "audit"),
		FirstRunFile: filepath.Join(tmpDir, ".first_run"),
	}

	m, _ := NewManager(cfg)
	defer func() { _ = m.Close() }()

	if m.Config().Mode != ModeReadOnly {
		t.Error("expected initial mode to be read-only")
	}

	m.SetMode(ModeNormal)

	if m.Config().Mode != ModeNormal {
		t.Error("expected mode to be normal after SetMode")
	}
}

func TestEnableWrites(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Mode:         ModeReadOnly, // Start in read-only mode
		EnableWrites: false,
		AuditLogPath: filepath.Join(tmpDir, "audit"),
		FirstRunFile: filepath.Join(tmpDir, ".first_run"),
	}

	// Mark not first run
	_ = os.MkdirAll(filepath.Dir(cfg.FirstRunFile), 0755)
	f, _ := os.Create(cfg.FirstRunFile)
	_ = f.Close()

	m, _ := NewManager(cfg)
	defer func() { _ = m.Close() }()

	// Should fail initially in read-only mode
	if err := m.ValidateWriteAccess(); err == nil {
		t.Error("expected error in read-only mode with writes disabled")
	}

	// Enable writes
	m.EnableWrites(true)

	// Should succeed now
	if err := m.ValidateWriteAccess(); err != nil {
		t.Errorf("expected no error after enabling writes, got: %v", err)
	}
}
