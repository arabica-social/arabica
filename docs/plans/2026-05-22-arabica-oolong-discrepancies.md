# Arabica vs Oolong discrepancy audit

Date: 2026-05-22

This compares the in-repo state of the coffee app (`arabica`) with the sister
tea app (`oolong`). It focuses on feature parity, intentionally deferred
surfaces, and test coverage gaps.

## Executive summary

Oolong has the core record model and basic CRUD/view surfaces for tea logging,
but it is not yet at Arabica parity. The largest missing pieces are:

1. **Recipes / reusable brew plans** — Arabica has recipe records, recipe
   exploration, forking, create-from-brew, and brew-form integration. Oolong has
   no tea recipe equivalent.
2. **Onboarding / readiness flow** — Arabica has `/onboarding`, `/add`, setup
   cards, station forms, and incomplete-record prompts. Oolong has only
   `/my-tea` and no guided setup.
3. **Cafe and drink launch surface** — Oolong has cafe/drink models, record
   converters, templates, and modal code, but the descriptors are intentionally
   not registered; no routes, OAuth scopes, refresh/list-all inclusion, or
   feed/manage exposure.
4. **OG images for typed records** — Arabica registers per-entity OG handlers
   for beans/roasters/grinders/brewers/brews/recipes. Oolong entity bundles do
   not set `OGImage` handlers.
5. **Integration test parity** — current integration harness is Arabica-only.
   Oolong has entity conversion/unit tests but no end-to-end CRUD, routing,
   authz, validation, snapshot, cache, or firehose integration coverage.

## Entity / product surface parity

| Capability          | Arabica                              | Oolong                                         | Gap / note                                                                                                                                           |
| ------------------- | ------------------------------------ | ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Primary item        | Bean                                 | Tea                                            | Present in both. Oolong tea uses full-page create/edit (`/teas/new`, `/teas/{id}/edit`) rather than modal-only.                                      |
| Vendor-like entity  | Roaster                              | Vendor                                         | Present in both.                                                                                                                                     |
| Equipment entities  | Grinder, Brewer                      | Vessel, Infuser                                | Present in both, domain-specific.                                                                                                                    |
| Brew/session record | Brew                                 | Brew                                           | Present in both. Oolong uses `/brews/new` steep form and `/brews/{id}/edit`.                                                                         |
| Recipes             | Recipe                               | None                                           | Oolong has no recipe model, NSID, route, templates, popular section, fork/copy flow, or brew-form recipe application.                                |
| Cafe/drink records  | Planned for Arabica, not implemented | Models/converters/templates exist but disabled | Oolong comments say cafe/drink are deferred and not registered as descriptors, so they do not appear in feed, `/my-tea`, OAuth scopes, or API cache. |
| Social records      | Like/comment                         | Like/comment converters exist                  | Shared app wiring appends social NSIDs, but Oolong does not register descriptors; verify route behavior with integration tests.                      |

## Routing and page discrepancies

### Arabica-only routes

From `internal/routing/routing.go`, these are gated to
`cfg.App.Name == "arabica"`:

- `/onboarding`, `/add`
- `/my-coffee`, `/manage`
- `/brews`, `/brews/new`, `/brews/{id}/edit`, `/brews/{actor}/{id}`,
  `/brews/{actor}/{id}/og-image`, `/brews/export`
- `/recipes`, `/recipes/{actor}/{id}`, `/recipes/{actor}/{id}/og-image`
- `/api/brews`, `/api/manage`, `/api/incomplete-records`,
  `/api/get-started-card`, `/api/onboarding/station-form/{kind}`,
  `/api/popular-recipes`, `/api/manage/refresh`
- `/api/recipes`, `/api/recipes/suggestions`, `/api/recipes/{id}`,
  `/api/recipes/from-brew/{id}`, `/api/recipes/fork/{id}`
- `/api/modals/recipe/new`, `/api/modals/recipe/{id}`

### Oolong-specific routes

Oolong currently adds:

- `/my-tea`
- `/api/tea/refresh`
- `/brews/new`, `/brews/{id}/edit`
- `/teas/new`, `/teas/{id}/edit`
- Generic entity bundle routes for registered tea/vendor/vessel/infuser/brew
  CRUD, views, and modals where handlers exist.

