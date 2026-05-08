# Tea Multi-Tenant Refactor — Spec

**Status:** Active
**Author:** ptdewey (with Claude)
**Created:** 2026-05-07
**Last updated:** 2026-05-07

## Goal

Enable a tea-tracking sister app ("matcha") to ship as a separate binary that
shares arabica's Go core. Both apps reuse identical infrastructure — auth,
session/witness/PDS cache stack, Jetstream firehose pipeline, feed/moderation,
likes/comments, OG card scaffolding, combo-select, suggestions — and provide
their own lexicons, descriptors, and entity-specific UI.

After this refactor, `cmd/arabica` and `cmd/matcha` differ only in the `App`
value they construct at startup. Bug fixes and cross-cutting features land
once and benefit both apps.

## Relationship to entity-descriptor-refactor

This refactor is a strict superset of `docs/entity-descriptor-refactor.md`:

- **ED Phase 0** (`internal/entities` registry) is the foundation. ✅ done
- **ED Phase 1** (templ data switches in `feed.templ`) is in progress
  independently and stays on its own track.
- **ED Phases 2-4** were deferred as not paying for themselves on
  entity-addition alone. This refactor activates them — cross-app reuse needs
  the data and handler layers to be entity-agnostic, with no escape hatch like
  "edit the brew handler directly" because there is no brew in matcha.
- **ED Phase 5'** (modal shell extraction) lands as part of Phase G here.

The honest tradeoff: re-activating ED's deferred phases is roughly 2-3 weeks of
churn through hot files. The matcha project has to justify that. We've decided
it does.

## What we're optimizing for

1. **Code reuse without forking.** One repo, one Go module, two binaries. Both
   apps import the same `internal/atplatform/...` packages. A bug fix in the
   witness cache benefits both.
2. **App startup is the only branching point.** At runtime, downstream code
   does not know which app it serves. It dispatches via the descriptor
   registry and reads from the `App` value threaded through the handler chain.
3. **Add-an-entity friction stays low for both apps.** The ~10-edit ceiling
   the descriptor refactor targets is preserved.

## Non-goals

- **Shared databases.** Each app has its own SQLite witness index and BoltDB
  session store. They can run on the same host but not the same files.
- **Shared OAuth client identity.** Each app registers as a separate AT
  Protocol OAuth client and serves its own client metadata.
- **Plugin or dynamic loading.** matcha is built from this repo's source. No
  Go plugin runtime, no shared-object wizardry.
- **Lexicon code generation.** Per ED non-goals, lexicons stay JSON +
  hand-written models.
- **Feed cross-pollination.** matcha users see matcha events; arabica users
  see arabica events. The federated feed within one app stays scoped to that
  app's collections.

## Architecture sketch

### The App struct

```go
package domain

type App struct {
    Name        string                  // "arabica", "matcha"
    NSIDBase    string                  // "social.arabica.alpha"
    Descriptors []*entities.Descriptor  // sourced from each app's register.go
    Brand       BrandConfig
}

func (a *App) NSIDs() []string         // descriptor NSIDs + .like + .comment
func (a *App) OAuthScopes() []string   // "atproto" + "repo:<nsid>" per NSID
func (a *App) DescriptorByNSID(string) *entities.Descriptor
```

### Layered package structure (after Phase H)

```
internal/
  atplatform/        ← shared across all apps
    domain/          ← App, BrandConfig
    atproto/         ← OAuth, store, cache (now generic)
    firehose/        ← consumer, generic indexer
    feed/            ← generic FeedItem, service, moderation
    database/        ← generic Store interface
    handlers/        ← cross-cutting (auth, feed, likes, comments) + factories
    web/components/  ← entity-agnostic shells (modal, feed card, combo-select)
    ogcard/          ← entity-agnostic card primitives

  arabica/           ← arabica-specific
    register.go      ← descriptor registrations + arabica's App
    handlers/        ← brew/recipe handlers (legitimately bespoke)
    web/             ← entity-specific templ pages, components, assets
    ogcard/          ← per-entity OG cards
    lexicons/        ← arabica's record types

  matcha/            ← tea-specific (sibling layout to arabica/)
    register.go
    ...

cmd/
  arabica/main.go    ← constructs arabica's App, starts server
  matcha/main.go     ← constructs matcha's App, starts server
```

We start with `internal/atplatform/` (not `pkg/`) because matcha is committed
to live in this repo. If a separate repo emerges later, promotion to `pkg/` is
mechanical.

### Hot files that change shape

- `internal/atproto/store.go` (2,239 LOC) → moves to atplatform with ~10
  generic methods; per-app entity registration handles ref resolution.
- `internal/atproto/cache.go` (297 LOC) → `UserCache.Records map[string][]any`
  keyed by NSID.
- `internal/firehose/index.go` (2,182 LOC) → `recordToFeedItem` becomes a
  registry-driven dispatch using a new `Descriptor.RecordToFeedPayload`.
- `internal/feed/service.go` → `FeedItem.Record` becomes `any` + RecordType.

## Phased rollout

