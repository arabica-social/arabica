# SvelteKit SPA Migration Plan

Date: 2026-07-08

## Goal

Migrate arabica's frontend from server-rendered Templ + HTMX to a SvelteKit
SPA consuming a Go JSON API. Keep the Go backend as the single deployable
binary; add a JSON API surface alongside existing HTML handlers, then port
pages one at a time to SvelteKit routes.

## Why

The current HTMX + Svelte-islands architecture works but has hard limits:

- **Frontend rendering is untested.** Smoke tests only assert `200 + non-empty
  body`. No component-level coverage of page output, interactions, or state
  transitions.
- **The API contract is implicit.** Some handlers return JSON, some HTML, some
  HTML fragments behind `RequireHTMXMiddleware`. There's no single source of
  truth for what the API returns.
- **Coupling.** 16K lines of templ interleave data-fetching, formatting, and
  markup. The `bff/` helpers, `feedviews.Registry` (returns `templ.Component`),
  and per-entity view configs are all templ-coupled.

A SvelteKit SPA with a complete JSON API separates these concerns: backend
tests verify data/logic, frontend tests verify presentation. The existing
integration test suite (449 assertions, already JSON-API-shaped) barely
moves; the frontend gains rich `@testing-library/svelte` coverage per page.

## Architecture Decision: SvelteKit SPA mode

- **Adapter:** `@sveltejs/adapter-static` with `fallback: "index.html"` (pure
  SPA, no Node runtime).
- **Routing:** File-based routes under `src/routes/` mapping 1:1 to current
  Go page routes.
- **Data loading:** `+page.ts` `load` functions `fetch()` from the Go JSON
  API. No SvelteKit form actions — mutations use regular `fetch` POST/PUT/DELETE
  to existing CSRF-protected Go endpoints.
- **Shell:** A thin Go catch-all route serves `index.html` with `<head>`
  populated server-side (OG tags, title, theme script, CSP nonce, traceparent).
  For entity view URLs, the shell resolves the record to inject correct OG
  tags. Everything inside `<body>` is SvelteKit-rendered.
- **Stay in Go forever:** OAuth flows, OG image generation (PNG), OG meta tag
  injection, CSRF mutation endpoints, static file serving, well-known
  endpoints.

### What stays vanilla Svelte (components, not routes)

Reusable widgets imported by SvelteKit routes: `EntityCombo.svelte`,
`BrewFormField.svelte`, `PoursEditor.svelte`, `BeanRatingIsland.svelte`,
`DisclosureIsland.svelte`, `ShareButtonIsland.svelte`,
`ScrollTopIsland.svelte`. These are already built and tested.

### Two-app strategy (arabica + oolong)

**Single SvelteKit build with runtime app detection.** The apps share 90%+
of component structure (entity views, feed, manage, profile are structurally
identical with different entity types). `data-app` on `<body>` already
identifies the running app; `appCache.ts`, `feedCache.ts`, and
`comboSelectRegistry.ts` already key by app name. Entity descriptors
(`domain.App.Descriptors`) abstract entity differences at runtime.

## Current State Summary

### Backend (already strong)

- `records.Store` — app-agnostic 5-method interface (FetchRecord,
  FetchAllRecords, PutRecord, RemoveRecord, DID).
- `entities.Descriptor` + `EntityRouteBundle` — descriptor-driven route
  registration, 8 route patterns per entity.
- `RecordCRUDWrite` — generic create/update that already returns JSON via
  `WriteJSON` for all simple entities (bean, roaster, grinder, brewer, and
  all oolong entities).
- `EntityViewLoader` — view-loading pipeline (own-store → witness → PDS
  with ref resolution), HTTP-independent. Reusable for JSON view endpoints.
- `FetchSocialData` — generic social data fetcher (likes, comments,
  moderation state).
- Integration tests are already JSON-API-shaped: POST `/api/roasters` →
  decode JSON → assert fields. Only 5 of ~449 assertions check HTML body.

### Frontend (already invested in Svelte)

- 37 Svelte islands (~9,440 LOC), Svelte 5 runes, Vite 6, vitest (29 tests).
- `appCache.ts` — client record cache talking to `/api/data`.
- `comboSelectRegistry.ts` — entity config, fully portable.
- `feedMasonry.ts`, `PoursEditor.svelte`, `EntityCombo.svelte` — reusable
  as-is.
