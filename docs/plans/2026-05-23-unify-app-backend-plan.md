# Unify App Backend Plan

Date: 2026-05-23

## Goal

Reduce the Arabica/Oolong seam in backend code while keeping the real product
differences isolated in entity models, rendering, and branding.

Target direction:

- **One deployable server binary** that can run both apps.
- **Two app databases** at first: one Arabica DB, one Oolong DB.
- **Shared backend plumbing** for AT Protocol record CRUD, cache invalidation,
  feed/index setup, OAuth bootstrap, and standard entity handlers.
- **Separate entity packages and web packages** for product-specific models,
  codecs, copy, templates, and design.

## Non-goals

- Do not merge the Arabica and Oolong lexicons.
- Do not move to one shared SQLite database in this pass.
- Do not redesign feed ranking, moderation policy, or onboarding UX.
- Do not remove the one-app binaries until the unified server has soaked in dev.

## Desired end state

```text
cmd/server/                 # one process can mount both apps
cmd/arabica/                # retained initially as one-app wrapper
cmd/oolong/                 # retained initially as one-app wrapper

internal/arabica/
  entities/                 # coffee models/codecs/descriptors
  web/                      # coffee-specific templ components/pages

internal/oolong/
  entities/                 # tea models/codecs/descriptors
  web/                      # tea-specific templ components/pages

internal/atplatform/domain/ # App, Brand, per-app config
internal/atplatform/server/ # app stack builder, can build N app stacks
internal/entities/          # shared descriptor registry
internal/handlers/          # shared app-aware HTTP plumbing
internal/atproto/           # concrete AT Protocol/PDS record store
```

Runtime data layout:

```text
$DATA_ROOT/
  arabica/arabica.db
  arabica/backups/
  oolong/oolong.db
  oolong/backups/
```

## Current pain points

1. **Backend split is inconsistent.** Arabica routes use the typed
   `arabicastore.Store` interface; Oolong routes often downcast to
   `*atproto.AtprotoStore` and use generic record primitives directly.

2. **Per-app handler packages duplicate backend flow.** Create/update/delete
   shape is mostly the same: decode, validate, encode, write PDS record,
   invalidate caches, respond.

3. **The app abstraction already exists but is underused.** `domain.App`,
   descriptor lists, NSID bases, and per-app like/comment NSIDs are the right
   seam. More backend code should dispatch through that seam instead of through
   `coffeehandlers` vs `teahandlers` package boundaries.

4. **Single shared DB is tempting but premature.** Several tables need explicit
   app scoping before one DB is safe: registered DIDs, backfill status, OAuth
   sessions/auth requests, notifications, moderation state, and backups.

## Architecture decision

Prefer **single binary + two databases** as the first consolidation step.

Why:

- Captures most operational simplification.
- Avoids cross-app data leakage and ambiguous app-scoping migrations.
- Matches the existing `server.Run(app, opts)` shape.
- Keeps rollback easy: one-app binaries can continue to run unchanged.

Defer **single binary + one database** until there is a deliberate app-scoping
schema plan.

## Phase 0 — Guardrails and inventory

Status: not started.

Tasks:

- [ ] Add/refresh an import and package inventory for app-specific backend code.
- [ ] List every route currently registered by Arabica and Oolong.
- [ ] List every table in the SQLite schema and mark it as app-local or
      potentially global.
- [ ] Identify tests that cover Arabica CRUD, Oolong CRUD, feed indexing,
      OAuth scopes, route registration, and app-specific assets.

Verification:

- [ ] `go test ./...`
- [ ] Inventory committed in this plan or a companion audit doc.

Rollback:

- Documentation-only phase.

## Phase 1 — Add a unified server binary, keep isolated app stacks

Status: implemented 2026-05-23.

Goal: prove one process can run both apps without changing handler/storage
architecture yet.

Tasks:

- [x] Add shared app constructors outside `cmd/arabica` and `cmd/oolong`, e.g.
      `internal/arabica/app` and `internal/oolong/app`, or an
      `internal/atplatform/apps` package.
- [x] Add `cmd/server` that constructs both apps and starts one app stack per
      app.
- [x] Keep separate ports initially:
      - Arabica default: `18910`
      - Oolong default: `18920`
- [x] Keep separate metrics ports initially:
      - Arabica default: `9101`
      - Oolong default: `9102`
- [x] Keep separate data dirs and DBs through existing per-app env prefixes:
      `ARABICA_DATA_DIR`, `OOLONG_DATA_DIR`.
- [x] Ensure both app stacks can shut down from the same process context.
- [x] Keep `cmd/arabica` and `cmd/oolong` working.

Verification:

- [x] `go test ./...`
- [x] `go run ./cmd/server` starts both HTTP listeners.
- [x] Arabica home/feed/login pages load on Arabica port.
- [x] Oolong home/feed/login pages load on Oolong port.
- [x] Both DB files are created in separate data dirs.

Rollback:

- Remove `cmd/server` and shared app constructor extraction. One-app binaries
  remain unchanged.

## Phase 2 — Introduce a generic record-store boundary

Status: implemented 2026-05-23.

Goal: create one storage abstraction both apps can use, without forcing all
Arabica typed methods to disappear immediately.

Proposed interface shape:

```go
type RecordStore interface {
    DID() string
    FetchRecord(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, err error)
    FetchAllRecords(ctx context.Context, nsid string) ([]atproto.RawRecord, error)
    PutRecord(ctx context.Context, nsid, rkey string, record any) (resultRKey, cid string, err error)
    RemoveRecord(ctx context.Context, nsid, rkey string) error
}
```

Tasks:

- [x] Define the interface in a package that is not app-specific. Candidate:
      `internal/records`.
