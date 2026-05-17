# Codebase Reorganization Plan

Date: 2026-05-17
Companion to: `2026-05-17-codebase-audit.md`

> Phased refactor to address the audit. Each phase is independently
> shippable, ends in a clean build + green tests, and can be paused
> between. Order is dependency-driven: earlier phases unblock later ones.

## Goal

Reorganize per-app code (arabica, oolong) into top-level
`internal/arabica/` and `internal/oolong/` packages so each app owns
its full vertical slice (entities, handlers, web). Decouple
`internal/atproto/` from app-specific code. Unify the duplicate
generic CRUD machinery.

## Architecture

Target layout:

```
internal/
  arabica/
    entities/     ← moved from internal/entities/arabica/
    handlers/     ← moved from handlers/{brew,recipe,arabica_crud_generic}.go
    web/          ← already exists (templ components + pages)
  oolong/
    entities/     ← moved from internal/entities/oolong/
    handlers/     ← moved from handlers/oolong_*.go
    web/          ← already exists
  entities/       ← keeps cross-app registry (entities.go) only
  handlers/       ← shared only: auth, admin, feed, profile, modals, ...
  atproto/        ← decoupling deferred (see note below)
  (everything else unchanged)
```

Three phases, in dependency order:

1. **Promote `internal/entities/{app}/` → `internal/{app}/entities/`** —
   mechanical rename that establishes the per-app root pattern.
   ✅ Completed 2026-05-17.
2. **Move per-app handlers into `internal/{app}/handlers/`** — the
   largest move; finishes the vertical-slice structure.
3. **Unify the duplicate generic CRUD helpers** — easier once both
   sides are isolated.

A fourth optional phase splits `atproto/store.go` internally.

### Phase originally numbered 1 (decouple `atproto` from arabica) — deferred

The audit suggested injecting a codec registry to make `atproto/`
app-agnostic. On closer inspection, the coupling runs much deeper than
a codec table: `*AtprotoStore` carries typed methods (`CreateBean`,
`ListBeans`, `resolveBeanRefs`, …) and the `database.Store` interface
is entirely arabica-typed. Unwinding all of that is a significant
refactor on its own — not a prerequisite for the package rename
(Phase 2 originally; now Phase 1), which is purely a sed over import
paths. Deferred to a dedicated future plan.

## Tech Stack

Go 1.21+, stdlib `net/http`, Templ, AT Protocol indigo SDK, BoltDB,
SQLite, testify. No new dependencies.

## Corrections to the audit

- "Move OAuth out of `atproto/`" — already done. OAuth lives in
  `handlers/auth.go` (HTTP flow), `database/sqlitestore/oauth.go`
  (storage), and `atproto/client.go` (thin session-resume wrapper, which
  is legitimately AT-Protocol concern).
- Session-cache vs witness-cache split in `atproto/` — dropped. Files
  are 202 + 82 LOC, splitting yields nothing.

---

## Phase 1: Promote per-app entity packages to top-level roots ✅

Completed 2026-05-17.

**Why first:** Establishes the `internal/{app}/` pattern that Phase 2
fills out. Pure sed-based rename — no architectural impediment from
`atproto/`'s coupling to arabica because the package name stays the
same; only the import path changes.

**Scope:** Two package moves. ~50 files relocate. Pure renames — no
code changes inside files except import paths.

**Moves:**

- `internal/entities/arabica/` → `internal/arabica/entities/`
- `internal/entities/oolong/` → `internal/oolong/entities/`
- `internal/entities/arabica/__snapshots__/` and `__snapshots__/` move
  with their packages
- `internal/entities/entities.go` and `internal/entities/entities_*test.go`
  **stay** at `internal/entities/` — this is the cross-app registry

**Steps (all completed):**

- [x] **1.1** — `mv internal/entities/arabica internal/arabica/entities`.
- [x] **1.2** — Verified package decl stays `arabica`. No change needed.
- [x] **1.3** — Bulk import rewrite via `find ... -print0 | xargs -0 sed -i`.
- [x] **1.4** — `go build ./...` clean.
- [x] **1.5** — Repeat for oolong.
- [x] **1.6** — `internal/entities/` now contains only registry files
  (`entities.go`, `entities_test.go`, `entities_allforapp_test.go`).
