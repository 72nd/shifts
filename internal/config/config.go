package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds all configuration values for the application
type Config struct {
	// Server configuration
	Host string `toml:"host"`
	Port int    `toml:"port"`

	// Database configuration
	DBPath string `toml:"db_path"`

	// Logging configuration
	LogLevel string `toml:"log_level"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     8080,
		DBPath:   "data/shifts.db",
		LogLevel: "info",
	}
}

// LoadConfig loads the configuration from a TOML file
func LoadConfig(path string) (Config, error) {
	var config Config

	// If no config file is specified, try to find it in the current directory
	if path == "" {
		path = "config.toml"
	}

	// Try to read the config file
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to a TOML file
func SaveConfig(path string, config Config) error {
	if err := toml.NewEncoder(os.Stdout).Encode(config); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := toml.NewEncoder(os.Stdout).Encode(config); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
