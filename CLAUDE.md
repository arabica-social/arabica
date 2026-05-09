# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Arabica is a coffee brew tracking application built on AT Protocol. User data
(beans, brews, cafes, drinks, etc.) lives in each user's Personal Data Server
(PDS), not locally. The app authenticates via OAuth, then performs CRUD through
XRPC calls to the user's PDS.

## Build & Development Commands

```bash
# Run development server (debug logging, moderator config, known-dids backfill)
just run

# Run tests
go test ./...

# Run a single test
go test ./internal/models/... -run TestBeanIsIncomplete

# After editing .templ files — regenerate Go code
templ generate            # all files
templ generate -f <file>  # single file

# Rebuild Tailwind CSS (required after CSS/class changes)
just style

# Verify after changes
go vet ./...
go build ./...
```

## Workflow Rules

Do NOT spend more than 2-3 minutes exploring/reading files before beginning
implementation. If the task is clear, start writing code immediately. Ask
clarifying questions rather than endlessly reading the codebase. When given a
specific implementation task, produce code changes in the same session.

## Work Management

This project uses **cells** for task tracking. See `.cells/AGENTS.md` for usage.

- `./cells list` / `./cells list --status open` / `./cells show <cell-id>`
- Do NOT use `./cells run` (spawns new agent session, humans only)

## Dependencies

Prefer standard library solutions over external dependencies. Only add a
third-party dependency if stdlib genuinely cannot handle the requirement.

## Task Agents

Hard limit of 3 agents maximum. Each agent must have a clearly scoped
deliverable. Do not poll agents in a loop. If agents aren't producing results
within 5 minutes, fall back to doing the work directly.

## Tech Stack

