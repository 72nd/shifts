# Shifts - Collaborative Shift Planning

Shifts is a modern, web-based application for collaborative shift planning and management. It provides an intuitive interface for creating, organizing, and signing up for shifts, making it ideal for volunteer organizations, event planning, and team coordination.

![Shifts Logo](internal/web/static/favicon.svg)

## 🌟 Features

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

## 📋 Requirements

- Go 1.16 or higher
- Modern web browser (Chrome, Firefox, Safari, Edge)
- Sqlite3

## 🚀 Installation

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


## 🔧 Configuration

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

## 📱 Usage Guide

### Creating a New Shift Plan

1. Access the home page and click "Create New Plan"
2. Enter a title, description, and optional admin password
3. Configure the day boundary time (default: 00:00)
4. Click "Create Plan" to generate your new shift plan

### Managing Shift Types

1. Log in as admin by clicking the admin icon and entering your password
2. Click "Manage Shift Types" to add, edit, or delete shift types
3. For each type, you can set:
   - Name and description
   - Color for visual identification
   - Minimum and maximum participants
   - Whether the type should be visible to all users

### Adding Shifts

1. Log in as admin
2. Click "Add Shift" and select a shift type
3. Set the date, start time, and duration
4. Add a description if needed
5. Click "Save" to create the shift

### Signing Up for Shifts

1. Browse the available shifts on the main plan view
2. Click on a shift you want to join
3. Enter your name in the modal that appears
4. Click "Join" to sign up for the shift

### Filtering and Sorting

- Use the filter panel to show/hide specific shift types
- Shifts are automatically grouped by date
- Admin users can reorder shift types for consistent display

## 🔒 Security Considerations

- Admin functions are protected by a password
- The application does not collect personal data beyond names
- All data is stored locally on the server

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgements

- [Go](https://golang.org/) - The programming language used
- [Alpine.js](https://alpinejs.dev/) - JavaScript framework for interactivity
- [DaisyUI](https://daisyui.com/) - Tailwind CSS component library
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework 