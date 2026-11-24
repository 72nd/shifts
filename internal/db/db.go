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
//
// There are two pools on the same file. The embedded *sql.DB is the write pool
// (one connection, every transaction starts BEGIN IMMEDIATE) and is also used for
// migrations and anything that runs inside a write flow. rdb is a read-only pool
// (WAL allows readers during a write) used by the pure read functions; each such
// function is a single statement, i.e. one consistent snapshot.
type DB struct {
	*sql.DB
	rdb    *sql.DB
	logger *slog.Logger
}

// New creates a new database connection and ensures the schema is up to date
func New(dbPath string, logger *slog.Logger) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite3", "file:"+dbPath+"?_txlock=immediate")
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

	// Bring the schema up to date (see migrate.go)
	if err := db.migrate(dbPath); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	// Open the read pool only now: the file exists, is in WAL mode and has its schema.
	rdb, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&_busy_timeout=5000&_query_only=1")
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to open read-only database: %w", err)
	}
	rdb.SetMaxOpenConns(4)
	rdb.SetMaxIdleConns(4)
	rdb.SetConnMaxLifetime(0)
	if err := rdb.Ping(); err != nil {
		rdb.Close()
		sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to read-only database: %w", err)
	}
	db.rdb = rdb

	return db, nil
}

// Close closes both connection pools
func (db *DB) Close() error {
	rerr := db.rdb.Close()
	if err := db.DB.Close(); err != nil {
		return err
	}
	return rerr
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

// transactionImmediate executes a function within a transaction. The write pool's DSN
// has _txlock=immediate, so the write lock is taken at BEGIN, preventing race conditions
// in check-then-act scenarios like capacity checks. (go-sqlite3 ignores the isolation
// level below; it is kept only as documentation of intent.)
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
