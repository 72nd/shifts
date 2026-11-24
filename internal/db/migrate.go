package db

import (
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Schema migrations: numbered SQL files in migrations/sqlite/, applied in
// ascending order at startup. Applied versions are recorded in the table
// schema_migrations. Migrations only go forward; the rollback path is the
// backup created before the first pending migration runs.
// See .claude/rules/db-migrations.md for how to add a migration.

//go:embed migrations/sqlite/*.sql
var migrationFiles embed.FS

const migrationDir = "migrations/sqlite"

type migration struct {
	version int
	name    string
	sql     string
}

// legacyColumn is a column that was added to the schema before the migration
// system existed. Databases created by older versions may lack it.
type legacyColumn struct {
	table, column, definition string
}

var legacyColumns = []legacyColumn{
	{"shift_plans", "is_multilingual", "BOOLEAN NOT NULL DEFAULT true"},
	{"shift_plans", "day_boundary_time", "TEXT NOT NULL DEFAULT '00:00'"},
	{"shift_plans", "collaborative_mode", "BOOLEAN NOT NULL DEFAULT false"},
	{"shifts", "is_locked", "BOOLEAN NOT NULL DEFAULT false"},
}

// loadMigrations reads and sorts the embedded migration files.
func loadMigrations() ([]migration, error) {
	entries, err := migrationFiles.ReadDir(migrationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations: %w", err)
	}

	var migrations []migration
	for _, e := range entries {
		base := strings.TrimSuffix(e.Name(), ".sql")
		prefix, name, ok := strings.Cut(base, "_")
		version, convErr := strconv.Atoi(prefix)
		if !ok || convErr != nil {
			return nil, fmt.Errorf("invalid migration file name %q (want NNNN_name.sql)", e.Name())
		}
		body, err := migrationFiles.ReadFile(path.Join(migrationDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %q: %w", e.Name(), err)
		}
		migrations = append(migrations, migration{version: version, name: name, sql: string(body)})
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	for i := 1; i < len(migrations); i++ {
		if migrations[i].version == migrations[i-1].version {
			return nil, fmt.Errorf("duplicate migration version %d", migrations[i].version)
		}
	}
	return migrations, nil
}

// migrate brings the database schema up to the latest version.
func (db *DB) migrate(dbPath string) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return fmt.Errorf("no migrations found")
	}
	latest := migrations[len(migrations)-1].version

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}

	applied, current, err := db.appliedVersions()
	if err != nil {
		return err
	}
	if current > latest {
		return fmt.Errorf("%w: database is at version %d, this binary knows up to %d", ErrSchemaTooNew, current, latest)
	}

	// A database with tables but without migration records predates the
	// migration system.
	legacy := len(applied) == 0
	if legacy {
		if legacy, err = db.tableExists("shift_plans"); err != nil {
			return err
		}
	}

	var pending []migration
	for _, m := range migrations {
		if !applied[m.version] {
			pending = append(pending, m)
		}
	}

	backedUp := false
	backup := func() error {
		if backedUp {
			return nil
		}
		backedUp = true
		return db.backup(dbPath)
	}

	if legacy {
		if err := backup(); err != nil {
			return err
		}
		if err := db.baselineLegacy(migrations[0]); err != nil {
			return fmt.Errorf("failed to baseline existing database: %w", err)
		}
		pending = pending[1:] // version 1 is covered by the baseline
	}

	for _, m := range pending {
		// Fresh databases (nothing applied yet) need no backup.
		if len(applied) > 0 || legacy {
			if err := backup(); err != nil {
				return err
			}
		}
		if err := db.applyMigration(m); err != nil {
			return fmt.Errorf("migration %04d_%s failed: %w", m.version, m.name, err)
		}
		applied[m.version] = true
		db.logger.Info("applied database migration", "version", m.version, "name", m.name)
	}

	return nil
}

// appliedVersions returns the set of applied versions and the highest one.
func (db *DB) appliedVersions() (map[int]bool, int, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int]bool{}
	highest := 0
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, 0, fmt.Errorf("failed to scan schema_migrations: %w", err)
		}
		applied[v] = true
		if v > highest {
			highest = v
		}
	}
	return applied, highest, rows.Err()
}

func (db *DB) tableExists(name string) (bool, error) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("failed to inspect sqlite_master: %w", err)
	}
	return n > 0, nil
}

func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query("SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// baselineLegacy adds columns missing in databases created by older versions
// and marks the initial migration as applied without running it.
func (db *DB) baselineLegacy(initial migration) error {
	return db.transaction(func(tx *sql.Tx) error {
		for _, c := range legacyColumns {
			exists, err := columnExists(tx, c.table, c.column)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.column, c.definition)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("%s: %w", stmt, err)
			}
			db.logger.Info("added missing column to existing database", "table", c.table, "column", c.column)
		}
		return recordMigration(tx, initial)
	})
}

func (db *DB) applyMigration(m migration) error {
	return db.transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(m.sql); err != nil {
			return err
		}
		return recordMigration(tx, m)
	})
}

func recordMigration(tx *sql.Tx, m migration) error {
	_, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		m.version, m.name, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// backup writes a consistent copy of the database next to the original.
// VACUUM INTO is safe with WAL mode.
func (db *DB) backup(dbPath string) error {
	target := fmt.Sprintf("%s.%s.pre-migration.bak", dbPath, time.Now().Format("20060102-150405"))
	if _, err := db.Exec("VACUUM INTO ?", target); err != nil {
		return fmt.Errorf("failed to back up database before migration: %w", err)
	}
	db.logger.Info("database backup created before migration", "file", target)
	return nil
}
