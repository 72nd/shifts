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
- Dropping or changing a column needs the table rebuild pattern (create new table, copy, drop old, rename). `PRAGMA foreign_keys` cannot be switched inside a transaction, so such a migration needs a Go step in `migrate.go` instead of a plain SQL file. Add that mechanism only when first needed.
- Times are stored as RFC3339 `TEXT`.

## PostgreSQL later

A second dialect would live in `internal/db/migrations/postgres/` with the same numbering. Query portability (`?` placeholders, `LastInsertId`, boolean comparisons, error text matching in `metrics.go`) is not solved yet.
