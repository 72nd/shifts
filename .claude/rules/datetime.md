---
paths:
  - "internal/**"
---

# Dates and times: floating local event time

Shift times are the wall-clock time at the event ("Saturday 09:00"). They are stored, transferred and shown **without offset and without conversion**. Nothing in the app converts them to another zone, so the result does not depend on the browser, the server or the reader's time zone.

## Formats

| Value | Storage (SQLite TEXT), JSON | Example |
|-------|-----------------------------|---------|
| Shift start | `2006-01-02T15:04:05` | `2024-12-20T09:00:00` |
| Plan start/end, day keys, statistics days | `2006-01-02` | `2024-12-20` |
| Day boundary | `HH:MM` | `04:00` |
| Real instants (`created_at`, `exported_at`, `applied_at`) | RFC3339 in UTC | `2024-12-01T10:00:00Z` |

- Input with an offset or `Z` is rejected with HTTP 400. There is no tolerance for old clients.
- Each plan has its own IANA zone in `shift_plans.time_zone`. It is chosen when the plan is created and cannot be changed afterwards. It is only used for "now" (past, current, future), never for converting shift times.
- The `timezone` option in `config.toml` (default `Europe/Berlin`) is only the preselection for new plans and the zone used when migrating existing plans (migration `0002`).
- Real instants are stored in UTC. Show them in the plan's zone (`time_zone`) only where a user reads them, for example `exported_at` in a PDF.

## Go (`internal/model/datetime.go`)

- Types: `model.DateTime` (shift start, wall clock, location always UTC) and `model.Date` (calendar day, midnight UTC). Never call `.In()`, `.Local()` or compare them with `time.Now()`.
- Creation and parsing: `NewDateTime`, `NewDate`, `NowIn`, `ParseDateTime`, `ParseDate`, `ParseBoundary`.
- Methods: `String`, `AddMinutes`, `AddDays`, `At`, `LogicalDay`.
- Default zone: `SetDefaultLocation`, `DefaultLocation`.
- JSON, SQL (`Value`, `Scan`) and swag: use `swaggertype:"string"` with an example in the layout of the type.
- `LogicalDay` is the only implementation of the day boundary. Exports, statistics and validation call it.
- Legacy conversion (RFC3339 with offset, the UTC+12h rule for plan dates) lives only in `internal/db/legacy_time.go` and the migration step, never in `model`.

## JavaScript (`internal/web/static/js/datetime.js`)

A floating value is a `Date` whose **UTC fields** hold the wall clock. Format with `Intl.DateTimeFormat` and `timeZone: 'UTC'`.

- Parse and serialise: `parseDateTime`, `parseDate`, `serializeDateTime`, `serializeDate`, `toInputValue`, `fromInputValue`.
- Arithmetic and days: `addMinutes`, `addDays`, `minutesOfDay`, `dateKey`, `logicalDay`, `daysBetween`.
- Display: `formatTime`, `formatDate`.
- Now: `nowInZone(timeZone)` with the plan's `time_zone`.

Outside `datetime.js` do not use:

- `new Date(apiString)`
- local getters and setters (`getHours`, `getDate`, `setHours`, `setDate`, ...)
- `toISOString`
- `getTimezoneOffset`
- `toLocale*String` without an explicit `timeZone`

The only exceptions are error timestamps in the modals (`modal.js`, `join-modal.js`, `admin-modal.js`), which are not event times.

## Plan dates and validation

- A shift is valid if its logical day (`LogicalDay` with the plan's day boundary) lies within `[start_date, end_date]`. A shift may end after the plan's end date.
- Shifts are sorted by comparing the offsetless strings (lexicographic order is correct for this format).
