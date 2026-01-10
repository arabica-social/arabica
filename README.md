# Arabica

Coffee brew tracking application using AT Protocol for decentralized storage.

## Tech Stack

- **Backend:** Go with stdlib HTTP router
- **Storage:** AT Protocol Personal Data Servers
- **Local DB:** BoltDB for OAuth sessions and feed registry
- **Templates:** html/template
- **Frontend:** HTMX + Alpine.js + Tailwind CSS

## Quick Start

```bash
# Using Nix
nix run

# Or with Go
go run cmd/server/main.go
```

Access at http://localhost:18910

## Configuration

Environment variables:

- `PORT` - Server port (default: 18910)
- `ARABICA_DB_PATH` - BoltDB path (default: ~/.local/share/arabica/arabica.db)
- `OAUTH_CLIENT_ID` - OAuth client ID (optional, uses localhost mode if not set)
- `OAUTH_REDIRECT_URI` - OAuth redirect URI (optional)
- `SECURE_COOKIES` - Set to true for HTTPS (default: false)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: console, json (default: console)

## Features

- Track coffee brews with detailed parameters
- Store data in your AT Protocol Personal Data Server
- Community feed of recent brews from registered users
- Manage beans, roasters, grinders, and brewers
- Export brew data as JSON
- Mobile-friendly PWA design

## Architecture

Data is stored in AT Protocol records on users' Personal Data Servers. The application uses OAuth to authenticate with the PDS and performs all CRUD operations via the AT Protocol API.

Local BoltDB stores:
- OAuth session data
- Feed registry (list of DIDs for community feed)

See docs/ for detailed documentation.

## Development

```bash
# Enter development environment
nix develop

# Run server
go run cmd/server/main.go

# Run tests
go test ./...

# Build
go build -o arabica cmd/server/main.go
```

## Deployment

See docs/nix-install.md for NixOS deployment instructions.

## License

MIT
