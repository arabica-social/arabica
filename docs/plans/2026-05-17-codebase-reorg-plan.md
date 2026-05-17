# Codebase Reorganization Plan

Date: 2026-05-17
Companion to: `2026-05-17-codebase-audit.md`

> Phased refactor to address the audit. Each phase is independently
> shippable, ends in a clean build + green tests, and can be paused
> between. Order is dependency-driven: earlier phases unblock later ones.

## Goal

Reduce coupling between `internal/atproto/` and arabica-specific code,
and pull app-specific HTTP handlers out of the shared `handlers/`
package so each app (arabica, oolong) owns its own HTTP surface.

## Architecture

Three foundational moves, in order:

1. Sever `atproto/store.go` from app-specific entity packages.
2. Relocate per-app HTTP handlers next to their domain models.
3. Unify the now-parallel generic CRUD machinery in `handlers/`.

A fourth optional phase splits `atproto/store.go` internally if its
size still hurts after the above.

## Tech Stack

Go 1.21+, stdlib `net/http`, Templ, AT Protocol indigo SDK, BoltDB,
SQLite, testify. No new dependencies.

## Correction to the audit

The audit recommended "move OAuth out of `atproto/`." On closer look,
OAuth already lives in the right places:

- `internal/handlers/auth.go` — HTTP flow (login, callback, logout)
- `internal/database/sqlitestore/oauth.go` — session/request storage
- `internal/atproto/client.go` — only thin session-resume wrapper around
  indigo's `atp.OAuthApp` (legitimately AT Protocol concern)

That recommendation is dropped. The session-cache vs witness-cache split
in `atproto/` is also dropped — both files are small (202 + 82 LOC) and
splitting yields no real benefit.

---

## Phase 1: Decouple `atproto/store` from arabica codecs

**Why first:** This is the smallest foundational change. While it sits
in place, Phase 2's per-app handler split is blocked because per-app
codec helpers live inside `atproto/`.

**Scope:** ~200 LOC moved, plus injection wiring.

**Files involved:**

- Move: `internal/atproto/store_arabica_codecs.go` →
  `internal/entities/arabica/codec.go` (or merge into existing
  `internal/entities/arabica/records.go`)
- Modify: `internal/atproto/store.go` — accept a codec registry at
  construction; remove direct imports of `internal/entities/arabica`
- Modify: `internal/atproto/store_entity.go`,
  `internal/atproto/store_generic.go` — read codecs from injected
  registry instead of import-level constants
- Modify: every `atproto.NewAtprotoStore(...)` caller (binary main +
  handler tests) — pass arabica codec registry explicitly

**Steps:**

- [ ] **1.1** — Define a `CodecRegistry` interface in `internal/atproto/`
  (or `internal/database/`). Methods: `EncodeRecord(nsid string, model any) (map[string]any, error)`
  and `DecodeRecord(nsid string, raw map[string]any) (any, error)`. One
  method per direction; the concrete arabica/oolong implementations live
  in their entity packages.
- [ ] **1.2** — Implement `arabica.NewCodecRegistry()` in
  `internal/entities/arabica/codec.go` that wraps the existing
  conversion functions. Add an oolong equivalent if any
  atproto-store-driven code path will hit it (check
  `handlers/oolong_*.go` callers).
- [ ] **1.3** — Add `WithCodecs(reg CodecRegistry)` option (or required
  constructor arg) to `NewAtprotoStore`. Keep the old constructor as a
  shim during migration if needed.
- [ ] **1.4** — Replace references to `store_arabica_codecs.go` symbols
  inside `atproto/` with registry lookups. Delete the file.
- [ ] **1.5** — Update every store constructor call site to pass the
  registry. Run `go vet ./... && go build ./... && go test ./...`.
- [ ] **1.6** — Commit: `refactor(atproto): inject codec registry instead of hardcoding arabica`.

**Verification:**

```
grep -r "entities/arabica" internal/atproto/   # expect zero hits
go test ./internal/atproto/...
go test ./...
```

**Rollback:** Revert the commit. The shim constructor (if kept) lets
callers fall back to compile-time codec wiring.

---

## Phase 2: Move per-app handlers to `internal/entities/{app}/handlers/`

**Why now:** With Phase 1 done, per-app handlers can import their own
entity package's codecs cleanly. This is the largest move — ~3-4K LOC
relocates across two sub-packages.

**Scope:** All handler files that are exclusively about one app's
entities. Shared infrastructure stays in `internal/handlers/`.

**What moves to `internal/entities/arabica/handlers/`:**

- `handlers/brew.go` (637 LOC) — arabica brew CRUD + view
- `handlers/recipe.go` (819 LOC) — arabica recipe CRUD + view
- The arabica-specific halves of `handlers/entities.go` and
  `handlers/entity_views.go`. Split these files: shared entity helpers
  stay; arabica-specific handlers move.
- `handlers/arabica_crud_generic.go` (112 LOC)

**What moves to `internal/entities/oolong/handlers/`:**

- `handlers/oolong_crud.go` (518 LOC)
- `handlers/oolong_pages.go` (481 LOC)
- `handlers/oolong_modals.go` (76 LOC)
- `handlers/oolong_api.go` (72 LOC)
- `handlers/oolong_modals_generic.go` (58 LOC)
- `handlers/oolong_crud_generic.go` (82 LOC)
- `handlers/entity_views_oolong.go` (215 LOC)

**What stays in `internal/handlers/`:**

- `auth.go`, `admin.go`, `feed.go`, `pages.go`, `profile.go`,
  `bsky_profile.go`, `modals.go`, `notifications.go`, `report.go`,
  `suggestions.go`
