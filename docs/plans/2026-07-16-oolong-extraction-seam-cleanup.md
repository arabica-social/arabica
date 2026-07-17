# Oolong Extraction and Shared-Seam Cleanup Plan

Date: 2026-07-16

## Goal

Enable Oolong to be removed from this repository into its own codebase, and
clean up the shared platform packages so the boundary between "platform" and
"app" is honest. The nearer-term expected outcome is that **Oolong forks the
Arabica repo** and continues on its current Templ + HTMX frontend for a while,
while **Arabica is free to make larger backend and frontend changes** without
dragging Oolong along or keeping coupling alive for its sake. A longer-term
possibility is that Oolong adopts an external AppView (e.g. happyview or
quickslice) instead of the hand-rolled firehose/FeedIndex indexing stack.

This plan records the seams as they exist today, the places where the seam
leaks, and the cleanup that makes extraction cheap regardless of which
longer-term direction Oolong takes. It is not a migration runbook: it is a map
of the boundary and the work needed to make it clean.

## Non-goals

- Do not perform the Oolong extraction as part of this plan. This plan prepares
  the seam; the actual fork/split is a separate, later decision.
- Do not change Oolong's frontend strategy here. Whether Oolong stays on
  Templ/HTMX or later ports to the SPA is tracked by the
  [2026-07-11 Oolong SvelteKit SPA port plan](2026-07-11-oolong-sveltekit-spa-port.md)
  and is independent of backend extraction readiness.
- Do not merge Arabica and Oolong lexicons or record types.
- Do not delete `cmd/server` (the unified two-app binary) prematurely; it is a
  useful dev convenience even if the flake packages the two apps separately.
- Do not decide now whether Oolong's eventual AppView is happyview, quickslice,
  or something else. That is a future product/architecture decision.
- Do not change AT Protocol record contracts, PDS ownership, OAuth, moderation
  policy, or the witness-cache-is-a-projection invariant.

## Context: how the two apps are wired today

### The intended seam

The codebase already has a clean intended boundary:

- `internal/atplatform/domain.App` (`domain/app.go`) is a config struct carrying
  `Name`, `NSIDBase`, `Descriptors`, `EntityRoutes`, `Brand`, and an optional
  `RecordStore` wrapper. It is the single object that makes shared code
  app-aware.
- Each app builds its `*domain.App` in a small constructor:
  `internal/arabica/app/app.go` (`arabicaapp.New`) and
  `internal/oolong/app/app.go` (`oolongapp.New`).
- Three entrypoints funnel through one shared bootstrap,
  `internal/atplatform/server.Run(ctx, app, opts)`:
  - `cmd/arabica/main.go` and `cmd/oolong/main.go` build one app and call
    `server.Run` directly (separate binaries, separate ports 18910 / 18920).
  - `cmd/server/main.go` runs both apps in one process with isolated listeners,
    metrics ports, data dirs, and SQLite databases. It is a dev convenience and
    is **not** wired into `flake.nix` packages.
- Shared packages receive app identity through `*domain.App`, the
  `routing.AppRoutes` interface (implemented per-app as `coffeehandlers.Routes{}`
  and `teahandlers.Routes{}`), and `handlers.StaticPageRenderers`.
- Per-app entity packages (`internal/{arabica,oolong}/entities`) register
  descriptors and `RecordBehavior` (codecs, rkey, title, reference hydration)
  into a shared global registry at `init()` time. `entities.AllForApp(nsidBase)`
  filters by NSID prefix so each app's constructor only sees its own descriptors.

### Codebase shape

Non-test, non-generated Go LOC (approximate):

| Area | LOC |
|---|---|
| `internal/arabica/*` | 10,298 |
| `internal/oolong/*` | 3,736 |
| Shared `internal/*` (platform) | 20,661 |
| — of which `internal/handlers` | 6,605 |
| — of which `internal/firehose` | 4,184 |

Oolong is roughly 18% of the app-specific code. The shared platform is about
twice both apps combined. The single largest shared package, `internal/handlers`
(6.6K LOC), does **not** import either app's entity packages; Oolong consumes
it through exported symbols (`Handler`, `ListRecords`, `PutRecord`,
`ValidateRKey`, `EntityViewConfig`, `StandardViewTriple`, etc.).

