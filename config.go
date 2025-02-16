package main

import (
	"os"
)

type Config struct {
	Port          string
	AdminPassword string
	Title         string
	InfoTextDE    string
	InfoTextEN    string
	DayZero       string // Format: "2024-03-20"
	DebugMode     bool   // Enables debug features
	Version       string // Schipla version
}

func LoadConfig() Config {
	config := Config{
		Port:          getEnvOrDefault("PORT", "8080"),
		AdminPassword: getEnvOrDefault("ADMIN_PASSWORD", "admin123"),
		Title:         getEnvOrDefault("SITE_TITLE", "25s3"),
		DayZero:       getEnvOrDefault("DAY_ZERO", ""),
		DebugMode:     getEnvOrDefault("DEBUG_MODE", "") == "true",
		Version:       "0.1.0", // Hardcoded for now, could be set via build flag later
	}

	// Read German info text
	if content, err := os.ReadFile("info_de.md"); err == nil {
		config.InfoTextDE = string(content)
	}

	// Read English info text
	if content, err := os.ReadFile("info_en.md"); err == nil {
		config.InfoTextEN = string(content)
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
