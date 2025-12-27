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

- [ ] Text overflow for shifts with a lot of participants
- [ ] Z-Index Filter Shift-Types
- [ ] Z-Index Mobile Admin Menu
- [ ] Dark mode Toggle hört auf nur auf Icon


## Feature Request:w

- Random Kickout
- Experience per Participant