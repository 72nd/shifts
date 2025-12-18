package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

	// Configure connection pool for SQLite
	// SQLite works best with a single writer connection to avoid lock contention
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0) // Connections don't expire

	// Set pragmas for better performance, safety, and concurrency handling
	if _, err := sqlDB.Exec(`
        PRAGMA foreign_keys = ON;
        PRAGMA journal_mode = WAL;
        PRAGMA synchronous = NORMAL;
        PRAGMA busy_timeout = 5000;
        PRAGMA cache_size = -64000;
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

// transactionImmediate executes a function within a transaction with IMMEDIATE mode.
// This acquires a write lock at the start of the transaction, preventing race conditions
// in check-then-act scenarios like capacity checks.
func (db *DB) transactionImmediate(fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return fmt.Errorf("failed to begin immediate transaction: %w", err)
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

// logError logs an error with context
func (db *DB) logError(operation string, err error, args ...any) {
	db.logger.Error("database error",
		"operation", operation,
		"error", err,
		"args", args,
	)
}
