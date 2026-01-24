# Arabica

Coffee brew tracking application build on ATProto

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

## Docker

```bash
# Build and run with Docker Compose
docker compose up -d

# Or build and run manually
docker build -t arabica .
docker run -p 18910:18910 -v arabica-data:/data arabica
```

For production deployments, configure environment variables in `docker-compose.yml`:

```yaml
environment:
  - SERVER_PUBLIC_URL=https://arabica.example.com
  - SECURE_COOKIES=true
```

## Configuration

### Command-Line Flags

- `--firehose` - Enable real-time feed via AT Protocol Jetstream (default: false)
- `--known-dids <file>` - Path to file with DIDs to backfill on startup (one per line)

### Environment Variables

- `PORT` - Server port (default: 18910)
- `SERVER_PUBLIC_URL` - Public URL for reverse proxy deployments (e.g., https://arabica.example.com)
- `ARABICA_DB_PATH` - BoltDB path (default: ~/.local/share/arabica/arabica.db)
- `ARABICA_FEED_INDEX_PATH` - Firehose index BoltDB path (default: ~/.local/share/arabica/feed-index.db)
- `ARABICA_PROFILE_CACHE_TTL` - Profile cache duration (default: 1h)
- `OAUTH_CLIENT_ID` - OAuth client ID (optional, uses localhost mode if not set)
- `OAUTH_REDIRECT_URI` - OAuth redirect URI (optional)
- `SECURE_COOKIES` - Set to true for HTTPS (default: false)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: console, json (default: console)

## Features

- Track coffee brews with detailed parameters
- Store data in your AT Protocol Personal Data Server
- Community feed of recent brews from registered users (polling or real-time firehose)
- Manage beans, roasters, grinders, and brewers
- Export brew data as JSON
- Mobile-friendly PWA design

### Firehose Mode

Enable real-time feed updates via AT Protocol's Jetstream:

```bash
# Basic firehose mode
go run cmd/server/main.go --firehose

# With known DIDs for backfill
go run cmd/server/main.go --firehose --known-dids known-dids.txt
```

**Known DIDs file format:**
```
# Comments start with #
did:plc:abc123xyz
did:plc:def456uvw
```

The firehose automatically indexes **all** Arabica records across the AT Protocol network. The `--known-dids` flag allows you to backfill historical records from specific users on startup (useful for development/testing).

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
