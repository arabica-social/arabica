# Oolong SvelteKit SPA Port Plan

Date: 2026-07-11

## Goal

Port Oolong from its current Templ + HTMX frontend to the same embedded
SvelteKit SPA model Arabica now uses, without changing Oolong's AT Protocol
records, public URLs, authentication model, or core product behavior.

The migration must be incremental and reversible. A page becomes SPA-owned
only after its JSON contract, Svelte route, direct-load behavior, and tests are
complete. Until then, the existing Templ/HTMX route remains the production
owner.

The finished Oolong surface should:

- use the existing single SvelteKit build with runtime app detection;
- consume typed Go JSON endpoints;
- preserve OAuth, CSRF, PDS ownership, witness-cache behavior, moderation, and
  server-generated metadata;
- support Oolong's five active entities: tea, vendor, vessel, infuser, and
  brew/steep;
- have strong Go contract, Svelte component, and Playwright lifecycle tests;
- remove Oolong-specific Templ/HTMX code only after equivalent SPA behavior is
  proven.

## Scope

### In scope

- Shared SPA shell, layout, session bootstrap, navigation, theme, toast, and
  error handling for the `oolong` runtime app.
- Oolong home/feed, static pages, account creation, notifications, settings,
  moderation admin, onboarding, profile, and My Tea pages.
- Detail and backlinks pages for:
  - tea (`/teas/...`)
  - vendor (`/vendors/...`)
  - vessel (`/vessels/...`)
  - infuser (`/infusers/...`)
  - brew/steep (`/brews/...`)
- Full-page create/edit flows for tea and steep sessions.
- SPA create/edit flows for vendor, vessel, and infuser, replacing their
  current modal-only management flow.
- JSON contracts, generated TypeScript types, app-specific frontend
  configuration, and deterministic test fixtures.
- Explicit route-by-route ownership in
  `internal/oolong/handlers.Routes.SPAOwnedRoutes`.
- Route-level retirement of Oolong Templ pages, HTMX fragments, and legacy
  island mounts after cutover.

### Non-goals

- Do not enable the currently deferred Oolong cafe or drink entities.
- Do not invent an Oolong recipe/planned-steep record as part of this port.
- Do not add an Oolong Explore product surface unless it is approved as a
  separate feature.
- Do not change Oolong lexicons or record semantics merely to fit the SPA.
- Do not replace Go OAuth, CSRF, OG/head generation, static serving, PDS CRUD,
  witness cache, firehose ingestion, moderation, or notifications.
- Do not redesign the Oolong visual identity. Preserve current content and
  behavior first; polish can follow after parity.
- Do not remove shared Templ/HTMX infrastructure while any route in either app
  still depends on it.

## Current state

The migration has useful foundations but Oolong itself has not started page
cutover:

- `internal/oolong/handlers/routes.go` returns no SPA-owned routes.
- `internal/routing.PageRoutes` already provides the required explicit
  SPA-versus-legacy ownership boundary.
- The Go SPA shell already injects app identity and session data.
- The Svelte layout and header already recognize `data-app="oolong"`.
- `/api/data`, entity suggestions, shared feed JSON, social JSON,
  notifications JSON, settings JSON, and moderation JSON are available or
  largely reusable.
- Oolong CRUD mutations already return JSON for create/update, but their error
  and delete response behavior needs contract coverage.
- Oolong entity detail and backlinks pages have reusable loader configs, but
  their route bundles do not register JSON view or JSON backlinks handlers.
- Oolong lacks typed JSON contracts for My Tea, public profile, and onboarding.
- Tygo currently generates Arabica entity and handler types but not Oolong
  entity or handler types.
- Vitest and Playwright coverage is broad for Arabica but almost entirely
  absent for Oolong user journeys.
- The integration and E2E harnesses construct Arabica directly and must become
  app-selectable before they can provide equivalent Oolong proof.

