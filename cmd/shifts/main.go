package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	_ "time/tzdata"

	"github.com/72nd/shifts/internal/config"
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/web"
)

// @title Shifts API
// @version 1.0
// @description API for collaborative shift planning and management. This API allows you to create shift plans, manage shift types, create and manage shifts, and handle participant sign-ups.

// @contact.name API Support
// @contact.url https://github.com/72nd/shifts

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey AdminToken
// @in header
// @name X-Admin-Token
// @description Admin authentication token. Required for admin-only endpoints. The token is the admin password set when creating a shift plan.

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
	if err := cfg.NormalizeDefaultView(); err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}

	loc, err := cfg.NormalizeTimezone()
	if err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}
	model.SetDefaultLocation(loc)

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

	logger.Info("default time zone", "timezone", loc.String())

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

	// Create server
	server, err := web.NewServer(database, logger, cfg.DefaultView, loc.String())
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Start metrics server on separate port if enabled
	if cfg.MetricsEnabled {
		go func() {
			if err := server.StartMetricsServer(cfg.MetricsPort); err != nil {
				logger.Error("metrics server error", "error", err)
			}
		}()
		logger.Info("metrics enabled", "port", cfg.MetricsPort)
	}

	// Start server
	if err := server.Start(cfg.Host, cfg.Port); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