- **Language:** Go 1.21+, stdlib `net/http` with Go 1.22 routing
- **Storage:** AT Protocol PDS (user data), BoltDB (sessions), SQLite (firehose index)
- **Frontend:** HTMX + Alpine.js + Tailwind CSS
- **Templates:** [Templ](https://templ.guide/) (type-safe Go templates)
- **Logging:** zerolog

## Architecture

### AT Protocol Integration

1. User authenticates via OAuth (indigo SDK handles PKCE/DPOP)
2. Handler creates `AtprotoStore` scoped to user's DID + session
3. Store methods make XRPC calls to user's PDS
4. Results rendered via Templ components or returned as JSON

**Collections (NSIDs)** — defined in `internal/atproto/nsid.go`:

- `social.arabica.alpha.bean` — Coffee beans (references roaster)
- `social.arabica.alpha.roaster` — Roasters
- `social.arabica.alpha.grinder` — Grinders
- `social.arabica.alpha.brewer` — Brewing devices
- `social.arabica.alpha.cafe` — Cafes (references roaster)
- `social.arabica.alpha.brew` — Brew sessions (references bean, grinder, brewer, recipe)
- `social.arabica.alpha.drink` — Drinks at cafes (references cafe, bean)
- `social.arabica.alpha.recipe` — Recipes (references brewer)
- `social.arabica.alpha.like` — Likes (strongRef to any record)
- `social.arabica.alpha.comment` — Comments (strongRef to any record, optional parent for threads)

Records reference each other via AT-URIs (`at://did/collection/rkey`). Record
keys use TID format (timestamp-based identifiers).

### Store Interface

`internal/database/store.go` defines the `Store` interface with CRUD methods for
all entity types. `AtprotoStore` is the production implementation backed by the
user's PDS with witness cache and session cache layers.

### Three-Layer Caching

1. **SessionCache** (`internal/atproto/cache.go`) — per-user in-memory cache
   (2-min TTL). Copy-on-write pattern, invalidated on writes. Dirty-collection
   tracking skips witness cache after local writes until firehose catches up.

2. **WitnessCache** (`internal/firehose/index.go`) — SQLite-backed local index
   populated by the Jetstream firehose consumer. Provides fast reads without PDS
   calls. Used as fallback when session cache misses.

3. **PDS fallback** — direct XRPC calls to the user's PDS when both caches miss.

Write path: PDS write -> write-through to witness cache -> invalidate session
cache (mark dirty).

### Firehose & Feed Pipeline

`internal/firehose/` subscribes to AT Protocol's Jetstream relay for real-time
events. Records are indexed into the SQLite feed index. The feed pipeline:

1. **FeedIndex** (`firehose/index.go`) — SQLite store, `recordToFeedItem()`
   converts indexed records to `feed.FeedItem` structs with resolved
   references.
2. **FeedIndexAdapter** (`firehose/adapter.go`) — implements
   `feed.FirehoseIndex` over `FeedIndex`. Items pass through as
   `*feed.FeedItem`; the adapter only bridges the
   `FirehoseFeedQuery`/`FeedQuery` query structs.
3. **Feed Service** (`feed/service.go`) — applies moderation filtering,
   caching, and pagination. The handler populates `IsLikedByViewer` and
   `IsOwner` on each item once a viewer is identified.

When adding a new entity type, fields live on a single `feed.FeedItem`
struct; entity-specific population lives in `recordToFeedItem`'s switch.

### Adding a New Entity Type (Checklist)

The full stack for a new entity requires changes across many files. Follow the
pattern of an existing entity (e.g., roaster for simple entities, brew for
entities with references):

1. **Lexicon JSON** in `lexicons/`
2. **NSID constant** in `internal/atproto/nsid.go`
3. **RecordType constant** in `internal/lexicons/record_type.go` (const + ParseRecordType + DisplayName)
4. **Model + request types + validation** in `internal/models/models.go`
5. **Record conversion** (`XToRecord`/`RecordToX`) in `internal/atproto/records.go`
6. **Store interface methods** in `internal/database/store.go`
7. **AtprotoStore implementation** in `internal/atproto/store.go` (CRUD + witness + cache)
8. **Cache fields + Set/Invalidate methods** in `internal/atproto/cache.go`
9. **OAuth scope** in `internal/atproto/oauth.go`
10. **Firehose config** (collection list) in `internal/firehose/config.go`
11. **`recordToFeedItem` switch case** in `internal/firehose/index.go` (entity-specific population on `feed.FeedItem`)
12. **`FeedItem` fields** in `internal/feed/service.go` if the new entity needs a field on the shared payload
13. **CRUD handlers** in `internal/handlers/entities.go` (also update `HandleManagePartial`, `HandleAPIListAll`, `HandleManageRefresh`)
14. **View + OG image handlers** in `internal/handlers/entity_views.go`
15. **Modal handlers** in `internal/handlers/modals.go`
16. **Routes** in `internal/routing/routing.go` (page views, API CRUD, modals, OG images)
17. **Templ view page** in `internal/web/pages/` (e.g., `cafe_view.templ`)
18. **Templ record content** in `internal/web/components/` (e.g., `record_cafe.templ`)
19. **Entity table component** in `internal/web/components/entity_tables.templ`
20. **Dialog modal** in `internal/web/components/dialog_modals.templ` (+ `getStringValue` cases)
21. **Manage partial** tab in `internal/web/components/manage_partial.templ`
22. **My Coffee tab** in `internal/web/pages/my_coffee.templ`
23. **Feed card** switch cases in `internal/web/pages/feed.templ` (card class, content, ActionText, share URL, title, delete URL)
24. **OG card** function in `internal/ogcard/entities.go` (+ accent color in `brew.go`)
25. **Suggestions** config in `internal/suggestions/suggestions.go` + handler map in `internal/handlers/suggestions.go`
26. **Client-side cache** entity case in `static/js/combo-select.js` `getUserEntities()`

### Templ Architecture

**Tabs only in `.templ` files** — never use spaces for indentation. A post-edit
hook runs `templ fmt` automatically. After editing `.templ` files, run
`templ generate` to regenerate Go code.

Pages (`internal/web/pages/`) accept `*components.LayoutData` + page-specific
props. Components (`internal/web/components/`) are reusable building blocks.

Pattern: `pages.PageName(layoutData, props).Render(r.Context(), w)`

### Combo-Select Component System

Entity selection dropdowns (bean, grinder, brewer, roaster, cafe) use a shared
combo-select pattern with typeahead search, community suggestions, and inline
creation:

- **Go config**: `components.ComboSelectConfig()` in `components/combo_select.templ`
  generates Alpine.js `x-data` with entity-specific label formatting and create
  data mapping.
- **Templ markup**: `components.ComboSelectInput()` renders the shared dropdown UI.
- **JS behavior**: `static/js/combo-select.js` — Alpine.js component that
  searches user records (from client-side cache), community suggestions (from
  `/api/suggestions/{entity}`), and creates new entities inline via POST.
- **Suggestions backend**: `internal/suggestions/suggestions.go` — entity configs
  define searchable fields and dedup keys.

To add a new entity to combo-select: add a case to `ComboSelectConfig`, add to
`getUserEntities()` in `combo-select.js`, add entity config to
`suggestions.go`, and add to the entity-to-NSID map in
`handlers/suggestions.go`.

### Entity View Handler Pattern

View handlers (`HandleXView`) support both authenticated (own records) and
public (via `?owner=` parameter) access. They:

1. Try witness cache first, fall back to PDS
2. Resolve references (e.g., roaster for cafe)
3. Populate OG metadata for social sharing
4. Fetch social data (likes, comments, moderation state)
5. Render the templ page with all props

### CSS Cache Busting

When making CSS/style changes, bump the version query parameter in
`internal/web/components/layout.templ`:

```html
<link rel="stylesheet" href="/static/css/output.css?v=0.1.3" />
```

## Testing Conventions

All tests MUST use [testify/assert](https://github.com/stretchr/testify). Do
NOT use `if` statements with `t.Error()`.

```go
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.Contains(t, haystack, needle)
assert.True(t, value)
assert.Nil(t, value)
```

## Using Go Tooling

- `go mod download -json MODULE` — get dependency source path
- `go doc foo.Bar` — read package/type/function docs
- `go run ./cmd/arabica` instead of `go build` to avoid artifacts

## Design Context

See `.impeccable.md` for the full design system reference. Key points:

### Brand Personality
**Cozy, social, inviting** — like a neighborhood specialty cafe. Warm, not
clinical. The emotional goals are calm satisfaction, geeky delight, community
belonging, and craft pride.

### Visual References
- Specialty coffee bag packaging (Counter Culture, Onyx) — craft labels, earthy
  tones, confident type
- Analog journals — Moleskine, handwritten brew logs, texture of paper and ink

### Design Principles
1. **Warmth over precision** — Brown paper, not graph paper
2. **Quiet confidence** — Strong typography, restrained color, let content shine
3. **Tactile texture** — Evoke the analog: ceramic, kraft, journal pages
4. **Community as atmosphere** — Cafe conversations, not social media timelines
5. **Respect the ritual** — No urgency, no gamification, intentional interactions

### Typography
Iosevka Patrick (custom monospace) is the core UI font. Open to pairing with a
warmer display font for headings.