The existing working copy also contains unrelated SPA test and component
changes. Implementation of this plan must preserve those changes and should be
split into focused jj changes.

## Architecture decisions

### 1. Keep one SvelteKit build

Oolong will use the same `web/` build as Arabica. The frontend reads the app
name from the server-injected session store and selects app-specific copy,
routes, entities, colors, and behavior from a typed frontend app registry.

Do not create a second `web-oolong/` project or a second compiled SPA. That
would duplicate shared components, API clients, session logic, caches, and
tests, and would allow the two products to drift again.

### 2. Introduce a real frontend app boundary

The existing runtime app detection is not enough. Many current routes and
components still hard-code Arabica paths, NSIDs, copy, cache keys, event names,
and entity assumptions.

Add an app configuration layer under `web/src/lib/app/`, with a shape similar
to:

```ts
type AppDefinition = {
  name: "arabica" | "oolong";
  displayName: string;
  tagline: string;
  libraryPath: string;
  sessionNoun: string;
  sessionAction: string;
  commentCollection: string;
  entityRoutes: Record<string, EntityRouteDefinition>;
  feedRecordTypes: string[];
};
```

This registry should own product-level facts. Shared components should accept
data or consult the active definition instead of branching on Oolong throughout
their markup.

Immediate hard-coded values to remove or parameterize include:

- `/my-coffee` redirects and delete destinations;
- `social.arabica.alpha.comment` in comment URIs;
- Arabica-only feed-card record rendering;
- backlinks collection-to-route mapping;
- profile and management NSIDs;
- Arabica terms/about/AT Protocol copy;
- the home-page My Coffee link when Oolong is active;
- cache storage keys and mutation event names where cross-app isolation is not
  already guaranteed;
- admin collection labels.

### 3. Preserve URLs and terminology

Public and mutation URLs remain unchanged. In particular:

- `/my-tea` remains Oolong's library route;
- `/teas/new` and `/teas/{id}/edit` remain tea form routes;
- `/brews/new` and `/brews/{id}/edit` remain steep form routes;
- public entity URLs remain `/{entity}/{actor}/{id}`;
- APIs remain under `/api/...`.

The backend and transport model may continue to call the record `brew`, because
that is the existing lexicon and route type. User-facing copy should call it a
“steep” or “steep session.” Avoid renaming persisted fields or NSIDs during the
frontend port.

### 4. Keep Go as the authorization and data-composition boundary

The SPA must not reconstruct authorization, profile visibility, moderation,
reference hydration, or source-selection policy in TypeScript.

Go JSON handlers remain responsible for:

- own-store versus witness-cache versus public-PDS reads;
- resolving tea vendor and brew tea/vessel/infuser references;
- owner and viewer capabilities;
- likes, comments, reports, and moderation state;
- canonical actor resolution;
- profile visibility rules;
- stable error envelopes and status codes.

Svelte components render these decisions and initiate mutations; they do not
reimplement them.

### 5. Use generated transport types, not `any`

Extend `tygo.yml` with:

- `internal/oolong/entities` →
  `web/src/lib/types/generated/oolong_entities.ts`;
- Oolong response DTOs from `internal/oolong/handlers` →
  `web/src/lib/types/generated/oolong_handlers.ts`.

Transport DTOs should use concrete Oolong model fields. Do not accept the
Arabica generator's current `any /* arabica.X */` output as the pattern for new
Oolong contracts. Frontend view-state types may remain handwritten where they
compose multiple generated DTOs.

### 6. Route ownership stays explicit

Do not add a global Oolong SPA catch-all. A route is added to
`SPAOwnedRoutes()` only in the same change that supplies:

1. a matching SvelteKit route;
2. a real Go JSON source where data is required;
3. route-loader and component tests;
4. a Go direct-request ownership test;
5. a Playwright journey or an explicitly documented reason it is covered by a
   shared journey;
6. crawler/head verification for public pages.

## JSON API work

