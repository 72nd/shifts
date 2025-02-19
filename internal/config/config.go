package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all configuration values for the application
type Config struct {
	// Server configuration
	Host string `json:"host"`
	Port int    `json:"port"`

	// Database configuration
	DBPath string `json:"db_path"`

	// Logging configuration
	LogLevel string `json:"log_level"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     8080,
		DBPath:   "data/schipla.db",
		LogLevel: "info",
	}
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()

	// If no config file is specified, return defaults
	if path == "" {
		return config, nil
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return config, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, create it with default values
			if err := SaveConfig(path, config); err != nil {
				return config, fmt.Errorf("failed to create default config: %w", err)
			}
			return config, nil
		}
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the config file
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to a JSON file
func SaveConfig(path string, config Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
