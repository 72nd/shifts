package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/72nd/schipla/internal/config"
	"github.com/72nd/schipla/internal/db"
	"github.com/72nd/schipla/internal/web"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file (optional)")
	flag.Parse()

	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Update log level if specified in config
	if cfg.LogLevel != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
			logger.Warn("invalid log level in config, using default", "level", cfg.LogLevel)
		} else {
			logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			}))
		}
	}

	// Override log level if debug mode is enabled
	if cfg.Debug {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		logger.Info("debug mode enabled, setting log level to debug")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		logger.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	// Initialize database
	database, err := db.New(cfg.DBPath, logger)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize HTTP server
	server, err := web.New(database, logger)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	// Start server
	if err := server.Start(cfg.Host, cfg.Port); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