### Gaps

- No Oolong equivalent of Arabica's async manage partial (`/api/manage`) or
  incomplete/get-started partials.
- No Oolong recipe routes or recipe API.
- No Oolong record OG image routes because bundle `OGImage` fields are unset.
- Oolong cafe/drink routes are absent despite modal/template/converter code
  existing.
- `HandleMyTea` comment says it fetches “all 7 entity types”, but it currently
  fetches 5 registered collections: tea, vendor, vessel, infuser, brew. This
  comment is stale unless cafe/drink are re-enabled.

## Feed / discovery discrepancies

- Arabica descriptors include feed filters for beans, grinders, brewers,
  recipes, and brews; roasters intentionally lack a feed tab.
- Oolong descriptors include feed filters for teas, vendors, vessels, infusers,
  and brews.
- Oolong cafe/drink feed content templates and descriptor bridge hooks exist,
  but descriptors are not registered, so they are unreachable.
- Arabica has popular recipe home partials and recipe explore; Oolong lacks a
  comparable discovery surface beyond the shared feed.

## API/cache/suggestions discrepancies

- Arabica `HandleAPIListAll` returns beans, roasters, grinders, brewers,
  recipes, and brews.
- Oolong `HandleOolongAPIListAll` returns teas, vendors, vessels, infusers, and
  brews only.
- Oolong `HandleTeaRefresh` refreshes those same five collections only.
- Oolong modal code references suggestions for cafes/drinks, but since
  cafe/drink descriptors are disabled, verify whether those suggestion endpoints
  should be hidden or enabled with the cafe/drink launch.

## Templates/UI discrepancies

Arabica has several UI surfaces without Oolong equivalents:

- `internal/arabica/web/pages/onboarding.templ`
- `internal/arabica/web/pages/manage.templ` plus
  `components/manage_partial.templ`
- `internal/arabica/web/pages/recipe_explore.templ`, `recipe_view.templ`
- `components/popular_recipes.templ`
- `components/brew_list_table.templ`
- profile partial tabs for coffee beans/roasters/equipment/brews

Oolong has its own tea-specific pages:

- `my_tea.templ`
- `tea_form.templ`
- `steep_form.templ`
- entity view pages for tea/vendor/vessel/infuser/brew
- about/terms pages under `internal/oolong/web/pages`, while shared routes also
  expose `/about` and `/terms` through shared handlers.

## Test coverage comparison

Command run:

```bash
go test ./... -coverprofile=/tmp/arabica-coverage.out
go tool cover -func=/tmp/arabica-coverage.out
```

Result: all tests passed. Overall statement coverage was **7.8%** across the
repository, heavily diluted by generated templates and untested handler
packages.

### Package-level app coverage

