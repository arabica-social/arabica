# AT Protocol Package Analysis

> Analysis of `internal/atproto/` — what to upstream to
> `tangled.org/pdewey.com/atp` and what to extract to its own package.

## Context

As arabica adds a second app (oolong tea tracker), the `internal/atproto/`
package has accumulated both general-purpose AT Protocol boilerplate and
arabica-specific business logic. This creates a problem: oolong needs the
protocol layer but not the coffee domain code. The shared `atp` library exists
to hold the general-purpose layer.

## Files in `internal/atproto/`

| File               | Lines | Purpose                                                       |
| ------------------ | ----- | ------------------------------------------------------------- |
| `nsid.go`          | 40    | RKey validation, proxies `atp.BuildATURI`/`atp.RKeyFromURI`   |
| `client.go`        | 308   | OTel/metrics/UA wrapping of `atp.Client` + I/O type wrappers  |
| `oauth.go`         | 360   | OAuth flow management, auth middleware, context helpers       |
| `handle.go`        | 30    | `NormalizeHandle`, `DisplayHandle` (IDN punycode/unicode)     |
| `resolver.go`      | 155   | AT-URI parsing + entity-specific reference resolvers          |
| `public_client.go` | 210   | Caching wrapper around `atp.PublicClient`                     |
| `store.go`         | 1250+ | `AtprotoStore` implementing `database.Store` for all entities |
| `store_generic.go` | 170   | Generic fetch/put/delete helpers used by store.go             |
| `cache.go`         | 226   | Copy-on-write per-session cache with typed entity accessors   |
| `witness.go`       | 62    | `WitnessCache` interface + `WitnessRecord` type               |

## What's in `atp` already

- **`atp.Client`** — PDS CRUD (`CreateRecord`, `GetRecord`, `ListRecords`,
  `PutRecord`, `DeleteRecord`, `UploadBlob`, `GetBlob`)
- **`atp.OAuthApp`** — OAuth flow management (`StartLogin`, `HandleCallback`,
  `LoginCLI`, `ResumeSession`, `Logout`, `DeleteSession`, `ClientMetadata`)
- **`atp.PublicClient`** — unauthenticated reads (`ResolveHandle`,
  `GetPDSEndpoint`, `GetProfile`, `ListPublicRecords`, `GetPublicRecord`)
- **`atp.uri`** — `BuildATURI`, `ParseATURI`, `RKeyFromURI`
- **`atp/middleware`** — `CookieAuth` middleware, `ClientMetadataHandler`,
  context getters
- **`atp/errors`** — `ErrSessionExpired`, `WrapPDSError`
- **`atp/scopes`** — `ScopesForCollections`
- **`atp/jetstream`** — Jetstream consumer
- **`atp/store/bolt`**, **`atp/store/sqlite`** — persistent OAuth session stores
- **`atp/tracing`** — OTel span helpers

## Recommendations

### Move to `atp`

These have zero arabica dependencies and are generally useful:

| Code                                                                                       | Destination        | Rationale                                                                  |
| ------------------------------------------------------------------------------------------ | ------------------ | -------------------------------------------------------------------------- |
| `NormalizeHandle`, `DisplayHandle`                                                         | `atp`              | Pure IDN utility; handle resolution is an atp concern                      |
| `ValidateRKey`                                                                             | `atp`              | URI/RKey validation alongside `ParseATURI`                                 |
| Caching layer on `PublicClient` (handle→DID, DID→PDS, `InvalidateHandle`, `InvalidateDID`) | `atp.PublicClient` | Generic; firehose consumers need cache invalidation                        |
| `PublicClient.ListAllRecords`, `PublicClient.GetRecord`                                    | `atp.PublicClient` | Convenience methods already in `atp.Client` but missing from public client |

### Keep in arabica (not upstreaming)

These are opinionated design choices — the caching architecture may not be the
best general approach for all atproto apps:

| Code                                                                       | Keep in                                          | Rationale                                                                                                                                                                                                                                                                |
| -------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `WitnessCache` interface + `WitnessRecord`                                 | `internal/atproto/` or `internal/store/atproto/` | Interface defines the contract between store and firehose index in arabica's specific caching strategy. Not clear this is the right pattern for general atproto apps                                                                                                     |
| `SessionCache` / `UserCache`                                               | `internal/atproto/` or `internal/store/atproto/` | Copy-on-write + dirty tracking is coupled to arabica's three-layer cache architecture (session → witness → PDS). Premature to impose on other apps                                                                                                                       |
| `store_generic.go` (fetchRecord, fetchAllRecords, putRecord, removeRecord) | `internal/store/atproto/` (with store.go)        | Not actually generic — every method depends on arabica's caching architecture: witness cache probes, dirty-collection checks, write-through, and session cache invalidation. The pure generic CRUD is already in `atp.Client`; these are the caching-orchestration layer |

