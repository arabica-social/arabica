# AT Protocol Package Analysis

> Analysis of `internal/atproto/` — what to upstream to
> `tangled.org/pdewey.com/atp` and what to extract to its own package.

## Context

As arabica adds a second app (oolong tea tracker), the `internal/atproto/`
package has accumulated both general-purpose AT Protocol boilerplate and
arabica-specific business logic. This creates a problem: oolong needs the
protocol layer but not the coffee domain code. The shared `atp` library exists
to hold the general-purpose layer.

## Current Files in `internal/atproto/`

| File               | Lines | Purpose                                                       | Status     |
| ------------------ | ----- | ------------------------------------------------------------- | ---------- |
| `client.go`        | 308   | OTel/metrics/UA wrapping of `atp.Client` + I/O type wrappers  | TODO       |
| `oauth.go`         | 360   | OAuth flow management, auth middleware, context helpers       | TODO       |
| `resolver.go`      | 155   | Record resolution engine (`resolveRef` generic) + AT-URI parsing wrappers + entity-specific resolvers | TODO       |
| `public_client.go` | ~70   | Thin OTel/UA wrapper around `atp.PublicClient`                | Slimmed ✅ |
| `store.go`         | 1250+ | `AtprotoStore` implementing `database.Store` for all entities | TODO       |
| `store_generic.go` | 170   | Generic fetch/put/delete helpers used by store.go             | TODO       |
| `cache.go`         | 226   | Copy-on-write per-session cache with typed entity accessors   | TODO       |
| `witness.go`       | 62    | `WitnessCache` interface + `WitnessRecord` type               | TODO       |

**Already deleted**: `handle.go`, `handle_test.go`, `nsid_test.go`, `oauth.go`

## What's in `atp` now

- **`atp.Client`** — PDS CRUD (`CreateRecord`, `GetRecord`, `ListRecords`,
  `PutRecord`, `DeleteRecord`, `UploadBlob`, `GetBlob`)
- **`atp.OAuthApp`** — OAuth flow management (`StartLogin`, `HandleCallback`,
  `LoginCLI`, `ResumeSession`, `Logout`, `DeleteSession`, `ClientMetadata`) **+
  `StartSignup`** ✅
- **`atp.PublicClient`** — unauthenticated reads (`ResolveHandle`,
  `GetPDSEndpoint`, `GetProfile`, `ListPublicRecords`, `GetPublicRecord`) **+
  caching, `InvalidateHandle`, `InvalidateDID`, `ListAllRecords`** ✅
- **`atp.uri`** — `atp.BuildATURI`, `ParseATURI`, `RKeyFromURI` **+
  `NormalizeHandle`, `DisplayHandle`, `ValidateRKey`** ✅
- **`atp.URI`** struct — parsed AT-URI components (DID, Collection, RKey) with `String()` method
- **`atp.ResolveRecord[T]`** — generic typed record fetch from AT-URI ✅
- **`atp/middleware`** — `CookieAuth` middleware, `ClientMetadataHandler`,
  context getters
- **`atp/errors`** — `ErrSessionExpired`, `WrapPDSError`
- **`atp/scopes`** — `ScopesForCollections`
- **`atp/jetstream`** — Jetstream consumer
- **`atp/store/bolt`**, **`atp/store/sqlite`** — persistent OAuth session stores
- **`atp/tracing`** — OTel span helpers

## Completed — Phase A ✅

| Item                                              | Details                                                                                                              |
| ------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `NormalizeHandle`, `DisplayHandle` → `atp/uri.go` | Deleted `handle.go`, updated all `.templ` + `.go` callers                                                            |
| `ValidateRKey` → `atp/uri.go`                     | Updated `handlers/handlers.go`, `handlers/brew.go` callers                                                           |
| `PublicClient` caching → `atp/public.go`          | TTL-based handle→DID and DID→PDS caches, `InvalidateHandle`, `InvalidateDID`, `ListAllRecords`                       |
| `StartSignup` → `atp/oauth.go`                    | `prompt=create` PAR flow as `OAuthApp.StartSignup`                                                                   |
| `public_client.go` slimmed                        | Now ~70 lines, embeds `*atp.PublicClient`, keeps backward-compat `ListRecords`/`GetRecord`/`ListAllRecords` wrappers |

## Completed — Phase B ✅

