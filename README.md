<p align="center">
  <img src="internal/web/static/img/favicon.svg" alt="Shifts Logo" style="width: 120px;" />
</p>

# Shifts - Collaborative Shift Planning

The Shifts shift planning web application enables the collaborative coordination of shift cover for an event. Organisers simply need to share a URL with participants. Participants can then easily sign up using a nickname of their choice. Organisers always have a clear overview of how many shifts have already been filled. The application also offers exports in various formats as well as printable PDFs.

The application was developed out of frustration that existing tools were either too complicated for participants to use (login, email, etc.) or simply could not handle more complex events with numerous shifts and shift types.

> [!WARNING]
> **LLM Disclaimer.** This tool is the result of an experiment designed to investigate the potential applications of Vibe Coding. The architecture was specified by humans. Nevertheless, it is likely that this code will contain the usual problems and errors associated with LLM coding agents. Use is therefore at your own risk, and you must carry out your own risk assessment. Apart from a freely chosen nickname, the tool does not store any user data.


## Showcase

<p align="center">
  <img src="opt/showcase-01.png" alt="List View" width="45%" style="display: inline-block; vertical-align: top; margin-right: 2%;">
  <img src="opt/showcase-02.png" alt="Table View" width="45%" style="display: inline-block; vertical-align: top;">
</p>
<p align="center">
  <span style="display: inline-block; width: 45%; text-align: center; margin-right: 2%;">List view: Optimized for small screens with clear overview and filter options.</span>
  <span style="display: inline-block; width: 45%; text-align: center;">Table/timetable view: Convenient overview for events with many shifts and types.</span>
</p>

<p align="center">
  <img src="https://raw.githubusercontent.com/72nd/shifts/main/docs/screenshots/admin_panel.png" alt="Admin Panel" width="45%" style="display: inline-block; vertical-align: top; margin-right: 2%;">
  <img src="https://raw.githubusercontent.com/72nd/shifts/main/docs/screenshots/statistics_view.png" alt="Statistics View" width="45%" style="display: inline-block; vertical-align: top;">
</p>
<p align="center">
  <span style="display: inline-block; width: 45%; text-align: center; margin-right: 2%;">Admin panel: Manage shift types, lock shifts, and access all exports.</span>
  <span style="display: inline-block; width: 45%; text-align: center;">Statistics view: Get quick daily overviews and helper rankings.</span>
</p>

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
- A REST API and its documentation are available.
- Light and dark modes are supported.




## Requirements

- Go 1.16 or higher
- Modern web browser (Chrome, Firefox, Safari, Edge)
- Sqlite3

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/72nd/shifts.git
   cd shifts
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build and run the application:
   ```bash
   go run main.go
   ```

4. Access the application at `http://localhost:8080` (or your configured port)


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

Example configuration:
```toml
# Server configuration
host = "localhost"
port = 8080

# Database configuration
db_path = "data/shifts.db"

# Logging configuration
log_level = "info"
```

## API Documentation

### Requirements

Install the swag CLI tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Generating Documentation

To generate the Swagger documentation, run from the project root:

```bash
swag init -g cmd/shifts/main.go -o docs --parseDependency --parseInternal
```

If you encounter errors about not finding types, ensure:
1. All handler files that reference `db.*` types import the `db` package
2. The command is run from the project root directory
3. All dependencies are installed: `go mod download`

This will generate:
- `swagger.json` - OpenAPI specification in JSON format
- `swagger.yaml` - OpenAPI specification in YAML format
- `docs.go` - Go code for serving the documentation

### Accessing Documentation

After generating the documentation and starting the server, you can access:

- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/swagger/doc.json`