- `main.ts` (1,075 LOC) — island mounting system; **deleted entirely** in
  SvelteKit (replaced by routing).

### CSS (~7,900 lines)

- Design tokens (`tokens.css`), reset, utilities — portable as global.
- Component CSS (24 files, ~5,200 lines) — tightly coupled to templ HTML
  structure. **Lowest-effort path:** keep as a global stylesheet imported by
  the SvelteKit shell. All class names continue to work.

## Concurrency Strategy

Three parallel work streams with clear interfaces:

### Stream A — Go JSON API (backend)

**Owner:** backend agent
**Interface contract:** JSON endpoints at documented paths returning
documented shapes. HTML handlers stay running throughout.

**Principle:** Add JSON endpoints alongside existing HTML handlers. Both
work simultaneously. The SvelteKit migration consumes new JSON endpoints;
existing templ pages keep functioning until retired.

**Work items:** See "Phase 1" below.

### Stream B — SvelteKit foundation + page migration

**Owner:** frontend agent
**Interface contract:** Consumes Stream A's JSON endpoints. Coexists with
templ pages during migration (page-by-page cutover).

**Principle:** Set up SvelteKit alongside the existing Vite island build
(both can coexist). Port pages one at a time. Each ported page is
independently shippable.

**Work items:** See "Phase 2" below.

### Stream C — Testing + type safety

**Owner:** testing agent
**Interface contract:** Contract tests that verify the JSON API shapes
match what SvelteKit expects. E2E tests for critical paths.

**Principle:** Lock down the API contract early with tests. Add E2E
coverage for critical user paths. Generate TypeScript types from Go
structs to prevent drift.

**Work items:** See "Phase 3" below.

### Coordination interface

The critical contract between streams is the **API shape document**. Stream
A implements it, Stream B consumes it, Stream C tests it. Maintain it as
a living spec:

```
docs/api/                    # NEW — API contract spec
  README.md                  # overview, conventions
  entities.md                # entity view, list, CRUD shapes
  feed.md                    # feed item, query, pagination
  profile.md                 # profile data bundle
  social.md                  # likes, comments, moderation
```

Stream A updates the spec when adding endpoints. Stream B reads it to build
Svelte types. Stream C generates contract tests from it.

---

## Phase 0 — Guardrails (done before parallel work starts)

**Goal:** establish the contract spec and SvelteKit skeleton so all three
streams can start in parallel.

- [ ] Create `docs/api/` contract spec with shapes for all endpoints to be
      added (from the gap analysis below).
- [ ] Run `pnpm dlx sv create` or manual setup to create a SvelteKit project
      alongside the existing Vite config. Use `adapter-static` with SPA
      fallback. Verify it builds and serves a hello-world route.
- [ ] Add a Go shell route that serves the SvelteKit `index.html` for
      unmigrated paths (catch-all, before the 404 handler). Keep existing
      templ routes taking priority.
- [ ] Establish the TypeScript type generation pipeline (see Phase 3) so
      Stream B has types from day one.
- [ ] Baseline snapshot: `go test ./...` + `pnpm test:svelte` green.

---

## Phase 1 — Go JSON API (Stream A)

**Goal:** complete the JSON API surface so every page's data is available
as JSON. Additive, low-risk — existing HTML handlers keep running.

### Priority order (by what unblocks Stream B earliest)

#### P1.1 — Feed JSON (unblocks home, feed pages)

`GET /api/feed?type=&sort=&cursor=` → JSON.

- Drop `RequireHTMXMiddleware` gate (or add a parallel `/api/feed.json`
  route — prefer reusing the path with content negotiation or a new
  dedicated JSON path to avoid breaking HTMX clients during migration).
- Response shape:
  ```json
  {
    "items": [FeedItem],
    "next_cursor": "string",
    "is_authenticated": true,
    "query": {"type": "brew", "sort": "recent"}
  }
  ```
- `FeedItem` fields: `record_type`, `action`, `record` (typed model),
  `author` {did, handle, display_name, avatar}, `timestamp`,
  `like_count`, `comment_count`, `subject_uri`, `subject_cid`,
  `is_liked_by_viewer`, `is_owner`.
