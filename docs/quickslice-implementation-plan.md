# Quickslice Integration Plan

Quickslice is a self-hostable AppView framework for AT Protocol. It connects to Jetstream,
indexes records matching your lexicons into SQLite/Postgres, and exposes a GraphQL API with
built-in joins across record types.

**Goal:** Reduce PDS fetches for reads by querying a local quickslice instance, keeping the
existing PDS path as a fallback.

---

## Value Proposition

### The Problem With PDS-First Architecture

Arabica's current architecture makes reads directly against users' Personal Data Servers.
This is correct for writes (AT Protocol mandates it), but for reads it has significant costs:

**Sequential, unbatched fetches.** When you view a brew, the app makes individual `getRecord`
calls — one for the brew, then one for the bean, then one for the roaster nested inside that
bean, then one for the grinder, then one for the brewer. These are sequential because each
reference is resolved after the previous record arrives. That's 4-5 round trips per brew
view, each one a network call to an external PDS.

**N+1 on the community feed.** The feed currently polls registered users' PDS instances
for their brews. Resolving each brew's references for display (bean name, roaster, grinder)
requires additional PDS calls per brew card. As the user base grows, this becomes
expensive fast.

**Cross-user reads have no cache.** The 5-minute session cache covers own-user data well.
But when viewing another user's public brew, there's no caching — every page load hits
their PDS for 4-6 calls.

**PDS polling for the feed.** The feed service periodically polls each registered user's
PDS. This doesn't scale well and is the reason the backfill system exists. Jetstream
already has all this data in a push model.

### What Quickslice Provides

Quickslice acts as a **local read index** — it subscribes to Jetstream and maintains a
local SQLite/Postgres copy of all arabica records from across the network. Because the data
is local, reads are fast and joined queries are cheap.

The key feature is **automatic join generation from lexicons**. Because arabica's lexicons
define typed references between records (`beanRef`, `grinderRef`, `brewerRef`), quickslice
generates GraphQL joins that follow those references in a single query. What currently
takes 4-5 sequential PDS calls becomes one local query.

```
Current:  brew → PDS  →  bean → PDS  →  roaster → PDS  →  grinder → PDS  →  brewer → PDS
          (5 sequential network calls, each blocked on the previous)

With QS:  ──────────────────► quickslice (1 local query, all joins in one response) ◄──────
```

### Where This Matters Most

**Public brew views (highest value).** Viewing another user's brew currently requires 4-6
calls to their PDS via `public_client`. There's no caching. Quickslice replaces this with
a single local query, and the result could be cached locally. This is the biggest latency
improvement for the most visible user-facing operation.

**Community feed enrichment.** Each brew card in the feed shows bean name, roaster, and
equipment. Currently this requires fetching that data per-brew. A single quickslice query
can return feed brews with all references resolved, replacing N×5 PDS calls with 1 query.

**Reference resolution after brew creation.** When you create a brew, the app immediately
resolves its references to populate the full model (for rendering the response). That's
2-4 extra PDS calls. Quickslice can't help immediately after a write (Jetstream lag), but
the references themselves (bean, grinder, brewer) are already indexed — so a targeted
query for those specific records is instant.

**Own-user reads on cold cache.** The session cache (5-minute TTL) covers most own-user
list operations after the first load. But on first load or after cache expiry, `ListBrews`
makes 5 PDS calls. Quickslice serves this from local storage instead.

**Future social features.** Likes and comments will need cross-user aggregation — "how many
people liked this brew?" requires querying many users' PDS instances. With quickslice, this
becomes a reverse-join query (find all `social.arabica.alpha.like` records pointing at a
given brew URI), served locally.

### What Quickslice Does Not Fix

- **Writes** always go to the user's PDS. OAuth, DPOP, and mutations are unaffected.
- **Freshness after writes**: Jetstream has seconds of lag. A record you just created
  won't be in quickslice immediately. The existing PDS path must remain for post-write reads.
- **Auth-gated data**: Quickslice indexes public records only (AT Protocol records are
  public by default, so this is fine for arabica).
- **Scale ceiling**: Quickslice is early-stage (v0.20.x). At very high scale, you'd
  eventually need a more mature indexing pipeline. But for arabica's current scale, SQLite
  is more than sufficient.

---

## Current Pain Points

| Operation | Current PDS calls |
|-----------|:-:|
| `GetBrewByRKey` | 1 (brew) + 1 (bean) + 1 (roaster) + 1 (grinder) + 1 (brewer) = **up to 5** |
| `ListBrews` (cold cache) | 1 (brews) + 1 (beans) + 1 (roasters) + 1 (grinders) + 1 (brewers) = **5** |
| Public brew view (cross-user) | 1 (brew) + 1 (bean) + 1 (roaster) + 1 (grinder) + 1 (brewer) = **up to 5** |
| Community feed brew cards | 5 calls × N brews (N+1 problem on ref resolution) |

