# Entity Descriptor Refactor — Spec

**Status:** Phases 0 and 1 complete
**Author:** ptdewey
**Created:** 2026-04-25
**Last updated:** 2026-04-25

## Goal

Replace the per-entity switch/method fan-out with a single `entities.Descriptor`
registry. Adding a new entity (cafe, drink, etc.) should require ~10 edits
instead of the ~27 documented in `CLAUDE.md`. Concretely: collapse the
dozen-plus switches scattered across the feed templ, OG cards, modals,
suggestions, and routing into one lookup table.

## Background

Arabica supports seven record types today (bean, brew, brewer, grinder, like,
recipe, roaster). The "Adding a New Entity Type" checklist in `CLAUDE.md`
enumerates ~27 separate edits across as many files. Most of those edits are
mechanical scaffolding rather than meaningful per-entity logic:

- `internal/atproto/cache.go`: 12+ identical `SetX`/`InvalidateX` methods
- `internal/atproto/store.go`: parallel `GetXByRKey`/`ListX` methods that all
  follow the same witness → convert → resolve refs → fallback-to-PDS chain
- `internal/web/pages/feed.templ`: five separate switches over `RecordType`
  (card class, action text, share URL, delete URL, delete confirm copy)
- `internal/web/components/dialog_modals.templ`: a nested type+field switch
  in `getStringValue`, plus five ~340-LOC modal components that share a
  common shell but differ in field bodies
- `internal/handlers/entity_views.go`: four near-identical entity view handlers
- `internal/ogcard/entities.go`: five Draw functions sharing one structure
- `internal/firehose/index.go`: a 135-line `recordToFeedItem` switch
- `internal/suggestions/suggestions.go` + `internal/handlers/suggestions.go`:
  parallel maps of per-entity config

The pattern that ties all of these together is "given a `RecordType`, do X."
A single registry keyed by `RecordType` removes the duplication.

## What we're optimizing for

The maintainer's stated priorities, ranked:

1. **Adding new entity types** (cafe, drink, future). The 27-step checklist
   is the headline pain point.
2. **Adding/changing fields** on existing entities. Today this means editing
   ~6-8 files; the refactor can bring it down to ~4-5 in some cases.
3. **Preserving room for unique behavior** on entities like brew (multi-ref
   resolution, espresso/pourover variants) and recipe (pours, computed
   ratios). The refactor must NOT force these through a one-size-fits-all
   abstraction.

LOC reduction is **not** a primary goal. Realistic net delta after the full
rollout is ~600-1000 LOC removed minus ~300-500 LOC of new descriptor/helper
code. That's meaningful but not the headline.

## Non-goals

- Replacing per-entity record conversion (`RecordToBean`, `BeanToRecord`,
  etc.) — these carry validation and shape logic; leave them
- Code generation from lexicon JSON — too much annotation overhead for the
  win, and the lexicons are not rich enough to drive Go types + templ markup
- Restructuring `SessionCache` or the firehose pipeline — separate concerns,
  with their own audit findings to address later
- Building cafe/drink scaffolding before the abstraction lands — avoid a
  moving target