| # | Phase | Status | Goal | Effort | LOC delta |
|---|---|---|---|---|---|
| A | Domain/App layer | ✅ done | Introduce `App` and thread it through startup. OAuth scopes and firehose collections flow from `App`, not constants. | 2-3 days | +150 |
| B | Finish ED phase 1 | ✅ done | Migrated remaining `feed.templ` data switches: card class block (collapsed onto `feedCardClass(item)` helper using `Descriptor.Noun`), share URL/title via `Descriptor.URLPath` + `RKey()`/`DisplayTitle()`, delete URL via descriptor + brew exception, OG card labels via `Descriptor.Noun`. | — | -150 |
| C | Cache map | ✅ done | `UserCache` typed fields → `map[string]any` keyed by NSID. Generic `SetRecords`/`InvalidateRecords` primitives. Typed wrappers retained for arabica call sites; Phase D removes them. | 2-3 days | -200 |
| D | Generic Store CRUD | ✅ done | Generic `fetchRecord`/`fetchAllRecords`/`putRecord`/`removeRecord` primitives in `store_generic.go`. All 6 entities migrated; per-entity wrappers handle only model construction + ref resolution. Typed cache wrappers removed. store.go shrank from 2239 LOC → 1424 LOC (-815). | 4-5 days | -700 |
| E | Generic feed pipeline | ✅ done | `FeedItem.Record any` replaces six typed pointer fields across `firehose.FeedItem`, `feed.FirehoseFeedItem`, `feed.FeedItem`. `recordToFeedItem` dispatches via `entities.GetByNSID` + `Descriptor.RecordToModel`; per-entity ref resolution lives in three named helpers. `Action` text comes from `Descriptor.Noun`. Templ feed.templ migrated 64 sites from typed-field reads to nil-safe accessor methods. `RKey()`/`DisplayTitle()` type-switch on `Record`. | 4-5 days | net ~+34 (LOC trade for entity-add friction reduction) |
| F | Handler/route parameterization | ◐ partial | Routes for the simple entities (bean, roaster, grinder, brewer) — view, OG image, JSON CRUD, modal partials — register via a loop over `App.Descriptors` + `Handler.EntityRouteBundles()`. `App.DescriptorByType` added. Recipe and brew stay explicit. View/OG handler internals (the per-entity `xViewConfig` builders) remain typed; matcha can ship its own bundle without touching arabica's handler internals, so the deeper consolidation can wait or stay deferred. | 5-7 days | net ~-30 (LOC), full matcha-enablement of routing |
| G | Templ shell extraction | ◐ partial | ModalShell already shipped during ED phase 5'; verified all 5 entity modals use it. Feed filter pills loop over descriptors with new `Descriptor.FeedFilterLabel` (empty hides). Brand strings (page title, header logo, footer name+tagline, og:site_name, meta description) thread through `domain.BrandConfig` via `Handler.SetBrand`/`LayoutData.BrandName/BrandTagline`/`HeaderProps.BrandName`/`FooterWithBrand`. Manage tab tables stay bespoke (each table is genuinely different); profile sections will land alongside Phase H templ split. | 3-4 days | net ~+50 (LOC) — matcha unblocked for branding |
| H | Package split + cmd binaries | pending | Move shared code to `internal/atplatform/`. `cmd/arabica`, `cmd/matcha`. Migrate `ArabicaCollections` readers to App-driven sources. | 2-3 days | net 0 |

Total: **~3-5 weeks** of refactor work. matcha's entity-specific work
(lexicons, models, descriptors, templ pages) begins after Phase H.

## Decisions made

1. **Shared-core location:** `internal/atplatform/` for now (single repo,
   atomic refactors). Promote to `pkg/` only if matcha lives in its own repo.
2. **Single Go module.** No multi-module setup. matcha imports
   `tangled.org/arabica.social/arabica/internal/atplatform/...`.
3. **Descriptor scope expands.** ED's descriptor will gain methods over the
   refactor: `RecordToFeedPayload` (Phase E), possibly `DefaultModerationPolicy`.
   Each addition is justified by a phase eliminating a specific switch site.
4. **No shared frontend assets between apps.** Each app embeds its own
   `static/` and CSS bundle. Shared `web/components` take templ inputs (`App`,
   `BrandConfig`) and per-app assets compose around them.
5. **Brand customization is config, not code.** Color palette, copy strings
   ("brews" vs "steeps"), default OG colors → `BrandConfig` on `App`.
6. **Phase A doesn't move files.** App layer threads through existing
   structure. Package movement is consolidated in Phase H to avoid repeated
   churn.

## Risks

- **Hot-file churn during refactor.** `atproto/store.go` and
  `firehose/index.go` will be touched repeatedly across phases C-F. Mitigate
  with TDD: every refactor preserves test green, and we add coverage where
  it's thin before moving fields around.
- **Refactor stalls before matcha ships.** If we land Phases A-C and then
  pause, we have less-tested code with no offsetting product win. Mitigate by
  sequencing matcha's lexicon JSONs and stub descriptors in parallel from
  Phase E onward — we should see a "matcha hello world" by end of Phase G.
- **Generics + reflection cost.** Phase D's `Get[T any]` likely needs
  reflection or per-descriptor decode functions. Test the hot path early
  (witness cache reads on a populated DB) to confirm no regression.
- **Templ ergonomics for app-themed components.** Some shells will need
  per-app slot content. Fallback is per-app component overrides; we avoid
  runtime template selection.

## Success criteria

- `cmd/arabica` builds and serves with no behavior change at any phase
  boundary.
- After Phase H, `cmd/matcha` (with stub descriptors) builds, runs, and
  refuses login because no descriptors are registered — proving the App layer
  is the only branching point.
- The 27-step entity checklist in `CLAUDE.md` collapses to ≤10 steps for both
  arabica and matcha.
- No test regressions throughout. Coverage is added where the refactor
  touches uncovered code.

## Related work

- `docs/entity-descriptor-refactor.md` — parent spec (ED phases 0/1/5')
- `docs/plans/2026-05-07-tea-phase-a-domain-layer.md` — Phase A detailed plan
- Subsequent phase plans get written when each phase begins (the codebase
  shape changes between phases; up-front detailed plans for D-H would be
  fiction).
- `docs/cafe-and-drinks.md` — entity additions that benefit from the same
  scaffolding