- [x] Make `*atproto.AtprotoStore` satisfy it.
- [x] Replace Oolong concrete `*atproto.AtprotoStore` requirements with the
      generic interface where practical.
- [x] Leave Arabica `arabicastore.Store` intact during this phase.
- [x] Add focused tests around generic create/update helpers using a fake store.

Verification:

- [x] `go test ./internal/oolong/... ./internal/handlers/... ./internal/atproto/...`
- [x] `go test ./...`

Rollback:

- Interface addition is additive. Revert Oolong call-site conversions if needed.

## Phase 3 — Move standard CRUD plumbing into shared app-aware handlers

Status: implemented 2026-05-23 for standard simple-entity CRUD plumbing.

Goal: replace per-app duplicated CRUD skeletons with shared descriptor-driven
helpers.

Tasks:

- [x] Extend `entities.Descriptor` only where needed for backend behavior. No new
      descriptor callbacks were needed for this pass; the shared helper stays
      handler-driven to avoid a descriptor god object.
- [x] Extract common write flow:
      decode request → validate → model → record map → `PutRecord` → invalidate
      feed/session state → JSON/HTMX response.
- [x] Convert Oolong simple entities first; they already use generic record
      primitives.
- [x] Convert Arabica simple entities next: roaster, grinder, brewer.
- [x] Defer complex entities with richer behavior: Arabica brew, recipe, and
      any entity with batch reference resolution or bespoke page flows.
- [x] Delete only helper code that has no remaining callers.

Verification:

- [x] Existing handler tests pass.
- [x] Add table tests for shared helper behavior: validation error, decode
      error, create success, update success, store error.
- [x] Handler smoke: create/update simple entities through the shared helper for
      Arabica and Oolong using fake stores; public server smoke starts both apps.

Rollback:

- Convert entity-by-entity so each step can be reverted independently.

## Phase 4 — Normalize Arabica onto the generic record path

Status: implemented 2026-05-23 for simple entity handlers; complex typed flows intentionally retained.

Goal: reduce reliance on the Arabica-only `arabicastore.Store` interface.

Tasks:

- [x] Convert Arabica simple entity handlers to generic record helpers.
- [x] Move Arabica-specific typed repository methods behind local helpers where
      they still add value. Complex brew/recipe/social flows remain on
      `internal/arabica/store` pending a dedicated refactor.
- [x] Reassess `internal/arabica/store.Store` after conversions:
      - It still supports complex Arabica flows and tests, so it now lives at
        `internal/arabica/store`.
- [x] Keep typed model conversion in `internal/arabica/entities`.

Verification:

- [x] `go test ./internal/arabica/... ./internal/atproto/...`
- [x] `go test ./...`
- [x] Handler smoke: Arabica roaster/grinder generic record CRUD covered by tests;
      bean remains intentionally bespoke because it can inline-create roasters.

Rollback:

- Keep old typed methods until each converted handler is proven. Delete last.

## Phase 5 — Consolidate route registration where behavior is app-generic

Status: implemented 2026-05-23 for descriptor-driven entity bundles.

Goal: remove routing branches that only exist because of current package layout.

Tasks:

- [x] Identify route bundles that are pure descriptor loops.
- [x] Move generic route registration into shared routing/handler code.
- [x] Keep app-specific route registration for bespoke pages and onboarding.
- [x] Ensure unsupported/deferred entities are gated by `App.Descriptors`, not
      hardcoded app switches.

Verification:

- [x] Route inventory from Phase 0 matches generated routes after refactor.
- [x] `go test ./internal/routing/... ./...`
- [x] Public server smoke both apps; descriptor route filtering covered by tests.

Rollback:

- Route registration can remain mixed; no data migration involved.

## Phase 6 — Clean package names and storage layout

Status: implemented 2026-05-23.

Goal: make package names match ownership after the backend merge.

Tasks:

- [x] Move SQLite infrastructure adapters out of `internal/database/sqlitestore`
      to `internal/sqlitestore`.
- [x] If `arabicastore.Store` remains Arabica-only, move it to
      `internal/arabica/store`.
- [x] Update docs and AGENTS guidance for the new package boundaries.

Verification:

- [x] `go test ./...`
- [x] `go vet ./...`

Rollback:

- Mechanical import-path revert.

## Phase 7 — Consider one shared database only after app scoping is explicit

Status: deferred.

Do not start this until the single-binary/two-DB setup is stable.

Questions to answer first:

- [ ] Are registered users global or app-local?
- [ ] Is moderation global or app-local?
- [ ] Are OAuth sessions shared between app brands?
- [ ] Should notifications co-mingle across apps?
- [ ] Should backfill status be per DID globally or per app/collection set?
- [ ] Should backups restore one app independently or both together?

Likely schema work if this proceeds:

- Add `app` column to app-local tables.
- Add composite indexes including `app`.
- Migrate existing Arabica/Oolong DBs into one DB.
- Add tests preventing cross-app leakage.

## Suggested commit sequence

1. `refactor: extract app constructors from cmd packages`
2. `feat: add unified server binary for arabica and oolong`
3. `refactor: introduce generic record store interface`
4. `refactor(oolong): use generic record store boundary`
5. `refactor: share standard entity CRUD helper`
6. `refactor(arabica): move simple entities to generic record CRUD`
7. `refactor: consolidate descriptor-driven route registration`
8. `refactor: clarify sqlite and arabica store package names`

## Completion criteria

- One command can run both apps in one process.
- Arabica and Oolong still use separate DB files.
- Simple entity CRUD no longer has parallel Arabica/Oolong backend helper
  implementations.
- Product-specific code remains in entity/web packages.
- `go test ./...`, `go vet ./...`, and manual smoke tests for both apps pass.
