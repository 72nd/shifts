# Shifts - Collaborative Shift Planning

Shifts is a modern, web-based application for collaborative shift planning and management. It provides an intuitive interface for creating, organizing, and signing up for shifts, making it ideal for volunteer organizations, event planning, and team coordination.

![Shifts Logo](internal/web/static/favicon.svg)

## Features

- **Intuitive Shift Management**
  - Create, edit, and delete shifts with customizable details
  - Set shift duration, required participants, and descriptions
  - Organize shifts by type with color-coding for easy identification

- **Collaborative Participation**
  - Sign up for shifts with a simple click
  - View who else has signed up for each shift
  - Remove yourself from shifts when needed

- **Flexible Organization**
  - Group shifts by date with customizable day boundaries
  - Filter shifts by type to focus on relevant information
  - Sort shift types to prioritize important categories

- **User-Friendly Interface**
  - Responsive design works on desktop and mobile devices
  - Dark/light theme support for comfortable viewing
  - Multilingual support (currently English and German)

- **Administrative Controls**
  - Password-protected admin functions
  - Manage shift types and their properties
  - Configure plan settings including day boundaries

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