### What is clean today

The following shared packages have **zero** imports of `internal/arabica` or
`internal/oolong` and no app-name/NSID branching in non-test Go code. They are
app-agnostic by construction:

- `internal/atplatform` — `domain.App` + `server.Run`. `"arabica"`/`"oolong"`
  appear only in comments.
- `internal/entities` — global descriptor registry + `AllForApp(nsidBase)`.
- `internal/feed`, `internal/notifications`, `internal/social`,
  `internal/records`, `internal/backlinks`, `internal/matching`,
  `internal/moderation`, `internal/profileprefs`, `internal/validation`,
  `internal/suggestions` — zero app references; `backlinks` and `suggestions`
  use clean `Register()` patterns filled by per-app `init()`.
- `internal/firehose` ingestion core — the Jetstream `Consumer` and
  `FeedIndex.UpsertRecord` dispatch record conversion through
  `entities.BehaviorByNSID`, the per-app `RecordBehavior` registry.
- `internal/handlers` (the 6.6K package) — no app entity imports; consumed by
  both apps through exported symbols.

**Conclusion:** An extracted Oolong codebase could import these packages as-is —
vendored or as a shared Go module — with zero structural changes to them. The
work is concentrated in the leaks below.

### What Oolong actually uses from the platform

This matters for sizing the extraction. Oolong:

- Has **no** explore references (grep-confirmed). `internal/explore` is
  Arabica-only in practice.
- Does **not** call any `BrewCountsBy*` / `BeanCountsBy*` / `AvgBrewRatingBy*`
  firehose query methods. Those are coffee-specific and called only by
  `internal/arabica/handlers`.
- Has an empty `SPAOwnedRoutes` inventory (`internal/oolong/handlers/routes.go`)
  and has not started SPA cutover; all Oolong pages are still Templ/HTMX.
- Uses the shared `internal/handlers` package through ~8 exported symbols,
  most heavily `handlers.ListRecords` and `handlers.EntityViewConfig`.

## Where the seam leaks

The leaks fall into three patterns: (a) string equality on `app.Name` /
`AppName == "arabica"` / `"oolong"`; (b) hardcoded `social.arabica.alpha.*` NSIDs
in SQL and defaults; (c) entire coffee-specific subsystems with no Oolong
counterpart living in shared packages.

### Leak 1 — `internal/explore` is coffee-only code posing as shared

`internal/explore/registry.go:74` defines `NewArabicaRegistry(...)`, which
hardcodes `def.App = "arabica"` and registers only bean/roaster/grinder/brewer/
recipe with coffee-specific filters (`origin`, `variety`, `process`,
`roast_level`, `grinder_type`, `burr_type`, `brewer_type`, `ratio`) and coffee
extractors. There is no `NewOolongRegistry`, and Oolong has zero explore
references. The explore *storage* and queries live in
`internal/firehose/explore_index.go:162,411`, which calls
`explore.NewArabicaRegistry` unconditionally and defaults `q.App = "arabica"`.

This is the single biggest "shared package that isn't."

### Leak 2 — Arabica-specific query methods in `internal/firehose`

`internal/firehose/index.go` carries methods called **only** by
`internal/arabica/handlers`:

- `BrewCountsByRecipeURI`, `BrewCountsByBeanURI`, `BrewCountsByGrinderURI`,
  `BrewCountsByBrewerURI`, `BeanCountsByRoasterURI`, and `AvgBrewRatingBy*`
  (approximately `index.go:903`–`1072`). These hardcode
  `social.arabica.alpha.brew` / `.bean` collection strings and coffee JSON field
  names (`beanRef`, `grinderRef`, `brewerRef`, `roasterRef`) in SQL.
- `commentCollection()` defaults to `social.arabica.alpha.comment`
  (`index.go:108`). It is overridden at boot via `SetCommentNSID`, but the
  fallback is Arabica-specific.

Oolong does not call any of these.

### Leak 3 — `internal/web` SPA shell and assets branch on app name

