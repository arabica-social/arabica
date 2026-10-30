# TODO

Unimplemented items from `docs/road-to-v1.md`, verified against the codebase on
2026-07-26, plus subsequent review follow-ups. Already-done items are listed at
the bottom for reference.

## Breaking lexicon changes

- [ ] Switch `coffeeAmount` precision in brew to decigrams (BREAKING)
  - `lexicons/social/arabica/alpha/brew.json` still describes `coffeeAmount`
    as "Amount of coffee used in grams" with `minimum: 0`. Stored numbers would
    increase by 10x. (Recipe already uses tenths-of-grams; brew does not.)
- [ ] Move `waterAmount` out of brew top-level into method params (BREAKING)
  - `brew.json` still has a top-level `waterAmount`; `pouroverParams`/
    `espressoParams` exist but do not own the water amount. `recipe.json` also
    still has a top-level `waterAmount`.
- [ ] Improve method-specific brew params
  - Espresso/pourover params exist, but water amount has not moved into them and
    other method-specific params are not fleshed out.

## Recipe system overhaul (probably BREAKING)

- [ ] Brainstorm ways to make recipes more modular
- [ ] Investigate partial recipes (use multiple recipes in one brew — e.g. one
      for brewer+pours, one for bean+grinder+grind size)
- [ ] Recipes should support non-pourover methods better
  - `recipe.json` is still coffee/water/pours-centric with only a `brewerType`
    fallback string.
- [x] Cross-user "Use this recipe in brew" link on brew view page
  - Fixed: `recipeViewURL` resolves via `brew.recipe_obj.author_did || owner`,
    and the recipe detail "Use in Brew" link passes `recipe_owner` to the brew
    form, which loads the recipe from the correct owner.

## New lexicons

- [ ] Cafe and drink lexicons
  - No `cafe`/`drink` lexicons exist under `lexicons/`. Deferred per AGENTS.md.
  - [ ] Decide whether drink is an extension of brew (for cafes and maybe home
        milk/other-drink logging).

## Design assets

- [ ] Replace placeholder design assets
  - `static/` still serves placeholder SVGs (`icon-placeholder.svg`,
    `favicon.svg` is 197 bytes, `icon-192.svg`/`icon-512.svg` are ~200 bytes).
    No hero/banner assets exist.
  - [ ] Make a new logo
  - [ ] Other graphical assets (hero, banner, etc.)
  - [ ] Replace the bean-variety / tea-leaf icon (currently the generic `leaf`
        icon, flagged as inadequate)

## Discovery & social

- [ ] Explore: personalized recommendations
  - `/explore` and the witness-backed Explore index exist, but there is no
    "things other people rated high that you might like based on your history"
    ranking. Current Explore is filter/sort-driven, not recommender-driven.
- [ ] Follows (social graph) + alternative following feed
  - No follows collection, follow store, or following feed exists. Open
    questions: reuse the Bluesky social graph? Allow importing it? Custom feeds
    (probably post-v1).

## Brew-form / onboarding UX

- [ ] Make the "missing roaster" warning dismissible (or replace the approach)
  - Beans already allow a nil roaster; onboarding `CheckBrewReadiness` gates
    initial brew logging on owning at least one roaster. But there is no
    dismissible per-bean "missing roaster" nudge in the SPA. The road-to-v1
    suggests removing roaster/bean creation from the brew form (brew form now
    uses `EntityCombo` pickers, no inline creation — done) and adding an
    onboarding flow / warnings that prevent brew creation when a user has no
    beans. The dismissible-warning piece remains.

## Comments

- [ ] Make comment threads look nicer
- [ ] Allow threads deeper than the current limit
  - `CommentSection.svelte` only renders a Reply button when
    `comment.depth < 2`, capping reply depth. The firehose index supports
    deeper nesting; the UI does not.
- [ ] Make comments collapsible
  - No collapse/expand behavior in `CommentSection.svelte`.

## Copy and UI cleanup follow-ups

Implemented from the 2026-10-29 review on 2026-10-30. The warm palette,
display typography, and coffee artwork are preserved.

- [x] Consolidate My Coffee's sidebar into compact collection tools
  - `web/src/routes/my-coffee/+page.svelte`: Refresh sits beside collection
    navigation; the equipment-summary/troubleshooting rail is removed. Refresh
    disables while pending and reports failures without replacing records.
    Page actions and the useful first-brew empty-state action remain.
- [x] Simplify entity-form sidebars and repeated instructions
  - `web/src/lib/components/{Bean,Roaster,Brewer,Grinder,Recipe}Form.svelte`:
    removed explanatory rails, completeness counters, and repetitive copy.
    Required indicators, validation, and meaningful field help remain. Live
    recipe ratio is beside measurements, which share a row on desktop.