### Replace with existing `atp` equivalents

| Arabica code                                     | `atp` replacement                                      | Notes                                              |
| ------------------------------------------------ | ------------------------------------------------------ | -------------------------------------------------- |
| `BuildATURI` proxy                               | `atp.BuildATURI` directly                              | Delete `nsid.go`                                   |
| `ExtractRKeyFromURI` proxy                       | `atp.RKeyFromURI` directly                             |                                                    |
| `ResolveATURI` / `ATURIComponents`               | `atp.ParseATURI`                                       | Slightly different return type (strings vs struct) |
| `OAuthManager`                                   | `atp.OAuthApp`                                         | Nearly identical API                               |
| `AuthMiddleware`                                 | `atp/middleware.CookieAuth`                            | Identical behavior                                 |
| `GetAuthenticatedDID`, `GetSessionIDFromContext` | `atp/middleware.GetDID`, `atp/middleware.GetSessionID` | Different context keys but same pattern            |
| `ContextWithAuthDID`, `ContextWithAuth`          | `atp/middleware` context keys                          | Adapt to atp context keys                          |

### Missing from `atp` — needs upstreaming

Before arabica can fully switch from `OAuthManager` to `atp.OAuthApp`, one
feature needs adding to `atp`:

| Feature                                             | Current home            | Notes                                                                                                                               |
| --------------------------------------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `InitiateSignup(pdsURL)` → "prompt=create" PAR flow | `oauth.go` (~105 lines) | Most of `sendAuthRequestWithPrompt` is copied from indigo SDK with `prompt` field injection. Add to `atp.OAuthApp` as `StartSignup` |

### Extract to arabica-specific packages

These are tightly coupled to arabica entities and don't belong in a
general-purpose library:

| Code                                                                                 | Recommended home                                          | Rationale                                                                                      |
| ------------------------------------------------------------------------------------ | --------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `AtprotoStore` + `store_generic.go`                                                  | `internal/store/atproto/`                                 | Implements `database.Store` for arabica entity types                                           |
| Entity-specific ref resolvers (`ResolveBeanRefWithRoaster`, `ResolveBrewRefs`, etc.) | `internal/store/atproto/resolve.go`                       | Only make sense in context of arabica's entity model                                           |
| `SessionCache` typed accessors (`Beans()`, `Roasters()`, etc.)                       | Remove or inline                                          | Use generic `cache.Get(sessionID).Records[nsid]` instead; generic cache core can move to `atp` |
| `userAgentTransport`                                                                 | Keep in arabica                                           | Application-specific branding                                                                  |
| `metricLabelFor`                                                                     | Keep in store                                             | Maps arabica NSIDs to Prometheus labels                                                        |
| `InitiateSignup` (if not upstreamed)                                                 | `internal/oauth/signup.go` or `internal/handlers/auth.go` | Signup flow specific to arabica                                                                |
| I/O types (`CreateRecordInput`, `GetRecordOutput`, etc.)                             | Remove                                                    | Use `atp.Client` directly                                                                      |

### Keep cache.go in arabica

The `SessionCache` / `UserCache` pattern (copy-on-write, TTL, dirty collection
tracking) is coupled to arabica's three-layer caching architecture. While the
pattern could theoretically be generalized, it's premature to impose this
opinionated design on other `atp` consumers. The typed entity accessors
(`Beans()`, `Roasters()`, etc.) are also arabica-specific.

**Recommendation:** Keep in arabica. When oolong or a third app needs session
caching, extract a shared arabica-internal pattern or re-evaluate upstreaming to
`atp`.

### Why `store_generic.go` stays in arabica

The `fetchRecord`, `fetchAllRecords`, `putRecord`, and `removeRecord` methods
look generic (they work with `map[string]any`), but every method is deeply
coupled to arabica's three-layer architecture:

- **`fetchRecord`** — tries witness cache (`getFromWitness` which checks
  `SessionCache.IsDirty`), then falls back to PDS; tracks `fromWitness` boolean
  so downstream ref resolution can stay in the witness cache