With quickslice, each of the above becomes **1 GraphQL query** with forward joins.

---

## Architecture

```
arabica server
  ├─ Writes  ──────────────────────────────► User's PDS (unchanged)
  └─ Reads ──► quickslice GraphQL client ──► quickslice HTTP API
                    │ (on error/miss)
                    └──────────────────────► existing PDS path (fallback)

quickslice service
  ├─ Subscribes to Jetstream (firehose)
  ├─ Indexes arabica lexicon records → SQLite or Postgres
  └─ Exposes GraphQL API at :8080/graphql (no auth required for reads)
```

Quickslice is a **read-only index**. All writes (create/update/delete) continue to go to the
user's PDS. Quickslice self-populates via Jetstream, so no backfill code changes are needed.

---

## Constraints and Caveats

- **Eventual consistency**: Jetstream has ~seconds of lag. A record just written to the PDS
  may not appear in quickslice immediately. After any write operation, skip quickslice and
  read directly from PDS for that request.
- **Early-stage software**: Quickslice is v0.20.x, "APIs may change without notice". Wrap it
  behind an interface so swapping it out is painless.
- **Gleam runtime**: Quickslice is written in Gleam. Debugging upstream issues requires
  learning a new language — treat it as a black box.
- **Writes stay on PDS**: OAuth sessions, DPOP tokens, and all mutations are unaffected.

---

## Phase 1: Infrastructure

**Goal:** Get quickslice running and indexing arabica records.

### 1a. Add quickslice to docker-compose

Add a `quickslice` service to the project's docker-compose (or nix deployment config):

```yaml
quickslice:
  image: ghcr.io/slices-network/quickslice:latest
  environment:
    DATABASE_URL: ""           # empty = use SQLite
    LEXICON_DIR: /lexicons
    JETSTREAM_URL: wss://jetstream2.us-east.bsky.network/subscribe
  volumes:
    - ./lexicons:/lexicons:ro
    - quickslice-data:/data
  ports:
    - "8080:8080"
```

Point `LEXICON_DIR` at the repo's existing `lexicons/` directory. Quickslice reads lexicon
JSON files and auto-generates its GraphQL schema from them.

### 1b. Verify indexing

After startup, check the GraphQL playground at `http://localhost:8080` and run a sample
query for `social.arabica.alpha.brew` records. Confirm that records from the firehose appear
with the expected fields.

### 1c. Add environment variable to arabica

```
QUICKSLICE_URL=http://localhost:8080/graphql
```

When unset or empty, quickslice integration is disabled and arabica falls back to PDS-only
mode. This makes it safe to deploy without quickslice running.

---

## Phase 2: Go GraphQL Client

**Goal:** Write a minimal quickslice client in Go. No codegen — just HTTP + JSON.

### File: `internal/quickslice/client.go`

```go
type Client struct {
    endpoint   string
    httpClient *http.Client
}

func New(endpoint string) *Client { ... }

// Returns nil, ErrNotFound if the record isn't indexed yet (freshness gap after writes).
func (c *Client) GetBrew(ctx context.Context, did, rkey string) (*models.Brew, error) { ... }
func (c *Client) ListBrews(ctx context.Context, did string) ([]*models.Brew, error) { ... }
func (c *Client) GetBeanWithRoaster(ctx context.Context, did, rkey string) (*models.Bean, error) { ... }

// helpers
func (c *Client) query(ctx context.Context, q string, vars map[string]any, out any) error { ... }
```

GraphQL queries use forward joins to resolve references in a single round-trip. Example for
`GetBrew`:

```graphql
query GetBrew($did: String!, $rkey: String!) {
  socialArabicaAlphaBrew(where: { did: { eq: $did }, rkey: { eq: $rkey } }, first: 1) {
    edges {
      node {
        uri cid did rkey createdAt
        # forward join: beanRef → bean record
        beanRefResolved {
          ... on SocialArabicaAlphaBeanRecord {
            uri rkey name origin process roastLevel
            # nested: roasterRef → roaster record
            roasterRefResolved {
              ... on SocialArabicaAlphaRoasterRecord { uri rkey name location }
            }
          }
        }
        grinderRefResolved {
          ... on SocialArabicaAlphaGrinderRecord { uri rkey brand model burrType }
        }
        brewerRefResolved {
          ... on SocialArabicaAlphaBrewerRecord { uri rkey brand model type }
        }
      }
    }
  }
}
```

**Note:** The exact GraphQL field names are generated from lexicon IDs. Verify them against
the quickslice playground after Phase 1 is complete.

