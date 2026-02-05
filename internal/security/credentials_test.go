package security

import (
	"os"
	"testing"
)

func TestCredentialManager_ValidateRequired(t *testing.T) {
	cm := NewCredentialManager()

	// Save original values
	origAnthropic := os.Getenv(EnvAnthropicKey)
	origOpenAI := os.Getenv(EnvOpenAIKey)
	defer func() {
		_ = os.Setenv(EnvAnthropicKey, origAnthropic)
		_ = os.Setenv(EnvOpenAIKey, origOpenAI)
	}()

	// Clear both
	_ = os.Unsetenv(EnvAnthropicKey)
	_ = os.Unsetenv(EnvOpenAIKey)

	// Should fail with no keys
	if err := cm.ValidateRequired(); err == nil {
		t.Error("expected error with no API keys set")
	}

	// Set Anthropic key
	_ = os.Setenv(EnvAnthropicKey, "sk-ant-test")

	if err := cm.ValidateRequired(); err != nil {
		t.Errorf("expected success with Anthropic key, got: %v", err)
	}

	// Clear and set OpenAI only
	_ = os.Unsetenv(EnvAnthropicKey)
	_ = os.Setenv(EnvOpenAIKey, "sk-test")

	if err := cm.ValidateRequired(); err != nil {
		t.Errorf("expected success with OpenAI key, got: %v", err)
	}
}

func TestCredentialManager_ValidateAll(t *testing.T) {
	cm := NewCredentialManager()

	// Save original values
	origAnthropic := os.Getenv(EnvAnthropicKey)
	origOpenAI := os.Getenv(EnvOpenAIKey)
	defer func() {
		_ = os.Setenv(EnvAnthropicKey, origAnthropic)
		_ = os.Setenv(EnvOpenAIKey, origOpenAI)
	}()

	// Set test values
	_ = os.Setenv(EnvAnthropicKey, "sk-ant-testkey12345")
	_ = os.Setenv(EnvOpenAIKey, "sk-openai-testkey999")

	statuses := cm.ValidateAll()

	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}

	for _, s := range statuses {
		if !s.Present {
			t.Errorf("expected %s to be present", s.Name)
		}
		if s.Masked == "" {
			t.Errorf("expected masked value for %s", s.Name)
		}
	}
}

func TestCredentialManager_HasKeys(t *testing.T) {
	cm := NewCredentialManager()

	// Save original values
	origAnthropic := os.Getenv(EnvAnthropicKey)
	origOpenAI := os.Getenv(EnvOpenAIKey)
	defer func() {
		_ = os.Setenv(EnvAnthropicKey, origAnthropic)
		_ = os.Setenv(EnvOpenAIKey, origOpenAI)
	}()

	_ = os.Setenv(EnvAnthropicKey, "test")
	_ = os.Unsetenv(EnvOpenAIKey)

	if !cm.HasAnthropicKey() {
		t.Error("expected HasAnthropicKey to return true")
	}

	if cm.HasOpenAIKey() {
		t.Error("expected HasOpenAIKey to return false")
	}
}

func TestCredentialManager_CheckConfigForCredentials(t *testing.T) {
	cm := NewCredentialManager()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "clean config",
			content: "schedule:\n  cron: '0 2 * * *'\nbudget:\n  max_percent: 75",
			wantErr: false,
		},
		{
			name:    "api_key present",
			content: "api_key: sk-test123456",
			wantErr: true,
		},
		{
			name:    "secret present",
			content: "secret: mysecretvalue",
			wantErr: true,
		},
		{
			name:    "password present",
			content: "password: mypassword123",
			wantErr: true,
		},
		{
			name:    "openai key pattern",
			content: "key: sk-abc123def456",
			wantErr: true,
		},
		{
			name:    "env var reference ok",
			content: "key: ${ANTHROPIC_API_KEY}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.CheckConfigForCredentials(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckConfigForCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskCredential(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sk-ant-12345678901234567890", "sk-...890"},
		{"short", "***"},
		{"sk-test", "***"},
		{"exactly8", "exa...ly8"},
	}

	for _, tt := range tests {
		result := maskCredential(tt.input)
		if result != tt.expected {
			t.Errorf("maskCredential(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidateCredentialFormat(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		value   string
		wantErr bool
	}{
		{
			name:    "valid anthropic key",
			envVar:  EnvAnthropicKey,
			value:   "sk-ant-api03-testing",
			wantErr: false,
		},
		{
			name:    "valid anthropic key alt format",
			envVar:  EnvAnthropicKey,
			value:   "sk-testing123",
			wantErr: false,
		},
		{
			name:    "invalid anthropic key",
			envVar:  EnvAnthropicKey,
			value:   "invalid-key",
			wantErr: true,
		},
		{
			name:    "valid openai key",
			envVar:  EnvOpenAIKey,
			value:   "sk-proj-testing123",
			wantErr: false,
		},
		{
			name:    "invalid openai key",
			envVar:  EnvOpenAIKey,
			value:   "invalid-key",
			wantErr: true,
		},
		{
			name:    "empty value",
			envVar:  EnvAnthropicKey,
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialFormat(tt.envVar, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/etc/config.yaml", true},
		{"/app/settings.json", true},
		{"credentials.toml", true},
		{"secrets.conf", true},
		{"main.go", false},
		{"README.md", false},
		{"/tmp/data.txt", false},
	}

	for _, tt := range tests {
		result := isConfigFile(tt.path)
		if result != tt.expected {
			t.Errorf("isConfigFile(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}
