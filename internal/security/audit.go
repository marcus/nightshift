// Package security provides audit logging for nightshift operations.
// All significant operations are logged to an append-only audit log.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEventType categorizes audit events.
type AuditEventType string

const (
	AuditAgentStart     AuditEventType = "agent_start"
	AuditAgentComplete  AuditEventType = "agent_complete"
	AuditAgentError     AuditEventType = "agent_error"
	AuditFileRead       AuditEventType = "file_read"
	AuditFileWrite      AuditEventType = "file_write"
	AuditFileDelete     AuditEventType = "file_delete"
	AuditGitCommit      AuditEventType = "git_commit"
	AuditGitPush        AuditEventType = "git_push"
	AuditGitOperation   AuditEventType = "git_operation"
	AuditSecurityCheck  AuditEventType = "security_check"
	AuditSecurityDenied AuditEventType = "security_denied"
	AuditConfigChange   AuditEventType = "config_change"
	AuditBudgetCheck    AuditEventType = "budget_check"
)

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	EventType  AuditEventType    `json:"event_type"`
	Agent      string            `json:"agent,omitempty"`
	TaskID     string            `json:"task_id,omitempty"`
	Project    string            `json:"project,omitempty"`
	Target     string            `json:"target,omitempty"`
	Action     string            `json:"action,omitempty"`
	Result     string            `json:"result,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	TokensUsed int               `json:"tokens_used,omitempty"`
	Error      string            `json:"error,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	RequestID  string            `json:"request_id,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
}

// AuditLogger writes audit events to an append-only log file.
type AuditLogger struct {
	logDir    string
	file      *os.File
	mu        sync.Mutex
	sessionID string
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(logDir string) (*AuditLogger, error) {
	if logDir == "" {
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, ".local", "share", "nightshift", "audit")
	}

	// Ensure audit directory exists with restricted permissions
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("creating audit log dir: %w", err)
	}

	// Generate session ID for this logger instance
	sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())

	logger := &AuditLogger{
		logDir:    logDir,
		sessionID: sessionID,
	}

	// Open current log file
	if err := logger.openLogFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// openLogFile opens or creates the current day's audit log.
func (l *AuditLogger) openLogFile() error {
	filename := fmt.Sprintf("audit-%s.jsonl", time.Now().Format("2006-01-02"))
	path := filepath.Join(l.logDir, filename)

	// Open in append-only mode
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}

	l.file = f
	return nil
}

// Log writes an audit event to the log.
func (l *AuditLogger) Log(event AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Add session ID
	event.SessionID = l.sessionID

	// Generate request ID if not set
	if event.RequestID == "" {
		event.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	// Serialize to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling audit event: %w", err)
	}

	// Append newline for JSONL format
	data = append(data, '\n')

	// Write to file
	if _, err := l.file.Write(data); err != nil {
		return fmt.Errorf("writing audit event: %w", err)
	}

	// Sync to disk for durability
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("syncing audit log: %w", err)
	}

	return nil
}

// LogOperation logs an operation from the security manager.
func (l *AuditLogger) LogOperation(op Operation) error {
	eventType := l.operationToEventType(op.Type)

	return l.Log(AuditEvent{
		EventType: eventType,
		Agent:     op.Agent,
		TaskID:    op.TaskID,
		Target:    op.Target,
		Action:    string(op.Type),
		Metadata:  op.Metadata,
	})
}

// operationToEventType maps operation types to audit event types.
func (l *AuditLogger) operationToEventType(opType OperationType) AuditEventType {
	switch opType {
	case OpAgentInvoke:
		return AuditAgentStart
	case OpFileRead:
		return AuditFileRead
	case OpFileWrite:
		return AuditFileWrite
	case OpGitCommit:
		return AuditGitCommit
	case OpGitPush:
		return AuditGitPush
	default:
		return AuditEventType(opType)
	}
}

// LogAgentStart logs the start of an agent execution.
func (l *AuditLogger) LogAgentStart(agent, taskID, project string) error {
	return l.Log(AuditEvent{
		EventType: AuditAgentStart,
		Agent:     agent,
		TaskID:    taskID,
		Project:   project,
		Action:    "start",
	})
}

// LogAgentComplete logs the completion of an agent execution.
func (l *AuditLogger) LogAgentComplete(agent, taskID, project string, duration time.Duration, tokensUsed int, result string) error {
	return l.Log(AuditEvent{
		EventType:  AuditAgentComplete,
		Agent:      agent,
		TaskID:     taskID,
		Project:    project,
		Action:     "complete",
		Duration:   duration,
		TokensUsed: tokensUsed,
		Result:     result,
	})
}

// LogAgentError logs an agent execution error.
func (l *AuditLogger) LogAgentError(agent, taskID, project string, err error) error {
	return l.Log(AuditEvent{
		EventType: AuditAgentError,
		Agent:     agent,
		TaskID:    taskID,
		Project:   project,
		Action:    "error",
		Error:     err.Error(),
	})
}

// LogFileModification logs a file write or delete operation.
func (l *AuditLogger) LogFileModification(eventType AuditEventType, path, agent, taskID string) error {
	return l.Log(AuditEvent{
		EventType: eventType,
		Agent:     agent,
		TaskID:    taskID,
		Target:    path,
		Action:    string(eventType),
	})
}

// LogGitOperation logs a git operation.
func (l *AuditLogger) LogGitOperation(operation, repo, branch, agent, taskID string, metadata map[string]string) error {
	return l.Log(AuditEvent{
		EventType: AuditGitOperation,
		Agent:     agent,
		TaskID:    taskID,
		Project:   repo,
		Target:    branch,
		Action:    operation,
		Metadata:  metadata,
	})
}

// LogSecurityCheck logs a security check result.
func (l *AuditLogger) LogSecurityCheck(checkType, target, result string, allowed bool) error {
	eventType := AuditSecurityCheck
	if !allowed {
		eventType = AuditSecurityDenied
	}

	return l.Log(AuditEvent{
		EventType: eventType,
		Target:    target,
		Action:    checkType,
		Result:    result,
	})
}

// Close closes the audit log file.
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// RotateIfNeeded checks if the log file needs rotation (new day).
func (l *AuditLogger) RotateIfNeeded() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	expectedFilename := fmt.Sprintf("audit-%s.jsonl", time.Now().Format("2006-01-02"))
	expectedPath := filepath.Join(l.logDir, expectedFilename)

	if l.file != nil {
		currentPath := l.file.Name()
		if currentPath == expectedPath {
			// No rotation needed
			return nil
		}

		// Close old file
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("closing old audit log: %w", err)
		}
	}

	// Open new file
	return l.openLogFile()
}

// GetLogFiles returns a list of all audit log files.
func (l *AuditLogger) GetLogFiles() ([]string, error) {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil, fmt.Errorf("reading audit log dir: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			files = append(files, filepath.Join(l.logDir, entry.Name()))
		}
	}

	return files, nil
}

// ReadEvents reads audit events from a specific log file.
func ReadEvents(path string) ([]AuditEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading audit log: %w", err)
	}

	var events []AuditEvent
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var event AuditEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// splitLines splits data by newlines without allocating strings.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}

	// Handle last line without trailing newline
	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}
