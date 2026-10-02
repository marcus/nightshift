// Package security provides credential management for nightshift.
// Credentials are loaded from environment variables only - never from config files.
package security

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Standard credential environment variables.
const (
	EnvAnthropicKey = "ANTHROPIC_API_KEY"
	EnvOpenAIKey    = "OPENAI_API_KEY"
)

// CredentialStatus represents the validation status of a credential.
type CredentialStatus struct {
	Name    string
	EnvVar  string
	Present bool
	Masked  string // Masked value for display (e.g., "sk-...abc")
}

// CredentialManager validates and provides access to credentials.
// Credentials are NEVER stored - only validated from environment.
type CredentialManager struct {
	warnings []string
}

// NewCredentialManager creates a new credential manager.
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		warnings: make([]string, 0),
	}
}

// ValidateRequired checks that required credentials are set.
// Returns error if any required credential is missing.
func (m *CredentialManager) ValidateRequired() error {
	// At least one AI provider key must be set
	anthropic := os.Getenv(EnvAnthropicKey)
	openai := os.Getenv(EnvOpenAIKey)

	if anthropic == "" && openai == "" {
		return fmt.Errorf("no AI provider credentials found: set %s or %s", EnvAnthropicKey, EnvOpenAIKey)
	}

	return nil
}

// ValidateAll checks all known credentials and returns their status.
func (m *CredentialManager) ValidateAll() []CredentialStatus {
	credentials := []struct {
		name   string
		envVar string
	}{
		{"Anthropic API Key", EnvAnthropicKey},
		{"OpenAI API Key", EnvOpenAIKey},
	}

	statuses := make([]CredentialStatus, 0, len(credentials))

	for _, cred := range credentials {
		value := os.Getenv(cred.envVar)
		status := CredentialStatus{
			Name:    cred.name,
			EnvVar:  cred.envVar,
			Present: value != "",
		}

		if status.Present {
			status.Masked = maskCredential(value)
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// HasAnthropicKey checks if Anthropic API key is available.
func (m *CredentialManager) HasAnthropicKey() bool {
	return os.Getenv(EnvAnthropicKey) != ""
}

// HasOpenAIKey checks if OpenAI API key is available.
func (m *CredentialManager) HasOpenAIKey() bool {
	return os.Getenv(EnvOpenAIKey) != ""
}

// CheckConfigForCredentials scans config content for potential credential leaks.
// Returns error if credentials appear to be stored in config.
func (m *CredentialManager) CheckConfigForCredentials(content string) error {
	// Patterns that suggest credentials in config
	dangerPatterns := []string{
		"api_key:",
		"apikey:",
		"api-key:",
		"secret:",
		"password:",
		"token:",
		"sk-", // OpenAI key prefix
	}

	contentLower := strings.ToLower(content)
	var found []string

	for _, pattern := range dangerPatterns {
		if strings.Contains(contentLower, pattern) {
			// Check if it's not just a reference to env var
			if !strings.Contains(contentLower, "${"+pattern) &&
				!strings.Contains(contentLower, "env(") {
				found = append(found, pattern)
			}
		}
	}

	if len(found) > 0 {
		return fmt.Errorf("potential credentials found in config (patterns: %s). Use environment variables instead", strings.Join(found, ", "))
	}

	return nil
}

// maskCredential returns a masked version of a credential for safe display.
func maskCredential(value string) string {
	if len(value) < 8 {
		return "***"
	}

	// Show first 3 and last 3 characters
	prefix := value[:3]
	suffix := value[len(value)-3:]
	return prefix + "..." + suffix
}

// isConfigFile checks if a file path looks like a config file.
func isConfigFile(path string) bool {
	configExtensions := []string{".yaml", ".yml", ".json", ".toml", ".conf", ".cfg"}
	configNames := []string{"config", "settings", "credentials", "secrets"}

	pathLower := strings.ToLower(path)

	for _, ext := range configExtensions {
		if strings.HasSuffix(pathLower, ext) {
			return true
		}
	}

	for _, name := range configNames {
		if strings.Contains(pathLower, name) {
			return true
		}
	}

	return false
}

// CredentialError represents a credential-related error.
type CredentialError struct {
	Credential string
	Message    string
}

// Error returns a formatted credential error message.
func (e *CredentialError) Error() string {
	return fmt.Sprintf("credential error (%s): %s", e.Credential, e.Message)
}

// Common credential errors.
var (
	ErrNoCredentials     = errors.New("no credentials available")
	ErrCredentialExpired = errors.New("credential may be expired")
	ErrInvalidCredential = errors.New("credential format invalid")
)

// ValidateCredentialFormat checks if a credential has a valid format.
func ValidateCredentialFormat(name, value string) error {
	if value == "" {
		return &CredentialError{Credential: name, Message: "empty value"}
	}

	switch name {
	case EnvAnthropicKey:
		// Anthropic keys typically start with "sk-ant-"
		if !strings.HasPrefix(value, "sk-ant-") && !strings.HasPrefix(value, "sk-") {
			return &CredentialError{Credential: name, Message: "unexpected format (expected sk-ant-* or sk-*)"}
		}
	case EnvOpenAIKey:
		// OpenAI keys typically start with "sk-"
		if !strings.HasPrefix(value, "sk-") {
			return &CredentialError{Credential: name, Message: "unexpected format (expected sk-*)"}
		}
	}

	return nil
}
