# Arabica - Coffee Brew Tracker

A self-hosted web application for tracking your coffee brewing journey. Built with Go and SQLite.

## Features

- 📝 Quick entry of brew data (temperature, time, method, flexible grind size entry, etc.)
- ☕ Organize beans by origin and roaster with quick-select dropdowns
- 📱 Mobile-first PWA design for on-the-go tracking
- 📊 Rating system and tasting notes
- 📥 Export your data as JSON
- 🔄 CRUD operations for all brew entries
- 🗄️ SQLite database with abstraction layer for easy migration

## Tech Stack

- **Backend**: Go 1.22+ (using stdlib router)
- **Database**: SQLite (via modernc.org/sqlite - pure Go, no CGO)
- **Templates**: html/template (Go standard library)
- **Frontend**: HTMX + Alpine.js
- **CSS**: Tailwind CSS
- **PWA**: Service Worker for offline support

## Project Structure

```
arabica/
├── cmd/server/          # Application entry point
├── internal/
│   ├── database/        # Database interface & SQLite implementation
│   ├── models/          # Data models
│   ├── handlers/        # HTTP handlers
│   └── templates/       # HTML templates
├── web/static/          # Static assets (CSS, JS, PWA files)
└── migrations/          # Database migrations
```

## Getting Started

### Prerequisites

Use Nix for a reproducible development environment with all dependencies:

```bash
nix develop
```

### Running the Application

1. Enter the Nix development environment:
```bash
nix develop
```

2. Build and run the server:
```bash
go run ./cmd/server
```

The application will be available at `http://localhost:8080`

## Usage

### Adding a Brew

1. Navigate to "New Brew" from the home page
2. Select a bean (or add a new one with the "+ New" button)
   - When adding a new bean, provide a **Name** (required) like "Morning Blend" or "House Espresso"
   - Optionally add Origin, Roast Level, and Description
3. Select a roaster (or add a new one)
4. Fill in brewing details:
   - Method (Pour Over, French Press, etc.)
   - Temperature (°C)
   - Brew time (seconds)
   - Grind size (free text - enter numbers like "18" or "3.5" for grinder settings, or descriptions like "Medium" or "Fine")
   - Grinder (optional)
   - Tasting notes
   - Rating (1-10)
5. Click "Save Brew"

### Viewing Brews

Navigate to the "Brews" page to see all your entries in a table format with:
- Date
- Bean details
- Roaster
- Method and parameters
- Rating
- Actions (View, Delete)

### Exporting Data

Click "Export JSON" on the brews page to download all your data as JSON.

## Configuration

Environment variables:

- `DB_PATH`: Path to SQLite database (default: `./arabica.db`)
- `PORT`: Server port (default: `8080`)

## Database Abstraction

The application uses an interface-based approach for database operations, making it easy to swap SQLite for PostgreSQL or another database later. See `internal/database/store.go` for the interface definition.

## PWA Support

The application includes:
- Web App Manifest for "Add to Home Screen"
- Service Worker for offline caching
- Mobile-optimized UI with large touch targets

## Future Enhancements (Not in MVP)

- Statistics and analytics page
- CSV export
- Multi-user support (database already has user_id column)
- Search and filtering
- Photo uploads for beans/brews
- Brew recipes and sharing

## Development Notes

### Why These Choices?

- **Go**: Fast compilation, single binary deployment, excellent stdlib
- **modernc.org/sqlite**: Pure Go SQLite (no CGO), easy cross-compilation
- **html/template**: Built-in Go templates, no external dependencies
- **HTMX**: Progressive enhancement without heavy JS framework
- **Nix**: Reproducible development environment

### Database Schema

See `migrations/001_initial.sql` for the complete schema.

Key tables:
- `users`: Future multi-user support
- `beans`: Coffee bean information
- `roasters`: Roaster information
- `brews`: Individual brew records with all parameters

## License

MIT

## Contributing

This is a personal project, but suggestions and improvements are welcome!