`internal/web/spa/handler.go` has five `if d.AppName == "oolong"` branches
(lines ~402, 409, 416, 430, 437) that hardcode brand, tagline, site description,
and theme colors for tea vs coffee, instead of reading `domain.App.Brand`
(which already carries `DisplayName` / `Tagline`).
`internal/web/assets/bundle.go` branches `b.appName == "arabica"` to pick the
CSS URL path (`/static/css/output.css` vs `output-<app>.css`) and defaults an
empty app name to `"arabica"`. Arabica is treated as the no-suffix default.
The SPA head marker is a hardcoded constant `arabica-spa-head`
(`spa/handler.go:595`).

### Leak 4 — hardcoded Arabica defaults in `internal/handlers`

- `handlers.go:195` — `CookieNames` branches on `app == "" || app == "arabica"`
  so Arabica keeps legacy unprefixed cookie names while other apps get prefixed
  names.
- `handlers.go:611-619` — OG titles hardcode `"on arabica.social"` instead of
  `domain.App.Brand`.
- `handlers.go:868` — comment NSID fallback `social.arabica.alpha.comment`
  (overridden when an app is set, but the default leaks).
- `auth.go:482` — `X-Client: arabica.social` request header hardcoded.
- `admin.go:844,1104` — witness/PDS export filenames hardcoded
  `arabica-witness-*` / `arabica-pds-*`.
- `feed.go:122` — default `SiteCardOpts` wordmark `arabica.social`.

The store NSID wiring (`handlers.go:482-486`) is correctly derived from
`h.app.LikeNSID()` / `CommentNSID()` and is a model for the fix.

### Leak 5 — smaller hardcoded Arabica identity

- `internal/atproto/client.go:15` — `userAgent = "Arabica (+https://alpha.arabica.social; abuse@mail.arabica.systems)"` hardcoded, not derived from `domain.App`.
- `internal/routing/routing.go:325` — `otelhttp.NewHandler(handler, "arabica", ...)`
  hardcoded span name; should use `app.Name`.
- `internal/backup/metrics.go:10-35` — six Prometheus metric names prefixed
  `arabica_backup_*`, not app-derived.
- `internal/signup/signup.go:70-72` — default signup provider URL/domain
  `arabica.systems`, presumably overridden per app but defaulted to Arabica.
- `internal/lexicons/record_type.go` — shared `RecordType` constants for **both**
  apps' record types in one package. Not branching, but co-located; extraction
  means duplicating Oolong's constants or keeping `internal/lexicons` as a
  shared dependency.

### Inert-but-present coupling

Compiled Svelte-island build artifacts under `internal/web/assets/js/*.js`
contain `arabica-theme`, `arabica_data_cache`, `arabica:dev-mode-change` event
names and `oolongBrewer` / `oolongRecipe` combo keys. These are generated from
`web/src` Svelte source; their source-of-truth is in the SPA, and renaming
browser storage keys / custom events is a user-visible migration concern
(orphaned `localStorage` / `sessionStorage`). Not a Go-seam blocker, but noted
for the frontend side of any fork.

## Cleanup targets

These cleanups make the boundary honest. They are worthwhile **independently of
extraction** — each one removes a fake-shared assumption or a string branch
that should have been config. Ordered by impact and independence.

> **Status (2026-07-16):** C1, C3, and the C4/C5 identity-derivation work
> are implemented in change `platform: make app-identity seam honest for
> oolong extraction`. C2 (moving coffee query methods out of firehose) is
> deferred — see its note. The `internal/atproto` User-Agent and
> `internal/backup` metric-name leaks are also deferred: both are deeper
> refactors (19 call sites; deferred Prometheus registration) with low
> control-flow impact, not extraction blockers.

### C1 — Make explore app-pluggable, or move it under `internal/arabica` ✅

**Implemented as app-pluggable.** Added `explore.Register(nsidBase,
factory)` + `explore.RegistryFor(nsidBase)` (mirroring `backlinks` /
`suggestions`). Arabica registers its factory from
`internal/arabica/entities/explore.go` init; Oolong registers nothing, so
`RegistryFor` returns nil and the firehose treats explore as a no-op for it.
`internal/firehose` no longer calls `explore.NewArabicaRegistry`
unconditionally or defaults `q.App = "arabica"`; it looks up the factory via
`FeedIndex.WithApp(nsidBase, appName)` and tags documents with the configured
app name. `NewArabicaRegistry` is retained as a test fallback for indexes built
without `WithApp`.

