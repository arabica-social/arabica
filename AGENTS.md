# Arabica - Project Context for AI Agents

Coffee brew tracking application using AT Protocol for decentralized storage.

## Tech Stack

- **Language:** Go 1.25+
- **HTTP:** stdlib `net/http` with Go 1.22 routing
- **Storage:** AT Protocol PDS (user data), BoltDB (sessions/feed registry)
- **Frontend:** Svelte SPA with client-side routing
- **Testing:** Standard library testing + [shutter](https://github.com/ptdewey/shutter) for snapshot tests
- **Logging:** zerolog

## Use Go Tooling Effectively

- To see source files from a dependency, or to answer questions
  about a dependency, run `go mod download -json MODULE` and use
  the returned `Dir` path to read the files.

- Use `go doc foo.Bar` or `go doc -all foo` to read documentation
  for packages, types, functions, etc.

- Use `go run .` or `go run ./cmd/foo` instead of `go build` to
  run programs, to avoid leaving behind build artifacts.

## Project Structure

```
cmd/arabica-server/main.go          # Application entry point
internal/
  atproto/                  # AT Protocol integration
    client.go               # Authenticated PDS client (XRPC calls)
    oauth.go                # OAuth flow with PKCE/DPOP
    store.go                # database.Store implementation using PDS
    cache.go                # Per-session in-memory cache
    records.go              # Model <-> ATProto record conversion
    resolver.go             # AT-URI parsing and reference resolution
    public_client.go        # Unauthenticated public API access
    nsid.go                 # Collection NSIDs and AT-URI builders
  handlers/
    handlers.go             # HTTP handlers (API endpoints)
    auth.go                 # OAuth login/logout/callback
    api_snapshot_test.go    # Snapshot tests for API responses
    testutil.go             # Test helpers and fixtures
    __snapshots__/          # Snapshot files for regression testing
  database/
    store.go                # Store interface definition
    store_mock.go           # Mock implementation for testing
    boltstore/              # BoltDB implementation for sessions
  feed/
    service.go              # Community feed aggregation
    registry.go             # User registration for feed
  models/
    models.go               # Domain models and request types
  middleware/
    logging.go              # Request logging middleware
  routing/
    routing.go              # Router setup and middleware chain
frontend/                   # Svelte SPA source code
  src/
    routes/                 # Page components
    components/             # Reusable components
    stores/                 # Svelte stores (auth, cache)
    lib/                    # Utilities (router, API client)
  public/                   # Built SPA assets
lexicons/                   # AT Protocol lexicon definitions (JSON)
static/                     # Static assets (CSS, icons, service worker)
  app/                      # Built Svelte SPA
```

## Key Concepts

### AT Protocol Integration

User data stored in their Personal Data Server (PDS), not locally. The app:

1. Authenticates via OAuth (indigo SDK handles PKCE/DPOP)
2. Gets access token scoped to user's DID
3. Performs CRUD via XRPC calls to user's PDS

**Collections (NSIDs):**

- `social.arabica.alpha.bean` - Coffee beans
- `social.arabica.alpha.roaster` - Roasters
- `social.arabica.alpha.grinder` - Grinders
- `social.arabica.alpha.brewer` - Brewing devices
- `social.arabica.alpha.brew` - Brew sessions (references bean, grinder, brewer)

**Record keys:** TID format (timestamp-based identifiers)

**References:** Records reference each other via AT-URIs (`at://did/collection/rkey`)

### Store Interface

`internal/database/store.go` defines the `Store` interface. Two implementations:

- `AtprotoStore` - Production, stores in user's PDS
- BoltDB stores only sessions and feed registry (not user data)

All Store methods take `context.Context` as first parameter.

### Request Flow

1. Request hits middleware (logging, auth check)
2. Auth middleware extracts DID + session ID from cookies
3. For SPA routes: Serve index.html (client-side routing)
4. For API routes: Handler creates `AtprotoStore` scoped to user
5. Store methods make XRPC calls to user's PDS
6. Results returned as JSON

### Caching

`SessionCache` caches user data in memory (5-minute TTL):

- Avoids repeated PDS calls for same data
- Invalidated on writes
- Background cleanup removes expired entries

### Backfill Strategy

User records are backfilled from their PDS once per DID:

- **On startup**: Backfills registered users + known-dids file
- **On first login**: Backfills the user's historical records
- **Deduplication**: Tracks backfilled DIDs in `BucketBackfilled` to prevent redundant fetches
- **Idempotent**: Safe to call multiple times (checks backfill status first)

This prevents excessive PDS requests while ensuring new users' historical data is indexed.

## Common Tasks

### Run Development Server

```bash
# Run server (uses firehose mode by default)
go run cmd/arabica-server/main.go

# Backfill known DIDs on startup
go run cmd/arabica-server/main.go --known-dids known-dids.txt

# Using nix
nix run
```

### Run Tests

```bash
go test ./...
```

#### Snapshot Testing

Backend API responses are tested using snapshot tests with the [shutter](https://github.com/ptdewey/shutter) library. Snapshot tests capture the JSON response format and verify it remains consistent across changes.

**Location:** `internal/handlers/api_snapshot_test.go`

**Covered endpoints:**

- Authentication: `/api/me`, `/client-metadata.json`
- Data fetching: `/api/data`, `/api/feed-json`, `/api/profile-json/{actor}`
- CRUD operations: Create/Update/Delete for beans, roasters, grinders, brewers, brews

**Running snapshot tests:**

```bash
cd internal/handlers && go test -v -run "Snapshot"
```

**Working with snapshots:**

```bash
# Accept all new/changed snapshots
shutter accept-all

# Reject all pending snapshots
shutter reject-all

# Review snapshots interactively
shutter review
```

**Snapshot patterns used:**

- `shutter.ScrubTimestamp()` - Removes timestamp values for deterministic tests
- `shutter.IgnoreKey("created_at")` - Ignores specific JSON keys
- `shutter.IgnoreKey("rkey")` - Ignores AT Protocol record keys (TIDs are time-based)

Snapshots are stored in `internal/handlers/__snapshots__/` and should be committed to version control.

### Build

```bash
go build -o arabica cmd/arabica-server/main.go
```

## Environment Variables

| Variable                    | Default                              | Description                                                      |
| --------------------------- | ------------------------------------ | ---------------------------------------------------------------- |
| `PORT`                      | 18910                                | HTTP server port                                                 |
| `SERVER_PUBLIC_URL`         | -                                    | Public URL for reverse proxy (enables secure cookies when HTTPS) |
| `ARABICA_DB_PATH`           | ~/.local/share/arabica/arabica.db    | BoltDB path (sessions, registry)                                 |
| `ARABICA_FEED_INDEX_PATH`   | ~/.local/share/arabica/feed-index.db | Firehose index BoltDB path                                       |
| `ARABICA_PROFILE_CACHE_TTL` | 1h                                   | Profile cache duration                                           |
| `LOG_LEVEL`                 | info                                 | debug/info/warn/error                                            |
| `LOG_FORMAT`                | console                              | console/json                                                     |

## Code Patterns

### Creating a Store

```go
// In handlers, store is created per-request
store, authenticated := h.getAtprotoStore(r)
if !authenticated {
    http.Error(w, "Authentication required", http.StatusUnauthorized)
    return
}

// Use store with request context
brews, err := store.ListBrews(r.Context(), userID)
```

### Record Conversion

```go
// Model -> ATProto record
record, err := BrewToRecord(brew, beanURI, grinderURI, brewerURI)

// ATProto record -> Model
brew, err := RecordToBrew(record, atURI)
```

### AT-URI Handling

```go
// Build AT-URI
uri := BuildATURI(did, NSIDBean, rkey)  // at://did:plc:xxx/social.arabica.alpha.bean/abc

// Parse AT-URI
components, err := ResolveATURI(uri)
// components.DID, components.Collection, components.RKey
```

## Future Vision: Social Features

The app currently has a basic community feed. Future plans expand social interactions leveraging AT Protocol's decentralized nature.

### Planned Lexicons

```
social.arabica.alpha.like      - Like a brew (references brew AT-URI)
social.arabica.alpha.comment   - Comment on a brew
social.arabica.alpha.follow    - Follow another user
social.arabica.alpha.share     - Re-share a brew to your feed
```

### Like Record (Planned)

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.like",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The brew being liked"
          },
          "createdAt": { "type": "string", "format": "datetime" }
        }
      }
    }
  }
}
```

### Comment Record (Planned)

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.comment",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["subject", "text", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The brew being commented on"
          },
          "text": {
            "type": "string",
            "maxLength": 1000,
            "maxGraphemes": 300
          },
          "createdAt": { "type": "string", "format": "datetime" }
        }
      }
    }
  }
}
```

