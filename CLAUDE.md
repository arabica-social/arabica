# Arabica - Project Context for AI Agents

Coffee brew tracking application using AT Protocol for decentralized storage.

## Tech Stack

- **Language:** Go 1.21+
- **HTTP:** stdlib `net/http` with Go 1.22 routing
- **Storage:** AT Protocol PDS (user data), BoltDB (sessions/feed registry)
- **Frontend:** HTMX + Alpine.js + Tailwind CSS
- **Templates:** html/template
- **Logging:** zerolog

## Project Structure

```
cmd/server/main.go          # Application entry point
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
    handlers.go             # HTTP handlers for all routes
    auth.go                 # OAuth login/logout/callback
  bff/
    render.go               # Template rendering helpers
    helpers.go              # View helpers (formatting, etc.)
  database/
    store.go                # Store interface definition
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
lexicons/                   # AT Protocol lexicon definitions (JSON)
templates/                  # HTML templates
web/static/                 # CSS, JS, manifest
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
3. Handler creates `AtprotoStore` scoped to user
4. Store methods make XRPC calls to user's PDS
5. Results rendered via BFF templates or returned as JSON

### Caching

`SessionCache` caches user data in memory (5-minute TTL):

- Avoids repeated PDS calls for same data
- Invalidated on writes
- Background cleanup removes expired entries

## Common Tasks

### Run Development Server

```bash
# Basic mode (polling-based feed)
go run cmd/server/main.go

# With firehose (real-time AT Protocol feed)
go run cmd/server/main.go --firehose

# With firehose + backfill known DIDs
go run cmd/server/main.go --firehose --known-dids known-dids.txt

# Using nix
nix run
```

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o arabica cmd/server/main.go
```

## Command-Line Flags

| Flag            | Type   | Default | Description                                           |
| --------------- | ------ | ------- | ----------------------------------------------------- |
| `--firehose`    | bool   | false   | Enable real-time firehose feed via Jetstream          |
| `--known-dids`  | string | ""      | Path to file with DIDs to backfill (one per line)     |

**Known DIDs File Format:**
- One DID per line (e.g., `did:plc:abc123xyz`)
- Lines starting with `#` are comments
- Empty lines are ignored
- See `known-dids.txt.example` for reference

## Environment Variables

| Variable                    | Default                           | Description                        |
| --------------------------- | --------------------------------- | ---------------------------------- |
| `PORT`                      | 18910                             | HTTP server port                   |
| `SERVER_PUBLIC_URL`         | -                                 | Public URL for reverse proxy       |
| `ARABICA_DB_PATH`           | ~/.local/share/arabica/arabica.db | BoltDB path (sessions, registry)   |
| `ARABICA_FEED_INDEX_PATH`   | ~/.local/share/arabica/feed-index.db | Firehose index BoltDB path     |
| `ARABICA_PROFILE_CACHE_TTL` | 1h                                | Profile cache duration             |
| `SECURE_COOKIES`            | false                             | Set true for HTTPS                 |
| `LOG_LEVEL`                 | info                              | debug/info/warn/error              |
| `LOG_FORMAT`                | console                           | console/json                       |

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

Key areas:

- Context should flow through methods (some fixed, verify all paths)
- Cache race conditions need copy-on-write pattern
- Missing CID validation on record updates (AT Protocol best practice)
- Rate limiting for PDS calls not implemented