| Item                                                                | Details                                                                                                 |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `OAuthManager` → `atp.OAuthApp`                                     | Handler struct + constructor switched; `InitiateLogin` → `StartLogin`, `InitiateSignup` → `StartSignup` |
| `AuthMiddleware` → `atp/middleware.CookieAuth`                      | Routing middleware replaced; OnAuth callback configurable via routing.Config                            |
| `GetAuthenticatedDID` → `atp/middleware.GetDID`                     | All 46+ callers in handlers, middleware, and tests updated (return changed from `error` to `bool`)      |
| `GetSessionIDFromContext` → `atp/middleware.GetSessionID`           | All callers updated (changed from `error` to `bool` return)                                             |
| `ContextWithAuthDID` / `ContextWithAuth` → raw context keys         | Test helpers updated to use `"atp_did"` / `"atp_session_id"` context keys                               |
| `ParseDID` → `syntax.ParseDID`                                      | Arabica wrapper deleted; direct indigo `syntax.ParseDID` used instead                                   |
| `SetOnAuthSuccess` → `CookieAuthConfig.OnAuth`                      | On-auth callback moved from OAuthManager setter to middleware config                                    |
| `NewOAuthManager` → `atp.NewOAuthApp`                               | Server bootstrap + integration harness updated                                                          |
| `NewClient(oauth *OAuthManager)` → `NewClient(oauth *atp.OAuthApp)` | Client constructor type changed; internal provider uses `oauth.ResumeSession`                           |
| **Deleted files**                                                   | `internal/atproto/oauth.go` (~360 lines)                                                                |

**Impact**: ~360 lines deleted, one less OAuth implementation to maintain.

---

## Remaining Work

### Phase C — Upstream `resolveRef` → `atp.ResolveRecord`

`resolveRef[T]` in `resolver.go` is a generic helper that parses an AT-URI,
validates the collection matches the expected NSID, fetches the record from the
PDS, and converts the raw `map[string]any` to a typed struct. It is the engine
behind all five entity-specific resolvers (`ResolveRoasterRef`,
`ResolveGrinderRef`, `ResolveBrewerRef`, `ResolveRecipeRef`) and is partially
inlined in `ResolveBeanRefWithRoaster` (which also resolves the nested roaster
reference).

This pattern — AT-URI → validate collection → fetch → convert — is
universal to any AT Protocol app. Adding a generic to `atp` avoids duplicating
it in oolong.

**Added to `atp/uri.go`**:

```go
type URI struct {
    DID        string
    Collection string
    RKey       string
}

func (u *URI) String() string
func ParseATURI(uri string) (*URI, error)
func RKeyFromURI(uri string) string  // unchanged signature
```

`ParseATURI` now returns a `*URI` struct instead of 4 strings, eliminating the
need for arabica's `ATURIComponents` wrapper.

**Added to `atp/record.go`**:

```go
type RecordFetcher interface {
    GetRecord(ctx context.Context, collection, rkey string) (*Record, error)
}

func ResolveRecord[T any](
    ctx context.Context,
    fetcher RecordFetcher,
    atURI string,
    expectedCollection string,
    convert func(value map[string]any, uri string) (*T, error),
) (*T, error)
```

Key changes from the arabica version:

- `fetcher RecordFetcher` (satisfied by `*atp.Client`) instead of arabica's
  `*Client` + `sessionID string` pass-through
- Uses `fetcher.GetRecord(collection, rkey)` directly — no `GetRecordInput`
  wrapper
- Uses `ParseATURI` returning `*URI` instead of `ResolveATURI` → `ATURIComponents`

**In arabica**:

1. Entity resolvers updated to call `atp.ResolveRecord` via an `*atp.Client`
   obtained from `client.AtpClient(ctx, did, sessionID)`.
2. `resolveRef` deleted from `resolver.go`.
3. `ResolveBeanRefWithRoaster` now uses `atpClient.GetRecord` + `RecordToBean`
   (needs raw `roasterRef` from the record value before conversion).
4. `ATURIComponents` / `ResolveATURI` deleted — all ~50 call sites cut over
   to `atp.RKeyFromURI` (22) or `atp.ParseATURI` (4).

### Phase D — Move arabica resolvers to `entities/arabica`