- [x] **1.7** — `go vet ./...` clean, `go test ./...` all 887 tests
  pass across 36 packages.
- [x] **1.8** — Commit: `refactor: promote per-app entity packages to internal/{app}/entities`.

**Verification (passed):**

```
grep -r "internal/entities/arabica\|internal/entities/oolong" .   # zero hits
ls internal/arabica/entities/ internal/oolong/entities/             # populated
go test ./...                                                       # 887 passed
```

**Rollback:** Revert commit. Pure rename so revert is clean.

**Risk:** Low. Mechanical. The main hazard is missing an import path
in a templ-generated file or test fixture — step 1.4's build catches
those.

---

## Phase 2: Move per-app handlers into `internal/{app}/handlers/`

**Why now:** With per-app entities at their final home, handlers can
move next to them. Largest move (~3-4K LOC), but each app's move is
independent.

**Scope:** All handler files exclusively about one app's entities.
Shared infrastructure stays in `internal/handlers/`.

**What moves to `internal/arabica/handlers/`:**

- `handlers/brew.go` (637 LOC)
- `handlers/recipe.go` (819 LOC)
- `handlers/arabica_crud_generic.go` (112 LOC)
- Arabica-specific halves of `handlers/entities.go` and
  `handlers/entity_views.go` (split required — see 3.6)

**What moves to `internal/oolong/handlers/`:**

- `handlers/oolong_crud.go` (518 LOC)
- `handlers/oolong_pages.go` (481 LOC)
- `handlers/oolong_modals.go` (76 LOC)
- `handlers/oolong_api.go` (72 LOC)
- `handlers/oolong_modals_generic.go` (58 LOC)
- `handlers/oolong_crud_generic.go` (82 LOC)
- `handlers/entity_views_oolong.go` (215 LOC)

**What stays in `internal/handlers/`:**

- Shared HTTP: `auth.go`, `admin.go`, `feed.go`, `pages.go`,
  `profile.go`, `bsky_profile.go`, `modals.go`, `notifications.go`,
  `report.go`, `suggestions.go`
- Core: `handlers.go` (the `Handler` struct + helpers — see 3.3)
- Cross-app pieces of `entities.go`, `entity_views.go`,
  `entity_routes.go`, `entity_view_helpers.go`

**Steps:**

- [ ] **2.1** — Create `internal/arabica/handlers/` and
  `internal/oolong/handlers/` (empty packages with a placeholder
  `doc.go`).
- [ ] **2.2** — Move `handlers/brew.go` and `handlers/recipe.go` to
  `internal/arabica/handlers/`. Update package declaration. Update
  imports. Build (will fail — proceed to 2.3).
- [ ] **2.3** — Per-app handlers currently hang methods off
  `*handlers.Handler`. Extract the subset of dependencies they actually
  use (atproto store getter, suggestions, feed service, templ
  renderer, session lookup) into a smaller `*Deps` struct exported
  from `internal/handlers/`. Per-app packages declare local
  `*ArabicaHandlers` / `*OolongHandlers` types that hold `*Deps` and
  carry the moved methods. Constructor:
  `arabicahandlers.New(deps *handlers.Deps) *ArabicaHandlers`.
- [ ] **2.4** — Move `arabica_crud_generic.go` into
  `internal/arabica/handlers/`. Helpers become package-level functions
  on `*Deps`.
- [ ] **2.5** — Repeat 2.2-2.4 for oolong (all six listed files). Move
  + repackage + wire to `*Deps`. Build green after this step.
- [ ] **2.6** — Split `handlers/entities.go` and
  `handlers/entity_views.go`. Cross-app helpers stay. The
  `switch RecordType` blocks split: each case body moves to its app
  package, and the switch becomes a dispatch table the router builds
  from app registrations (each app registers a handler set for the
  record types it owns).
- [ ] **2.7** — Update `internal/routing/routing.go`. Per-app route
  registration moves into `arabicahandlers.Register(mux, deps)` /
  `oolonghandlers.Register(mux, deps)`. Shared routes continue to be
  registered from `internal/handlers/`.
- [ ] **2.8** — `go test ./...` + manual smoke test of both apps
  (login, CRUD for one entity per app, view page, feed cards). Run
  `templ generate` first if any `.templ` imports changed.