- [x] Make mobile Explore prioritize results
  - `web/src/routes/explore/+page.svelte`: compact search/type/sort controls,
    collapsed additional filters, applied-filter summary, and established icon.
    Filtered direct loads, reset, failed searches, and stale pending responses
    are covered without changing API parameters.
- [x] Remove the duplicate existing-account login invitation
  - `web/src/routes/join/create/+page.svelte`: kept the contextual top Log in
    invitation and removed the repeated invitation below provider choices.
- [x] Adjust About spacing for the shorter content
  - `web/src/routes/about/+page.svelte`: page-scoped spacing removes stacked
    prose margins; the framed introduction, texture, and warnings remain.
- [x] Distinguish general Add choices from first-time setup
  - `web/src/lib/components/GetStartedCard.svelte`: library mode has equal
    choices without numbered stations or setup badges; onboarding retains
    prerequisites and completion states. Removed redundant countdown/ready copy.
- [x] Make user-facing terminology consistent
  - Removed "Coffee ledger" on profiles and unsupported "drinks" in roaster
    help. Recipe UI consistently says "copy"; persisted fields and API names
    such as `fork_count` and `most_forked` remain unchanged. See the glossary.
- [x] Review and refresh stale home screenshot baselines
  - Reviewed both themes and refreshed seven dark-mode mastheads and the
    whole-page home composition, plus changed collection, bean-form, and
    recipe-catalog baselines. The baseline-update run passed all 50 tests.

Final verification: 659 unit tests passed; Svelte check reported zero errors
and warnings; 87 targeted browser/layout/snapshot tests passed without updating
snapshots. The responsive matrix covers 390px and 1280px in both themes, empty
and populated collections, filters/reset, forms and live ratio, Add/onboarding,
and account creation. The full browser run's separate pagination failure is
tracked below. Preserve those states when revising these surfaces.

## UX improvements

- [ ] Investigate Community feed pagination's duplicate-key error
  - The 2026-10-30 full browser run passed 122/123 tests; `view-feed.spec.ts`
    failed after Load more with Svelte `each_key_duplicate`. The home feed
    append/render path was unchanged by the copy/UI follow-ups. Reproduce with
    enough indexed records to guarantee a second page and inspect overlapping
    `subject_uri` values before choosing a fix; do not mask the browser error.
  - Read-only diagnosis: `internal/firehose/feed_reader.go` uses timestamp/URI
    cursors but orders SQL by timestamp alone. Equal-timestamp page overlap is
    a hypothesis to test, not a confirmed root cause of this run. Also inspect
    the separate cursor/truncation boundary in `internal/feed/service.go`.

- [ ] My coffee page (`web/src/routes/my-coffee/+page.svelte`)
  - [ ] Brew cards: clicking the bean name should view the bean (currently the
        whole card links to the brew; the bean name is plain text, not a link)
  - [ ] Roaster name on bean/brew cards should link to the roaster view page
        (currently plain text)
  - [ ] Show and make clickable the `link`/`website` fields for bean, grinder,
        and brewer on these cards (FeedCard in Explore shows them; my-coffee
        cards do not)
- [ ] Profile page (`web/src/routes/profile/[actor]/+page.svelte`)
  - [ ] Make click targets consistent across entity types (brew/bean cards use
        whole-card links; roaster cards link only the name)
  - [ ] Roaster name on bean/brew cards is not clickable
  - [ ] Show links
- [ ] Cards: "more actions" menu gets cut off by the top of a card
  - `.action-menu` is `position: absolute; right: 0` with no explicit vertical
    positioning; verify it doesn't clip when the action bar sits high on a card.

---

## Already implemented (do not redo)

- [x] Optional `link` fields on bean, grinder, brewer lexicons; `website` on
      roaster — all present in `lexicons/social/arabica/alpha/*.json`.
- [x] Drop BoltDB — `bbolt`/`boltdb` no longer in `go.mod`; OAuth sessions now
      stored in SQLite via `internal/atproto/oauthsqlite` (wired in
      `internal/atplatform/server/server.go`). Join flow is OAuth-based account
      creation, not BoltDB-backed join requests.
- [x] Explore page — `/explore` route + `internal/explore` +
      `internal/firehose/explore_index.go` + `internal/arabica/handlers/explore*`.
- [x] Improve the about and atproto pages — `web/src/routes/about` and
      `web/src/routes/atproto` have full, non-placeholder content.