- **Handler:** `internal/handlers/feed.go` — reuse `feedService.GetFeedWithQuery`
  / `GetCachedPublicFeed`. The data pipeline is HTTP-independent; only the
  response serialization changes.

#### P1.2 — Entity view JSON (unblocks all entity view pages)

`GET /api/{entity}/{actor}/{id}` → JSON.

- Reuse `EntityViewLoader.Load` (own-store → witness → PDS + ref resolution).
- Reuse `FetchSocialData` (likes, comments, moderation).
- Reuse `fetchBacklinks`.
- Response shape:
  ```json
  {
    "record": <typed model>,
    "subject_uri": "at://...",
    "subject_cid": "string",
    "author": {"did", "handle", "display_name", "avatar"},
    "social": {
      "is_liked": false,
      "like_count": 0,
      "comment_count": 0,
      "comments": [IndexedComment],
      "is_moderator": false,
      "can_hide_record": false,
      "can_block_user": false,
      "is_record_hidden": false
    },
    "backlinks": <backlinks.Result>,
    "is_own_profile": false,
    "is_authenticated": true,
    "share_url": "/beans/...",
    "entity_type": "bean",
    "entity_count": 0
  }
  ```
- Add a `JSONView` slot to `EntityRouteBundle` (or a separate generic
  handler) so this is generated for all entities via the descriptor loop.
- **Files:** `internal/handlers/entity_views.go`,
  `internal/handlers/entity_view_loader.go`, `internal/handlers/entity_routes.go`.

#### P1.3 — Brew create/update JSON (unblocks brew form)

`POST /api/brews` and `PUT /api/brews/{id}` → JSON.

- Currently `HandleBrewCreate`/`HandleBrewUpdate` always set `HX-Redirect`.
- Add a JSON path: when no `__redirect` form value (or when `Accept:
  application/json`), return the brew model as JSON via `WriteJSON`.
- Preserve the `X-Incomplete-Nudge` response header (or fold into JSON
  response: `{ "brew": {...}, "incomplete_nudge": "..." }`).
- **File:** `internal/arabica/handlers/brew.go`.

#### P1.4 — Manage JSON with stats (unblocks manage/my-coffee)

`GET /api/manage` → JSON.

- `HandleAPIListAll` (`/api/data`) returns raw records but lacks usage
  counts and avg ratings.
- `HandleManagePartial` computes: bean/grinder/brewer brew counts,
  roaster bean counts, avg brew ratings by bean/roaster URI.
- **Option A:** extend `/api/data` with `?include=stats` query param.
- **Option B (preferred):** new `GET /api/manage` returning records +
  stats in one response. Keep `/api/data` for the lightweight appCache use
  case.
- Response shape:
  ```json
  {
    "did": "string",
    "beans": [Bean],
    "roasters": [Roaster],
    "grinders": [Grinder],
    "brewers": [Brewer],
    "recipes": [Recipe],
    "stats": {
      "bean_brew_counts": {"<uri>": 3},
      "grinder_brew_counts": {"<uri>": 5},
      "brewer_brew_counts": {"<uri>": 2},
      "roaster_bean_counts": {"<uri>": 4},
      "bean_avg_brew_ratings": {"<uri>": 4.2},
      "roaster_avg_brew_ratings": {"<uri>": 4.5}
    }
  }
  ```
- **File:** `internal/arabica/handlers/entities.go`.

#### P1.5 — Brew list JSON (unblocks brew list / my-coffee brew table)

`GET /api/brews?offset=&limit=` → JSON.

- `HandleBrewListPartial` currently returns HTML.
- Response: `{ "brews": [Brew], "has_more": true, "next_offset": 20 }`.
- **File:** `internal/arabica/handlers/brew.go`.

#### P1.6 — Profile JSON (unblocks profile page)

`GET /api/profile/{actor}?brews_offset=&brews_limit=` → JSON.

- Combines `HandleProfile` (shell) + `HandleProfilePartial` (heavy data).
- Response shape: profile, is_own_profile, is_arabica_user, brews
  (paginated), entity lists (beans, roasters, grinders, brewers), total_brews,
  brew social stats (like/comment counts, liked-by-user, CIDs), entity
  usage counts, avg ratings (respecting visibility prefs).
- **File:** `internal/arabica/handlers/profile.go`.

#### P1.7 — Notifications JSON (unblocks notifications page)

