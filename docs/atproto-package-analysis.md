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
| `resolver.go`      | 155   | AT-URI parsing + entity-specific reference resolvers          | TODO       |
| `public_client.go` | ~70   | Thin OTel/UA wrapper around `atp.PublicClient`                | Slimmed ✅ |
| `store.go`         | 1250+ | `AtprotoStore` implementing `database.Store` for all entities | TODO       |
| `store_generic.go` | 170   | Generic fetch/put/delete helpers used by store.go             | TODO       |
| `cache.go`         | 226   | Copy-on-write per-session cache with typed entity accessors   | TODO       |
| `witness.go`       | 62    | `WitnessCache` interface + `WitnessRecord` type               | TODO       |

**Already deleted**: `handle.go`, `handle_test.go`, `nsid_test.go`

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

| Item                                                                   | Details                                                                                          |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `OAuthManager` → `atp.OAuthApp`                                        | Handler struct + constructor switched; `InitiateLogin` → `StartLogin`, `InitiateSignup` → `StartSignup` |
| `AuthMiddleware` → `atp/middleware.CookieAuth`                         | Routing middleware replaced; OnAuth callback configurable via routing.Config                     |
| `GetAuthenticatedDID` → `atp/middleware.GetDID`                        | All 46+ callers in handlers, middleware, and tests updated (return changed from `error` to `bool`) |
| `GetSessionIDFromContext` → `atp/middleware.GetSessionID`              | All callers updated (changed from `error` to `bool` return)                                      |
| `ContextWithAuthDID` / `ContextWithAuth` → raw context keys            | Test helpers updated to use `"atp_did"` / `"atp_session_id"` context keys                    |
| `ParseDID` → `syntax.ParseDID`                                         | Arabica wrapper deleted; direct indigo `syntax.ParseDID` used instead                            |
| `SetOnAuthSuccess` → `CookieAuthConfig.OnAuth`                         | On-auth callback moved from OAuthManager setter to middleware config                             |
| `NewOAuthManager` → `atp.NewOAuthApp`                                  | Server bootstrap + integration harness updated                                                   |
| `NewClient(oauth *OAuthManager)` → `NewClient(oauth *atp.OAuthApp)`    | Client constructor type changed; internal provider uses `oauth.ResumeSession`                    |
| **Deleted files**                                                      | `internal/atproto/oauth.go` (~360 lines)                                                         |

**Impact**: ~360 lines deleted, one less OAuth implementation to maintain.

---

## Remaining Work

### Phase C — Package extraction

Extract the store, cache, and witness code into `internal/store/atproto/`:

1. **Create `internal/store/atproto/`**: Move `store.go`, `store_generic.go`,
   `cache.go`, `witness.go` there.

2. **Move entity resolvers**: Extract `ResolveBeanRefWithRoaster`,
   `ResolveRoasterRef`, `ResolveGrinderRef`, `ResolveBrewerRef`,
   `ResolveRecipeRef`, `ResolveBrewRefs` from `resolver.go` into
   `internal/store/atproto/resolve.go`.

3. This separates the coffee-domain data access layer from general protocol
   utilities. Oolong would get its own `internal/store/atproto/` with tea
   entities.

### Phase D — Cleanup

Remove remaining duplication and thin abstractions:

1. **Delete `resolver.go`**: Replace `ResolveATURI` / `ATURIComponents` usage
   with `atp.ParseATURI` (same 4-string return: did, collection, rkey, err).
   Entity-specific resolvers already moved to `internal/store/atproto/`.

2. **Delete `client.go` I/O types**: `CreateRecordInput`, `GetRecordOutput`,
   etc. are thin wrappers around `atp.Client` methods. Replace with direct
   `atp.Client` usage. Keep only the UA/metrics constructor wiring.

3. **Delete `nsid.go` proxies**: Update remaining `atproto.atp.BuildATURI` and
   `atproto.ExtractRKeyFromURI` callers to use `atp.atp.BuildATURI` and
   `atp.RKeyFromURI` directly. This affects ~15 files (mostly tests).

4. **Delete `public_client.go` entirely**: Once all callers use
   `atp.PublicClient` methods directly, the thin
   `PublicListRecordsOutput`/`PublicRecordEntry` wrappers can go away.

### End state

```
arabica/internal/
├── store/atproto/          ← store.go, generic.go, cache.go, witness.go, resolve.go
├── atproto/                ← (empty or single OTel constructor file)
├── handlers/
└── ...
```

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
