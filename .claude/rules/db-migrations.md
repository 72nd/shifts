---
paths:
  - "internal/db/**"
---

# Database schema migrations

The schema is defined **only** by the numbered files in `internal/db/migrations/sqlite/`. There are no inline `CREATE TABLE` strings in Go code. The runner is `internal/db/migrate.go`; it runs from `db.New` at startup.

## Changing the schema

1. Add a new file `internal/db/migrations/sqlite/NNNN_short_name.sql` (next free four-digit number, snake_case name).
2. **Never edit or delete an existing migration file.** Installed databases have already applied it.
3. Plain SQL, multiple statements allowed. The runner wraps each file in one transaction and records it in `schema_migrations`.
4. Update the Go code (structs in `internal/db`, queries) in the same change.
5. Mention the schema change in `CHANGELOG.md` (type `Feature` or `Development`) and say that an automatic backup is created.

## How it behaves

- Forward only, no down migrations. Before the first pending migration on an existing database the runner writes `<db_path>.<timestamp>.pre-migration.bak` (`VACUUM INTO`). Fresh databases get no backup.
- A database that has tables but no `schema_migrations` rows is treated as pre-migration (v0.5.x and older): `legacyColumns` in `migrate.go` are added if missing and version 1 is marked applied without running `0001_init.sql`. Do not extend `legacyColumns`; new columns belong in a new migration.
- A database with a higher version than the binary knows aborts startup with `ErrSchemaTooNew`.

## SQLite pitfalls

- `ALTER TABLE ... ADD COLUMN` with `NOT NULL` needs a `DEFAULT`.
- Dropping or changing a column needs the table rebuild pattern (create new table, copy, drop old, rename). `PRAGMA foreign_keys` cannot be switched inside a transaction, so such a migration needs a Go step (see below) instead of a plain SQL file.
- Shift times, plan dates and their formats are defined in `.claude/rules/datetime.md`. Timestamps like `created_at` are RFC3339 UTC. Do not write new time formats by hand.

## Go steps

When a migration needs Go code (for example time zone conversion or a table rebuild), register a function in the `goSteps` map in `migrate.go`, keyed by the migration version:

```go
var goSteps = map[int]func(tx *sql.Tx, logger *slog.Logger) error{
    2: migrateLocalTimes,
}
```

- The runner executes the file's SQL first (skipped if the file has no statements besides comments), then the Go step, both in the same transaction. A failing step rolls back the whole migration.
- The step is called for its version even if the file contains only a comment. Such a file is the pattern for a Go-only migration: `0002_local_times.sql` holds just a comment that points to `migrate_local_times.go`.
- Write the step defensively: check whether a column exists before adding it, and skip rows already in the new format, so it can be re-run safely.
- Log the number of rewritten rows and fail with the table, row ID and value on unreadable data. Do not skip silently.

## PostgreSQL later

A second dialect would live in `internal/db/migrations/postgres/` with the same numbering. Query portability (`?` placeholders, `LastInsertId`, boolean comparisons, error text matching in `metrics.go`) is not solved yet.
