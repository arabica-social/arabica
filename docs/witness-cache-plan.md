# Witness Cache Implementation Plan

**Goal:** Eliminate redundant PDS requests by using the firehose SQLite index as
a local read cache for Arabica records.

**Date:** 2026-03-22

---

## Problem

Several pages make excessive PDS XRPC calls per page load:

| Page      | PDS Calls | Breakdown                                                                     |
| --------- | --------- | ----------------------------------------------------------------------------- |
| Brew View | 5-7       | 1 brew + 4-5 sequential ref resolves (bean, roaster, grinder, brewer, recipe) |
| Profile   | 6-7       | 5 listRecords (all collections) + handle resolve + profile fetch              |
| Manage    | 5+        | listRecords for all entity types                                              |
| Brew Form | 5+        | All entities for select dropdowns                                             |

The in-memory `SessionCache` (2min TTL, per-session) helps on rapid
re-navigation but is volatile — server restarts and new sessions trigger full
PDS re-fetches.

## Architecture

Three-tier read path:

```
Request → L1: SessionCache (in-memory, 2min TTL, parsed Go structs)
            ↓ miss
          L2: WitnessCache (firehose SQLite, updated in real-time via relay)
            ↓ miss
          L3: PDS (remote XRPC calls)
```

The firehose index already stores every Arabica record it sees from the relay.
We expose it as a read-only `WitnessCache` interface and inject it into
`AtprotoStore`.

Writes always go to PDS. The firehose keeps the witness cache consistent
asynchronously (typically <1s propagation delay).

## Design Decisions

### Use firehose index directly, not a separate SQLite cache

The firehose index already has all the data. A dedicated cache would duplicate
storage and require its own invalidation logic. The firehose's relay
subscription handles consistency for free.

### Resolve references from witness cache too

The WIP branch's biggest flaw: after fetching a brew from the witness cache, it
still called `ResolveBrewRefs` which made 4-5 individual PDS calls for the
referenced bean/grinder/brewer/recipe. All of those records are also in the
firehose index.

A single brew view should go from 5-7 PDS calls to 0 when the witness cache is
warm.

### Keep SessionCache as L1

SessionCache holds parsed Go structs — zero deserialization cost on hit. The
witness cache (L2) requires JSON → struct conversion but avoids network. Both
layers serve different performance profiles.

### Don't cache profiles/handles in witness cache

Profiles come from the AppView/PLC directory, not Arabica lexicons. The existing
`ARABICA_PROFILE_CACHE_TTL` (1h) handles those. The witness cache only covers
Arabica record collections.

### Fall through to PDS after writes

When a user creates/updates/deletes, the SessionCache is invalidated (already
happens today). The next read falls through to PDS since the firehose may not
have propagated the change yet. This preserves read-your-writes consistency
without any new invalidation logic.

---

## Tasks

### Task 1: WitnessCache interface and FeedIndex implementation

**Files:**

- Create: `internal/atproto/witness.go`
- Modify: `internal/firehose/index.go`

Define the interface in `atproto` (avoids import cycle with `firehose`):

```go
// internal/atproto/witness.go
package atproto

type WitnessRecord struct {
    URI        string
    DID        string
    Collection string
    RKey       string
    CID        string
    Record     json.RawMessage
    IndexedAt  time.Time
    CreatedAt  time.Time
}

type WitnessCache interface {
    // Returns (nil, nil) when not found.
    GetWitnessRecord(ctx context.Context, uri string) (*WitnessRecord, error)
    // Returns empty slice when none found.
    ListWitnessRecords(ctx context.Context, did, collection string) ([]*WitnessRecord, error)
}
```

Add composite index to firehose schema for efficient DID+collection queries:

```sql
CREATE INDEX IF NOT EXISTS idx_records_did_coll ON records(did, collection, created_at DESC);
```

Implement `GetWitnessRecord` and `ListWitnessRecords` on `FeedIndex`, with
compile-time interface check:

```go
var _ atproto.WitnessCache = (*FeedIndex)(nil)
```

---

### Task 2: Witness cache helpers in AtprotoStore

**Files:**

- Modify: `internal/atproto/store.go`

Add `witnessCache WitnessCache` field to `AtprotoStore`. Add
`NewAtprotoStoreWithWitness` constructor.

Add two private helpers:

```go
// getFromWitness fetches a single record by collection+rkey.
// Returns nil when cache is not configured or record not found.
func (s *AtprotoStore) getFromWitness(ctx context.Context, collection, rkey string) *WitnessRecord

// listFromWitness returns all cached records for a collection.
// Returns nil when cache is not configured or empty.
func (s *AtprotoStore) listFromWitness(ctx context.Context, collection string) []*WitnessRecord
```

Add `witnessRecordToMap` for converting `json.RawMessage` into the
`map[string]interface{}` format the existing `Record*` conversion functions
expect.

---

### Task 3: Cache-first List operations

**Files:**

- Modify: `internal/atproto/store.go`

For each `List*` method (ListBeans, ListRoasters, ListGrinders, ListBrewers,
ListRecipes, ListBrews):

1. Check SessionCache (unchanged)
2. Try `listFromWitness` — if records found, convert and populate SessionCache
3. Fall through to PDS on miss

The read path becomes:

```go
func (s *AtprotoStore) ListBeans(ctx context.Context) ([]*models.Bean, error) {
    // L1: session cache
    if cached := s.cache.Get(s.sessionID); cached != nil && cached.Beans != nil && cached.IsValid() {
        return cached.Beans, nil
    }
    // L2: witness cache
    if wRecords := s.listFromWitness(ctx, NSIDBean); wRecords != nil {
        beans := convertWitnessRecordsToBeans(wRecords)
        s.cache.SetBeans(s.sessionID, beans)
        return beans, nil
    }
    // L3: PDS (existing code, unchanged)
    ...
}
```

---

### Task 4: Cache-first Get operations with reference resolution from witness

**Files:**

- Modify: `internal/atproto/store.go`

This is the highest-impact change. For `GetBrewByRKey`:

1. Get brew from witness cache
2. Get each reference (bean, grinder, brewer, recipe) from witness cache too
3. Only fall back to PDS if any witness lookup misses

```go
func (s *AtprotoStore) GetBrewByRKey(ctx context.Context, rkey string) (*models.Brew, error) {
    if wr := s.getFromWitness(ctx, NSIDBrew, rkey); wr != nil {
        brew := convertWitnessRecordToBrew(wr)

        // Resolve refs from witness cache — NOT from PDS
        if beanURI != "" {
            if beanWR, _ := s.witnessCache.GetWitnessRecord(ctx, beanURI); beanWR != nil {
                brew.Bean = convertWitnessRecordToBean(beanWR)
                // Resolve roaster ref from witness too
                if roasterURI != "" {
                    if roasterWR, _ := s.witnessCache.GetWitnessRecord(ctx, roasterURI); roasterWR != nil {
                        brew.Bean.Roaster = convertWitnessRecordToRoaster(roasterWR)
                    }
                }
            }
        }
        // Same for grinder, brewer, recipe...

        return brew, nil
    }

    // PDS fallback (existing code)
    ...
}
```

**Impact:** Brew view page goes from 5-7 PDS calls to 0.

Apply the same witness-first reference resolution to `GetBeanByRKey` (resolves
roaster ref) and any other `Get*ByRKey` that resolves references.

---

### Task 5: Witness cache for public/profile reads

**Files:**

- Modify: `internal/handlers/profile.go`
- Modify: `internal/handlers/brew.go` (public brew view path)
- Modify: `internal/handlers/entity_views.go` (public entity view paths)

The profile page and shared view pages use `PublicClient` which bypasses both
SessionCache and WitnessCache entirely. The firehose index has these records for
all known users.

