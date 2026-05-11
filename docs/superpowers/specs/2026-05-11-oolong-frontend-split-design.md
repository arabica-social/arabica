## Oolong Frontend Split — Design

**Status:** Active
**Author:** ptdewey (with Claude)
**Created:** 2026-05-11

## Goal

Carve `internal/web/` into shared shells (stays in `internal/web/`) and
per-app entity-specific templates (`internal/arabica/web/`, future
`internal/oolong/web/`), so oolong can ship a frontend that reuses
arabica's shells and contributes its own entity pages and record
components.

After this work, `internal/web/` contains only entity-agnostic templ
files; arabica- and oolong-specific UI lives next to its `entities/`
and `handlers/` packages.

## Relationship to tea-multitenant-refactor

This spec is the deferred templ split from Phase H of
`docs/tea-multitenant-refactor.md` — that phase split `cmd/`, `entities/`,
`atproto`/`firehose`/`feed`/`handlers` along app lines but left the templ
tree untouched. We diverge from the original tea-refactor plan in two
places:

1. **Shared shells stay in `internal/web/`**, not `internal/atplatform/web/`.
   The 49 existing import sites keep pointing where they already point;
   only files that move *out* (to `internal/arabica/web/`) need import
   updates. A later rename to `internal/core/web/` (or back to
   `atplatform/`) is a single `git mv` away.
2. **Two app web packages, not three**. There is no `atplatform/web/`
   intermediate package — the shared package is just `internal/web/`.

## Non-goals

- No new shared frontend assets between apps. Each app embeds its own
  `static/` overlay and CSS bundle (already in place via
  `assets/css/themes/{arabica,oolong}.css`).
- No `internal/core/`-style rename. If we want it later we do it
  mechanically; right now it would 3× the diff size for no behavior win.
- No oolong page authoring in scope of this refactor — Phase 4 is
  net-new oolong UI work and gets its own spec when we get there.
- No behavior change for arabica. Phases 1-3 are net-zero on the user
  experience; CI stays green throughout.

## Architecture

### Final layout

```
internal/web/                  ← shared
  components/                  layout, header, footer, action_bar, modal_shell,
                               comments, forms, icons, buttons, card, avatar,
                               combo_select (markup), incomplete_records,
                               generic bits of shared.templ
  pages/                       feed, profile, notifications, settings, atproto,
                               create_account, join, notfound, manage (shared
                               shell — see "manage / my_coffee" below)

internal/arabica/web/          ← coffee-specific
  components/                  dialog_modals, entity_tables, record_bean,
                               record_brew, record_grinder, record_brewer,
                               record_recipe, record_roaster, record_cafe,
                               record_drink, profile_brew_card,
                               popular_recipes, BeanSummary
  pages/                       home, about, terms, admin, my_coffee (route
                               wrapper around shared manage shell),
                               bean_view, brew_view, brewer_view,
                               grinder_view, roaster_view, brew_form,
                               brew_list, recipe_view, recipe_explore

internal/oolong/web/           ← tea-specific (Phase 4, out of scope)
  components/, pages/          mirror of arabica/web/ for tea entities
```

### Entity dispatch — descriptor render hooks

The remaining type-switches in `internal/web/pages/feed.templ` (lines 217,
482, 516) and equivalent sites in `profile.templ`, `manage.templ` are the
only thing keeping those pages from being entity-agnostic. We extend
`entities.Descriptor` with templ-component function pointers so the shells
dispatch through the registry instead of switching on `RecordType`:

```go
type Descriptor struct {
    // ...existing fields (Type, NSID, DisplayName, Noun, URLPath,
    //   FeedFilterLabel, GetField, RecordToModel)...

    // RenderFeedContent returns the entity-specific card content for
    // feed.templ. nil means the entity does not appear in the feed.
    RenderFeedContent func(item *feed.FeedItem) templ.Component

    // RenderManageTable returns the table body for the manage page's
    // tab. nil means the entity does not get a manage tab.
    RenderManageTable func(props ManageTableProps) templ.Component

    // RenderDialogModal returns the create/edit modal for this entity.
    // nil means inline-only or no modal.
    RenderDialogModal func(entity any, refs DialogModalRefs) templ.Component
}
```

Each app's `register.go` (e.g. `internal/entities/arabica/register.go`)
populates these fields with references to its own per-app
`web/components`. Shared shells like `feed.templ` then call
`@desc.RenderFeedContent(item)` with no per-app imports.

This matches the existing descriptor pattern (`GetField`, `RecordToModel`
are already function pointers on the descriptor). Tradeoff: templ
components become reachable only through function-pointer indirection,
which slightly hurts grep-ability — accepted as the cost of removing
hard-coded entity knowledge from `internal/web/`.

The exact set of render hooks is empirical: we add a hook only when a
switch site appears in a file we want to share. If `profile.templ` and
`manage.templ` need additional hooks (e.g. `RenderProfileCard`), those
get added during phase 2 as the migration encounters them.

### Manage / my_coffee shell

`internal/web/pages/manage.templ` (249 LOC) and `my_coffee.templ`
currently coexist — `my_coffee.templ` is arabica's renamed,
reorganized version of the manage UI (entity tabs + a "Brews" primary
tab). Phase 2 consolidates these:

- The **shared shell** lives at `internal/web/pages/manage.templ` and
  is descriptor-driven: tab list comes from `App.Descriptors`
  (filtered by which descriptors have `RenderManageTable != nil`), tab
  content comes from `desc.RenderManageTable(...)`. Page title comes
  from `BrandConfig` (e.g. "My Coffee" / "My Tea").
- Arabica's `internal/arabica/web/pages/my_coffee.templ` is a thin
  wrapper (or just a route to the shared shell with the arabica
  `App`). The page name "My Coffee" remains the arabica-facing label;
  oolong analogously gets "My Tea".
- The old `manage.templ` and `my_coffee.templ` arabica content gets
  unified into the new shared shell + arabica wrapper. No double
  implementation survives.

### Combo-select

`internal/web/components/combo_select.templ` markup is entity-agnostic
and stays in `internal/web/components/`. The accompanying Go config
helper `ComboSelectConfig` is coffee-coupled and either (a) moves to
`internal/arabica/web/components/` alongside arabica's record_*.templ
files, or (b) becomes descriptor-driven (descriptor exposes
`ComboSelectConfig()` returning the Alpine.js config struct). Decision
deferred to Phase 2 implementation — option (b) is cleaner if multiple
combo-select sites exist; (a) is fine if there's only one wiring
location per app.

### Brand-driven page text

Phase G of the tea-refactor already threaded `BrandConfig` through
`LayoutData.BrandName/BrandTagline` and `HeaderProps.BrandName`. No
additional brand wiring is needed for the shared shells — they read
`BrandConfig` from `LayoutData` exactly as today.

## Phased rollout

| # | Phase | Goal | Commit |
|---|-------|------|--------|
| 1 | Descriptor render hooks | Add `RenderFeedContent`, `RenderManageTable`, `RenderDialogModal` (and any others discovered) to `entities.Descriptor`. Arabica's `register.go` wires them to existing components in `internal/web/components/`. Convert switches in `feed.templ`, `manage.templ`/`my_coffee.templ`, and `dialog_modals.templ` callers into dispatch. Zero behavior change for arabica. | 1 |
| 2 | Carve out `internal/arabica/web/` | Create `internal/arabica/web/{components,pages}/`. Move all coffee-specific templ files (dialog_modals, entity_tables, all record_*, BeanSummary extracted from shared.templ, profile_brew_card, popular_recipes, all entity view pages, home, about, terms, admin, my_coffee, brew_form, brew_list, recipe_*). Consolidate `manage.templ` into the shared shell and have arabica's `my_coffee.templ` use it. Update 49 import sites. Run `go vet` + `go build` + tests green. | 1 |
| 3 | (Optional) Cleanup | Verify `internal/web/` contains only entity-agnostic files. Delete dead code. Update CLAUDE.md to reflect the new layout. | 1 |
| 4 | Author `internal/oolong/web/` | Out of scope for this spec — oolong entity templates, record components, view pages, and per-app shells (home, about, manage wrapper). Builds atop everything above. | separate spec |

Each numbered phase ships as one commit. Phases 1-3 are arabica-only and
preserve full test green.

## Success criteria

- `internal/web/` contains no `lexicons.RecordType` switches and no
  imports of `internal/entities/arabica` or `internal/entities/oolong`.
- `cmd/arabica` builds, all tests green, no visible UX changes (manual
  smoke: feed renders, manage tabs work, entity views render, modals
  open).
- `entities.Descriptor` carries the render hooks needed for the shared
  shells to function via dispatch.
- The 27-step entity checklist in `CLAUDE.md` can be updated to point at
  per-app paths for templ files (no behavior change to the count, just
  the locations).
- Phase 4 (oolong frontend authoring) can begin without further
  refactor work on `internal/web/`.

## Risks

- **Phase 1 churn in `feed.templ`.** That file is hot. The switch
  conversion is mechanical but every entity needs its render hook wired
  before the switch can be removed. Mitigate: convert one entity at a
  time inside the switch, keep the switch as fallback until all entities
  are migrated, then remove the switch and the fallback in the same
  commit.
- **`shared.templ` split.** `BeanSummary` lives mixed in with generic
  helpers. We need to extract it cleanly so the remaining 600 LOC stays
  generic. Mitigate: grep callers of `BeanSummary`, extract to its own
  file in arabica, leave the rest as-is.
- **Combo-select decision deferred to Phase 2.** If we discover during
  Phase 2 that the descriptor-driven config is the wrong shape, we may
  have to redesign it mid-migration. Mitigate: implement option (a)
  first (move config to arabica), promote to descriptor only if oolong
  duplicates the wiring.
- **`templ generate` after large file moves.** Moving .templ files
  invalidates generated `_templ.go` files. Mitigate: run `templ
  generate` after each move batch; commit only after generated files
  are consistent.

## Open questions

- Whether `combo_select.go` config moves to arabica (option a) or
  becomes descriptor-driven (option b). Decide during Phase 2.
- Whether the entity-table tab list inside `manage.templ` includes
  drinks/cafes from day 1 of the shared shell, or only the original
  five (bean/roaster/grinder/brewer/recipe). Decide during Phase 2.

## Related work

- `docs/tea-multitenant-refactor.md` — parent spec (Phase H deferred
  the templ split this resolves)
- `docs/entity-descriptor-refactor.md` — descriptor pattern this
  extends
- `docs/superpowers/specs/2026-05-10-oolong-tea-lexicons-design.md` —
  oolong lexicons (prerequisite for Phase 4)