### Implementation Approach

**Cross-user interactions:**

- Likes/comments stored in the actor's PDS (not the brew owner's)
- Use `public_client.go` to read other users' brews
- Aggregate likes/comments via relay/firehose or direct PDS queries

**Feed aggregation:**

- Current: Poll registered users' PDS for brews
- Future: Subscribe to firehose for real-time updates
- Index social interactions in local DB for fast queries

**UI patterns:**

- Like button on brew cards in feed
- Comment thread below brew detail view
- Share button to re-post with optional note
- Notification system for interactions on your brews

### Key Design Decisions

1. **Strong references** - Likes/comments use `com.atproto.repo.strongRef` (URI + CID) to ensure the referenced brew hasn't changed
2. **Actor-owned data** - Your likes live in your PDS, not the brew owner's
3. **Public by default** - Social interactions are public records, readable by anyone
4. **Portable identity** - Users can switch PDS and keep their social graph

## Deployment Notes

### CSS Cache Busting

When making CSS/style changes, bump the version query parameter in `templates/layout.tmpl`:

```html
<link rel="stylesheet" href="/static/css/output.css?v=0.1.3" />
```

Cloudflare caches static assets, so incrementing the version ensures users get the updated styles.

## Known Issues / TODOs

See @BACKLOG.md
