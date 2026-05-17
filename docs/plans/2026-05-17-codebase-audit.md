# Codebase Organization Audit

Date: 2026-05-17

Audit of `internal/` package structure looking for bad abstractions, bad
splits, and packages doing too much.

## Summary

The codebase is well-architected overall — no circular dependencies, most
packages are focused. Issues are at the margins, concentrated in two places:
`internal/atproto/` (overloaded) and `internal/handlers/` (kitchen sink with
app-specific code mixed in).

## Findings

### `internal/database/` — keep it

Thin interface + mock (~1.3K LOC). It's the boundary between handlers and
`atproto/` internals. Removing it would tightly couple handlers to AT
Protocol details. No behavior bloat; this package earns its existence.

### `internal/atproto/` — overloaded (the biggest problem)

Conflates 4+ distinct concerns in ~3K LOC across 11 files:

- **OAuth** (`oauth.go`) — web auth, not AT Protocol records
- **Record codecs** (`records.go`, `store_arabica_codecs.go`) — hardcodes
  arabica entities, breaking app-agnosticism
- **Session cache** (`cache.go`) + **witness cache** (`witness.go`) — two
  different caches with different lifetimes/storage, conflated in one package
- **Store impl** (`store.go`, ~39K) — implements `database.Store` via PDS XRPC

This is the most-imported package in the codebase (18 imports from handlers
alone). Every change ripples widely.

### `internal/handlers/` — kitchen sink

27 files / ~10K LOC. App-specific code mixed with shared:

- `brew.go`, `recipe.go` — arabica-only
- `oolong_crud.go`, `oolong_pages.go`, `oolong_modals.go` — oolong-only
- `arabica_crud_generic.go` and `oolong_crud_generic.go` are nearly duplicate
  — should be one parameterized generator over `entities.Descriptor`

### `internal/entities/{arabica,oolong}/` — well-designed

Per-app models + codecs + suggestions + NSID + registry. The naming clash
with the repo name is mildly confusing in directory listings but the pattern
is sound. The registry approach avoids import cycles and lets each binary
load only its own entity set.

### Clean packages — leave alone

`firehose/`, `backup/`, `feed/`, `web/`, `atplatform/`, `logging/`,
`tracing/`, `metrics/`, `lexicons/`, `matching/`, `signup/`, `moderation/`,
`ogcard/`, `suggestions/`, `middleware/`, `routing/`.

No mutual dependencies, focused responsibilities. The tiny utility packages
(`logging`, `tracing`, `metrics`, `lexicons`, `matching`) are appropriately
isolated — none of them are one-file wrappers in need of inlining.

### Dependency graph (clean)

```
handlers → atproto → database
         ↘ atplatform/domain → entities
         ↘ feed → entities, atproto, moderation
         ↘ firehose → entities, atproto
         ↘ web/components → entities, feed
```

No cycles. `handlers/` is the top-level orchestrator; only `routing/` and
`atplatform/server/` import it, which is correct.

## Recommendations (priority order)

### 1. Move OAuth out of `internal/atproto/`

Relocate to `internal/auth/` (new) or fold into `internal/signup/`.
`atproto/` should mean "AT Protocol records + XRPC," not web auth. Keep
NSID scope constants in `atproto/` if they're app-independent.

### 2. Pull per-app handlers into `internal/entities/{app}/handlers.go`

Keep `internal/handlers/` for shared infrastructure only: auth, feed,
profile, notifications, admin, moderation. Move per-app CRUD/view/modal
handlers to live next to their entity models. Lets a binary include
exactly one app cleanly without dragging in the other's HTTP surface.

### 3. Collapse the duplicate `*_crud_generic.go` files

`arabica_crud_generic.go` and `oolong_crud_generic.go` are nearly identical.
Replace with one descriptor-driven generator in `handlers/` that takes an
`*entities.Descriptor`.

### 4. Decouple `atproto/store.go` from arabica codecs

Inject a codec registry at `NewAtprotoStore()` time so the store
implementation is app-agnostic. Currently `store.go` imports arabica
entity packages directly, which prevents the same store from serving
oolong cleanly.

### 5. (Optional) Split session cache from witness-cache wiring in `atproto/`

`cache.go` (session, in-memory, per-request) and `witness.go`
(SQLite-backed, populated by firehose) have different lifetimes and
storage. Splitting would clarify ownership. Not urgent.

## What the user asked specifically

> there is a store package — should _it_ exist?

Yes. `internal/database/` (the `Store` interface package) is small but
load-bearing — it's what keeps handlers from importing `atproto/`
internals. The real mess is inside `atproto/` itself, which has accreted
OAuth + caching + records + store implementation and should be peeled
apart. The DB story is "spread across packages" because that's correct
layering (interface in `database/`, PDS impl in `atproto/`, SQLite index
in `firehose/`, backups in `backup/`), not because it's disorganized.