---

## Phase 3: Wire Into AtprotoStore

**Goal:** Add quickslice as an optional fast path in `AtprotoStore`, falling back to existing
PDS code on any error.

### 3a. Add field to AtprotoStore

```go
// internal/atproto/store.go
type AtprotoStore struct {
    client     *Client
    quickslice *quickslice.Client // nil if QUICKSLICE_URL is unset
    did        string
    sessionID  string
    cache      *SessionCache
}
```

Wire `quickslice.Client` in during store construction (in `handlers.go` or wherever
`AtprotoStore` is instantiated).

### 3b. Wrap Get* methods

Pattern for all Get* methods:

```go
func (s *AtprotoStore) GetBrewByRKey(ctx context.Context, rkey string) (*models.Brew, error) {
    if s.quickslice != nil {
        brew, err := s.quickslice.GetBrew(ctx, s.did, rkey)
        if err == nil {
            return brew, nil
        }
        // log at debug level, fall through
    }
    // existing implementation unchanged below
    ...
}
```

Do **not** wrap immediately after a write. The handler already has the created record in
hand at that point — no PDS read needed.

### 3c. Wrap ListBrews (cold-cache path only)

`ListBrews` already checks `SessionCache` first. The quickslice path sits between cache miss
and PDS:

```go
func (s *AtprotoStore) ListBrews(ctx context.Context, userID string) ([]*models.Brew, error) {
    if brews := s.cache.GetBrews(s.sessionID); brews != nil {
        return brews, nil
    }
    if s.quickslice != nil {
        brews, err := s.quickslice.ListBrews(ctx, s.did)
        if err == nil {
            s.cache.SetBrews(s.sessionID, brews)
            return brews, nil
        }
    }
    // existing PDS implementation unchanged
    ...
}
```

Apply the same pattern to `ListBeans`, `ListRoasters`, `ListGrinders`, `ListBrewers`.

---

## Phase 4: Cross-User Reads (Highest Value)

**Goal:** Replace the `resolveBrewReferences` function in `internal/handlers/brew.go` that
handles public brew view (other users' brews). This is the highest-value target because
there is no session cache here — every page view currently makes 4-6 PDS calls to
`public_client`.

### Current flow (brew.go ~line 313)

```go
func resolveBrewReferences(ctx context.Context, client PublicClient, brew *models.Brew) error {
    // Individual GetRecord calls for bean, roaster, grinder, brewer
}
```

### New flow

```go
func (h *Handler) resolveBrewReferences(ctx context.Context, brew *models.Brew) error {
    if h.quickslice != nil {
        resolved, err := h.quickslice.GetBrew(ctx, brew.AuthorDID, brew.RKey)
        if err == nil {
            *brew = *resolved // copy resolved refs onto brew
            return nil
        }
    }
    // existing public_client fallback
}
```

This also benefits the community feed — brew cards in the feed can be enriched with bean
names and roaster info in a single batch query instead of N×5 PDS calls.

---

## Phase 5: Community Feed (Optional, Later)

The feed service (`internal/feed/service.go`) polls registered users' PDS for brews.
Quickslice already indexes all arabica records from the firehose network-wide.

Long-term, the feed query could become a single GraphQL query with sorting and pagination:

```graphql
query CommunityFeed($limit: Int!, $after: String) {
  socialArabicaAlphaBrew(
    first: $limit, after: $after
    sortBy: [{ field: "createdAt", direction: DESC }]
  ) {
    edges {
      node {
        uri did actorHandle createdAt
        beanRefResolved { ... }
      }
      cursor
    }
    pageInfo { hasNextPage endCursor }
  }
}
```

This replaces the polling-based aggregation entirely. Hold off until the quickslice API
stabilizes (it's still early-stage).

---

## Implementation Order

1. **Phase 1** — Infrastructure (quickslice running, lexicons loaded, env var wired)
2. **Phase 2** — Go client with `GetBrew` and verify joins work against the live instance
3. **Phase 4** — Cross-user reads (public brew view) — highest value, no auth complexity
4. **Phase 3** — Own-user Get* reads with fallback
5. **Phase 3c** — List* reads (lower priority — session cache already covers most of this)
6. **Phase 5** — Feed service replacement (deferred until API stabilizes)

---

## Testing Strategy

- Unit test the quickslice client with a mock HTTP server returning known GraphQL responses
- Integration test: start quickslice in docker, write a record to a test PDS, wait for
  indexing, query via client
- Smoke test fallback: with `QUICKSLICE_URL` unset, all existing behavior must be unchanged
- After any write, assert the next read uses the PDS path (not quickslice) to avoid
  stale-data bugs
