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
- `SERVER_PUBLIC_URL` - Public URL for reverse proxy deployments (e.g., https://arabica.example.com)
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

### Reverse Proxy Setup

When deploying behind a reverse proxy (nginx, Caddy, Cloudflare Tunnel, etc.), set the `SERVER_PUBLIC_URL` environment variable to your public-facing URL:

```bash
# Example with nginx reverse proxy
SERVER_PUBLIC_URL=https://arabica.example.com
SECURE_COOKIES=true
PORT=18910

# The server listens on localhost:18910
# But OAuth callbacks use https://arabica.example.com/oauth/callback
```

The `SERVER_PUBLIC_URL` is used for OAuth client metadata and callback URLs, ensuring the AT Protocol OAuth flow works correctly when the server is accessed via a different URL than it's running on.

### NixOS Deployment

See docs/nix-install.md for NixOS deployment instructions.

## License

MIT