`GET /api/notifications?cursor=` → JSON.

- `HandleNotifications` currently renders HTML; `POST
  /api/notifications/read` returns a redirect.
- Response: `{ "notifications": [{...Notification, actor: {...}, link,
  action_text}], "next_cursor": "string" }`.
- `POST /api/notifications/read` → `{ "read": true }`.
- **File:** `internal/handlers/notifications.go`.

#### P1.8 — Explore JSON (unblocks explore page)

`GET /api/explore?type=&q=&sort=&cursor=&{filters}` → JSON.

- `HandleExplore` currently returns HTML (page + HTMX append).
- Response: `{ "items": [FeedItem], "documents": {uri: ExploreDocument},
  "facet_counts": [...], "next_cursor": "...", "health": {...} }`.
- **File:** `internal/arabica/handlers/explore.go`.

#### P1.9 — Social endpoints JSON (unblocks likes, comments, report)

Convert HTMX-partial mutation endpoints to JSON:

- `POST /api/likes/toggle` → `{ "is_liked": true, "like_count": 5,
  "subject_uri": "..." }` (currently renders LikeButton HTML).
- `GET /api/comments?subject_uri=` → `{ "comments": [IndexedComment],
  "subject_uri": "...", "is_authenticated": true }` (currently HTML partial).
- `POST /api/comments` → `{ "comment": Comment, "comments":
  [IndexedComment] }` (currently re-renders CommentSection).
- `DELETE /api/comments/{id}` → `{ "deleted": true }` (currently HX-Trigger).
- `POST /api/report` → `{ "report_id": "...", "submitted": true }` +
  error shape.
- **Files:** `internal/handlers/feed.go`, `internal/handlers/handlers.go`,
  `internal/handlers/report.go`.

#### P1.10 — Settings JSON (unblocks settings page)

- `GET /api/settings` → `{ profile_stats_visibility, user_preferences,
  bluesky_profile }`.
- `POST /api/settings/preferences` → `{ "saved": true }` (currently
  `<span>Saved</span>`).
- `POST /api/settings/profile-visibility` → `{ "saved": true }`.
- `POST /api/settings/bluesky-profile` → `{ "saved": true }`.
- **File:** `internal/handlers/pages.go`, `internal/handlers/bsky_profile.go`.

#### P1.11 — Onboarding + incomplete records + popular recipes JSON

- `GET /api/onboarding` → `{ readiness, beans[], brewers[], grinders[],
  roasters[] }`.
- `GET /api/incomplete-records` → `{ "records": [{entity_type, rkey, name,
  missing_fields[]}] }`.
- `GET /api/recipes/popular?limit=3` → `[Recipe]` (or extend
  `/api/recipes/suggestions` with `?sort=popular`).
- **Files:** `internal/arabica/handlers/onboarding.go`,
  `internal/arabica/handlers/entities.go`,
  `internal/arabica/handlers/recipe.go`.

#### P1.12 — Admin JSON (unblocks admin page, lowest priority)

- `GET /api/_mod` → `AdminProps` as JSON.
- `GET /api/_mod/content` → `AdminProps` (partial).
- `GET /api/_mod/stats` → `AdminStats` + backups.
- `POST /_mod/*` (hide, unhide, dismiss-report, block, etc.) →
  `{ "ok": true, "action": "hide" }` (currently HX-Trigger).
- **File:** `internal/handlers/admin.go`.

#### P1.13 — Backlinks JSON

`GET /api/{entity}/{actor}/{id}/backlinks` → JSON.

- `RenderBacklinksView` currently renders HTML.
- Response: `{ entity_noun, entity_name, back_url, detail_url, result:
  backlinks.Result }`.
- **File:** `internal/handlers/entity_views.go`.

#### P1.14 — Signup categories JSON

`GET /api/signup/categories` → `{ "categories": [...] }` (for
`/join/create` page).
- **File:** `internal/handlers/handlers.go`.

### Backend verification per endpoint

- [ ] New JSON endpoint returns 200 with `Content-Type: application/json`.
- [ ] Existing HTML handler still works (200 + HTML body).
- [ ] `go test ./...` passes.
- [ ] Integration test added or existing test covers the new JSON path.
- [ ] `docs/api/` spec updated with the endpoint shape.

---

## Phase 2 — SvelteKit migration (Stream B)