Because Oolong does not use explore at all, the lowest-risk option is to **move
`internal/explore` into `internal/arabica/explore`** and move the
explore-specific storage/query code out of `internal/firehose/explore_index.go`
into Arabica-owned code. The `firehose.FeedIndex` would stop owning explore
tables entirely.

If explore is ever wanted for Oolong, the pluggable alternative is to introduce
an `explore.Register(nsidBase, registryFn)` pattern (mirroring `backlinks` and
`suggestions`) and have each app register its own registry; Oolong would
register a tea registry. Either way, `firehose/explore_index.go` must stop
calling `explore.NewArabicaRegistry` unconditionally and defaulting
`q.App = "arabica"`.

Decision deferred to implementation: move-first (simplest, since Oolong has no
explore) vs. pluggable-first (more general). Prefer move-first unless an
Oolong explore surface is approved.

### C2 — Move coffee-specific query methods out of `internal/firehose`

Move `BrewCountsBy*`, `BeanCountsBy*`, and `AvgBrewRatingBy*` out of
`internal/firehose/index.go` into Arabica-owned code (e.g.
`internal/arabica/handlers` or a new `internal/arabica/feedstats`) that queries
the witness SQLite database directly. The firehose package then owns only
app-agnostic ingestion and storage.

Also make `FeedIndex.commentCollection()` default derive from the configured
comment NSID rather than a hardcoded `social.arabica.alpha.comment` literal, so
the fallback is never silently wrong for a non-Arabica app.

### C3 — Drive the SPA shell and assets from `domain.App.Brand` ✅

**Implemented.** Added `SiteDescription`, `LightThemeColor`, and
`DarkThemeColor` to `domain.BrandConfig`; both app constructors populate them.
`spa.NewShellHandler` now takes a `domain.BrandConfig` and reads brand,
tagline, description, and theme colors from it. The five
`if d.AppName == "oolong"` branches and the `brandNameForApp` /
`brandTaglineForApp` helpers are removed. (The CSS-bundle URL-path convention
— arabica as the unnamed default `/static/css/output.css` — is left as a stable
URL contract, not an identity leak.)

Replace the five `if d.AppName == "oolong"` branches in
`internal/web/spa/handler.go` with reads from `domain.App.Brand`
(`DisplayName`, `Tagline`) plus a theme-color field added to `BrandConfig`.
Replace the `b.appName == "arabica"` CSS-path branching in
`internal/web/assets/bundle.go` with a deterministic, app-named path scheme
with no implicit "no-suffix default" special case.

This is strictly an improvement even if Oolong never moves: it removes string
branching in favor of the config struct that already exists.

### C4 — Derive `internal/handlers` defaults from `domain.App` ✅ (partial)

**Implemented:** OG titles use a new `siteName` parameter (from `h.brandName()`)
instead of hardcoded `arabica.social`; `X-Client` header, witness/PDS export
filenames, and the fallback `SiteCardOpts` wordmark derive from `h.app`.
Cookie names now branch on an explicit `App.LegacyUnprefixedCookies` boolean
(Arabica = true) instead of `app == "arabica"` string comparison; `CookieNames`
takes `*domain.App`.

**Deferred:** the `commentNSID` fallback in `FilterHiddenComments` already
derives from `h.app.CommentNSID()` in production; the hardcoded literal is only
a nil-app test fallback and is left as-is.

- OG title/wordmark/site-card from `domain.App.Brand` instead of
  `arabica.social` literals.
- `X-Client` header from `app.Name` or `Brand`.
- Witness/PDS export filenames from `app.Name`.
- Comment NSID fallback from `app.CommentNSID()` rather than a hardcoded
  literal.
- Cookie names: the `app == "" || app == "arabica"` branch exists for legacy
  compatibility with existing Arabica sessions. Keep Arabica's unprefixed
  cookies by an explicit `LegacyUnprefixedCookies bool` on `domain.App` (or a
  brand field) rather than a name string comparison, so the rule is declared
  rather than inferred from identity.

### C5 — Parameterize remaining hardcoded identity ✅ (partial)

**Implemented:** `internal/tracing` `Init` now takes a `serviceName` (passed
`app.Name` from `server.Run`); `internal/routing` otel span name uses
`cfg.App.Name`.