Document Oolong additions in `docs/api/`, either by adding explicit Oolong
sections to the existing domain files or by adding focused `oolong_*.md`
contracts. The contract must distinguish shared envelopes from app-specific
record payloads.

### Existing endpoints to retain and test

| Endpoint | Use |
| --- | --- |
| `GET /api/data` | Authenticated Oolong record bootstrap and combo-select cache |
| `GET /api/feed` | Home/feed data with Oolong filter tabs and records |
| `GET /api/suggestions/{entity}` | Community suggestions and inline creation |
| `POST /api/likes/toggle` | Like mutation |
| `GET/POST /api/comments` | Comment list/create |
| `DELETE /api/comments/{id}` | Comment delete |
| `POST /api/report` | Report mutation |
| `GET /api/notifications` | Notification list |
| `POST /api/notifications/read` | Notification read state |
| `GET/POST /api/settings...` | Settings reads and mutations |
| `GET/POST /api/_mod...` | Moderation admin |

### Oolong endpoints or response contracts to add

#### Entity detail and backlinks

For every active Oolong entity, register the existing shared JSON renderer
using the same `EntityViewConfig` already used by Templ:

- `GET /api/teas/{actor}/{id}`
- `GET /api/vendors/{actor}/{id}`
- `GET /api/vessels/{actor}/{id}`
- `GET /api/infusers/{actor}/{id}`
- `GET /api/brews/{actor}/{id}`
- matching `/backlinks` endpoints

Add `JSONView` and `JSONBacklinks` handlers to each Oolong
`EntityRouteBundle`. This should reuse `RenderEntityViewJSON` and
`RenderBacklinksViewJSON`, not duplicate the loader pipeline.

#### My Tea

Add `GET /api/manage` for Oolong. Since Arabica and Oolong run as separate app
stacks, the same path can return an app-specific typed payload:

```json
{
  "did": "did:plc:...",
  "actor": "handle.example",
  "teas": [],
  "vendors": [],
  "vessels": [],
  "infusers": [],
  "brews": [],
  "stats": {
    "tea_brew_counts": {
      "at://did:plc:.../social.oolong.alpha.tea/rkey": 3
    }
  }
}
```

The response must hydrate the same references the current `HandleMyTea`
hydrates. The SPA should not join references independently.

Keep `/api/data` as the lighter app-cache endpoint. Do not overload it with
management-only statistics.

#### Public profile

Add `GET /api/profile/{actor}` for Oolong. The response should preserve the
current profile behavior:

- handle-or-DID lookup;
- canonical handle information;
- public profile identity;
- teas, vendors, vessels, infusers, and brews;
- hydrated brew references;
- `is_own_profile` and viewer capabilities;
- visibility filtering before serialization;
- stable 400/404/503/error-envelope behavior.

Do not copy Arabica's coffee-specific stats shape. Define an Oolong response
whose fields match the actual Oolong profile UI.

#### Onboarding

Add `GET /api/onboarding` returning:

```json
{
  "readiness": {
    "has_tea": true,
    "has_vendor": true,
    "has_vessel": false,
    "has_infuser": false,
    "ready": false
  },
  "teas": [],
  "vendors": [],
  "vessels": [],
  "infusers": []
}
```

Preserve the current readiness rule: tea + vendor + vessel are required;
infuser is optional. Once the SPA onboarding page is complete, replace the
HTMX station-form fragments with native Svelte drawers that submit to the
existing entity CRUD APIs.

#### Refresh

Keep `POST /api/tea/refresh`, but add a JSON response for SPA callers, for
example `{ "refreshed": true }`, while retaining the legacy `204` behavior
until `/my-tea` is cut over. Tests must verify session-cache invalidation and
witness write-through remain unchanged.

#### Form initialization and mutations

Tea and steep create/edit routes may initially load owner records from the
typed `/api/manage` or `/api/data` payload. If this becomes wasteful or makes
edit direct loads fragile, add owner-only `GET /api/{entity}/{id}` endpoints
rather than relying on stale session storage.

