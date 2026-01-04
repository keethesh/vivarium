// Package config handles Vivarium configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds the application configuration.
type Config struct {
	// Authorized must be set to true to use attack features.
	// This confirms the user understands the tool is for authorized testing only.
	Authorized bool `toml:"authorized"`

	// Verbose enables detailed output.
	Verbose bool `toml:"verbose"`
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "vivarium.toml"
	}
	return filepath.Join(home, ".vivarium", "config.toml")
}

// Load loads configuration from a file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save saves configuration to a file.
func Save(cfg *Config, path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	// Add header comment
	content := `# Vivarium Configuration
# https://github.com/keethesh/vivarium

# Set to true to confirm you have authorization to use this tool.
# This tool is for educational purposes and authorized testing only.
` + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Exists checks if a config file exists.
func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigPath()
	}
	_, err := os.Stat(path)
	return err == nil
}

// CreateDefault creates a default config file with authorized=false.
func CreateDefault(path string) error {
	cfg := &Config{
		Authorized: false,
		Verbose:    false,
	}
	return Save(cfg, path)
}
