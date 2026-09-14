# Arabica

Coffee brew logging application built on ATProto.

Development is on Tangled, and is mirrored to GitHub:

- [Tangled](https://tangled.org/arabica.social/arabica)
- [GitHub](https://github.com/arabica-social/arabica)

## Quick Start

With Nix (builds the SPA and Go binary):

```bash
nix run
```

With Go directly (requires Node.js and pnpm for the SPA build):

```bash
pnpm install
./scripts/build-spa.sh
go run ./cmd/arabica
```

Access at http://127.0.0.1:18910

## Configuration

### Command-Line Flags

- `--known-dids <file>` - Path to file with DIDs to backfill on startup (one per
  line)

### Environment Variables

App-scoped variables accept an `ARABICA_`-prefixed form (e.g. `ARABICA_PORT`) or
an unprefixed form (e.g. `PORT`); the prefixed form wins when both are set.

- `ARABICA_PORT` / `PORT` - Server port (default: 18910)
- `ARABICA_BIND_ADDR` / `BIND_ADDR` - Bind address (default: 0.0.0.0)
- `ARABICA_PUBLIC_URL` / `SERVER_PUBLIC_URL` - Public URL for reverse proxy
  deployments (e.g., https://arabica.example.com)
- `ARABICA_DATA_DIR` - Directory for the SQLite database (feed index, OAuth
  sessions, moderation state). Defaults to `$XDG_DATA_HOME/arabica`, then
  `~/.local/share/arabica`.
- `ARABICA_PROFILE_CACHE_TTL` - Profile cache duration (default: 1h)
- `ARABICA_METRICS_PORT` / `METRICS_PORT` - Prometheus metrics port, bound to
  `127.0.0.1` (default: 9101)
- `ARABICA_MODERATORS_CONFIG` - Path to the moderator roles JSON file
- `ARABICA_OAUTH_CLIENT_ID` / `OAUTH_CLIENT_ID` - OAuth client ID (optional,
  uses loopback mode if not set)
- `ARABICA_OAUTH_REDIRECT_URI` / `OAUTH_REDIRECT_URI` - OAuth redirect URI
  (optional)
- `ARABICA_DEV` - Enable dev mode: re-read CSS and the SPA build from disk on
  each request, and unlock dev-only signup providers
- `SECURE_COOKIES` - Set to true for HTTPS (default: false)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: console, json (default: console)
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP HTTP endpoint for tracing (default:
  `localhost:4318`)

## Development

### Prerequisites

- [Go](https://go.dev/) 1.26+
- [Node.js](https://nodejs.org/) and [pnpm](https://pnpm.io/)
- [just](https://github.com/casey/just) (optional but recommended); run helpers
  in `justfile`
- [inotify-tools](https://github.com/inotify-tools/inotify-tools) for the SPA
  watch workflows (`just run-spa-dev`, `just spa-watch`)

### Setup

1. `pnpm install` to install the SPA workspace dependencies.
2. Create `roles.json` with moderator roles. Env var:
   `ARABICA_MODERATORS_CONFIG=roles.json`
3. (Optional) Create `known-dids.txt` with one DID per line. Flag:
   `--known-dids known-dids.txt`

### Running

The SvelteKit SPA builds into `internal/web/spa/build` and is embedded into the
Go binary, so `web/` changes require a rebuild before the server serves them.
The CSS bundle, by contrast, is assembled in-process at server startup.

With Nix:

```bash
nix develop
just run
```

Without Nix, run helpers in `justfile`:

```sh
just run          # build the SPA, then run the Go server (debug logging, dev mode, moderator config)
just run-spa-dev  # Go backend + SPA rebuild watcher, live-rebuilds on web/ changes
just spa-dev      # Vite HMR dev server (proxies APIs to a Go backend on :18910)
just herdr-spa-dev  # launch the backend and SPA watcher as panes in a Herdr workspace
```

`just run-spa-dev` is the usual local workflow once `inotifywait` is available:
edit a Svelte file, refresh the browser, and the server picks up the rebuilt
assets. `just spa-dev` gives full Vite HMR but does not render Go's injected
document head, so pair it with a running server for API and OAuth routes.

### Testing

```sh
just test                 # build the SPA and run Go tests with coverage
just integration-test     # build the SPA and run integration tests
cd web && pnpm run check  # Svelte/TypeScript type checking
cd web && pnpm run test   # Vitest component tests
just e2e                  # build and run Playwright specs against an e2e server
just ci-check             # full local CI checkpoint
just format               # prettier + gofmt
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

# The server listens on 127.0.0.1:18910
# But OAuth callbacks use https://arabica.example.com/oauth/callback
```

The `SERVER_PUBLIC_URL` is used for OAuth client metadata and callback URLs,
ensuring the AT Protocol OAuth flow works correctly when the server is accessed
via a different URL than it's running on.

## License

MIT