**Goal:** port templ pages to SvelteKit routes, one at a time. Each page
is independently shippable.

### P2.0 — SvelteKit foundation

- [ ] Initialize SvelteKit project with `adapter-static`, SPA fallback.
- [ ] Set up `src/lib/` for shared components (port `EntityCombo.svelte`,
      `PoursEditor.svelte`, `BrewFormField.svelte`, `BeanRatingIsland.svelte`,
      `DisclosureIsland.svelte`, `ShareButtonIsland.svelte`).
- [ ] Port `appCache.ts` to a Svelte store (`$appCache`).
- [ ] Port `comboSelectRegistry.ts` as-is (pure config).
- [ ] Rewrite `feedCache.ts` to cache JSON instead of HTML strings.
- [ ] Set up `+layout.svelte` with header, footer, theme runtime, toast
      container, modal container.
- [ ] Configure Vite to output to a directory the Go embed FS picks up.
- [ ] Add the Go shell route serving SvelteKit's `index.html`.

### P2.1 — Static pages (easiest, proves the pattern)

Port: `/about`, `/terms`, `/atproto`, `/join/create`.

- These are mostly static content. `/join/create` needs
  `GET /api/signup/categories`.
- **Proof of concept:** verify Go shell serves the SPA, SvelteKit route
  renders, OG tags inject correctly.

### P2.2 — Simple entity views

Port: roaster view, grinder view, brewer view, bean view.

- Consume `GET /api/{entity}/{actor}/{id}` (Stream A P1.2).
- Reuse existing `record_*.templ` content as Svelte component markup.
- Port `ActionBar`, `Comments`, `BacklinksSection` as shared Svelte
  components.
- **Bean view** is slightly more complex (actions island, rating).

### P2.3 — Brew view + recipe view

Port: `/brews/{actor}/{id}`, `/recipes/{actor}/{id}`.

- Same entity view pattern, more complex record display.
- Recipe view has source recipe resolution + fork button (already a Svelte
  island).
- Brew view has save-as-recipe (already a Svelte island).

### P2.4 — Manage / My Coffee

Port: `/manage`, `/my-coffee`.

- Consume `GET /api/manage` (P1.4) + `GET /api/brews` (P1.5).
- Port `ManageTabsIsland` → SvelteKit tab component.
- Port `ManageCollectionsIsland` search/sort logic.
- Port entity tables as Svelte components.

### P2.5 — Feed + Home

Port: `/` (home), `/api/feed` (feed partial → SvelteKit route).

- Consume `GET /api/feed` JSON (P1.1).
- Port `FeedFiltersIsland` → SvelteKit route logic.
- Port `feedMasonry.ts` (reusable) or replace with CSS columns.
- Port feed card rendering (currently `feedviews.Registry` →
  `templ.Component`; becomes Svelte component per entity type).
- **This is the highest-risk port** — feed is complex (masonry, filters,
  pagination, moderation, per-entity cards).

### P2.6 — Brew form + Recipe form (hardest)

Port: `/brews/new`, `/brews/{id}/edit`, recipe form.

- `BrewFormIsland.svelte` (686 LOC) already has most of the logic; becomes
  a full SvelteKit route component.
- Consume `POST /api/brews` JSON (P1.3).
- Recipe form (`RecipeFormIsland.svelte`, 248 LOC) similar.

### P2.7 — Profile

Port: `/profile/{actor}`.

- Consume `GET /api/profile/{actor}` (P1.6).
- Port `ProfileStatsIsland` + `TasteProfileIsland` (radar chart).
- Port profile tabs.

### P2.8 — Notifications, Settings, Explore

Port: `/notifications`, `/settings`, `/explore`.

- Consume P1.7, P1.8, P1.10.
- Explore is complex (search + facets + documents map).

### P2.9 — Onboarding + Admin

Port: `/onboarding`, `/add`, `/_mod`.

- Onboarding consumes P1.11.
- Admin consumes P1.12 (lowest priority; admin is internal-only).

### P2.10 — Retire templ + HTMX

- [ ] Remove all templ page handlers (keep OG image + shell route).
- [ ] Remove `RequireHTMXMiddleware` and all HX-Request checks.
- [ ] Remove `htmx.min.js` from embed + layout.
- [ ] Remove `main.ts` island mounting system.
- [ ] Remove `feedCache.ts` HTML caching (replaced by JSON cache).
- [ ] Remove `domContracts.ts` HTMX helpers (`fetchHTMXPartial`,
      `extractFragment`).
