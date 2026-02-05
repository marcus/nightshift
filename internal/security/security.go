// Package security provides security and safety features for nightshift.
// Implements credential validation, sandboxed execution, audit logging,
// and safe defaults to protect against misuse and runaway costs.
package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SafetyMode determines the level of access nightshift has.
type SafetyMode string

const (
	// ModeReadOnly allows only read operations (first run default).
	ModeReadOnly SafetyMode = "read_only"
	// ModeNormal allows normal operations with safety checks.
	ModeNormal SafetyMode = "normal"
)

// Config holds security configuration.
type Config struct {
	Mode              SafetyMode // Operating mode
	EnableWrites      bool       // Allow write operations
	MaxBudgetPercent  int        // Max budget per run (default 75%)
	AllowGitPush      bool       // Allow pushing to remote repos
	AllowNetworkAgent bool       // Allow network access for agents
	AuditLogPath      string     // Path for audit logs
	FirstRunFile      string     // File to track first run state
}

// DefaultConfig returns safe default configuration.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Mode:              ModeReadOnly,
		EnableWrites:      false,
		MaxBudgetPercent:  75,
		AllowGitPush:      false,
		AllowNetworkAgent: false,
		AuditLogPath:      filepath.Join(home, ".local", "share", "nightshift", "audit"),
		FirstRunFile:      filepath.Join(home, ".local", "share", "nightshift", ".first_run_complete"),
	}
}

// Manager coordinates all security features.
type Manager struct {
	config      Config
	credentials *CredentialManager
	audit       *AuditLogger
	mu          sync.RWMutex
}

// NewManager creates a security manager with given config.
func NewManager(cfg Config) (*Manager, error) {
	// Ensure audit log directory exists
	if cfg.AuditLogPath != "" {
		if err := os.MkdirAll(cfg.AuditLogPath, 0700); err != nil {
			return nil, fmt.Errorf("creating audit log dir: %w", err)
		}
	}

	m := &Manager{
		config:      cfg,
		credentials: NewCredentialManager(),
	}

	// Initialize audit logger
	audit, err := NewAuditLogger(cfg.AuditLogPath)
	if err != nil {
		return nil, fmt.Errorf("initializing audit logger: %w", err)
	}
	m.audit = audit

	return m, nil
}

// IsFirstRun checks if this is the first run of nightshift.
func (m *Manager) IsFirstRun() bool {
	_, err := os.Stat(m.config.FirstRunFile)
	return os.IsNotExist(err)
}

// MarkFirstRunComplete marks that first run setup is complete.
func (m *Manager) MarkFirstRunComplete() error {
	dir := filepath.Dir(m.config.FirstRunFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dir: %w", err)
	}
	f, err := os.Create(m.config.FirstRunFile)
	if err != nil {
		return fmt.Errorf("marking first run complete: %w", err)
	}
	return f.Close()
}

// ValidateWriteAccess checks if write operations are allowed.
func (m *Manager) ValidateWriteAccess() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.IsFirstRun() && !m.config.EnableWrites {
		return errors.New("first run: write operations disabled. Use --enable-writes to allow modifications")
	}

	if m.config.Mode == ModeReadOnly && !m.config.EnableWrites {
		return errors.New("read-only mode: write operations disabled")
	}

	return nil
}

// ValidateGitPush checks if git push operations are allowed.
func (m *Manager) ValidateGitPush() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.config.AllowGitPush {
		return errors.New("git push disabled: use --allow-push to enable")
	}

	return nil
}

// ValidateBudgetSpend checks if spending within budget limits.
func (m *Manager) ValidateBudgetSpend(currentPercent int) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if currentPercent > m.config.MaxBudgetPercent {
		return fmt.Errorf("budget limit exceeded: %d%% > max %d%%", currentPercent, m.config.MaxBudgetPercent)
	}

	return nil
}

// Credentials returns the credential manager.
func (m *Manager) Credentials() *CredentialManager {
	return m.credentials
}

// Audit returns the audit logger.
func (m *Manager) Audit() *AuditLogger {
	return m.audit
}

// Config returns current security config.
func (m *Manager) Config() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetMode updates the safety mode.
func (m *Manager) SetMode(mode SafetyMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Mode = mode
}

// EnableWrites enables write operations.
func (m *Manager) EnableWrites(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.EnableWrites = enable
}

// Close cleans up security manager resources.
func (m *Manager) Close() error {
	if m.audit != nil {
		return m.audit.Close()
	}
	return nil
}

// CheckPreExecution validates all safety checks before agent execution.
func (m *Manager) CheckPreExecution(op Operation) error {
	// Log operation attempt
	if m.audit != nil {
		_ = m.audit.LogOperation(op)
	}

	// Check credentials
	if err := m.credentials.ValidateRequired(); err != nil {
		return fmt.Errorf("credential check failed: %w", err)
	}

	// Check write access for write operations
	if op.Type == OpFileWrite || op.Type == OpGitCommit {
		if err := m.ValidateWriteAccess(); err != nil {
			return err
		}
	}

	// Check git push permission
	if op.Type == OpGitPush {
		if err := m.ValidateGitPush(); err != nil {
			return err
		}
	}

	return nil
}

// Operation types for safety checks.
type OperationType string

const (
	OpAgentInvoke OperationType = "agent_invoke"
	OpFileRead    OperationType = "file_read"
	OpFileWrite   OperationType = "file_write"
	OpGitCommit   OperationType = "git_commit"
	OpGitPush     OperationType = "git_push"
	OpNetworkCall OperationType = "network_call"
)

// Operation represents an action being performed.
type Operation struct {
	Type        OperationType
	Target      string            // File path, git repo, URL, etc.
	Agent       string            // Agent performing the operation
	TaskID      string            // Associated task ID
	Description string            // Human-readable description
	Metadata    map[string]string // Additional context
}