Before any form cutover, normalize all Oolong CRUD endpoints to the shared JSON
conventions:

- success returns the persisted typed record;
- validation uses the standard `{error, code, fields?}` envelope;
- authentication and authorization use stable codes;
- delete returns `{deleted: true}` for JSON callers;
- HTML/HTMX behavior remains available until the corresponding legacy form or
  modal is retired.

## Frontend structure

Recommended new structure:

```text
web/src/lib/app/
  definitions.ts          # AppDefinition and active-app lookup
  arabica.ts
  oolong.ts

web/src/lib/oolong/
  components/
    TeaForm.svelte
    VendorForm.svelte
    VesselForm.svelte
    InfuserForm.svelte
    SteepForm.svelte
    TeaRecord.svelte
    VendorRecord.svelte
    VesselRecord.svelte
    InfuserRecord.svelte
    SteepRecord.svelte
  types.ts                # handwritten composed view-state types only

web/src/routes/
  teas/...
  vendors/...
  vessels/...
  infusers/...
  brews/...               # dispatch or app-aware rendering where paths overlap
  my-tea/...
```

Shared chrome and behavior should remain in `web/src/lib/components`.
Oolong-specific record markup and forms should not be forced into a generic
schema-driven renderer. Share stable primitives—field wrappers, entity combo,
action bar, comments, backlinks, layout—not entire domain forms.

For routes shared by both apps, such as `/`, `/brews/...`, `/profile/...`,
`/about`, `/terms`, `/atproto`, `/notifications`, `/settings`, and `/_mod`, the
route should select app-specific content or record rendering from the active
app definition. Do not duplicate the route merely to avoid a small runtime
dispatch.

## Migration phases

## Phase 0 — Establish an Oolong test baseline

Do this before moving any route.

Tasks:

- Generalize `tests/integration.HarnessOptions` so the harness can construct
  either `arabicaapp.New()` + `coffeehandlers.Routes{}` or
  `oolongapp.New()` + `teahandlers.Routes{}`.
- Keep app-specific databases, descriptors, feed registries, NSIDs, assets,
  cookies, and SPA shell names isolated in the selected harness mode.
- Make the E2E server app-selectable rather than creating a separate copied
  server implementation.
- Add an Oolong route inventory test proving that SPA ownership starts empty.
- Add Oolong HTTP contract tests for current `/api/data` and each CRUD
  lifecycle before changing responses.
- Add raw PDS record snapshots for tea, vendor, vessel, infuser, and brew.
- Add Oolong firehose/witness tests proving each active NSID is indexed and can
  be read back.
- Record the exact green baseline commands.

Exit criteria:

- The integration harness can run the same shared conformance tests for both
  apps.
- Oolong CRUD, validation, authz, PDS serialization, and cache/index behavior
  are covered independently of Templ output.
- No Oolong page route is SPA-owned yet.

## Phase 1 — Make the shared SPA truly app-aware

Tasks:

- Add the typed frontend app registry.
- Parameterize shared navigation, redirect destinations, entity route lookup,
  comment collection, cache namespace, feed mutation events, and admin labels.
- Make home, header, footer, settings, and shared error states render correct
  Oolong branding and links.
- Update shared component tests to run table cases for both apps.
- Add an Oolong SPA-shell integration test that asserts:
  - `data-app="oolong"`;
  - Oolong title/tagline/session values;
  - API and static routes remain outside SPA page ownership;
  - unowned Oolong pages still render legacy HTML.

Exit criteria:

- Loading the SPA shell in Oolong mode does not display or navigate to
  coffee-only surfaces.
- Shared stores cannot reuse Arabica user data when the active app changes.
- Existing Arabica frontend tests remain green.

## Phase 2 — Add typed Oolong JSON contracts

Tasks:

- Add Tygo generation for Oolong models and response DTOs.
- Register JSON entity view and backlinks handlers for all five active
  entities.
- Add Oolong `/api/manage`, `/api/profile/{actor}`, and `/api/onboarding`.
- Normalize CRUD/delete/refresh response envelopes for JSON callers.
- Verify shared feed, social, notifications, settings, and admin JSON using the
  Oolong harness.
- Update `docs/api/` before or with each endpoint.

Exit criteria:

- Every data-backed page planned below has a documented, typed Go JSON source.
- `just types-check` detects Oolong model or DTO drift.
- Contract tests cover success, anonymous, unauthorized, invalid, missing, and
  backend-failure paths where applicable.

## Phase 3 — First tracer bullet: vendor detail and backlinks

Vendor is the simplest Oolong entity and should prove the full migration path.

Tasks:

- Add `/vendors/{actor}/{id}` and `/vendors/{actor}/{id}/backlinks` Svelte
  routes.
- Render through shared `EntityViewLayout`, `ActionBar`, comments, share, and
  backlinks components plus an Oolong `VendorRecord` component.
- Add crawler/head resolution for the public vendor route.
- Add only these two patterns to Oolong `SPAOwnedRoutes()`.
- Keep vendor create/edit modal routes legacy-owned for now.

Required tests:

- Go JSON contract: owner, public viewer, anonymous viewer, missing record,
  handle and DID actor.
- Vitest: all vendor fields, safe website link, action capabilities, empty
  backlinks, populated backlinks, API error state.
- Routing test: vendor detail/backlinks use SPA while vessel/tea pages remain
  Templ.
- Playwright: direct-load vendor detail, reload, navigate to backlinks, return
  to detail.
- Raw HTTP head test: title, description, canonical URL, app identity, and OG
  image behavior are no worse than the legacy page.

Exit criteria:

- The tracer bullet is deployable by itself and its rollback is removing two
  ownership patterns.

## Phase 4 — Simple equipment and vendor CRUD

Tasks:

- Port vessel and infuser detail/backlinks routes using the vendor pattern.
- Build full-page vendor, vessel, and infuser create/edit routes in Svelte.
- Preserve their existing mutation URLs and inline-create suggestion behavior.
- Update My Tea links to target the new pages only after those pages exist.
- Retain legacy modal routes until My Tea and onboarding no longer request
  them.

Required Playwright lifecycle for each entity:

1. create;
2. verify persisted record after direct navigation or reload;
3. edit;
4. verify updated persisted record after reload;
5. delete;
6. verify both page and `/api/data` no longer contain it.

Also test validation failures, another user's mutation denial, keyboard form
operation, cancel navigation, and API error display.

Exit criteria:

- Vendor, vessel, and infuser public and owner flows are SPA-owned.
- Modal partials remain only where onboarding or legacy My Tea still consumes
  them.

## Phase 5 — Tea vertical slice

Tasks:

- Port tea detail and backlinks.
- Build `TeaForm.svelte` for `/teas/new` and `/teas/{id}/edit`.
- Preserve name, category, origin, harvest year, vendor, description, link,
  rating, closed, and source-ref semantics.
- Reuse `EntityCombo` for vendor selection and inline vendor creation.
- Preserve safe external-link rendering and incomplete-field behavior.

Required tests:

- Table-driven Go validation/serialization tests for every field and boundary.
- Component tests for required name, category options, harvest-year parsing,
  rating bounds, selected vendor, inline vendor creation, server field errors,
  and edit initialization.
- Playwright tea lifecycle with reload persistence and vendor relationship.
- Public viewer and owner action tests.
- Backlinks/source-ref tests with records from a second account.

Exit criteria:

- All tea detail, backlinks, create, and edit routes are SPA-owned.
- The lifecycle proves the exact PDS record survives create and edit.

## Phase 6 — Steep session vertical slice