**Deferred:** `internal/atproto` `userAgent` is a package-level `const` reached
via 19 `NewPublicClient()` call sites; making it app-derived requires threading
app identity through all of them (or a package-level setter set at startup) and
is low control-flow impact, so deferred. `internal/backup` metric names
(`arabica_backup_*`) are registered at package `init()` via `promauto` before
app config is known, and the unified `cmd/server` binary runs both apps in one
process (shared metric instances, distinguished by `source` label); deferring
registration to app-config time is a non-trivial refactor, deferred.

- `internal/atproto/client.go` — derive `User-Agent` from `domain.App` (passed
  into the store/client constructor).
- `internal/routing/routing.go:325` — use `app.Name` for the otel span name.
- `internal/backup/metrics.go` — derive metric name prefix from `app.Name` so
  Oolong gets `oolong_backup_*`.
- `internal/signup` — make the `arabica.systems` default explicit per-app in
  each app constructor rather than a shared default.

### C6 — Decide the `internal/lexicons` shared-type story

`internal/lexicons/record_type.go` holds both apps' `RecordType` constants.
For extraction this is a packaging choice, not a behavior change:

- Option A: keep `internal/lexicons` as a shared dependency both repos import.
- Option B: split Oolong's `RecordType` constants into Oolong's own lexicon
  package on extraction.

No work needed now; record the decision at extraction time.

## Extraction shapes

The cleanup above is independent of which way Oolong eventually goes. Two
shapes are plausible.

### Shape A — Oolong forks the repo and imports the platform as a shared module

This is the nearer-term, lower-effort path and matches the expected outcome.

- Oolong's new repo carries only: `cmd/oolong`, `internal/oolong/*` (app,
  entities, handlers, web/Templ pages), `lexicons/social/oolong/`,
  `web/src/lib/app/oolong.ts` plus tea SPA routes/components (when ported),
  and `nix/oolong-module.nix`.
- The shared platform (`internal/{atplatform,atproto,handlers,feed,firehose,...}`)
  is either vendored into Oolong's repo or published as a shared Go module that
  both repos depend on.
- Oolong keeps the hand-rolled firehose/FeedIndex stack for now. It does **not**
  adopt an external AppView yet.

After C1–C5, this is mostly a file move plus import-path rewriting. The shared
packages need no structural changes because the seam already holds.

Risk: the shared platform evolves in the Arabica repo; if Oolong vendors a
copy, the two drift. A shared module avoids drift but adds a release boundary.
This plan does not choose between vendoring and a shared module; that is an
extraction-time decision.

### Shape B — Oolong extracts and adopts an external AppView (longer term)

The hand-rolled derived-state stack — firehose Jetstream consumer, `FeedIndex`
SQLite witness cache, social storage, notifications, explore — is effectively a
bespoke AppView living in `internal/firehose` + `internal/feed` +
`internal/explore` + `internal/notifications` + `internal/social`. If Oolong
points at a real AppView (happyview / quickslice), it can drop that stack and
replace local FeedIndex reads with HTTP calls to the AppView's API.

Why this is attractive for Oolong specifically:

- Oolong barely uses the indexing stack: no explore, no brew-count/rating
  aggregations. It carries the full weight of a firehose/SQLite indexing
  pipeline for feed, notifications, and likes/comments only.
- A mature AppView already serves those shapes.

What Oolong would keep under Shape B: app config, entity
descriptors/`RecordBehavior`, lexicons, Templ pages, tea routes/handlers, and a
thin client layer reading feed/notifications/social from the external AppView.
What it drops: `internal/firehose`, `internal/feed`, `internal/explore`,
`internal/backlinks` (if unused).

Open questions for Shape B (not resolved here):

- Does an external AppView serve the feed/notifications/social shapes Oolong's
  handlers currently read from `FeedIndex`, or are Oolong's reads bespoke
  enough to need adapter shims?
- The exact `FeedIndex`-backed read paths in `internal/oolong/handlers` (feed
  list, notifications, likes/comments) that would need re-pointing need to be
  mapped precisely before committing to Shape B.

Shape B is explicitly future work. The cleanup C1–C5 makes Shape B cheaper if
it is chosen later, because the coffee-specific code will have already left the
shared packages.

