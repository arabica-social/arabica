## Project Overview

Arabica is a coffee brew tracking application built on AT Protocol. User data
(beans, brews, cafes, drinks, etc.) lives in each user's Personal Data Server
(PDS), not locally. The app authenticates via OAuth, then performs CRUD through
XRPC calls to the user's PDS.

## Build & Development Commands

```bash
# Run tests
go test ./...

# Regenerate templ after changes
templ generate

# Verify after changes
go vet ./...
go build ./...

# Format
just format
```

## Version Control

Prefer `jj` over `git` for all version control operations in this repo. Fall
back to `git` only when `jj` cannot accomplish the task.

## Dependencies

Prefer standard library solutions over external dependencies. Only add a
third-party dependency if stdlib genuinely cannot handle the requirement.

## Tech Stack

- Language: Go 1.26+, stdlib `net/http` with Go 1.22 routing
- Storage: atproto PDS (user data), SQLite (firehose index, auth sessions)
- Frontend: HTMX + Svelte islands + plain CSS (utility-class pattern) + Templ
  (templates)

## Architecture

### AT Protocol Integration

1. User authenticates via OAuth (indigo SDK handles PKCE/DPOP)
2. Handler creates `AtprotoStore` scoped to user's DID + session
3. Store methods make XRPC calls to user's PDS
4. Results rendered via Templ components or returned as JSON

Collections (NSIDs) — defined in `internal/atproto/nsid.go`:

- `social.arabica.alpha.bean` — Coffee beans (references roaster)
- `social.arabica.alpha.roaster` — Roasters
- `social.arabica.alpha.grinder` — Grinders
- `social.arabica.alpha.brewer` — Brewing devices
- `social.arabica.alpha.cafe` — Cafes (references roaster)
- `social.arabica.alpha.brew` — Brew sessions (references bean, grinder, brewer,
  recipe)
- `social.arabica.alpha.drink` — Drinks at cafes (references cafe, bean)
- `social.arabica.alpha.recipe` — Recipes (references brewer)
- `social.arabica.alpha.like` — Likes (strongRef to any record)
- `social.arabica.alpha.comment` — Comments (strongRef to any record, optional
  parent for threads)

Records reference each other via AT-URIs (`at://did/collection/rkey`). Record
keys use TID format (timestamp-based identifiers).

### Store Interface

`internal/arabica/store/store.go` defines the `Store` interface with CRUD
methods for all entity types. `AtprotoStore` is the production implementation
backed by the user's PDS with witness cache and session cache layers.

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
   converts indexed records to `feed.FeedItem` structs with resolved references.
2. **FeedIndexAdapter** (`firehose/adapter.go`) — implements
   `feed.FirehoseIndex` over `FeedIndex`. Items pass through as
   `*feed.FeedItem`; the adapter only bridges the
   `FirehoseFeedQuery`/`FeedQuery` query structs.
3. **Feed Service** (`feed/service.go`) — applies moderation filtering, caching,
   and pagination. The handler populates `IsLikedByViewer` and `IsOwner` on each
   item once a viewer is identified.

When adding a new entity type, fields live on a single `feed.FeedItem` struct;
entity-specific population lives in `recordToFeedItem`'s switch.

### Adding a New Entity Type (Checklist)

The full stack for a new entity requires changes across many files. Follow the
pattern of an existing entity (e.g., roaster for simple entities, brew for
entities with references):

1. **Lexicon JSON** in `lexicons/<namespace>/` (path mirrors the NSID, e.g.
   `lexicons/social/arabica/alpha/bean.json`)
2. **NSID constant** in `internal/atproto/nsid.go`
3. **RecordType constant** in `internal/lexicons/record_type.go` (const +
   ParseRecordType + DisplayName)
4. **Model + request types + validation** in `internal/models/models.go`
5. **Record conversion** (`XToRecord`/`RecordToX`) in
   `internal/atproto/records.go`
6. **Store interface methods** in `internal/arabica/store/store.go`
7. **AtprotoStore implementation** in `internal/atproto/store.go` (CRUD +
   witness + cache)
8. **Cache fields + Set/Invalidate methods** in `internal/atproto/cache.go`
9. **OAuth scope** in `internal/atproto/oauth.go`
10. **Firehose config** (collection list) in `internal/firehose/config.go`
11. **`recordToFeedItem` switch case** in `internal/firehose/index.go`
    (entity-specific population on `feed.FeedItem`)
12. **`FeedItem` fields** in `internal/feed/service.go` if the new entity needs
    a field on the shared payload
13. **CRUD handlers** in `internal/handlers/entities.go` (also update
    `HandleManagePartial`, `HandleAPIListAll`, `HandleManageRefresh`)
14. **View + OG image handlers** in `internal/handlers/entity_views.go`
15. **Modal handlers** in `internal/handlers/modals.go`
16. **Routes** in `internal/routing/routing.go` (page views, API CRUD, modals,
    OG images)
