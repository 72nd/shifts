package config

import (
	"fmt"
	"os"
	"time"

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

	// Metrics configuration
	MetricsEnabled bool `toml:"metrics_enabled"`
	MetricsPort    int  `toml:"metrics_port"`

	// UI: default shift plan view when URL and localStorage do not specify one ("list" or "timetable")
	DefaultView string `toml:"default_view"`

	// Default IANA time zone for new plans and start value for migrating existing plans
	Timezone string `toml:"timezone"`

	// Live updates via Server-Sent Events; nil (key missing) means enabled
	RealtimeEnabled *bool `toml:"realtime_enabled"`
}

// RealtimeOn reports whether live updates are enabled (default true).
func (c Config) RealtimeOn() bool {
	return c.RealtimeEnabled == nil || *c.RealtimeEnabled
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() Config {
	return Config{
		Host:           "localhost",
		Port:           8080,
		DBPath:         "data/shifts.db",
		LogLevel:       "info",
		MetricsEnabled: false,
		MetricsPort:    9090,
		DefaultView:    "timetable",
		Timezone:       "Europe/Berlin",
	}
}

// NormalizeDefaultView sets DefaultView to "timetable" if empty and validates it is "list" or "timetable".
func (c *Config) NormalizeDefaultView() error {
	if c.DefaultView == "" {
		c.DefaultView = "timetable"
		return nil
	}
	if c.DefaultView != "list" && c.DefaultView != "timetable" {
		return fmt.Errorf("invalid default_view %q: must be %q or %q", c.DefaultView, "list", "timetable")
	}
	return nil
}

// NormalizeTimezone sets Timezone to "Europe/Berlin" if empty and loads the zone.
func (c *Config) NormalizeTimezone() (*time.Location, error) {
	if c.Timezone == "" {
		c.Timezone = "Europe/Berlin"
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", c.Timezone, err)
	}
	return loc, nil
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