- `handlers.go` (the `Handler` struct + helpers) — but see step 2.3
- `entity_routes.go`, `entity_view_helpers.go`, the cross-app entity
  helpers from `entities.go`/`entity_views.go`

**Steps:**

- [ ] **2.1** — Create `internal/entities/arabica/handlers/` and
  `internal/entities/oolong/handlers/` (empty packages).
- [ ] **2.2** — Move `handlers/brew.go` and `handlers/recipe.go` to
  `internal/entities/arabica/handlers/`. Update imports. Build.
- [ ] **2.3** — The per-app handlers currently hang methods off
  `*handlers.Handler`. Extract the dependencies they actually use
  (atproto store getter, suggestions, feed service, templ renderer)
  into a smaller `*Deps` struct passed from the main `Handler`. Per-app
  handler files become package-level functions taking `*Deps`, or
  methods on an app-specific `*ArabicaHandlers` / `*OolongHandlers` type
  that holds `*Deps`. Decide which when wiring the first file in 2.2;
  apply consistently.
- [ ] **2.4** — Move arabica generic CRUD
  (`arabica_crud_generic.go`) into the arabica handlers package.
  Re-export the generic helpers as package-level functions on `*Deps`.
- [ ] **2.5** — Repeat 2.2-2.4 for oolong files (all six listed above).
- [ ] **2.6** — Split `handlers/entities.go` and
  `handlers/entity_views.go`. Cross-app helpers stay; per-app
  handlers move. The split point is usually the `switch` on
  `RecordType` — each case body moves to the relevant app package, and
  the switch becomes a dispatch table that the router builds from app
  registrations.
- [ ] **2.7** — Update `internal/routing/routing.go`. App routes are
  now registered from `internal/entities/{app}/handlers/`. Shared routes
  continue to be registered from `internal/handlers/`.
- [ ] **2.8** — Run full test suite + manual smoke test of both arabica
  and oolong (CRUD + view + feed cards). Templ generate first if any
  components moved.
- [ ] **2.9** — Commit as a single change: `refactor: relocate per-app handlers into entities/{app}/handlers`.

**Verification:**

```
grep -r "Oolong\|oolong" internal/handlers/    # expect only shared cross-references
grep -r "Brew\|Recipe" internal/handlers/      # expect only shared references (feed cards, etc.)
go test ./...
just run                                       # exercise both apps in browser
```

**Rollback:** Single commit. `git revert` cleanly restores. The
intermediate steps don't ship — they happen inside a worktree.

**Risk:** Highest-risk phase. Handler interdependencies are subtle
(shared session helpers, response writers, error formatters). The
`*Deps` shape from step 2.3 is the linchpin — get it wrong and every
per-app handler refactors twice. Keep `*Deps` minimal at first; add
fields as needed.

---

## Phase 3: Unify the generic CRUD helpers

**Why after Phase 2:** Right now `arabica_crud_generic.go` and
`oolong_crud_generic.go` look similar but call different store APIs
(arabica uses typed `store.CreateBean`-style methods; oolong uses
generic `putRecord(nsid, ...)`). Unifying them requires either pushing
arabica onto the generic putRecord path or pushing oolong onto typed
methods. Either way, with both sets of handlers already isolated in
their own packages (Phase 2), the unification becomes a focused diff
rather than a sprawling one.

**Scope:** ~200 LOC of dedupe; possibly refactors `AtprotoStore` API.

**Steps:**

- [ ] **3.1** — Pick a target shape. Recommended: a generic
  `CRUDHelper[Req, Model]` type with hooks for build/encode/decode,
  living in a new `internal/entities/crud/` package (since it's shared
  between arabica and oolong handler packages — neither owns it).
- [ ] **3.2** — Implement the helper. Cover create, update, delete
  paths. Reuse the existing oolong shape (generic putRecord) since it
  doesn't depend on entity-specific store methods.
- [ ] **3.3** — Migrate oolong handlers to use it. Run tests.
- [ ] **3.4** — Migrate arabica handlers to use it. This step removes
  the typed `store.CreateBean`/`CreateBrew`/etc. methods from
  `AtprotoStore` if no other callers remain — verify with grep.
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

**Why optional:** `store.go` is 1198 LOC. Large but not yet
unmanageable. After Phases 1-3 it should be smaller (codec helpers
extracted; some typed methods removed). Reassess then.

**Trigger:** Skip unless `store.go` still exceeds ~1000 LOC after
Phase 3 and developers report friction navigating it.

**Suggested split:**

- `store.go` — `AtprotoStore` struct, constructor, top-level methods
- `store_read.go` — list/get/resolve operations
- `store_write.go` — put/delete operations
- `store_session.go` — session cache integration
- `store_witness.go` — witness cache integration (or merge into
  existing `witness.go`)

**Steps:**

- [ ] **4.1** — `git mv`-style split by responsibility. No behavior
  changes. Compile, test, commit.

---

## Out of scope (deliberately)

- **Renaming `internal/entities/arabica/`** to disambiguate from the
  repo name. The directory listing confusion is real but rename churn
  isn't worth it — the audit confirmed the structure is sound.
- **Splitting session cache from witness cache wiring.** Files are
  small and serve clearly different roles already.
- **Pulling `signup` into `auth`.** Different concerns (account
  creation vs. login), keep separate.

## Execution notes

- Run each phase in a separate worktree (`jj` or `git worktree`),
  merge to main when green.
- Templ regeneration (`templ generate`) is required after any move that
  touches `.templ` imports — Phase 2 likely needs this.
- `just run` should be smoke-tested at the end of Phases 1, 2, and 3
  for both arabica and oolong app flows.
- Commit messages follow Conventional Commits per `CLAUDE.md`. Use
  `refactor:` scope; no breaking-change footers needed (no public API).
