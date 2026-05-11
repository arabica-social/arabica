# Arabica

Coffee brew logging application built on ATProto

Development is on Tangled, and is mirrored to GitHub:

- [Tangled](https://tangled.org/arabica.social/arabica)
- [GitHub](https://github.com/arabica-social/arabica)

## Quick Start

```bash
# Using Nix
nix run

# Or with Go
go run ./cmd/arabica
```

Access at http://localhost:18910

## Configuration

### Command-Line Flags

- `--known-dids <file>` - Path to file with DIDs to backfill on startup (one per
  line)

### Environment Variables

- `PORT` - Server port (default: 18910)
- `SERVER_PUBLIC_URL` - Public URL for reverse proxy deployments (e.g.,
  https://arabica.example.com)
- `ARABICA_DB_PATH` - BoltDB path (default: ~/.local/share/arabica/arabica.db)
- `ARABICA_FEED_INDEX_PATH` - Firehose index BoltDB path (default:
  ~/.local/share/arabica/feed-index.db)
- `ARABICA_PROFILE_CACHE_TTL` - Profile cache duration (default: 1h)
- `OAUTH_CLIENT_ID` - OAuth client ID (optional, uses localhost mode if not set)
- `OAUTH_REDIRECT_URI` - OAuth redirect URI (optional)
- `SECURE_COOKIES` - Set to true for HTTPS (default: false)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: console, json (default: console)

## Development

With Nix:

```bash
# Enter development environment
nix develop

# Run server
go run ./cmd/arabica

# Run tests
go test ./...

# Build
go build -o arabica ./cmd/arabica
```

Without Nix, you'll need to have Go and Templ installed (just is optional but
recommended). CSS is bundled in-process at server startup — no external build
tool needed.

```sh
# Compile Templ
templ generate
# Run the appview
go run ./cmd/arabica

# with just
just run
```

---

## Deployment

### Reverse Proxy Setup

When deploying behind a reverse proxy (nginx, Caddy, Cloudflare Tunnel, etc.),
set the `SERVER_PUBLIC_URL` environment variable to your public-facing URL:

```bash
# Example with nginx reverse proxy
SERVER_PUBLIC_URL=https://arabica.example.com
SECURE_COOKIES=true
PORT=18910

# The server listens on localhost:18910
# But OAuth callbacks use https://arabica.example.com/oauth/callback
```

The `SERVER_PUBLIC_URL` is used for OAuth client metadata and callback URLs,
ensuring the AT Protocol OAuth flow works correctly when the server is accessed
via a different URL than it's running on.

## License

MIT