- [ ] Remove `TransitionRuntimeIsland`, `LayoutRuntimeIsland` HTMX wiring.
- [ ] Delete templ components and pages (16K lines).
- [ ] Run `templ generate` is no longer needed; remove from justfile.
- [ ] Update `AGENTS.md` architecture docs.

### Frontend verification per page

- [ ] Page renders with data from JSON API.
- [ ] `@testing-library/svelte` test covers rendering + key interactions.
- [ ] OG tags still inject correctly via Go shell route.
- [ ] Existing Go integration tests still pass.
- [ ] Visual parity with templ version (screenshot compare if needed).

---

## Phase 3 — Testing + type safety (Stream C)

**Goal:** lock down the API contract, add E2E coverage, prevent type drift.

### P3.1 — TypeScript type generation

Generate TS types from Go structs to prevent drift between backend and
frontend.

- **Option A:** `tygo` (Go → TypeScript codegen from struct definitions).
  Generates `.ts` files from Go models in `internal/arabica/entities/`,
  `internal/feed/`, `internal/handlers/` (SocialData, EntityViewBase).
- **Option B:** Generate types from the lexicon JSON schemas
  (`lexicons/social/arabica/alpha/*.json`) using a JSON Schema → TS tool.
  Covers record shapes but not view/social data.
- **Option C (preferred):** tygo for Go structs + hand-maintained types for
  API response envelopes (`{items, next_cursor, ...}`).
- Output to `src/lib/types/generated.ts`. CI runs `tygo generate` and fails
  on diff.

### P3.2 — API contract tests

- [ ] For each new JSON endpoint (P1.1–P1.14), add a contract test that
      asserts the response shape matches `docs/api/` spec.
- [ ] Tests live in `tests/integration/` (existing harness) or a new
      `tests/api/` package.
- [ ] Contract test asserts: status code, content-type, required fields
      present, field types correct.
- [ ] These tests protect Stream B from shape changes during migration.

### P3.3 — E2E tests (Playwright)

Add E2E tests for critical user paths. This is a test type that doesn't
exist today and should have existed regardless of migration.

- [ ] Set up Playwright against the integration test harness (real Go server
      + test PDS).
- [ ] **Critical path: create brew** — login → /brews/new → fill form →
      submit → see brew in feed/my-coffee.
- [ ] **Critical path: view feed** — home → feed loads → filter → paginate.
- [ ] **Critical path: manage entities** — my-coffee → create roaster →
      create bean referencing roaster → edit bean.
- [ ] **Critical path: social** — view brew → like → comment → see
      notification.
- [ ] **Critical path: profile** — view own profile → view other user's
      profile.
- [ ] Run in CI alongside `go test` + `pnpm test:svelte`.

### P3.4 — Expand component tests

- [ ] For each ported SvelteKit page (P2.1–P2.9), add
      `@testing-library/svelte` tests.
- [ ] Test rendering with mocked API responses.
- [ ] Test key interactions (form submit, filter change, pagination,
      like toggle).
- [ ] Test error states (API 401, 500, network error).
- [ ] Target: every ported page has at least one component test.

### P3.5 — Snapshot tests for API responses

- [ ] Add JSON snapshot tests for key endpoints (feed, entity view,
      profile) using the existing `shutter` library.
- [ ] Scrub dynamic values (DIDs, rkeys, timestamps) as existing snapshot
      tests do.
- [ ] These catch unintended shape changes during refactors.

---

## Risk Register

### High risk

1. **Feed port (P2.5)** — masonry layout, per-entity card rendering,
   moderation filtering, pagination. The `feedviews.Registry` → Svelte
   component mapping is the most complex translation. Mitigate: port feed
   last, after the pattern is proven on simpler pages.

2. **Brew form port (P2.6)** — 686 LOC island + complex multi-method form
   with pours editor. Already has a Svelte island with most logic; risk is
   in the SvelteKit route wrapper + data loading. Mitigate: the island
   already works; the port is mostly removing the templ shell.