This is the highest-risk Oolong form and should follow the established entity
pattern, not precede it.

Tasks:

- Build an Oolong-specific `SteepForm.svelte` rather than adapting Arabica's
  coffee brew form through conditionals.
- Preserve tea, style, vessel, infusion method, infuser, temperature, leaf
  grams, water amount, time, tasting notes, rating, and created-at behavior.
- Preserve the current readiness redirect for `/brews/new`.
- Port Oolong brew detail rendering with hydrated tea, vessel, and infuser.
- Make the shared `/brews/...` route dispatch between Arabica Brew and Oolong
  Steep components by active app/record type.
- Port brew backlinks if exposed in the current route bundle.

Required tests:

- Go validation and record round-trip tests for long steep and cold brew,
  infusion-method constraints, numeric zero/omitted semantics, rating bounds,
  and missing references.
- Component tests for conditional infuser controls, numeric parsing, edit
  initialization, validation focus, server errors, and payload field names.
- Playwright create → direct-load → edit → reload → delete lifecycle.
- Playwright readiness redirect when tea/vendor/vessel setup is incomplete.
- Missing related-record rendering test; a deleted vessel or infuser must not
  make the steep page crash.
- Feed-index persistence test using deterministic polling, never sleeps.

Exit criteria:

- Oolong brew/steep view, backlinks, new, and edit routes are SPA-owned.
- A full lifecycle is proven against the real test PDS and firehose index.

## Phase 7 — My Tea and onboarding

Tasks:

- Port `/my-tea` against Oolong `/api/manage`.
- Preserve tabs, counts, empty states, refresh, create/edit/delete links, tea
  brew counts, and hydrated steep summaries.
- Port `/onboarding` against Oolong `/api/onboarding`.
- Replace HTMX station drawers with native Svelte drawers/forms using the same
  entity form components or focused compact variants.
- Refresh onboarding readiness after each successful mutation without a full
  document reload.
- Retire `/api/get-started-card` and station-form HTML fragments only after no
  route consumes them.

Required tests:

- Vitest for every readiness combination, next-step copy, optional infuser,
  tab state, counts, empty states, refresh behavior, and mutation failures.
- Playwright new-user onboarding from no records to ready state.
- Playwright My Tea refresh and all five entity sections.
- Direct-request auth tests for anonymous access.
- Test that `/brews/new` stops redirecting once required setup exists.

Exit criteria:

- `/my-tea` and `/onboarding` are SPA-owned.
- Legacy Oolong management modals and onboarding fragments have no callers.

## Phase 8 — Profile, home/feed, and shared pages

Tasks:

- Port Oolong `/profile/{actor}` using its own typed profile response.
- Make shared home/feed rendering support all Oolong feed record types and
  correct My Tea/Log Steep links.
- Verify Oolong feed filters, pagination, likes, comments, reports, deletes,
  and moderation state.
- Make `/about`, `/terms`, and `/atproto` app-aware while preserving Oolong's
  existing copy.
- Verify `/join/create`, `/notifications`, `/settings`, and `/_mod` in Oolong
  mode; remove remaining coffee-only assumptions before taking ownership.
- Add each route to Oolong `SPAOwnedRoutes()` only when its own checks pass.

Required tests:

- Two-user Playwright profile and social journeys.
- Feed tests containing tea, vendor, vessel, infuser, and steep cards.
- Filter and pagination tests using bounded index polling.
- Notifications test proving a second user's action creates a navigable Oolong
  notification.
- Static-page copy/link tests for Oolong.
- Settings persistence and theme tests under an Oolong-scoped storage key.
- Admin tests with Oolong collection labels.
- Mobile navigation and representative visual snapshots.

Exit criteria:

- Every current Oolong user-facing page has an explicit SPA or intentional
  non-SPA owner.
- No SPA page displays Arabica-only entity names, links, NSIDs, or copy in
  Oolong mode.

## Phase 9 — Route-level legacy retirement

