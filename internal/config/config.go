// Package config handles loading and validating nightshift configuration.
// Supports YAML config files and environment variable overrides.
package config

// Config holds all nightshift configuration.
type Config struct {
	// TODO: Add config fields (budget limits, provider settings, etc.)
}

// Load reads configuration from file and environment.
func Load() (*Config, error) {
	// TODO: Implement config loading
	return &Config{}, nil
}