3. **SEO degradation** — if the Go shell route doesn't correctly resolve
   entity data for OG tags, social sharing breaks. Mitigate: keep all OG
   computation logic in Go; the shell route reuses `EntityViewLoader` to
   resolve records for `<head>` injection. Test with the OG image routes
   that already exist.

4. **Two-app divergence** — arabica and oolong have different entity sets
   and branding. A single SvelteKit build must handle both at runtime.
   Mitigate: entity descriptors and `data-app` already abstract this; the
   `comboSelectRegistry` already supports both. Test both apps in CI.

### Medium risk

5. **CSS coupling** — 5,200 lines of component CSS reference templ HTML
   structure. If Svelte components use different class names, styling
   breaks. Mitigate: keep CSS global, use same class names in Svelte
   components. Only migrate to scoped styles if a component is rewritten.

6. **Type drift** — Go structs and TS types can diverge silently.
   Mitigate: tygo codegen in CI (P3.1) + contract tests (P3.2).

7. **Session/auth in SPA** — OAuth callback sets cookies; the SPA must
   detect auth state. Mitigate: `data-user-did` on `<body>` already
   signals auth state; `appCache.ts` already reads it. The Go shell route
   injects it server-side.

### Low risk

8. **Existing integration tests** — already JSON-API-shaped, barely change.
   The 5 HTML-body assertions in `social_test.go` and `cache_test.go` need
   updating when those pages are ported.

9. **Performance** — SvelteKit SPA adds a JS bundle download. Mitigate:
   Svelte 5 compiles to small bundles; code-splitting per route; service
   worker caches JS.

10. **Service worker** — currently caches `htmx.min.js`. Update to cache
    SvelteKit JS chunks. Low effort.

---

## Migration Sequencing (parallel timeline)

```
Week 1-2:
  Stream A: P1.1 (feed JSON), P1.2 (entity view JSON), P1.3 (brew CRUD JSON)
  Stream B: P2.0 (SvelteKit foundation), P2.1 (static pages)
  Stream C: P3.1 (type gen), P3.2 (contract test framework)

Week 3-4:
  Stream A: P1.4 (manage JSON), P1.5 (brew list), P1.6 (profile JSON)
  Stream B: P2.2 (simple entity views), P2.3 (brew/recipe view)
  Stream C: P3.2 (contract tests for P1.1-P1.3), P3.3 (Playwright setup)

Week 5-6:
  Stream A: P1.7 (notifications), P1.8 (explore), P1.9 (social endpoints)
  Stream B: P2.4 (manage/my-coffee), P2.5 (feed/home)
  Stream C: P3.3 (E2E critical paths), P3.4 (component tests for P2.2-P2.3)

Week 7-8:
  Stream A: P1.10 (settings), P1.11 (onboarding), P1.12 (admin), P1.13 (backlinks)
  Stream B: P2.6 (brew/recipe form), P2.7 (profile)
  Stream C: P3.4 (component tests), P3.5 (API snapshots)

Week 9-10:
  Stream A: P1.14 (signup), cleanup, oolong parity
  Stream B: P2.8 (notifications/settings/explore), P2.9 (onboarding/admin)
  Stream C: E2E for all critical paths, type drift audit

Week 11-12:
  Stream B: P2.10 (retire templ + HTMX)
  Stream C: full regression, visual parity audit
```

This is aggressive. The streams have clean interfaces (API spec), so
parallelism is real. The critical path is Stream A P1.1–P1.3 unblocking
Stream B's first real pages.

---

## Completion Criteria

- [ ] Every page renders as a SvelteKit route consuming JSON from the Go API.
- [ ] No templ page handlers remain (OG image + shell route excepted).
- [ ] `htmx.min.js` removed; `RequireHTMXMiddleware` removed.
- [ ] `main.ts` island mounting system deleted.
- [ ] Every ported page has `@testing-library/svelte` tests.
- [ ] Playwright E2E tests cover all critical user paths.
- [ ] TypeScript types generated from Go structs in CI (no drift).
- [ ] `go test ./...` + `pnpm test:svelte` + Playwright all green in CI.
- [ ] Both arabica and oolong apps work end-to-end.
- [ ] OG social sharing works (verified with crawler-style fetch).
- [ ] `AGENTS.md` updated to reflect the new architecture.

---

## Open Questions

