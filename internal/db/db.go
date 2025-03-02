package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// New creates a new database connection and ensures the schema is up to date
func New(dbPath string, logger *slog.Logger) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for better performance and safety
	if _, err := sqlDB.Exec(`
        PRAGMA foreign_keys = ON;
        PRAGMA journal_mode = WAL;
        PRAGMA synchronous = NORMAL;
    `); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to set database pragmas: %w", err)
	}

	db := &DB{
		DB:     sqlDB,
		logger: logger,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema initializes the database schema
func (db *DB) initSchema() error {
	// Combine all schemas in the correct order (respecting foreign key constraints)
	schemas := []string{
		shiftPlanSchema,   // No dependencies
		shiftTypeSchema,   // Depends on shift_plans
		shiftSchema,       // Depends on shift_types
		participantSchema, // Depends on shifts
	}

	// Execute all schemas in a transaction
	return db.transaction(func(tx *sql.Tx) error {
		for _, schema := range schemas {
			if _, err := tx.Exec(schema); err != nil {
				return fmt.Errorf("failed to execute schema: %w", err)
			}
		}
		return nil
	})
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// transaction executes a function within a transaction
func (db *DB) transaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// logError logs database errors with context
func (db *DB) logError(operation string, err error, args ...any) {
	// Bestimme den Log-Level basierend auf dem Fehlertyp
	level := slog.LevelError

	// Für bestimmte Fehlertypen können wir einen anderen Level verwenden
	if err == sql.ErrNoRows {
		// "No rows" ist oft ein erwarteter Zustand, daher nur Info
		level = slog.LevelInfo
	}

	// Erstelle ein strukturiertes Log-Attribut für die Argumente
	var argsAttr slog.Attr
	if len(args) > 0 {
		argsAttr = slog.Any("args", args)
	} else {
		argsAttr = slog.Any("args", nil)
	}

	// Erstelle ein strukturiertes Log mit Kontext
	db.logger.Log(context.Background(), level,
		"database operation",
		"operation", operation,
		"error", err.Error(),
		argsAttr,
		"timestamp", slog.TimeValue(time.Now()),
	)
}
