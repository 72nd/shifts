# Changelog Shifts

## v0.5.0 – 39c3 Release

This release was prepared for 39c3 and now contains (hopefully) all features necessary for Congress.

- **Feature: (Open) API Documentation** The API is now documented and can be accessed via `your-domain.com/swagger/index.html` and `your-domain.com/swagger/doc.json`.
- **Feature: API Batch Operations** The API now supports batch operations for the creation, (partial) updating and deletion of shifts.
- **Feature: Export Shifts as CSV** Shifts can now be exported to CSV files.
- **Feature: Metrics Endpoint** The application now exposes a Prometheus metrics endpoint at `127.0.0.1:METRICS_PORT/metrics`. This provides insight into both the performance and business metrics of the application.
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



## Feature Request / Backlog

- [ ] Dark mode Toggle hört auf nur auf Icon
- Random Kickout
- Experience per Participant