1. **JSON endpoint path strategy** — reuse existing `/api/feed` path (drop
   HTMX gate) or add parallel `/api/feed.json`? Reusing the path is cleaner
   long-term but risks breaking HTMX clients during migration. A dedicated
   JSON path (e.g., `/api/v1/feed`) is safer during transition but creates
   a namespace to clean up later. **Recommendation:** use `Accept:
   application/json` content negotiation on existing paths where an HTML
   handler exists; use plain JSON for new endpoints.

2. **Oolong parity** — oolong's `HandleMyTea` fetches everything
   server-side (no partials), unlike arabica's shell+partial pattern. The
   JSON API additions need oolong equivalents. How much does oolong
   diverge in its data fetching? The `HandleOolongAPIListAll` exists; does
   oolong need `/api/manage`, `/api/profile/{actor}`, etc. as separate
   endpoints or can it reuse shared handlers?

3. **Moderation toasts** — HTMX `HX-Trigger` events drive toast
   notifications (e.g., "Record hidden"). In a JSON API, the response body
   carries the result, but toast dispatch moves client-side. Need a
   Svelte toast store + convention for mutation responses to indicate
   toast messages.

4. **Profile stats visibility** — `profileprefs.ProfileStatsVisibility`
   gates whether avg ratings are shown on profiles. The JSON profile
   endpoint must respect this server-side (it already does in the HTML
   handler). Confirm the JSON path preserves this.

5. **Feed cache invalidation** — `feedCache.ts` currently caches HTML and
   clears on `arabica:feed-mutation` events. The JSON feed cache needs an
   equivalent invalidation strategy. Svelte stores + a mutation event
   bus, or rely on short TTL + manual refetch on mutation?

6. **`atproto.Profile` struct** — the external `tangled.org/pdewey.com/atp`
   module defines `Profile` (DID, Handle, DisplayName, Avatar). Confirm
   exact field names for the TS type generation and API spec.

---

## Appendix: Existing JSON API Surface (already working)

These endpoints already return JSON and need no changes:

| Endpoint | Method | Response shape |
|----------|--------|----------------|
| `/.well-known/oauth-client-metadata.json` | GET | OAuth client metadata |
| `/healthz` | GET | `{status, firehose, feed_index}` |
| `/api/resolve-handle` | GET | `{did, handle, displayName?, avatar?}` |
| `/api/search-actors` | GET | `{actors: [...]}` |
| `/api/suggestions/{entity}` | GET | `[]EntitySuggestion` |
| `/api/data` | GET | `{did, beans[], roasters[], ...}` |
| `/api/recipes` | GET | `[]Recipe` |
| `/api/recipes/suggestions` | GET | `[]Recipe` |
| `/api/recipes/{id}` | GET | `Recipe` |
| `/api/recipes` | POST | `Recipe` |
| `/api/recipes/from-brew/{id}` | POST | `Recipe` |
| `/api/recipes/fork/{id}` | POST | `Recipe` |
| `/api/{entity}` | POST | `<model>` (simple entities) |
| `/api/{entity}/{id}` | PUT | `<model>` or HX-Redirect |
| `/brews/export` | GET | `[]Brew` |
| `/_mod/export` | GET | export data |
| `/_mod/purge` | POST | `{did, purged, purgedAt}` |
| `/_mod/rebuild` | POST | `{did, handle, rebuilt, rebuiltAt}` |
| `/_mod/refresh-handles` | POST | `{refreshed, failed, durationMs}` |
| `/_mod/pds-records` | GET | PDS records data |

## Appendix: Key Reusable Backend Patterns

- **`records.Store`** — 5-method app-agnostic interface. Primary generic CRUD
  surface.
- **`entities.Descriptor` + `domain.EntityRoute`** — entity metadata + URL
  mapping.
- **`EntityRouteBundle`** — groups per-entity handlers; add a `JSONView` slot
  for generic JSON view endpoints.
- **`RecordCRUDWrite`** — generic create/update, already returns JSON.
- **`EntityViewLoader.Load`** — view-loading pipeline (store → witness → PDS
  + ref resolution), HTTP-independent. Reuse for JSON view endpoints.
- **`FetchSocialData`** — generic social data (likes, comments, moderation).
- **`ListRecords[T]` / `ListPublicRecords[T]`** — generic record listers.
- **`StandardViewTriple`** — builds view config lambdas from a decoder.
