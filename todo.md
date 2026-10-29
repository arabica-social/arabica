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

From the 2026-10-29 post-cleanup review. Preserve the warm palette, display
typography, and coffee artwork; simplify redundant framing rather than adding
filler or stripping the visual identity. Suggested order: My Coffee tools,
entity-form sidebars, then mobile Explore.

- [ ] Consolidate My Coffee's remaining sidebar into compact collection tools
  - `web/src/routes/my-coffee/+page.svelte`: the dedicated right column and
    heavy divider now frame only View beans, equipment counts, and refresh
    instructions. Remove redundant summaries (especially zero counts), keep
    Refresh discoverable, and review the detached tools on mobile.
  - Review repeated logging actions across the header, page toolbar, and empty
    state; retain the empty-state action where it helps.
- [ ] Simplify entity-form sidebars and repeated instructions
  - `web/src/lib/components/{Bean,Roaster,Brewer,Grinder,Recipe}Form.svelte`:
    remove duplicate explanations, decorative eyebrow labels, and unnecessary
    completeness counters. Keep required-field guidance beside controls and
    useful live previews, such as recipe ratio.
- [ ] Make mobile Explore prioritize results
  - `web/src/routes/explore/+page.svelte`: the expanded controls occupy most
    of the first screen at 390px. Explore compact search with collapsible
    filters and an active-filter summary. Use an established icon instead of
    the magnifying-glass emoji in the empty state.
- [ ] Remove the duplicate existing-account login invitation
  - `web/src/routes/join/create/+page.svelte`: the new introduction and the
    bottom of the provider chooser both invite existing users to log in.
    Keep one contextual invitation.
- [ ] Adjust About spacing for the shorter content
  - `web/src/routes/about/+page.svelte` and its shared styles: review the large
    framed introduction and broad section gaps. Make a modest spacing pass,
    not a wholesale visual redesign.
- [ ] Distinguish general Add choices from first-time setup
  - `web/src/lib/components/GetStartedCard.svelte`: in library mode on `/add`,
    remove numbered stations and required badges that imply a setup sequence.
    Preserve meaningful prerequisites and completion states on onboarding.
- [ ] Make user-facing terminology consistent
  - Review "Coffee ledger" on profiles and "fork" versus "copy" for recipes;
    use clear coffee-domain language consistently without changing persisted
    identifiers or API contracts.
  - Remove unsupported "drinks" references in `RoasterForm.svelte` help text.
- [ ] Review and refresh stale home screenshot baselines
  - The focused 2026-10-29 Playwright run had 54 passes and 8 screenshot
    comparison failures: seven dark-mode masthead baselines and the whole-page
    home baseline still show the old composition. Review current renders in
    both themes before updating; do not blindly accept snapshots.

Check affected layouts with empty and populated content, narrow and wide
screens, and both themes. Update current feature/design documentation when
implementing changes; do not reintroduce filler to occupy removed-copy space.

## UX improvements

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
