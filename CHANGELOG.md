# Changelog Shifts

## v0.5.0 – 39c3 Release

This release was prepared for 39c3 and now contains (hopefully) all features necessary for Congress.

- **Feature: (Open) API Documentation** The API is now documented and can be accessed via `YOUR_DOMAIN/swagger/index.html` and `YOUR_DOMAIN/swagger/doc.json`.
- **Feature: API Batch Operations** The API now supports batch operations for the creation, (partial) updating and deletion of shifts.
- **Feature: Export Shifts as CSV** Shifts can now be exported to CSV files.
- **Feature: Metrics Endpoint** The application now exposes a Prometheus metrics endpoint at `YOUR_DOMAIN:METRICS_PORT/metrics`. This provides insight into both the performance and business metrics of the application.
- **Feature: Validation of Shift Date and Time** Shifts can now only be created within the start and end date of the shift plan.
- **Enhancement: Transactional DB Operations** All database operations are now wrapped in a transaction. This should make the database layer more robust.
- **Enhancement: Table View** Shifts with descriptions that overflow the cell height are now handled much better. The description is always rendered as markdown and the expanding/collapsing is now much more stable.
- **Enhancement: Shift Type Modal Now Shows ID** The settings modal of a shift type now shows the database ID of the shift type. This is useful for API operations.
- **Enhancement: Table View is Now Default View for New Users** The table view is now the default view for new users.
- **Development: Load Testing with Locust** The repository now contains a load testing script using Locust.
- **Development: Makefile for Deployment** A makefile provides a convenient way to deploy and update the application from a local repository for systems with Go installed.
- **Refactor: Data Model** Previously, the data structure definitions were scattered throughout the codebase. Now, (almost) all data structure definitions have been moved to the `internal/model` package. This greatly enhances the API documentation. No models from the database package are any longer exposed to the API.
- **Refactor: DB Error Handling** The handling of errors in the database layer has been refactored to use a more consistent and reusable approach, using typed sentinel errors instead of string-based error matching.
- **Refactor: Validation** Previously, the validation logic was scattered throughout the codebase. Now it is centralised in the `internal/validation` package.
- **Refactor: HTTP Handler** Code repetition within the HTTP handlers has been substantially reduced by introducing helper methods for common functionality.
- **Fix: AlpineJS Template Directive** Bug in the frontend code.
- **Fix: Database Deadlock** A deadlock in the database layer when updating the settings for a shift plan has been fixed.


## v0.5.1 – Quality of life features

- **Feature: Overbooked shifts filter** The shift view filters now include a "overbooked" filter. This filter shows shifts that have more participants than the required number of participants.
- **Feature: Collision detection** The shift view now shows a warning icon if a participant has joined two shifts that overlap in time.


## v0.5.2 – Bug fixes

- **Fix: Text overflow for shifts with a lot of participants** The text overflow for shifts with a lot of participants has been fixed.
- **Fix: Z-Index** Th z-index of the filter and the mobile admin menu has been fixed. They do not obscure the header of the table view anymore.


## v0.5.3 – Statistics view enhancements

- **Feature: Daily participant statistics** The statistics view now includes a new section showing participant shifts per day, grouped by shift type with counts. Includes participant name filtering for quick overview.
- **Enhancement: Collapsible helper rankings** The helper rankings table in the statistics view is now collapsible and collapsed by default, showing only the top 5 entries when collapsed.


## v0.5.4 – Daily statistics improvements

- **Enhancement: Collapsible days in daily participant statistics** Each day section in the daily participant statistics view can now be collapsed or expanded individually, making it easier to focus on specific days.
- **Enhancement: Sortable columns in daily participant statistics** All columns in the daily participant statistics tables are now sortable (name, shifts, shift types, total hours), with visual indicators showing the current sort direction.


## v0.5.5 – Bug fixes

- **Fix: Collision detection** Collision detection now only checks shifts on the same day, preventing false positives for shifts that span across day boundaries.


## v0.5.6 – Timetable performance improvements

- **Feature: Configurable default view** The default view for shift plans can now be configured in the `config.toml` file.
- **Enhancement: Default view preference reset** When the configured default view changes, previously saved per-plan view preferences in the browser are cleared so the new default applies.
- **Enhancement: Timetable view performance** Large shift plans load faster: background grid lines are drawn with CSS instead of hundreds of DOM nodes, overflow detection runs in one batched pass, list view uses pre-grouped shifts by date, and day sections use `content-visibility` so off-screen content is skipped during layout.


## v0.5.7 – Documentation and installation improvements

- **Enhancement: Landing page rewritten for clarity** The start page was simplified and de-cluttered.
- **Enhancement: README rewritten** The README has been completely revised to be clearer and more organized.
- **Enhancement: Installation section added to README** Pre-built Linux binaries (amd64/arm64) are linked from GitHub Releases; all other platforms require a manual build due to CGo (SQLite).
- Enhancement: Older changelog entries were improved for better clarity and understanding.
- **Development: `make dist` target** The Makefile now includes a `dist` target that cross-compiles the application for Linux and Windows on both amd64 and arm64. Binaries are written to a `dist/` folder with names indicating the target platform and architecture (e.g. `shifts-linux-arm64`). `make clean` also removes the `dist/` folder.


## v0.6.0

- **Feature: Automatic database migrations** The database schema is now versioned and upgraded automatically on startup, so existing databases keep working across updates. Before the first migration of an existing database, a backup is written next to it (`<db_path>.<timestamp>.pre-migration.bak`). Databases from before v0.6.0 are recognised and brought up to date without data loss. A database migrated by a newer version is not opened by an older binary.
- **Development: Migration files** The schema is defined by numbered SQL files in `internal/db/migrations/sqlite/` instead of inline `CREATE TABLE` statements. Rules for agents and contributors are in `.claude/rules/db-migrations.md`.
- **Feature: Time zone per shift plan** Each shift plan has its own IANA time zone. It is chosen in the form for creating a plan (preselected with the `timezone` option) and cannot be changed afterwards.
- **Feature: `timezone` option in `config.toml`** Default `Europe/Berlin`. It preselects the time zone for new plans and is the zone used to convert existing plans during migration. An invalid zone stops the server at startup.
- **Fix: Times independent of the browser time zone** Shift times are shown as local event time on every device. Before, browsers in another time zone showed times shifted by the offset, for example Firefox with `privacy.resistFingerprinting`, which reports UTC.
- **Fix: PDF and text exports show local event time** PDF exports and the text export of open shifts showed UTC times, which were 1 to 2 hours early in Germany. The day boundary is now applied to local times in exports and statistics, so day filters and day groups match.
- **Fix: Shifts on the last day of a plan** Shifts on the last day of a plan were rejected. Plan dates are now calendar days. A shift is valid if its logical day (respecting the day boundary) lies within the plan period. A shift may end after the plan's end date, for example a night shift.
- **Enhancement: API date and time format** `start_time` is a local time without offset (`2024-12-20T09:00:00`). Plan dates `start_date` and `end_date` are `YYYY-MM-DD`. Plans return `time_zone`. Requests with an offset or `Z` are rejected with HTTP 400. Browser tabs open during the update must be reloaded.
- **Development: Migration 0002 converts stored times** Migration `0002_local_times` adds `shift_plans.time_zone` and converts existing shift times and plan dates to local event time, using the configured `timezone` for all existing plans. Before the migration runs, a backup is written as `<db_path>.<timestamp>.pre-migration.bak`. If the event took place in another time zone, set `timezone` before the first start.