The remaining entity resolvers (`ResolveBeanRefWithRoaster`,
`ResolveRoasterRef`, `ResolveGrinderRef`, `ResolveBrewerRef`,
`ResolveRecipeRef`, `ResolveBrewRefs`) are arabica-specific: they hardcode
`arabica.Bean`, `arabica.NSIDRoaster`, `arabica.RecordToBean`, etc. After
Phase C they are thin callers of `atp.ResolveRecord`. They belong next to
the typed models and `RecordToX` converters they wrap
(`internal/entities/arabica/records.go`), not in the store/cache
infrastructure layer. Putting them under `internal/store/atproto/` would
either fork that package per app or force it to import every app's entities
— both are wrong. Co-locating with the typed models is symmetric: oolong
gets `entities/oolong/resolve.go` for free.

**Prerequisite**: Phase C done. The resolvers should take an `*atp.Client`
(or the `RecordFetcher` interface) directly instead of the OTel-wrapped
`*atproto.Client`. Either flatten the wrapper first (Phase E step 3) or just
change the resolver signatures as part of this phase — `*atp.Client` is
already DID-scoped, so the `sessionID` parameter falls away.

1. **Create `internal/entities/arabica/resolve.go`** with `ResolveBean`,
   `ResolveRoaster`, `ResolveGrinder`, `ResolveBrewer`, `ResolveRecipe`,
   `ResolveBrewRefs`. Each takes `(ctx, *atp.Client, atURI)` and calls
   `atp.ResolveRecord(ctx, client, atURI, NSIDx, RecordToX)`. `ResolveBean`
   keeps the nested-roaster logic (raw record fetch → `RecordToBean` →
   delegate roaster ref to `ResolveRoaster`).

2. **Update `internal/atproto/store.go` callers** (the brew read fan-out for
   bean/grinder/brewer, recipe attachment on brew, brewer attachment on
   recipe, roaster attachment on cafe — ~6 call sites). Calls become
   `arabica.ResolveBean(ctx, atpClient, beanRef)` etc.; drop the `sessionID`
   argument.

3. **Delete the resolver functions from `internal/atproto/resolver.go`**.
   Combined with Phase C's removal of `resolveRef` and `ATURIComponents`,
   `resolver.go` is now empty and is deleted in Phase E.

4. **Store/cache/witness package extraction is deferred and out of scope
   here.** With resolvers gone, `store.go`/`store_generic.go`/`cache.go`/
   `witness.go` can later move to `internal/store/atproto/` as app-agnostic
   infrastructure, but the per-entity store methods living there today are a
   separate multitenancy problem (see `docs/tea-multitenant-refactor.md`).
   Phase D does not block on that move and is not a prerequisite for it.

### Phase E — Cleanup

Remove remaining duplication and thin abstractions:

1. ~~**Delete `ATURIComponents` / `ResolveATURI`**~~ ✅ — done in Phase C.
   22 callers switched to `atp.RKeyFromURI`, 4 switched to `atp.ParseATURI`
   (returning `*atp.URI`).

2. ~~**Delete `resolveRef`**~~ ✅ — done in Phase C.

3. **Delete `client.go` I/O types**: `CreateRecordInput`, `GetRecordOutput`,
   etc. are thin wrappers around `atp.Client` methods. Replace with direct
   `atp.Client` usage. Keep only the UA/metrics constructor wiring.

4. **Delete `public_client.go` entirely**: Once all callers use
   `atp.PublicClient` methods directly, the thin
   `PublicListRecordsOutput`/`PublicRecordEntry` wrappers can go away.

5. **Delete `resolver.go`** — after Phase D (entity resolvers moved) and the
   above cleanup, the file is empty.

### End state

```
arabica/internal/
├── entities/arabica/       ← models, records, resolve.go (entity ref resolvers)
├── atproto/                ← store.go, generic.go, cache.go, witness.go (+ OTel wiring)
├── handlers/
└── ...
```

A later refactor (tracked in `docs/tea-multitenant-refactor.md`) moves the
store/cache/witness layer out to `internal/store/atproto/` once its per-entity
methods are made multitenant. That work is independent of this plan.

---

## Things Kept in Arabica (by Design)

| What                                       | Why                                                             |
| ------------------------------------------ | --------------------------------------------------------------- |
| `WitnessCache` interface + `WitnessRecord` | Opinionated caching contract between store and firehose         |
| `SessionCache` / `UserCache`               | Copy-on-write + dirty tracking tied to three-layer architecture |
| `store_generic.go` helpers                 | Orchestrate witness → PDS → cache, not generic CRUD             |
| `cache.go` typed accessors                 | Arabica entity-specific                                         |
| `userAgentTransport`                       | Application branding                                            |
| `metricLabelFor`                           | Arabica NSIDs → Prometheus labels                               |
