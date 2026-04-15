<p align="center">
  <img src="internal/web/static/img/favicon.svg" alt="Shifts Logo" style="width: 120px;" />
</p>

# Shifts - Collaborative Shift Planning

The Shifts shift planning web application enables the collaborative coordination of shift cover for an event. Organisers simply need to share a URL with participants. Participants can then easily sign up using a nickname of their choice. Organisers always have a clear overview of how many shifts have already been filled. The application also offers exports in various formats as well as printable PDFs.

The application was developed out of frustration that existing tools were either too complicated for participants to use (login, email, etc.) or simply could not handle more complex events with numerous shifts and shift types.

> [!WARNING]
> **LLM Disclaimer.** This tool is the result of an experiment designed to investigate the potential applications of Vibe Coding. The architecture was specified by humans. Nevertheless, it is likely that this code will contain the usual problems and errors associated with LLM coding agents. Use is therefore at your own risk, and you must carry out your own risk assessment. Apart from a freely chosen nickname, the tool does not store any user data.


## Showcase

A full list of features can be found [below](#features).

<table>
  <tr>
    <td align="center" width="50%">
      <img src="opt/showcase-01.png" alt="Create a new shift plan." width="100%" />
      <br /><em>Create a new shift plan.</em>
    </td>
    <td align="center" width="50%">
      <img src="opt/showcase-02.png" alt="List view" width="100%" />
      <br /><em>List view</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="50%">
      <img src="opt/showcase-03.png" alt="Table view." width="100%" />
      <br /><em>Table view.</em>
    </td>
    <td align="center" width="50%">
      <img src="opt/showcase-04.png" alt="Edit shifts and shift types." width="100%" />
      <br /><em>Edit shifts and shift types.</em>
    </td>
  </tr>
</table>


## Features

- Written in Go, with SQLite used as the database backend. Deployment is straightforward, requiring just a single binary.
- Participants simply need the link to the shift schedule and can then sign up using a nickname of their choice. No login or email address is required. The aim was to minimise the hassle for event participants as much as possible.
- No login is required to create shift schedules either. A single Shifts instance can manage any number of shift schedules. 
- The administration mode is secured with an admin password, which is displayed once after the schedule has been created.
- Each shift schedule includes a description (Markdown is supported) as well as shift types and shifts. The number of people required is defined for each shift. An unlimited number of people per shift is supported.
- Filter functions
- Two different views of the shifts: one as a list and one as a table/timetable. The list view is particularly suitable for small screens (smartphones), whilst the table view offers a convenient overview of many shifts.
- The app aims to make applying multiple layers as easy as possible.
- Organisers have access to comprehensive statistical functions.
- Shift schedules can be exported as CSV or PDF files. For example, this allows for a transition to a paper-based shift schedule during the event via the poster export feature.
- The collaborative planning mode supports scenarios where a significantly larger number of people are interested in shifts and/or where organisers wish to retain control over the final shift allocation themselves. In this mode, any number of participants can ‘apply’ for a shift. The organisers then select the participants and lock the shift. After that, it can no longer be modified by non-admins.
- Bilingualism: Shift schedules can be created in either a single language or in two languages (currently German and English).
- A REST API is available, with interactive Swagger/OpenAPI documentation and support for batch operations on shifts (creation, partial updates, and deletion). The documentation is accessible at `/swagger/index.html` once the server is running.
- Light and dark modes are supported.
- Built in documentation system for the help pages.
- An optional **metrics** endpoint exposes both performance and business metrics in a Prometheus-compatible format, served on a dedicated port (configurable via `metrics_port` in `config.toml`).


## Installation

### Requirements

- [Go 1.24](https://go.dev/dl/) or higher
- SQLite3 development headers (required because the Go SQLite driver uses CGo)
- The Swag CLI tool is also required to generate the API documentation. When building via the Makefile it is installed automatically; for a manual build see Option B below.

### Option A — Using the Makefile (recommended)

1. Install the SQLite development headers:
   - **Debian / Ubuntu:** `sudo apt install libsqlite3-dev`
   - **Fedora / RHEL:** `sudo dnf install sqlite-devel`
   - **Arch Linux:** `sudo pacman -S sqlite`
   - **macOS (Homebrew):** `brew install sqlite`
   - **Windows:** headers are bundled with the driver; no separate installation is needed.

2. Clone the repository:
   ```bash
   git clone https://github.com/72nd/shifts.git
   cd shifts
   ```

3. Create and adjust the configuration file (see [Configuration](#configuration) for all available options):
   ```bash
   cp config.template.toml config.toml
   ```

4. Build the application — Swag is installed automatically if missing:
   ```bash
   make install
   ```

5. Run the application:
   ```bash
   ./shifts
   ```

Run `make help` to see all available targets.

### Option B — Manual

1. Install the SQLite development headers (see platform list in Option A above).

2. Install the Swag CLI tool:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

3. Clone the repository:
   ```bash
   git clone https://github.com/72nd/shifts.git
   cd shifts
   ```

4. Install Go dependencies:
   ```bash
   go mod download
   ```

5. Generate the Swagger API documentation:
   ```bash
   swag init -g cmd/shifts/main.go -o docs --parseDependency --parseInternal
   ```

6. Create and adjust the configuration file (see [Configuration](#configuration) for all available options):
   ```bash
   cp config.template.toml config.toml
   ```

7. Build the binary:
   ```bash
   CGO_ENABLED=1 go build -o shifts cmd/shifts/main.go
   ```

8. Run the application:
   ```bash
   ./shifts
   ```


## Updating

To update an existing installation to the latest release, run:

```bash
make all
```

This fetches the latest release tag, regenerates the Swagger docs, and recompiles the binary in one step. To update to the latest development branch instead:

```bash
make update MODE=dev && make install
```


## Configuration

The application uses a TOML configuration file for its settings. By default, it looks for a `config.toml` file in the current working directory. You can specify a different configuration file location using the `--config` flag.

### Configuration File

Copy the template configuration file to create your own:
```bash
cp config.template.toml config.toml
```

Available configuration options:

- `host`: Server host address (default: "localhost")
- `port`: Server port (default: 8080)
- `db_path`: Path to the SQLite database file (default: "data/shifts.db")
- `log_level`: Logging level (default: "info")
- `metrics_enabled`: Enable Prometheus metrics endpoint (default: false)
- `metrics_port`: Port for the Prometheus metrics endpoint (default: 9090)
- `default_view`: Default view for shift plans when the visitor has no `?view=` or saved preference (`"timetable"` or `"list"`) (default: "timetable")


Example configuration:
```toml
# Server configuration
host = "localhost"
port = 8080

# Database configuration
db_path = "data/shifts.db"

# Logging configuration
log_level = "info"

# Metrics configuration
metrics_enabled = true
metrics_port = 9090

# Default view for shift plans when the visitor has no ?view= or saved preference ("timetable" or "list")
default_view = "timetable"
```

## API Documentation

The Swagger documentation is generated automatically during every build (via `make install` or `make docs`). To regenerate it manually, run from the project root:

```bash
swag init -g cmd/shifts/main.go -o docs --parseDependency --parseInternal
```

This produces `docs/swagger.json`, `docs/swagger.yaml`, and `docs/docs.go`. Once the server is running, the documentation is available at:

- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/swagger/doc.json`


## Metrics

When `metrics_enabled = true` is set in `config.toml`, the application exposes a Prometheus-compatible `/metrics` endpoint on a dedicated port (default: 9090). Four groups of metrics are collected:

- **HTTP**: request count, latency histograms, and current in-flight requests — labelled by method, path, and status code.
- **Database**: query count and duration per operation and table; errors categorised by type (lock contention, constraint violations, schema errors).
- **Business**: counters for shift plans, shifts, and participant operations (joins, removals, and failures with failure reason).
- **Exports & Authentication**: generation count, duration, and errors per export format; admin login attempt count and duration.

The endpoint is designed to be scraped by a Prometheus instance and is fully compatible with standard dashboarding tools such as Grafana.