- A declarative form-spec DSL for modals (see "What we changed our minds
  about" below)

## What goes in the descriptor

Pure data and small accessors that vary across entities:

- `Type` (`RecordType`), `NSID`
- `DisplayName` (`"Bean"`), `Noun` (`"bean"`), `URLPath` (`"beans"`)
- `GetField(entity, field) (string, bool)` — for templ form prefill
- (later phases) `CardClass`, `OGAccentColor`, `SuggestionConfig`

What stays as code (NOT in the descriptor):

- Record conversion (rich, hand-written, type-safe)
- Templ form bodies (genuinely different layouts per entity)
- Validation rules
- Container-specific record accessors (e.g. `(*FeedItem).Record()`) — see
  "Design choice" below

## Design choice: descriptor describes records, containers know themselves

Earlier versions of this spec put a `Record func(*feed.FeedItem) any`
accessor on `Descriptor`. That coupled `entities` to `feed` and made
FeedItem a privileged container — but most future descriptor consumers
(OG cards, view handlers, store CRUD) operate on raw records, not feed
items.

The current shape: `entities` only depends on `lexicons` (and `models`
for field accessors). Each container that holds typed record fields
exposes its own way to retrieve a record by `RecordType`. For
`FeedItem`, that's a `Record() any` method; the small switch lives next
to the typed fields, where adding a new entity field obviously requires
updating it.

This keeps the descriptor reusable across every container without
re-design.

## Phased rollout

**Updated rollout** based on what the maintainer actually values. Phases
2-4 are deprioritized because they're invisible to day-to-day work; the
focus is on changes that affect entity addition and field-level edits.

Each phase ships independently. Stop anywhere if the win plateaus.

| Phase | Scope | Status | Approx. LOC |
|---|---|---|---|
| **0. Foundation** | New `internal/entities` package + registry. Migrate `ActionText` and `getStringValue` as proof. | ✅ done | net 0 |
| **1. Templ data switches** | Remaining four switches in `feed.templ` (card class, title, share URL, delete URL+confirm). OG card accent/label lookup. | ✅ done | −150 |
| **5'. Modal shell extraction** *(replaces old phase 5)* | Extract a shared `EntityModalShell` component (dialog mechanics, header, error display, footer/buttons). Per-entity field bodies stay as their own templ files. | planned | −100 to −150 |
| **2. Cache map** | `UserCache.Beans/Roasters/...` → `map[string]any` keyed by NSID. | deferred | −200 |
| **3. Generic store CRUD** | `Get[T](ctx, nsid, rkey)`, `List[T]` on `AtprotoStore`. | deferred | −400 |
| **4. View handler unification** | Collapse simple entity view handlers; brew/recipe stay bespoke. | deferred | −500 |
| **6. Cleanup pass** | Modal route loop, suggestions config from descriptor, dirty-tracking → TTL, PublicClient resolver cache. | deferred | −80 |

Phases 2-4 are still legitimate refactors, but their day-to-day value is
low for the maintainer's stated priorities. They land if they fall out
cheaply during other work, or if the storage layer is being refactored
for another reason (e.g. quickslice integration).

## What we changed our minds about

### Original phase 5 (full FieldSpec consolidation): rejected

The original plan was to replace all five dialog modals with a single
`EntityModal(descriptor, []FieldSpec, entity)` driven by a declarative
field spec. After looking at what the modals actually contain, this is
the wrong scope:

- Plain text/number/dropdown fields fit cleanly into FieldSpec (~70%)
- The bean modal's roaster picker (~80 lines of Alpine state) doesn't
- The brewer modal's espresso/pourover conditional sections don't
- The bean rating slider (stateful Alpine widget) doesn't

To absorb the bespoke widgets, FieldSpec would need predicates, slots,
init-state passthrough, and raw-templ escape hatches. At that point
it's a forms DSL, debugging means reading the spec interpreter, and
adding a field requires reasoning about how the renderer interprets
your spec.

### Replacement: modal shell extraction (phase 5')

Extract only the cross-cutting consistency layer: dialog open/close
mechanics, header, validation error display, footer with cancel/submit
buttons, dirty-tracking. Each entity's modal still has its own templ
file with its own field body — the body is where editing actually
happens, and it stays readable as plain templ.

Trade-offs vs the rejected design:

- LOC saved: ~100-150 instead of ~300, but **zero DSL risk**
- Adding a new entity: copy an existing modal, change the fields. Same
  friction as today for the body, but the shell is free.
- Cross-cutting modal changes (e.g., new error display): single edit,
  same as the rejected design.
- Field changes: same friction as today (edit the entity's modal templ
  directly). The "field changes get easier" pitch was always weak.

## Risks

- **Templ ergonomics**: descriptors return primitives, not templ
  components. Switches that dispatch to *different rendering* must
  stay (we only flatten *data* switches).
- **Over-applying the abstraction**: some entities (brew, recipe) are
  legitimately different. The discipline is to leave them bespoke,
  not strain the descriptor to fit them.
- **Refactor stalling**: phases 0-1 + 5' ship the real win even if 2-4
  never happen. Don't gate the value on the deferred phases.

## Success criteria

- Phase 0 lands with no behavior change and tests green. ✅
- Adding a hypothetical 8th entity (cafe) requires touching strictly
  fewer files than the current checklist documents.
- The 27-step checklist in `CLAUDE.md` shrinks to reflect the new
  ceiling after each phase.
- No new abstractions are introduced beyond the descriptor, the modal
  shell, and (later, if we get there) the generic CRUD helper. We are
  collapsing duplication, not building a framework.
- Brew and recipe view/modal code remains bespoke and uncomplicated by
  the descriptor system.

## Decisions made

1. **Package location**: `internal/entities/` ✅
2. **`Get` return shape**: `*Descriptor` (nil on miss) ✅
3. **Descriptor scope**: record-centric, not container-centric. `entities`
   does not import `feed`. Containers expose their own record accessors. ✅
4. **Like registration**: skipped (no entity page). Revisit if a feed
   migration needs it.
5. **Cafe & drink**: defer registration until phase 1 ships. Adding them
   should be the smoke test for the refactor.
6. **Phase 5**: rejected as originally specified. Replaced with shell
   extraction (phase 5'). ✅

## Related work

- `docs/cafe-and-drinks.md` — upcoming entity types this refactor unblocks
- `docs/quickslice-implementation-plan.md` — separate read-path refactor;
  composes cleanly with this one
- `docs/design-audit.md` — broader audit notes
- `docs/plans/2026-04-25-entity-descriptor-phase-0.md` — phase 0 plan
- `docs/plans/2026-04-25-entity-descriptor-phase-1.md` — phase 1 plan