| Package                           | Coverage | Notes                                                                                                                                                 |
| --------------------------------- | -------: | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/arabica/entities`       |    59.8% | Good model/record coverage; some gaps in recipe/social validation/conversion and ref resolution.                                                      |
| `internal/arabica/handlers`       |     6.6% | A few helper/delete paths covered; most route handlers, modals, CRUD, pages are untested by unit tests. Arabica does have broad integration coverage. |
| `internal/arabica/web/components` |     0.3% | Descriptor bridge only. Templates effectively uncovered.                                                                                              |
| `internal/arabica/web/pages`      |     0.0% | No page unit coverage.                                                                                                                                |
| `internal/oolong/entities`        |    55.2% | Similar entity conversion coverage for tea records. Missing tests for `models_infuser.go`/infuser record conversion and field helpers.                |
| `internal/oolong/handlers`        |     0.0% | No Oolong handler unit tests.                                                                                                                         |
| `internal/oolong/web/components`  |     0.6% | Descriptor bridge only. Templates effectively uncovered.                                                                                              |
| `internal/oolong/web/pages`       |     0.0% | No page unit coverage.                                                                                                                                |

### Test file counts

| Area                                         | Arabica | Oolong |
| -------------------------------------------- | ------: | -----: |
| App-specific Go files under `internal/<app>` |      59 |     65 |
| App-specific `_test.go` files                |       9 |     10 |
| Entity package tests                         |       6 |      9 |
| Handler package tests                        |       2 |      0 |
| Web component tests                          |       1 |      1 |
| Page tests                                   |       0 |      0 |

### Integration test discrepancy

`tests/integration/harness.go` constructs an Arabica app (`Name: "arabica"`,
`NSIDBase: arabica.NSIDBase`). Existing integration tests exercise Arabica CRUD,
validation, authz, cache, PDS snapshots, firehose, social, suggestions,
cross-user views, handle resolution, and smoke pages.

There is no equivalent Oolong integration harness or integration test suite.
This means the following Oolong paths are currently mostly unprotected by
end-to-end tests:

- `/my-tea`
- `/api/tea/refresh`
- `/api/all` Oolong shape, if routed for Oolong in the active app
- tea/vendor/vessel/infuser/brew CRUD via generic bundles
- `/teas/new`, `/teas/{id}/edit`, `/brews/new`, `/brews/{id}/edit`
- Oolong entity public views and owner/actor handling
- Oolong combo-select/cache/suggestions behavior
- Oolong witness-cache/firehose indexing for tea NSIDs
- Oolong authz and cross-user mutation protections
- Oolong validation failures preserving no records
- Raw PDS snapshot compatibility for tea record schemas

## Recommended discrepancy backlog

### P0: Decide intended Oolong scope

- Decide whether cafe/drink are v1, post-v1, or should be removed/hidden more
  completely until launch.
- Decide whether Oolong needs a tea recipe/planned-steep equivalent or
  intentionally omits recipes.
- Decide whether Oolong should have per-record OG images at launch.

### P1: Bring tests to minimum parity

- Add an Oolong integration harness variant that constructs `Name: "oolong"`,
  `NSIDBase: oolong.NSIDBase`, and `entities.AllForApp(oolong.NSIDBase)`.
- Port Arabica integration patterns for Oolong:
  - CRUD lifecycle for tea/vendor/vessel/infuser/brew.
  - Authz mutation denial for another user's Oolong records.
  - Validation failure cases for Oolong forms.
  - Snapshot tests for raw PDS records.
  - Smoke tests for `/my-tea`, tea/steep forms, public views, and modal
    endpoints.
  - Cache/fallback tests for Oolong collections.
- Add handler unit tests for `internal/oolong/handlers`, especially
  `HandleMyTea`, refresh, generic CRUD wrappers, and view handlers.

### P2: Resolve disabled cafe/drink state

If cafe/drink should ship:

- Register descriptors in `internal/oolong/entities/register.go`.
- Add them to route bundles.
- Include NSIDs in refresh/list-all/OAuth scopes/firehose config as needed.
- Add `/api/cafes`, `/api/drinks`, modal, view, suggestions, and combo-select
  coverage.
- Add integration snapshots and lifecycle tests.

If cafe/drink should remain deferred:

- Keep descriptor omission, but remove or clearly isolate unreachable
  modal/template code from launch surfaces.
- Fix stale comments claiming 7 entity types.
- Ensure suggestions and client cache cannot expose disabled entities
  accidentally.

### P3: Product parity decisions

- Add Oolong onboarding or a tea-specific first-run checklist.
- Add Oolong manage partial or keep `/my-tea` as the canonical management
  surface and document that choice.
- Add Oolong OG image handlers or explicitly accept no typed record OG images.
- Decide on Oolong recipe/discovery: no-op, tea-session templates, steep
  recipes, or curated popular steeps.

### P4: Low-risk cleanup

- Update stale comments in `HandleMyTea` and descriptor bridge around disabled
  cafe/drink behavior.
- Add missing tests for Oolong infuser conversions and model validation helpers.
- Consider package-level coverage targets for entity packages and handler
  helpers.

## Second pass: frontend and functional gaps

This pass looked more narrowly at templates, client JS, page flows, and user-facing copy. The gap is more substantial than the first backend/entity audit suggested: Oolong has enough shared plumbing to render pages, but many product-level front-end behaviours are either missing, coffee-branded, or wired only for the happy path.

### High-impact frontend gaps

#### 1. Oolong has no guided first-run or readiness gate

Arabica prevents a dead-end first brew through a real setup journey:

- Authenticated home checks readiness and shows “finish setup” before brew logging.
- `/onboarding` and `/add` guide the user through brewer, roaster, and bean creation.
- `/api/get-started-card`, `/api/incomplete-records`, and `/api/onboarding/station-form/{kind}` support progressive setup and repair.
- `internal/web/components/incomplete_records.templ` nudges users to complete sparse records.

Oolong only shows `Log Steep`, `My Tea`, and `Profile`. There is no readiness check that a user has at least a tea before logging a steep, no setup sequence for tea/vendor/vessel/infuser, and no incomplete-record repair path. A new Oolong user can hit `/brews/new` immediately and encounter a form whose required tea picker has no obvious pre-created options unless inline creation works perfectly.

#### 2. Oolong management is a static page, not the richer Arabica manage system

Arabica's `My Coffee` page is backed by `managePage()` and HTMX partials:

- Separate async brew list and manage partial loaders.
- Refresh events update the page without a full reload.
- Loading skeletons per tab.
- Incomplete record refresh integration.
- Recipe tab and recipe actions.

Oolong's `My Tea` renders all records server-side and toggles tabs locally. This is simpler, but it means:

- No async partial equivalent to `/api/manage`.
- No load-more pagination for large steep histories.
- Delete swaps remove cards locally, but counts and empty states do not recalculate.
- Sync from PDS reloads the whole page.
- Empty states are generic (“No teas yet.”), lacking Arabica's action-cue copy and setup guidance.

#### 3. Oolong profile is read-only and lacks social fidelity

Arabica profile content is HTMX-loaded, updates stats through `profile-stats.js`, and renders brew cards with real social counts/state through the shared feed/action pipeline.

Oolong profile renders all records upfront, which is fine for small data sets, but it currently has functional limitations:

- No profile pagination or lazy loading.
- Steep action bars render `SubjectURI` but no `SubjectCID`, so likes are unavailable in `ActionBar` for profile steep cards.
- Comments/likes are noted as zeros in comments, not fetched social aggregates.
- `IsAuthenticated` is set from `props.IsOwnProfile` on profile steep action bars, so a signed-in non-owner cannot like/report from another user's Oolong profile even if the shared action system would otherwise support it.
- Owner-only editing is pushed to `/my-tea`; that may be intentional, but it is a UX divergence from Arabica's richer owner action affordances.

#### 4. Shared feed copy still says Arabica for Oolong

`internal/web/pages/feed.templ` is shared, but its helper copy is hardcoded:

- `getFeedItemShareTitle` falls back to `Arabica`.
- `getFeedItemShareText` says `Check out this {type} by {name} on Arabica`.
- Empty feed state uses a coffee emoji and tells the user to “log your first brew”.

Because Oolong uses the same feed template, Oolong shares can leak Arabica branding and coffee language.

#### 5. Shared layout metadata is coffee-branded for Oolong

`internal/web/components/layout.templ` uses app-aware brand name for titles/site name, but default metadata is still coffee-specific:

- `ogDescription()` fallback: “Track your coffee brewing journey...”
- `<meta name="description">`: “coffee brew tracking app” and “brewing data” for every app.
- Theme storage key is `arabica-theme`; harmless technically, but reinforces that app identity was only partially split.
- Favicon/touch icons are shared coffee-oriented assets unless the static files are already neutral.

Oolong record view handlers set some OG fields, but the default site metadata remains Arabica/coffee-oriented.

#### 6. Header and login modal are only partly app-aware

The header selects Oolong create-menu items, but several user-facing pieces are still inherited from Arabica:

- Header logo always uses the coffee emoji: `☕ {brand}`.
- Login modal says “use it across Arabica, Bluesky, Leaflet...” even on Oolong.
- `WelcomeLoginCard` in shared components has the same Arabica copy.
- Authenticated Oolong header omits Recipes, correctly, but there is no direct “Explore” or discovery destination replacing it.

This makes Oolong feel like a themed Arabica instance rather than a first-class tea app.

#### 7. Authenticated home page still shows an Arabica about card on Oolong

`HomeContent` always renders `components.AboutInfoCard()` for authenticated users. That card is hardcoded:

- Heading: `About Arabica`
- Coffee brew/beans/equipment language
- “Own your coffee brewing history”

Unauthenticated Oolong home has tea-specific hero copy, but authenticated Oolong home still gets a coffee-specific informational card.

#### 8. Oolong full-page forms are less capable than Arabica brew forms

Arabica brew form includes a mature recipe mode, prerequisite flow, richer method-specific sections, and no-create combo-selects where prerequisites should be explicit.

Oolong steep form is much thinner:

- No recipe/template mode.
- No multi-infusion/pour/step capture, despite tea often needing repeated infusions.
- No session-style fields such as rinse, infusion count, per-infusion time, or notes per infusion.
- Required Tea picker allows inline creation, which is convenient but also masks the missing onboarding/readiness flow.
- Form errors are generic `serverError` blocks; there is no field-level validation affordance beyond browser required fields.

This is probably the biggest domain functionality gap: Oolong can log a single steep, but not the richer tea session shape implied by gongfu or multi-infusion tea workflows.

#### 9. Combo-select has Oolong remnants and unreachable entity affordances

`combo-select.js` includes `oolongBrewer` and `oolongRecipe`, but current Oolong descriptors/routes expose vessel/infuser/brew/tea/vendor, not brewer or recipe. The shared registry also includes `cafe`, and Oolong modal templates reference cafe/drink combo-selects, while cafe/drink descriptors remain disabled.

Risks:

- Dead registry entries make it harder to know what is actually shippable.
- Disabled cafe/drink UI code can drift from real endpoints.
- Inline creation for Oolong depends on `/api/data` cache shapes and generic CRUD behaviour that currently has no Oolong integration coverage.

#### 10. Record OG images are absent, but Oolong still has social share buttons

Oolong view pages have share buttons through `EntityViewLayout` and `ActionBar`, and handlers provide some OG metadata. But unlike Arabica, route bundles do not set per-record `OGImage` handlers. Shared social cards therefore lack the rich generated images Arabica has for brews, beans, roasters, grinders, brewers, and recipes.

This is especially visible because the UI invites sharing.

#### 11. Oolong has no equivalent to Arabica's recipe/discovery frontend

The first audit noted missing recipes. The frontend consequence is broader:

- No `/recipes` navigation equivalent.
- No popular/community curated module on home.
- No “use this recipe/template in steep” flow.
- No fork/copy affordance for community tea procedures.
- No replacement discovery surface for tea vendors, teas, or steep methods.

Oolong's only discovery is the shared community feed filter bar.

#### 12. Public/static pages are split inconsistently

Oolong has `internal/oolong/web/pages/about.templ` and `terms.templ`, but shared routing calls `h.HandleAbout` and `h.HandleTerms`. Verify those handlers actually dispatch to the Oolong pages. If not, Oolong may still be serving shared Arabica about/terms despite app-specific templates existing.

Even where Oolong-specific static pages render, shared login/about snippets still point back to Arabica copy.

### Functional parity matrix, frontend edition

| User-facing capability | Arabica | Oolong | Gap severity |
| --- | --- | --- | --- |
| First-run setup | Full onboarding + add-records flow | None | High |
| Home dashboard | Readiness-aware, incomplete records, popular recipes | Basic CTAs + feed, plus coffee about card | High |
| Manage page | Async tabs, refresh events, skeletons, recipes, brew list pagination | Server-rendered tabs, full reload sync | Medium |
| Public profile | HTMX-loaded content, stats script, social data via feed pipeline | Server-rendered, no pagination, weak social state | High |
| Feed branding | Coffee-appropriate | Shared template leaks Arabica/coffee copy | Medium |
| Record share cards | Rich OG images for records | Metadata only, no generated record images | Medium |
| Recipe/template flows | Explore, fork, create from brew, apply to brew | None | High |
| Brew/steep form depth | Rich coffee-specific params + recipes | Single steep/session-lite fields | High |
| Incomplete records | Detects and nudges repair | None | Medium |
| Empty states | More contextual in many Arabica surfaces | Mostly generic “No X yet.” | Medium |
| Cafe/drink UI | Not shipped | Partially present but unreachable/disabled | Medium |
| App-specific metadata/copy | Native | Partial; shared coffee strings remain | Medium |

### Revised recommended backlog

#### P0: Fix app identity leaks before broader Oolong launch

- Make shared feed text app-aware: empty state, share title fallback, share text.
- Make layout metadata app-aware: description, OG fallback description, favicon/icon if needed.
- Make header logo/icon app-aware; use a tea/leaf mark for Oolong.
- Replace `AboutInfoCard()` with `AboutInfoCardFor(appName)` or stop rendering it on Oolong until tea copy exists.
- Audit all shared components for “Arabica”, “coffee”, “brew”, “bean”, and coffee emoji in Oolong contexts.

#### P1: Add an Oolong activation path

- Design a tea-specific first-run flow: add a tea, optionally vendor, optionally vessel/infuser, then log first steep.
- Add readiness state for Oolong home and steep form.
- Add action-cue empty states in `/my-tea` and profile tabs.
- Decide whether inline creation is enough, or whether prerequisites should be explicit like Arabica.

#### P1: Close Oolong social/profile gaps

- Fetch social aggregates and viewer like state for Oolong profile steeps.
- Provide `SubjectCID` on profile steep action bars or hide like affordances consistently.
- Use real `IsAuthenticated` rather than `IsOwnProfile` for non-owner actions.
- Add pagination/lazy loading if profile/steep history can grow.

#### P2: Decide and design Oolong's core steep model

- Decide whether Oolong should log a single steep, a session with multiple infusions, or both.
- If multi-infusion matters, add frontend fields for infusion rows and summary display on record cards/views.
- Consider a “steep template” or “session recipe” feature instead of copying Arabica recipes directly.

#### P2: Bring My Tea closer to product quality

- Update counts and empty states after HTMX deletes.
- Add load-more or paging for steeps.
- Add richer section CTAs and hints.
- Consider whether to reuse the manage partial pattern or keep server-rendered tabs with targeted HTMX fragments.

#### P3: Clean dead/deferred frontend code

- Remove or quarantine `oolongRecipe`, `oolongBrewer`, cafe, and drink combo-select affordances until corresponding descriptors/routes are enabled.
- If cafe/drink are upcoming, promote them deliberately: add route bundles, visible navigation, cache inclusion, and tests.
- Fix stale comments that still say “seven entity types”.

#### P3: Add visual/social polish

- Add Oolong record OG image handlers.
- Add Oolong-specific empty-feed illustration/copy.
- Add an Oolong discovery surface to replace Arabica's recipe/popular module.

## Third pass: form design gaps

This pass focuses on form UX, information architecture, error handling, accessibility, and the domain fit of the Arabica and Oolong form surfaces.

### Summary

Arabica's forms are more mature, but both apps share a form-system debt: fieldsets and inputs are visually serviceable, yet the forms rely heavily on long stacked sections, generic labels, hidden HTMX behaviour, and global JS. Oolong inherits the structure but lacks the domain-specific form design work that would make tea logging feel intentional rather than a renamed coffee brew flow.

### Shared form-system issues

#### 1. Labels are not programmatically associated with most controls

`components.FormField` renders `<label class="form-label">` without a `for` attribute, and `TextInput`, `NumberInput`, `TextArea`, and `Select` do not emit ids. Combo-select labels also wrap no input and use repeated internal ids such as `combo-dropdown`.

Impact:

- Clicking labels will not focus the control.
- Screen readers lose reliable label/control association.
- Repeated `id="combo-dropdown"` breaks valid DOM semantics when multiple combo-selects appear on a page.
- Error summaries cannot target fields cleanly without ids.

#### 2. Required/optional semantics are inconsistent

Examples:

- Arabica brew form marks bean required, but many required relationships are implicit in server validation rather than visible form structure.
- Oolong steep form marks `Tea` and `Style` required, but section headings say “References (optional)” while `Infusion method` inside that section conditionally controls whether `Infuser` matters.
- Tea form labels “Category” as optional by omission, while other sections use `(optional)` in the legend.

Recommendation: use a consistent required/optional tag pattern per field or section, and reserve section-level “optional” only when every field inside is optional.

#### 3. Error handling is global, not field-level

Oolong full-page forms show a single `serverError` block. Arabica brew form posts to `body` and relies on full-page/server replacement. Neither pattern gives robust per-field feedback:

- No field-level error text.
- No `aria-invalid` or `aria-describedby` wiring.
- No summary that links to failed fields.
- No preservation of partial client-side state beyond what HTMX/server happens to return.

This is especially risky for combo-selects and conditional fields, where the hidden input can be invalid while the visible control looks fine.

#### 4. Form sections look alike and do not communicate progress

Both brew and steep forms use repeated bordered fieldsets with small legends. They are readable, but not very expressive:

- No step/progress sense.
- No “what matters most” emphasis beyond order.
- No summary of what has been completed.
- No tactile/brand treatment from the established design patterns: entity tints, required/optional/done tags, or state-aware CTA panels.

This is a missed opportunity because logging a brew/steep is the core ritual of both products.

#### 5. Numeric fields lack unit ergonomics

Many labels bake units into text: `Water (g/ml)`, `Temperature (°C)`, `Time (seconds)`, `Coffee Amount (grams)`, `Leaf (g)`.

Problems:

- Units are not visually aligned, so scanning a recipe is harder.
- `Water (g/ml)` mixes two units without asking the user which one they mean.
- Temperature assumes Celsius in Oolong and Arabica.
- Mobile numeric keyboards could be improved with `inputmode`, min/max, and better step defaults.

A better pattern would use inline unit pills or suffixes, plus app-level unit preferences later.

#### 6. Range rating is always biased to 5 on steep/brew creation

Both brew and steep forms default the rating slider to 5, meaning an unintentional default rating can be submitted. Tea form handles this better: rating is hidden until the user chooses “Add Rating”.

Recommendation: make brew/steep rating optional by default, matching Tea form, or clearly label 5 as an intentional default.

#### 7. Inline creation in combo-selects is doing too much

Combo-selects combine search, community suggestions, selecting existing records, creating local records, and sometimes nested creation (bean can create roaster). This is powerful, but from a form-design standpoint it creates hidden complexity:

- Required hidden inputs can remain empty while the visible query looks filled.
- Inline create details can interrupt the flow inside a dropdown.
- Oolong steep form depends on inline creation for required tea because there is no onboarding path.
- There is no strong visual difference between selecting an existing record and drafting a new one.

Recommendation: preserve combo-select for expert speed, but add clearer states: “selected”, “will create”, “needs selection”, and “community suggestion”.

### Arabica form gaps

#### 1. Brew form is powerful but cognitively dense

Arabica's brew form has recipe mode, bean/grinder/brewer refs, pours, espresso params, pourover params, tasting, and rating in one long page. It is functional, but the IA asks users to understand everything at once.

Potential improvements:

- Progressive disclosure by brew method before showing method-specific fields.
- A sticky or inline brew summary that updates as fields are completed.
- “Minimum required” vs “dial-in details” grouping.
- Make recipe mode a stronger top-level choice: “Start blank” vs “Use recipe”.

#### 2. Recipe mode creates hidden/conditional complexity

When a recipe is active, some fields collapse behind the recipe summary. That is useful, but it can obscure what will actually be saved to the brew. Users need clearer copy around:

- Which fields came from the recipe.
- Which fields are overrides.
- Whether changing a field changes the recipe or only this brew.
- How to detach from the recipe.

#### 3. Brew form prevents inline creation for required bean

The bean combo-select uses `ComboSelectConfigNoCreate`, so a user without beans must leave the brew form and add records elsewhere. Arabica mitigates this with onboarding/readiness, but the form itself could still give a helpful path when no beans exist: “Add a bean first” with a modal or link that preserves form state.

#### 4. Pours editor is visually and semantically thin

The pours editor is a small repeated pair of water/time inputs. It lacks:

- Total water reconciliation.
- Bloom vs later pour semantics.
- Drag/reorder or insert affordance.
- Validation for empty rows.
- A compact visual timeline that would match the ritual/journal design language.

### Oolong form gaps

#### 1. Steep form does not match tea's real workflow

The Oolong steep form captures a single style/method/time/temperature/water/leaf set. For many tea users, especially gongfu drinkers, a “brew” is a session with multiple infusions.

Missing concepts likely needed in the form design:

- Session vs single mug mode.
- Rinse/wash.
- Infusion rows: number, time, temperature, water, notes/rating per infusion.
- Vessel capacity vs total water.
- Leaf-to-water ratio preview.
- Brewing style presets (gongfu, western, grandpa, cold brew) that reshape the form.

Without this, Oolong's core form feels like a coffee brew form with tea labels rather than a tea-native tool.

#### 2. Oolong “Style” and “Infusion method” are ambiguous

The steep form has both `Style` and `Infusion method`. Depending on the option labels, users may not understand the distinction. There is no helper text explaining whether style is the tea preparation style, drink style, or brew context.

Recommendation: clarify taxonomy in copy and maybe rename:

- “Preparation style” for gongfu/western/cold/etc.
- “Infuser setup” for basket/gaiwan/teapot/no infuser.

#### 3. Infuser conditional logic is too narrow

The infuser picker only appears when `infusionMethod === 'infuser'`. But tea equipment relationships are more nuanced: a gaiwan, teapot, basket infuser, or grandpa glass may each imply different fields. The current conditional hides complexity rather than modeling it.

#### 4. Tea form underplays vendor/source and purchase context

Tea form has name/category/origin/harvest/vendor/notes/link/rating. It lacks common tea inventory details that affect real use:

- Amount on hand / package size.
- Open/finished state controls during edit/create.
- Purchase date, price, vendor product link distinction.
- Cultivar/sub-style in the full form, even though combo-select inline creation knows about `sub_style` and `cultivar`.
- Processing/elevation/region fields if lexicon supports or may support them.

Even if not all are in the lexicon, the form design should make the inventory vs tasting distinction clearer.

#### 5. Tea form rating is better than steep rating, but still visually isolated

The “Add Rating” toggle is a good pattern. It could be reused for brew/steep. But on Tea form it sits alone under “Notes & Rating” without context: is this rating for the dry tea, the purchase, or overall experience? Copy should clarify.

#### 6. Oolong submit behaviour is invisible

Oolong forms use `hx-swap="none"` and `PageForm()`/`SteepForm()` for redirects and errors. There is no visible saving state in the template, no disabled submit state, and no optimistic confirmation. If the request is slow, the form may feel inert.

### Modal form gaps

Oolong vendor/vessel/infuser modals are simple, which is good, but they are also not at parity with the richer combo/suggestion affordances implied elsewhere:

- Some comments still say Oolong modals are intentionally simple and suggestions will land later, while the same file now uses combo-selects in brew/cafe/drink modals.
- Modal forms rely on `ModalShell` but do not appear to share a consistent required/optional/done pattern.
- Cafe/drink modal forms exist while cafe/drink are disabled, making the design state ambiguous.

### Form design backlog

#### P0: Accessibility and semantics

- Add ids to shared form controls and `for` to labels.
- Generate unique ids for combo-select input/listbox pairs.
- Add `aria-invalid`, `aria-describedby`, and field error slots.
- Ensure hidden combo-select inputs expose invalid state through the visible control.

#### P1: Oolong-native steep redesign

- Decide whether Oolong logs a single steep, a multi-infusion session, or both.
- Introduce preparation-style presets that change visible fields.
- Add infusion rows/timeline for session mode.
- Add computed ratio/capacity summary.
- Rename/clarify `Style` and `Infusion method`.

#### P1: Better required/empty/precondition flows

- Add an Oolong readiness state before steep logging.
- In form empty states, show concrete next actions when required entities are missing.
- Make required vs optional fields visually consistent.
- Do not rely on inline creation as the only path to satisfy required prerequisites.

#### P2: Improve save/error feedback

- Add submit loading states and disabled duplicate-submit protection in templates.
- Render field-level validation errors.
- Preserve client-side conditional state after errors.
- Add success toasts or redirects that clearly confirm the saved record.

#### P2: Bring rating and units into a reusable pattern

- Make rating optional by default for brew/steep, matching Tea form.
- Use a shared rating component with “Add rating”, “Remove rating”, and clear semantics.
- Add unit suffix/pill components for grams, ml, seconds, °C/°F.
- Add `inputmode`, min/max, and step rules for mobile ergonomics.

#### P3: Make forms feel crafted, not generic

- Use entity tinting and “Required / Optional / Done” tags in form sections.
- Add a live summary panel for brew/steep forms.
- Replace repeated identical fieldsets with richer section hierarchy.
- Use action-cue helper copy that matches the ritual: “Start with the leaf”, “Tune the water”, “Capture the cup”.