17. **Templ view page** in `internal/web/pages/` (e.g., `cafe_view.templ`)
18. **Templ record content** in `internal/web/components/` (e.g.,
    `record_cafe.templ`)
19. **Entity table component** in `internal/web/components/entity_tables.templ`
20. **Dialog modal** in `internal/web/components/dialog_modals.templ` (+
    `getStringValue` cases)
21. **Manage partial** tab in `internal/web/components/manage_partial.templ`
22. **My Coffee tab** in `internal/web/pages/my_coffee.templ`
23. **Feed card** switch cases in `internal/web/pages/feed.templ` (card class,
    content, ActionText, share URL, title, delete URL)
24. **OG card** function in `internal/ogcard/entities.go` (+ accent color in
    `brew.go`)
25. **Suggestions** config in `internal/suggestions/suggestions.go` + handler
    map in `internal/handlers/suggestions.go`
26. **Client-side cache** entity case in
    `internal/web/assets/svelte/src/EntityCombo.svelte` `cachedEntities()`

### Templ Architecture

**Tabs only in `.templ` files** — never use spaces for indentation. A post-edit
hook runs `templ fmt` automatically. After editing `.templ` files, run
`templ generate` to regenerate Go code.

Pages (`internal/web/pages/`) accept `*components.LayoutData` + page-specific
props. Components (`internal/web/components/`) are reusable building blocks.

Pattern: `pages.PageName(layoutData, props).Render(r.Context(), w)`

### Svelte Islands & Combo-Select

Entity selection dropdowns (bean, grinder, brewer, roaster, cafe) use a shared
combo-select pattern with typeahead search, community suggestions, and inline
creation:

- **Templ markup**: complex forms render Svelte island mount points and use
  `EntityCombo.svelte` inside the island for entity-specific selections.
- **Svelte behavior**: `internal/web/assets/svelte/src/EntityCombo.svelte`
  searches user records from the Svelte app cache, community suggestions from
  `/api/suggestions/{entity}`, and creates new entities inline via POST.
- **Entity config**: `internal/web/assets/svelte/src/comboSelectRegistry.ts`
  owns entity-specific label formatting, extra fields, and create data mapping.
- **Suggestions backend**: `internal/suggestions/suggestions.go` — entity
  configs define searchable fields and dedup keys.

To add a new entity to combo-select: add entity config to
`comboSelectRegistry.ts`, add the cached entity case to `cachedEntities()` in
`EntityCombo.svelte`, add entity config to `suggestions.go`, and add to the
entity-to-NSID map in `handlers/suggestions.go`.

### Entity View Handler Pattern

View handlers (`HandleXView`) support both authenticated (own records) and
public (via `?owner=` parameter) access. They:

1. Try witness cache first, fall back to PDS
2. Resolve references (e.g., roaster for cafe)
3. Populate OG metadata for social sharing
4. Fetch social data (likes, comments, moderation state)
5. Render the templ page with all props

### Static assets (CSS + JS)

The `internal/web/assets` package owns the front-end source tree:

- `assets/css/tokens.css`, `reset.css`, `utilities.css` — base CSS layers
- `assets/css/components/*.css` — numbered (`01-dark-mode.css`, …) so the prefix
  preserves cascade under alphabetical glob expansion
- `assets/css/themes/<app>.css` — per-app theme overlay
- `assets/js/*.js` — every JS module the templates reference

All files are `go:embed`ed and served from in-memory caches:

- **CSS**: concatenated into one bundle per app at startup; URL is
  `/static/css/output.css` (or `/static/css/output-<app>.css`) with
  `?h=<sha256-prefix>` cache buster auto-derived from content
- **JS**: served per-file at `/static/js/<name>` with the same `?h=...` query
  param; templates reference each file via `{ assets.JSHrefFor("name.js") }`

Both subsystems set `Cache-Control: public, max-age=31536000, immutable` in
production and honor `If-None-Match` for 304s. Manual `?v=X.Y.Z` bumps are gone.

For dev, set `ARABICA_DEV=1` (already on in the `just run` recipe). Each
subsystem then re-reads its source directory on every request, so editing a CSS
or JS file and refreshing picks up the change without a server restart. There is
no separate build step beyond `go run` / `just run`.

`static/service-worker.js` stays a regular static file (the browser handles its
own update lifecycle). Fonts and images under `static/` are also still served by
the plain FileServer — they don't change often enough to warrant the
asset-handler treatment.

## Testing Conventions

All tests MUST use testify:

```go
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.Contains(t, haystack, needle)
assert.True(t, value)
```

Prefer table driven tests.

## Using Go Tooling

- `go mod download -json MODULE` — get dependency source path
- `go doc foo.Bar` — read package/type/function docs
- `go run ./cmd/arabica` instead of `go build` to avoid artifacts

## Design Context

See `DESIGN.md` and `PRODUCT.md` for the full design system reference.