Give handlers access to the witness cache for public reads:

```go
func (h *Handler) HandleProfilePage(w http.ResponseWriter, r *http.Request) {
    // Try witness cache for the target user's records
    if beans := h.listFromWitnessPublic(ctx, targetDID, NSIDBean); beans != nil {
        // Use cached data, skip PDS calls
    }
}
```

Or create a `PublicWitnessStore` wrapper:

```go
type PublicWitnessStore struct {
    witness      WitnessCache
    publicClient *PublicClient  // fallback
}
```

This could be a follow-up if scoping is a concern — the authenticated path
(Tasks 3-4) covers the most common case.

---

### Task 6: Metrics

**Files:**

- Modify: `internal/metrics/metrics.go`

Add Prometheus counters:

```go
var (
    WitnessCacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "arabica_witness_cache_hits_total",
        Help: "Total witness cache hits (PDS request avoided)",
    }, []string{"collection"})

    WitnessCacheMissesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "arabica_witness_cache_misses_total",
        Help: "Total witness cache misses (fell back to PDS)",
    }, []string{"collection"})
)
```

Instrument every witness cache check point in store.go. This is critical for
validating hit rates and identifying remaining cold-start gaps.

---

### Task 7: Wire into main.go and handlers

**Files:**

- Modify: `internal/handlers/handlers.go`
- Modify: `cmd/server/main.go`

Add `witnessCache atproto.WitnessCache` field to `Handler`. Add
`SetWitnessCache` method.

In `getAtprotoStore`, use `NewAtprotoStoreWithWitness` when witness cache is
configured:

```go
func (h *Handler) getAtprotoStore(r *http.Request) (database.Store, bool) {
    ...
    if h.witnessCache != nil {
        return atproto.NewAtprotoStoreWithWitness(client, did, sessionID, h.sessionCache, h.witnessCache), true
    }
    return atproto.NewAtprotoStore(client, did, sessionID, h.sessionCache), true
}
```

In main.go, wire `feedIndex` as the witness cache after it's initialized.

---

### Task 8: Public feed cache invalidation

**Files:**

- Modify: `internal/feed/service.go`
- Modify: `internal/handlers/brew.go`, `internal/handlers/entities.go`

Add `InvalidatePublicFeedCache()` method to `feed.Service`. Call it from
create/update/delete handlers so unauthenticated feed views reflect changes
immediately.

This was included in the WIP and is a good standalone improvement regardless of
the witness cache.

---

## Non-Goals

- **Profile/handle caching** — handled by existing `ARABICA_PROFILE_CACHE_TTL`
- **CID-based staleness detection** — future optimization, not needed for v1
- **Separate SQLite cache database** — firehose index already has the data
- **Increasing SessionCache TTL** — 2 minutes is correct for multi-device sync

## Risks and Mitigations

| Risk                                                   | Mitigation                                                                |
| ------------------------------------------------------ | ------------------------------------------------------------------------- |
| Firehose lag causes stale reads                        | PDS fallback on SessionCache invalidation after writes                    |
| New user before firehose backfill                      | Witness miss → PDS fallback (transparent)                                 |
| Firehose down                                          | Witness returns empty → PDS fallback (existing behavior)                  |
| JSON parsing differences between witness and PDS paths | Same `Record*` conversion functions, same `map[string]interface{}` format |
| Schema changes to firehose records table               | WitnessCache interface isolates AtprotoStore from schema details          |

## Expected Impact

| Page      | Before        | After (warm cache) |
| --------- | ------------- | ------------------ |
| Brew View | 5-7 PDS calls | 0                  |
| Profile   | 6-7 PDS calls | 0 (with Task 5)    |
| Manage    | 5+ PDS calls  | 0                  |
| Brew Form | 5+ PDS calls  | 0                  |

Server restarts no longer cause a thundering herd of PDS requests — the witness
cache survives restarts since it's backed by SQLite on disk.