Retire legacy code in small, evidence-backed changes.

Tasks:

- Remove Oolong Templ pages only after their route has been SPA-owned and green
  for at least one preceding change.
- Remove modal and onboarding fragments only after caller searches prove none
  remain.
- Remove Oolong-specific island mount code and generated bundles after source
  and built-asset references are gone.
- Remove HTMX attributes and middleware only where no remaining route in either
  app depends on them.
- Keep server-side metadata, OG image, auth, API, and static handlers.
- Update `AGENTS.md`, product docs, API docs, build commands, and architecture
  descriptions.
- Keep route ownership tests as permanent regression tests after legacy
  handlers disappear.

Exit criteria:

- No Oolong page rendering depends on Templ or HTMX.
- Oolong's active route inventory is fully SPA-owned except intentional Go-only
  routes such as OAuth, OG images, APIs, assets, and well-known endpoints.
- Unknown routes remain real 404s rather than receiving the SPA shell.
- Arabica remains green throughout retirement.

## Test strategy

The migration uses a layered test pyramid. Browser tests are required for real
journeys, but they do not replace faster contract and component tests.

### 1. Domain and record tests

Location: `internal/oolong/entities`.

Cover:

- validation boundaries and known enum values;
- model ↔ AT Protocol record round trips;
- omitted versus zero numeric fields;
- reference URI/rkey conversion;
- reference hydration and missing-reference tolerance;
- sourceRef preservation;
- timestamps and update semantics.

All Go tests use testify and table-driven cases where practical.

### 2. Go HTTP contract tests

Location: focused handler tests plus `tests/integration`.

For every Oolong endpoint assert:

- status and `Content-Type`;
- stable success and error envelopes;
- required fields and concrete JSON types;
- anonymous, authenticated, owner, and non-owner behavior;
- invalid rkey, missing record, invalid input, and backend failure;
- handle and DID actor resolution;
- public versus private/owner-only data;
- persisted PDS record after mutations;
- witness-cache and session-cache effects;
- legacy content negotiation until retirement.

Prefer structural assertions over brittle full snapshots. Use scrubbed snapshots
for a small set of high-value payloads: entity view, My Tea, profile, feed, and
onboarding.

### 3. Svelte unit, component, and route tests

Location: `web/tests`.

Every ported route needs tests for:

- normal rendering;
- empty state;
- loading state where visible;
- 401/403/404/422/500 or network error behavior as applicable;
- primary interaction;
- keyboard-accessible controls and focus movement;
- app-specific copy and destinations;
- no Arabica leakage in Oolong mode.

Form tests must assert the exact request method, URL, field names, values, and
post-success navigation. A test that only fills inputs is not sufficient.

Add Vitest coverage reporting after the Oolong baseline is established. Apply
thresholds to new app-registry and Oolong modules rather than using generated
files or the whole legacy tree to dilute the signal. Initial target: at least
80% line and branch coverage for new `web/src/lib/app` and
`web/src/lib/oolong` code, with critical form mapping and loader utilities at
100% branch coverage.

### 4. Playwright E2E tests

Add an Oolong project or app fixture to the existing Playwright setup. It must
boot the same compiled SPA against an Oolong-configured Go integration harness.

Required journeys:

1. shell, login/session, header, and static pages;
2. onboarding from empty account to ready state;
3. tea lifecycle;
4. vendor lifecycle;
5. vessel lifecycle;
6. infuser lifecycle;
7. steep lifecycle;
8. My Tea tabs, counts, refresh, and empty states;
9. public profile and entity detail as a second user;
10. like, comment, report, and notification flow;
11. feed filters and pagination;
12. settings and theme persistence;
13. authorization and error paths;
14. representative mobile navigation;
15. representative visual snapshots.

Rules:

- Each test or worker gets isolated accounts and record names.
- Mutations must be verified after reload or direct navigation.
- Firehose/index synchronization uses bounded polling with useful timeout
  diagnostics, never arbitrary sleeps.
