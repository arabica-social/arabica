# Infrastructure/data package review

## Scope

Reviewed current backend code in:

- `internal/atproto`
- `internal/atplatform/domain`
- `internal/atplatform/server`
- `internal/database`
- `internal/database/sqlitestore`
- `internal/firehose`
- `internal/feed`
- `internal/backup`
- `internal/logging`
- `internal/metrics`
- `internal/tracing`

## Strongest structural findings

### 1. Feed public cache returns shared mutable `*FeedItem` objects

Evidence:

- `internal/feed/service.go:297` defines `GetCachedPublicFeed`.
- `internal/feed/service.go:300` reads `items := s.cache.items`.
- `internal/feed/service.go:314` returns those cached items.
- `internal/feed/service.go:355` stores `s.cache.items = items`.
- `internal/handlers/feed.go:236-254` hydrates request-specific viewer state by
  mutating `item.IsOwner` and `item.IsLikedByViewer`.

Impact:

- Cached feed entries can carry per-viewer state across requests if cached
  pointers are reused.
- Even if current control flow avoids some cases, the abstraction is unsafe:
  callers receive mutable shared cache internals.

Code-judo move:

- Make cached feed items immutable to callers.
- Return cloned `[]*FeedItem` values from cache reads, or cache a DTO without
  request-specific fields.
- Move viewer hydration to operate only on per-request copies.

Risk: high correctness and privacy risk around request-specific state leakage.

Approval bar: not acceptable as a cache boundary.

---

### 2. `internal/firehose/index.go` is a god object for unrelated persistence concerns

Evidence:

- `internal/firehose/index.go:75` defines `FeedIndex`.
- `internal/firehose/index.go:132` begins schema creation for many tables.
- `internal/firehose/index.go:407` handles witness record lookup.
- `internal/firehose/index.go:533` handles record upsert.
- `internal/firehose/index.go:584` handles record delete.
- `internal/firehose/index.go:810` handles feed query.
- `internal/firehose/index.go:1068` maps records to feed items.
- `internal/firehose/index.go:1139` handles profile lookup.
- `internal/firehose/index.go:1417` handles known DID paths.
- `internal/firehose/index.go:1514+` handles aggregate/stat/explore queries.
- `internal/firehose/index.go:2147` handles comments query paths.

Impact:

- Feed serving, witness cache, profile cache, moderation, comments, likes,
  notifications, known-DID backfill, and exploration aggregates are coupled into
  one file/type.
- Schema and query changes are risky because unrelated features share one
  persistence surface.

Code-judo move:

- Keep one SQLite database, but split repositories around invariants:
  `WitnessRepository`, `FeedRepository`, `ProfileRepository`,
  `SocialRepository`, `NotificationRepository`, and `BackfillRepository`.
- Let `FeedIndex` become composition/root wiring rather than the implementation
  of every table.

Risk: high maintainability risk. This is the largest obvious decomposition
target in the backend.

Approval bar: functionally plausible, structurally overgrown. Decompose before
more firehose/feed features land.

---

### 3. `database.Store` still exposes SQL-era shape that does not match ATProto storage

Evidence:

- `internal/database/store.go:20` has
  `ListBrews(ctx, userID, offset, limit)`.
- Other list methods omit pagination/user ID, for example
  `internal/database/store.go:27`, `:34`, `:41`, `:48`, and `:55`.
- `internal/atproto/store.go:455` implements
  `ListBrews(ctx, userID, offset, limit)`.
- The `userID` parameter is not meaningful in the ATProto implementation.
- `internal/atproto/store.go:459-460` special-cases brew pagination.
- Other entities use generic `ListEntity`, for example
  `internal/atproto/store.go:681`, `:745`, `:798`, `:847`, and `:951`.

Impact:

- The interface describes neither the old local-SQL model nor the current
  ATProto/PDS model cleanly.
- Brew has bespoke paging semantics while other entities do not.
- `userID` is misleading dead API surface.

Code-judo move:

- Replace SQL-shaped store signatures with ATProto-shaped collection APIs.
- A DID/session-scoped store should not accept `userID`.
- Represent pagination consistently across entities, or isolate it to feed/list
  views.
- Consider splitting `Store` into focused interfaces used by handlers instead
  of one app-wide CRUD interface.

Risk: medium-high boundary risk. The wrong abstraction makes every handler and
store implementation harder to reason about.

Approval bar: not a clean abstraction boundary.

---

### 4. `AtprotoStore` remains too broad despite useful generic helpers

Evidence:

- `internal/atproto/store.go:20` defines `AtprotoStore` with client, session
  cache, witness cache, and app-config responsibilities.
- `internal/atproto/store.go:87` handles client acquisition.
- `internal/atproto/store.go:99` handles witness reads.
- `internal/atproto/store.go:121` handles witness list paths.
- `internal/atproto/store.go:376+` contains bespoke brew CRUD.
- `internal/atproto/store_entity.go:40` defines generic `CreateEntity`.
- `internal/atproto/store_entity.go:79` defines generic `ListEntity`.
- `internal/atproto/store.go:681`, `:745`, `:798`, `:847`, and `:951` show
  generic entity adoption.

Impact:

- Generic CRUD is the right direction, but `AtprotoStore` still mixes transport,
  cache policy, relation hydration, witness fallback, and domain CRUD.
- Brew remains a gravitational exception that keeps cache/list semantics
  divergent.

Code-judo move:

- Extract a low-level XRPC collection repository from domain entity services.
- Make cache policy a wrapper/decorator rather than embedded in every store
  path.
- Convert brew to the same codec/listing model, or explicitly isolate it as a
  separate aggregate service.

Risk: medium. Directionally good, but the boundary is still doing too much.

Approval bar: not yet a clean backend data boundary.

---

### 5. Server bootstrap is centralized enough to become fragile

Evidence:

- `internal/atplatform/server/server.go:74` defines `Run`.
- `internal/atplatform/server/server.go:80` resolves data directories.
- `internal/atplatform/server/server.go:149` opens databases.
- `internal/atplatform/server/server.go:187` initializes OAuth.
- `internal/atplatform/server/server.go:244` starts goroutines.
- `internal/atplatform/server/server.go:295` continues lifecycle/server setup.

Impact:

- `Run` owns environment resolution, persistence setup, OAuth, firehose,
  metrics, router/server lifecycle, and background workers.
- This is acceptable today but will become hard to test as app variants grow.

Code-judo move:

- Introduce a small `AppRuntime`/`Dependencies` builder split into:
  `OpenPersistence`, `BuildServices`, `BuildHTTP`, and
  `StartBackgroundWorkers`.
- Keep `Run` as orchestration only.

Risk: medium. This is not yet a blocker, but it is a predictable growth point.

Approval bar: approved for current size; refactor before more platform concerns
are added.

## Positive findings

- `internal/atproto/store_entity.go` is a strong simplification direction:
  codecs plus generic CRUD remove repeated entity boilerplate.
- `internal/firehose/adapter.go` appears intentionally thin and appropriate as
  a boundary adapter.
- `internal/backup` is small and understandable; no high-conviction structural
  issue was found.
- `internal/logging`, `internal/metrics`, and `internal/tracing` are compact.
  The main concern is global metric registration/import side effects, but no
  blocker was found.