## Tests

The cleanup is structural and should be verified with the narrowest checks that
catch regressions in each changed area, per the AGENTS.md verification table.

- After C1/C2 (firehose/explore moves): targeted `go test` on
  `internal/firehose`, `internal/explore`, and `internal/arabica/handlers` to
  confirm the moved query methods still return the same results for Arabica.
  Oolong has no explore/brew-count callers, so its behavior is unchanged by
  construction.
- After C3 (SPA shell/assets): `pnpm run check:svelte` and a smoke load of the
  SPA shell for both `data-app=arabica` and `data-app=oolong` to confirm brand,
  tagline, and theme colors render from `BrandConfig` rather than hardcoded
  branches.
- After C4/C5 (handlers/atproto/routing/backup identity): targeted `go test` on
  `internal/handlers`, `internal/atproto`, `internal/routing`, `internal/backup`
  and an integration check that OG metadata, `X-Client` header, span names, and
  metric names reflect the active app.
- After any change touching `.templ` files: run `templ generate` and include
  generated Go.

> **Verification run (2026-07-16):** `go build ./...` and
> `go build -tags=integration ./...` both clean; `go vet ./...` clean;
> `go test ./...` (excluding integration) all pass;
> `go test -tags=integration ./tests/integration/...` passes (52s);
> `tests/architecture` package-seam guard (`TestSharedPackagesDoNotAddAppImports`)
> still passes with an empty baseline — no shared package imports app code.
> `gofmt` clean on all changed files. No `.templ` files were touched, so no
> `templ generate` was needed.
- Broad checkpoint once a cleanup batch lands: `just ci-check`.

Do not claim full verification when only a partial signal was run.

## Acceptance criteria

A later agent can consider the seam cleanup complete when:

1. No shared package under `internal/*` (excluding `internal/arabica` and
   `internal/oolong`) imports `internal/arabica` or `internal/oolong`. (Already
   true today; must remain true.)
2. No shared package branches on the string `"arabica"` or `"oolong"` for
   control flow. Identity is read from `domain.App`. The only remaining
   `arabica`/`oolong` literals in shared packages are comments, test fixtures,
   or default values explicitly documented as app-specific.
3. `internal/explore` is either app-pluggable (registry pattern) or moved under
   `internal/arabica`. `firehose/explore_index.go` does not call
   `explore.NewArabicaRegistry` unconditionally.
4. `internal/firehose` owns no coffee-specific query methods; `BrewCountsBy*`,
   `BeanCountsBy*`, and rating aggregations live in Arabica-owned code.
5. The SPA shell and CSS bundle read brand/tagline/theme from `domain.App` /
   `BrandConfig`, with no `AppName == "oolong"` / `== "arabica"` branches.
6. `User-Agent`, otel span name, backup metric prefix, OG metadata, `X-Client`
   header, and export filenames all derive from `domain.App`.
7. `go test ./...`, `pnpm run check:svelte`, and `just ci-check` pass after the
   cleanup.
8. An extracted Oolong repo (Shape A) could be produced by moving
   `internal/oolong/*`, `cmd/oolong`, `lexicons/social/oolong/`, and Oolong SPA
   assets, plus depending on the shared platform, with no further changes to
   shared packages.

## Future work

- **The Oolong extraction itself.** This plan prepares the seam; the fork is a
  separate decision and may take Shape A (near term) or Shape B (longer term).
- **Shape B read-path mapping.** Before Oolong adopts an external AppView, map
  every `FeedIndex`-backed read in `internal/oolong/handlers` and confirm the
  AppView's API can serve each shape, or design adapter shims.
- **AppView choice.** Decide between happyview, quickslice, or another AppView
  for Oolong if/when Shape B is pursued.
- **Shared module vs. vendoring.** If Shape A is taken, decide whether the
  platform is a shared Go module (avoids drift, adds a release boundary) or
  vendored into Oolong's repo (simple, drifts).
- **Browser storage key migration.** If Oolong forks and the SPA eventually
  renames `arabica_*` storage keys / custom events, plan a user-visible
  migration for orphaned `localStorage` / `sessionStorage`.
- **`cmd/server` fate.** Decide whether the unified two-app binary stays as a
  dev convenience or is retired once the apps ship in separate repos.