- Failed API requests and unexpected browser console errors fail the test.
- A route test must assert an SPA shell marker so it cannot accidentally pass
  against the legacy Templ page.
- Social tests use two accounts.

### 5. Raw direct-request and crawler tests

For every public entity/profile route, fetch the document without executing
JavaScript and assert:

- HTTP status;
- app-specific title and description;
- canonical URL;
- OG image behavior;
- correct not-found response;
- no authenticated data leakage.

This protects social sharing and direct loads independently from Playwright's
client rendering.

## Quality gates

### Per vertical slice

A route cannot enter `SPAOwnedRoutes()` until:

- API docs and generated types are current;
- focused Go tests pass;
- Oolong integration contract tests pass;
- Svelte check and route/component tests pass;
- the relevant Playwright journey passes;
- direct-load/crawler behavior passes for public pages;
- existing Arabica tests still pass;
- a rollback consists of removing only that route's ownership entry and SPA
  files, without API/data loss.

### Per change stack

Run at minimum:

```bash
just types-check
go vet ./...
go build ./...
go test ./... -count=1
cd tests/integration && go test -tags=integration -count=1 ./...
cd web && pnpm run check
cd web && pnpm run test
```

Run the app-specific Playwright project for each slice. Before merging a phase,
run both Arabica and Oolong Playwright projects from clean databases.

The current `just ci-check` references root-level Svelte scripts and a CI file
that may not match the working tree. Before relying on it as the final gate,
repair or verify the executable recipes and check in CI that invokes the same
commands used locally.

## Suggested jj change sequence

Keep the work reviewable and rollback-friendly:

1. Oolong-capable integration/E2E harness and baseline tests.
2. Frontend app registry and shared-component parameterization.
3. Oolong Tygo types and API contracts.
4. Vendor detail/backlinks tracer bullet.
5. Vessel and infuser detail/backlinks.
6. Vendor/vessel/infuser forms and lifecycles.
7. Tea detail/form lifecycle.
8. Steep detail/form lifecycle.
9. My Tea JSON + SPA page.
10. Onboarding JSON + SPA page.
11. Oolong profile.
12. Home/feed and shared social behavior.
13. Static/shared pages, notifications, settings, and admin.
14. Route-level Templ/HTMX retirement.
15. Documentation and final dual-app regression audit.

Do not combine API contract invention, a complex form port, route cutover, and
legacy deletion in one change.

## Acceptance criteria

The port is complete when:

- all current Oolong user-facing pages render through the embedded SvelteKit
  SPA;
- Oolong's five active entities support tested create, read, update, delete,
  detail, and backlinks behavior as applicable;
- the steep flow preserves every existing record field and readiness rule;
- My Tea, onboarding, profile, feed, social, notifications, settings, and admin
  work in Oolong mode;
- all frontend data comes from documented JSON contracts with generated Oolong
  transport types;
- no Oolong SPA page contains coffee-only links, labels, NSIDs, cache data, or
  redirect destinations;
- public direct loads and crawler metadata work without JavaScript;
- authz, CSRF, visibility, moderation, PDS persistence, witness cache, and
  firehose behavior remain enforced by Go and covered by tests;
- Oolong Templ/HTMX page code and legacy island mounts have no remaining
  callers and are removed;
- unknown routes remain 404s;
- Go tests, dual-app integration tests, Svelte checks/tests, type drift checks,
  and Arabica + Oolong Playwright suites run in CI and pass from clean state.

## Future work

These should remain separate decisions after the port:

- enabling Oolong cafe and drink records;
- adding planned-steep/recipe records;
- adding an Oolong Explore surface;
- visual redesign beyond parity;
- extracting more schema-driven entity UI after both apps demonstrate a stable
  common pattern;
- removing all repository-wide Templ/HTMX infrastructure after both apps have
  no remaining consumers.
