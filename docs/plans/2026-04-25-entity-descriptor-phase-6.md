# Entity Descriptor — Phase 6: Cleanup Pass

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Three targeted cleanups that use the descriptor or remove manually
maintained per-entity maps. No behavior change except PublicClient latency.

**Parent spec:** `docs/entity-descriptor-refactor.md`
**Previous phase:** `docs/plans/2026-04-25-entity-descriptor-phase-5.md`

---

## Scope (revised from original spec)

| Item | Decision | Why |
|---|---|---|
| `entityTypeToNSID` map in suggestions handler | ✅ do it | Descriptor owns this data; the map must be updated when cafe/drink land |
| PublicClient resolver cache | ✅ do it | Small, low-risk, reduces network calls in firehose hot path |
| Modal route loop | ✅ do it (minor) | Removes 5 repetitive `HandleFunc` pairs |
| Dirty-tracking → TTL | ❌ defer | Cache correctness concern; belongs to a future phase 2 |
| Suggestions `entityConfigs` from descriptor | ❌ skip | Dedup functions and field lists are suggestions-specific, not descriptor data |

---

## Task 1: Replace `entityTypeToNSID` with descriptor lookup

**Files:**

- Modify: `internal/handlers/suggestions.go`

**Current:**

```go
var entityTypeToNSID = map[string]string{
    "roasters": atproto.NSIDRoaster,
    "grinders": atproto.NSIDGrinder,
    "brewers":  atproto.NSIDBrewer,
    "beans":    atproto.NSIDBean,
    "recipes":  atproto.NSIDRecipe,
}
```

**After:** Build the map from the descriptor registry so new entities
(cafe, drink) appear automatically when registered.

```go
var entityTypeToNSID = func() map[string]string {
    m := make(map[string]string)
    for _, d := range entities.All() {
        if d.NSID != "" {
            m[d.URLPath] = d.NSID
        }
    }
    return m
}()
```

Remove the `atproto` import from this file if it becomes unused.

## Task 2: Compact modal route registration

**Files:**

- Modify: `internal/routing/routing.go`

**Current:** 10 individual `HandleFunc` calls for 5 entity modals (new + edit each).

**After:** Loop over a compact slice of `{noun, new handler, edit handler}`:

```go
for _, m := range []struct {
    noun string
    new  http.HandlerFunc
    edit http.HandlerFunc
}{
    {"bean", h.HandleBeanModalNew, h.HandleBeanModalEdit},
    {"grinder", h.HandleGrinderModalNew, h.HandleGrinderModalEdit},
    {"brewer", h.HandleBrewerModalNew, h.HandleBrewerModalEdit},
    {"roaster", h.HandleRoasterModalNew, h.HandleRoasterModalEdit},
    {"recipe", h.HandleRecipeModalNew, h.HandleRecipeModalEdit},
} {
    mux.HandleFunc("GET /api/modals/"+m.noun+"/new", m.new)
    mux.HandleFunc("GET /api/modals/"+m.noun+"/{id}", m.edit)
}
```

Note: handler references can't come from the descriptor (they're business
logic, not metadata), so this still has 5 data entries. The win is that
the pattern is explicit and each entity is one line.

## Task 3: Cache PublicClient resolver results

**Files:**

- Modify: `internal/atproto/public_client.go`

**Current:** `GetPDSEndpoint` and `ResolveHandle` make raw network calls
with no caching. In the firehose processing path these can be called
repeatedly for the same DID or handle.

**After:** Add a simple TTL cache (1 hour) using `sync.RWMutex` + a map of
`cachedValue{value string, expiry time.Time}`. Cache miss falls through to
the existing inner client call. No change to the public API.

```go
type cachedValue struct {
    value  string
    expiry time.Time
}

type PublicClient struct {
    inner      *atp.PublicClient
    pdsMu      sync.RWMutex
    pdsCache   map[string]cachedValue // DID → PDS URL
    handleMu   sync.RWMutex
    handleCache map[string]cachedValue // handle → DID
}

const resolverCacheTTL = time.Hour
```

Keep it simple: no explicit invalidation, just TTL expiry. A 1-hour TTL
means stale PDS endpoints are served for at most an hour, which is
acceptable (PDS migrations are rare).

## Task 4: Verify

```bash
go vet ./...
go build ./...
go test ./...
```

Check: suggestions endpoint still returns results for bean/grinder/brewer/
roaster/recipe. Modal routes still open and submit correctly.

---

## Out of scope

- Dirty-tracking → TTL (cache correctness; deferred to phase 2)
- Suggestions `entityConfigs` from descriptor (field lists and dedup
  functions are suggestions-specific; descriptor is the wrong home)
- Any new entity registrations (cafe/drink) — those belong to feature work
