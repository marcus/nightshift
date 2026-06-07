package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CommandRunner executes commands for tmux interactions.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner executes commands using os/exec.
type ExecRunner struct{}

// Run runs the command and returns combined output.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// SessionOption configures a tmux Session.
type SessionOption func(*Session)

// WithWorkDir sets the session working directory.
func WithWorkDir(dir string) SessionOption {
	return func(s *Session) {
		s.workDir = dir
	}
}

// WithSize sets the session pane size.
func WithSize(width, height int) SessionOption {
	return func(s *Session) {
		s.width = width
		s.height = height
	}
}

// WithRunner sets the command runner.
func WithRunner(runner CommandRunner) SessionOption {
	return func(s *Session) {
		s.runner = runner
	}
}

// Session wraps a tmux session.
type Session struct {
	name    string
	workDir string
	width   int
	height  int
	runner  CommandRunner
}

// NewSession constructs a tmux session wrapper.
func NewSession(name string, opts ...SessionOption) *Session {
	s := &Session{
		name:   name,
		runner: ExecRunner{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start creates the tmux session.
func (s *Session) Start(ctx context.Context) error {
	args := []string{"new-session", "-d", "-s", s.name}
	if s.workDir != "" {
		args = append(args, "-c", s.workDir)
	}
	if _, err := s.run(ctx, args...); err != nil {
		return fmt.Errorf("tmux new-session: %w", err)
	}

	if s.width > 0 && s.height > 0 {
		if err := s.Resize(ctx, s.width, s.height); err != nil {
			return err
		}
	}

	return nil
}

// Resize sets the pane size.
func (s *Session) Resize(ctx context.Context, width, height int) error {
	if _, err := s.run(ctx, "resize-pane", "-t", s.name, "-x", fmt.Sprint(width), "-y", fmt.Sprint(height)); err != nil {
		return fmt.Errorf("tmux resize-pane: %w", err)
	}
	return nil
}

// SendKeys sends keys to the session.
func (s *Session) SendKeys(ctx context.Context, keys ...string) error {
	args := append([]string{"send-keys", "-t", s.name}, keys...)
	if _, err := s.run(ctx, args...); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}
	return nil
}

// CapturePane captures the current pane contents.
func (s *Session) CapturePane(ctx context.Context, captureArgs ...string) (string, error) {
	args := []string{"capture-pane", "-t", s.name, "-p"}
	args = append(args, captureArgs...)
	out, err := s.run(ctx, args...)
	if err != nil {
		return string(out), fmt.Errorf("tmux capture-pane: %w", err)
	}
	return string(out), nil
}

// Kill terminates the session.
func (s *Session) Kill(ctx context.Context) error {
	_, err := s.run(ctx, "kill-session", "-t", s.name)
	if err != nil {
		return fmt.Errorf("tmux kill-session: %w", err)
	}
	return nil
}

// WaitForPattern polls capture-pane until regex match or timeout.
func (s *Session) WaitForPattern(ctx context.Context, pattern *regexp.Regexp, timeout, pollInterval time.Duration, captureArgs ...string) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("timeout must be positive")
	}
	if pollInterval <= 0 {
		pollInterval = 200 * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastOutput string
	for {
		output, err := s.CapturePane(ctx, captureArgs...)
		if err == nil {
			lastOutput = output
			if pattern.MatchString(StripANSI(output)) {
				return output, nil
			}
		}

		select {
		case <-ctx.Done():
			return lastOutput, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Session) run(ctx context.Context, args ...string) ([]byte, error) {
	if s.runner == nil {
		s.runner = ExecRunner{}
	}
	return s.runner.Run(ctx, "tmux", args...)
}

var ansiRegexp = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\][^\x07]*\x07|[()][A-Z0-9])`)

// StripANSI removes ANSI escape codes from text.
func StripANSI(input string) string {
	return strings.TrimSpace(ansiRegexp.ReplaceAllString(input, ""))
}
