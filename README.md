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
templ generate
go run ./cmd/arabica
```

Access at http://127.0.0.1:18910 (arabica) or http://127.0.0.1:18920 (oolong)

## Configuration

### Command-Line Flags

- `--known-dids <file>` - Path to file with DIDs to backfill on startup (one per
  line)

### Environment Variables

- `PORT` - Server port (default: 18910)
- `SERVER_PUBLIC_URL` - Public URL for reverse proxy deployments (e.g.,
  https://arabica.example.com)
- `ARABICA_DB_PATH` - OAuth session database path. Defaults to
  <XDG_DATA_HOME or ~/.local/share>/arabica/arabica.db. Only needed to override
  the default location.
- `ARABICA_PROFILE_CACHE_TTL` - Profile cache duration (default: 1h)
- `OAUTH_CLIENT_ID` - OAuth client ID (optional, uses loopback mode if not set)
- `OAUTH_REDIRECT_URI` - OAuth redirect URI (optional)
- `SECURE_COOKIES` - Set to true for HTTPS (default: false)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: console, json (default: console)

## Development

### Prerequisites

- [Go](https://go.dev/) 1.26+
- [Templ](https://templ.guide/):
  `go install github.com/a-h/templ/cmd/templ@latest`
- [just](https://github.com/casey/just) (optional but recommended); run helpers
  in `justfile`

### Setup

1. Create `roles.json` with moderator roles. Env var:
   `ARABICA_MODERATORS_CONFIG=roles.json`
2. (Optional) Create `known-dids.txt` with one DID per line. Flag:
   `-known-dids known-dids.txt`

### Running

With Nix:

```bash
nix develop
templ generate
go run ./cmd/arabica
```

Without Nix, there are run helpers in `justfile`:

```sh
just run          # dev server: debug logging, hot reload, moderator config
just test         # run tests (regenerates templ first)
just templ-watch  # run with live template regeneration
```

CSS and JS are bundled in-process at server startup — no external build step
needed. For development set `ARABICA_DEV=1` to enable CSS+JS hot reload and
unlock dev-only signup providers (e.g. pds.rip on `/join/create`).

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

# The server listens on 127.0.0.1:18910
# But OAuth callbacks use https://arabica.example.com/oauth/callback
```

The `SERVER_PUBLIC_URL` is used for OAuth client metadata and callback URLs,
ensuring the AT Protocol OAuth flow works correctly when the server is accessed
via a different URL than it's running on.

## License

MIT