- [ ] **2.9** — Commit: `refactor: relocate per-app handlers into internal/{app}/handlers`.

**Verification:**

```
grep -r "Oolong\|oolong" internal/handlers/    # expect only shared cross-references
grep -r "Brew\|Recipe" internal/handlers/      # expect only shared references (feed cards, etc.)
go test ./...
just run                                       # exercise both apps in browser
```

**Rollback:** Single commit revert.

**Risk:** Highest-risk phase. Handler interdependencies are subtle
(shared session helpers, response writers, error formatters). The
`*Deps` shape from 2.3 is the linchpin — too narrow and per-app
handlers refactor twice; too wide and `*Deps` is just `*Handler`
renamed. Keep it minimal; add fields as compile errors demand them.

---

## Phase 3: Unify the generic CRUD helpers

**Why after Phase 2:** `arabica_crud_generic.go` and
`oolong_crud_generic.go` look similar but call different store APIs
(arabica uses typed `store.CreateBean`-style methods; oolong uses
generic `putRecord(nsid, ...)`). Unifying requires aligning one onto
the other. With both sets isolated in their own packages, the
unification is a focused diff.

**Scope:** ~200 LOC of dedupe; possibly removes typed CRUD methods
from `AtprotoStore` API.

**Steps:**

- [ ] **3.1** — Pick target shape. Recommended: a generic
  `CRUDHelper[Req, Model]` type with hooks for build/encode/decode,
  living in a new shared `internal/entities/crud/` package (neither
  app owns it; both depend on it).
- [ ] **3.2** — Implement the helper covering create, update, delete.
  Reuse the existing oolong shape (generic `putRecord`) since it
  doesn't depend on entity-specific store methods.
- [ ] **3.3** — Migrate oolong handlers to use it. Run tests.
- [ ] **3.4** — Migrate arabica handlers to use it. Removes typed
  `store.CreateBean`/`CreateBrew`/etc. methods from `AtprotoStore`
  if no other callers remain — verify with grep.
- [ ] **3.5** — Delete both old `*_crud_generic.go` files.
- [ ] **3.6** — Commit: `refactor(handlers): unify arabica/oolong CRUD generators`.

**Verification:**

```
grep -r "arabicaCRUDCreate\|oolongCRUDWrite" internal/   # expect zero
go test ./...
```

**Rollback:** Revert commit.

---

## Phase 4 (optional): Split `atproto/store.go`

**Why optional:** `store.go` is 1198 LOC. Large but not unmanageable.
After Phases 1-3 it should be smaller (codecs extracted; typed CRUD
methods possibly removed). Reassess then.

**Trigger:** Skip unless `store.go` still exceeds ~1000 LOC and
developers report navigation friction.

**Suggested split:**

- `store.go` — struct, constructor, top-level methods
- `store_read.go` — list/get/resolve operations
- `store_write.go` — put/delete operations
- `store_session.go` — session cache integration
- (keep existing `witness.go`, `store_entity.go`, `store_generic.go`)

**Steps:**

- [ ] **4.1** — Split by responsibility. No behavior changes. Compile,
  test, commit.

---

## Out of scope (deliberately)

- **Renaming the `internal/entities/` registry package.** After Phase
  2, this package holds only the cross-app `Descriptor` registry. The
  name still fits — it's the registry of entity descriptors. Leave it.
- **Splitting session cache from witness cache wiring.** Files are
  small and serve clearly different roles.
- **Pulling `signup` into a unified `auth` package.** Different
  concerns (account creation vs. login).

## Execution notes

- Run each phase in a separate worktree (`jj` or `git worktree`);
  merge to main when green.
- `templ generate` is required after any move that touches `.templ`
  imports. Phase 2 and Phase 2 both need this.
- `just run` smoke-tested at the end of Phases 1, 2, 3 for both
  arabica and oolong app flows.
- Commit messages follow Conventional Commits per `CLAUDE.md`. Use
  `refactor:` scope; no breaking-change footers (no public API).
- Phases 1 and 2 produce churn-heavy diffs. Coordinate with any
  in-flight feature branches before starting — rebases through these
  commits will be painful.