- **`fetchAllRecords`** — tries witness cache (`listFromWitness` which also
  checks dirty collections), then falls back to PDS pagination
- **`putRecord`** — PDS write → `writeThroughWitness` →
  `SessionCache.InvalidateRecords`
- **`removeRecord`** — PDS delete → `deleteFromWitness` →
  `SessionCache.InvalidateRecords`
- **`metricLabelFor`** — maps arabica NSIDs to Prometheus labels

The pure generic CRUD (call PDS, get records back) already lives in
`atp.Client`. These methods are the caching-orchestration layer between the
protocol primitives and arabica's chosen caching strategy. They belong alongside
`store.go`.

### Thin down `client.go`

The CRUD methods in `client.go` are thin delegations that add:

- OTel tracing spans
- Prometheus metrics counters
- zerolog logging
- Custom User-Agent header
- Ergonomic I/O type conversions (pointer optionals)

These are mostly wiring patterns that could be done at the constructor level or
replaced by direct `atp.Client` usage. The I/O types add unnecessary abstraction
over `atp.Client`'s methods.

## Target Architecture

```
tangled.org/pdewey.com/atp              (shared library)
├── client.go                           (unchanged)
├── oauth.go                            (+ StartSignup)
├── public.go                           (+ caching, InvalidateHandle, InvalidateDID, ListAllRecords, GetRecord)
├── uri.go                              (+ NormalizeHandle, DisplayHandle, ValidateRKey)
├── middleware/auth.go                  (unchanged)
├── ...

arabica/internal/
├── store/atproto/                      (NEW - extracted from atproto/)
│   ├── store.go                        AtprotoStore implementing database.Store
│   ├── generic.go                      fetchRecord, fetchAllRecords, putRecord, removeRecord (caching-orchestration layer)
│   ├── resolve.go                      entity-specific ref resolvers
│   ├── cache.go                        SessionCache / UserCache (arabica's caching architecture)
│   ├── witness.go                      WitnessCache interface + WitnessRecord (arabica's contract)
│   └── write.go                        write-through witness helpers
├── atproto/                            (KEPT - but slimmed)
│   ├── client.go                       slimmed to UA/metrics constructor
│   └── public_client.go                slimmed to UA-wrapped constructor
├── handlers/auth.go                    (+ signup if not upstreamed)
├── handlers/entities.go                (imports store/atproto)
└── entity_views.go                     (uses store for view handlers)
```

## Dependency Graph (desired)

```
handlers/entities.go
  └─→ internal/store/atproto/ (arabica Store impl)
         └─→ tangled.org/pdewey.com/atp (protocol layer)
                └─→ indigo (underlying SDK)
  └─→ internal/database/store.go (interface only)
```

No arabica-specific code in `atp`. The `atp` library is pure AT Protocol
boilerplate.

## Migration Strategy

1. **Phase A — Quick wins** (no structural change)
   - Move `handle.go` → `atp`
   - Delete `nsid.go`, update callers to `atp.BuildATURI` / `atp.RKeyFromURI`
   - Add caching + `InvalidateHandle`/`InvalidateDID` to `atp.PublicClient`

2. **Phase B — OAuth switch** (needs `StartSignup` upstreamed first)
   - Add `StartSignup` to `atp.OAuthApp`
   - Replace `OAuthManager` with `atp.OAuthApp`
   - Replace `AuthMiddleware` with `atp/middleware.CookieAuth`
   - Replace context helpers with `atp/middleware.GetDID` / `GetSessionID`
   - Delete `oauth.go`

3. **Phase C — Package extraction**
   - Create `internal/store/atproto/`
   - Move `store.go`, `store_generic.go`, `cache.go`, `witness.go`, entity
     resolver code there
   - Move entity-specific resolver code out of `resolver.go` into
     `internal/store/atproto/resolve.go`
   - Shrink `client.go` to just constructor wiring

4. **Phase D — Cleanup**
   - Delete `resolver.go` (all entity-specific code moved; `ResolveATURI`
     callers use `atp.ParseATURI`)
   - Delete `client.go` I/O types, use `atp.Client` directly
   - Delete `public_client.go` (upstreamed to `atp`)
   - `internal/atproto/` shrinks to just slimmed `client.go` (UA/metrics
     constructor) — the remaining protocol concerns live in
     `internal/store/atproto/`
