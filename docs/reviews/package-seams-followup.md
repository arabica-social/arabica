# Package seams follow-up review

Additional thermo-nuclear worker passes focused specifically on package seams, code organization, package boundaries, and abstraction lines.

## Progress notes

Updated 2026-05-29: the following follow-up items have been addressed in the current package-seams cleanup stack.

- Added an architecture seam guard so shared-package imports of `internal/arabica` and `internal/oolong` cannot grow unnoticed; the baseline now ratchets down as seams are fixed.
- App-specific route registration moved out of shared `internal/routing`; Arabica and Oolong route packages now register their own app routes, and the integration harness wires Arabica routes explicitly.
- Suggestion entity path resolution now uses the active app descriptors instead of the global all-app descriptor registry.
- Notification links/action labels now derive record URLs and entity names from the active app descriptors instead of hard-coded Arabica route maps.
- Static About/Terms page selection moved behind app-provided static page renderers.
- Arabica brew readiness onboarding moved from generic `internal/onboarding` to `internal/arabica/onboarding`; shared home rendering no longer imports the Arabica onboarding package directly.
- Request logging middleware now depends on a small request observer interface; Prometheus metrics are wired from routing instead of imported directly by middleware.
- Admin page view models/templates moved from `internal/arabica/web/pages` to shared `internal/web/pages`, so shared admin handlers no longer type against Arabica page structs.
- Home page readiness, OpenGraph description, and site-card behavior moved behind app-provided home behavior callbacks; shared home/feed handling no longer imports Oolong entities for readiness checks.
- SQLite store adapters split by owner: OAuth sessions moved to `internal/atproto/oauthsqlite`, and moderation persistence moved to `internal/moderation/sqlite`.
- Arabica profile handling moved from shared `internal/handlers` into `internal/arabica/handlers`; shared routing no longer owns the Arabica profile routes/partials.
- Layout rendering now receives an explicit asset manifest from server wiring, reducing reliance on package-level CSS/JS href registries for the main layout.
- FeedIndex profile persistence moved behind a narrow internal profile storage type while keeping `FeedIndex` as the public facade.
- The backend unification branch was integrated on top of the package-seams stack; generic record CRUD now uses `internal/records.Store`, Arabica typed store accessors live with Arabica handlers, and app constructors moved into app-owned packages.
- FeedIndex notification persistence moved behind a narrow internal notification storage type while keeping `FeedIndex` as the public facade.
- FeedIndex social persistence for likes and comments moved behind a narrow internal social storage type while keeping `FeedIndex` as the public facade.
- Remaining direct template script hrefs now flow through explicit layout/page data instead of calling asset package globals from templates.
- Feed rendering hooks moved out of domain descriptors into an explicit app-owned feed view registry; app constructors no longer blank-import web components for descriptor mutation.
- Feed edit action URLs moved out of domain descriptors and into app-owned feed views; architecture tests now guard descriptors against feed/web action fields.
- Feed filter labels moved out of domain descriptors and into app-owned feed views; descriptor guards now reject feed UI fields.
- Feed card nouns plus share/delete URL construction moved out of the shared feed page's descriptor lookups and into app-owned feed views; the seam guard now rejects descriptor route-field reads from `feed.templ`.
- Entity route paths and nouns moved from domain descriptors into app-owned `domain.App.EntityRoutes`; routing, suggestions, notifications, modal action URLs, and entity share URLs now resolve through app route metadata.
- Record codec/accessor hooks moved out of domain descriptors into a separate `entities.RecordBehavior` registry; descriptors now carry identity/display metadata only, while feed title/rkey extraction, firehose record conversion/ref hydration, and modal field prefill use behavior lookups.
- Arabica-specific typed feed accessors moved from shared `internal/feed` into Arabica feed components; the shared feed package no longer imports Arabica entities.
- Firehose feed conversion now follows app-owned record behavior reference metadata and shared profile preference types, so `internal/firehose/index.go` no longer imports Arabica or Oolong entity packages.
- Notification value types moved to shared `internal/notifications`; firehose and notification handlers no longer import Arabica entities for notification structs/constants.
- Brew pour JSON serialization moved beside the Arabica brew form view model; shared `internal/web/bff` no longer imports Arabica entities.

Verification for the cleanup stack:

```bash
env GOCACHE=/tmp/arabica-go-cache go test ./...
```

Remaining large seams include descriptor/rendering split, `AtprotoStore`, further `FeedIndex` decomposition, feed/firehose ownership, remaining asset globals, middleware package boundaries, and the web BFF/view helper split.

## App/shared package seams follow-up review

Scope reviewed: `internal/arabica/**`, `internal/oolong/**`, `internal/entities`, `internal/handlers`, `internal/routing`, `cmd/arabica`, `cmd/oolong`.

Review standard: thermo-nuclear structural review focused on package seams, ownership, abstraction lines, app-specific leakage into shared packages, duplicated abstractions, and split/merge candidates.

## Findings

### 1. `internal/handlers` is not actually shared; it imports both apps directly

**Severity: critical structural debt**

`internal/handlers` is supposed to be shared infrastructure, but it imports app-specific entity and page packages:

- `internal/handlers/feed.go:7` imports `internal/arabica/entities`
- `internal/handlers/feed.go:16` imports `internal/oolong/entities`
- `internal/handlers/profile.go:10-12` imports Arabica entities/components/pages
- `internal/handlers/pages.go:6-7` imports Arabica entities and Oolong pages
- `internal/handlers/admin.go:11` imports Arabica pages
- `internal/handlers/notifications.go:8` imports Arabica entities

This package is shared in name only. It is currently an app-aware composition root plus generic web helper package plus Arabica legacy package.

Concrete leaks:

- `internal/handlers/feed.go:68` branches on `h.app.Name == "oolong"`.
- `internal/handlers/feed.go:92-95` checks Oolong readiness by querying `oolong.NSIDVendor`, `oolong.NSIDVessel`, and `oolong.NSIDTea`.
- `internal/handlers/pages.go:21-24` chooses Oolong vs default About page by string app name.
- `internal/handlers/notifications.go:94-100` hard-codes Arabica NSID-to-route mappings.
- `internal/handlers/profile.go:24-31` defines profile data in Arabica terms: beans, roasters, grinders, brewers, brews.

**Code-judo proposal**

Split `internal/handlers` into:

1. `internal/webapp` or `internal/platform/handlers`
   - auth
   - sessions
   - moderation/reporting plumbing
   - generic entity view helpers
   - generic suggestion endpoint
   - no imports from `internal/arabica` or `internal/oolong`

2. `internal/arabica/handlers`
   - Arabica profile
   - Arabica admin page binding if the page is Arabica-specific
   - Arabica feed adornments
   - Arabica onboarding/home behavior

3. `internal/oolong/handlers`
   - Oolong profile/home/onboarding/page behavior

4. A tiny composition layer in `cmd/{arabica,oolong}` or `internal/routing`
   - wires the selected app handler set into shared auth/server infrastructure

The shared package should depend on an interface or app contract supplied by the app, not import app packages.

---

### 2. `internal/routing` is a central god switch for both apps

**Severity: critical structural debt**

`internal/routing/routing.go` imports both app handler packages:

- `internal/routing/routing.go:8` imports `internal/arabica/handlers`
- `internal/routing/routing.go:15` imports `internal/oolong/handlers`

It eagerly constructs both handler sets for every binary:

- `internal/routing/routing.go:42-43`
  - `coffee := coffeehandlers.New(h)`
  - `tea := teahandlers.New(h)`

Then it branches by app name:

- `internal/routing/routing.go:76-79` selects `/api/data` handler by `cfg.App.Name`.
- `internal/routing/routing.go:122` starts an Arabica-specific routing block.
- Multiple Oolong-specific routes are registered in the same shared router.

This destroys app isolation. The Arabica binary has to know about Oolong routes and handlers, and the Oolong binary has to compile Arabica route wiring.

**Code-judo proposal**

Invert routing ownership:

```go
type AppRoutes interface {
    RegisterRoutes(mux *http.ServeMux, base *handlers.Handler)
}
```

Then:

- `internal/arabica/handlers` owns Arabica route registration.
- `internal/oolong/handlers` owns Oolong route registration.
- `internal/routing` registers only truly shared routes:
  - login/logout/auth callback
  - static assets
  - health checks
  - generic profile lookup if truly app-neutral
  - moderation/admin only if app-neutral

The composition root should be:

```go
routing.SetupRouter(routing.Config{
    Platform: sharedHandler,
    App: app,
    AppRoutes: arabica.Routes{},
})
```

not a shared package importing every app.

---

### 3. The global descriptor registry mixes domain metadata, rendering hooks, and mutable side effects

**Severity: high**

`internal/entities/entities.go` is presented as a generic descriptor registry:

- `internal/entities/entities.go:18-33` defines `Descriptor` with domain-ish fields and `GetField`.
- `internal/entities/entities.go:39` adds `RecordToModel`.
- `internal/entities/entities.go:47` adds `RenderFeedContent`.
- `internal/entities/entities.go:88-90` stores descriptors in global `registry` and `nsidIndex`.
- `internal/entities/entities.go:95-100` globally registers descriptors.

The descriptor object is doing too much:

1. Domain identity: NSID, record type, display names.
2. Serialization: record/model conversion.
3. UI behavior: templ rendering hooks.
4. Routing-ish behavior: URL paths and modal URLs.
5. Suggestion/search metadata.
6. Runtime global registration.

Then app web component packages mutate descriptors via side effects:

- `cmd/arabica/app.go:8-10` blank-imports Arabica web components to run `init`.
- `cmd/oolong/app.go:8-10` blank-imports Oolong web components to run `init`.
- `internal/arabica/web/components/descriptor_bridge.go:1-4` says importing the package for side effects wires templ hooks.
- `internal/arabica/web/components/descriptor_bridge.go:20-33` mutates descriptors with `RenderFeedContent`.
- `internal/oolong/web/components/descriptor_bridge.go:1-3` does the same for Oolong.
- `internal/oolong/web/components/descriptor_bridge.go:21-34` mutates descriptors with render hooks.

This is fragile: app correctness depends on blank imports and init ordering. It also forces the shared entity registry to import `templ`, which makes a domain registry know about rendering.

**Code-judo proposal**

Split descriptors into separate immutable contracts:

```go
type EntitySchema struct {
    Type lexicons.RecordType
    NSID string
    DisplayName string
    URLPath string
    Fields []Field
    Codec EntityCodec
}

type EntityViews struct {
    FeedContent func(feed.Item) templ.Component
    RecordContent func(any) templ.Component
    Table func(any) templ.Component
}

type AppManifest struct {
    Schemas []EntitySchema
    Views map[lexicons.RecordType]EntityViews
    Routes AppRoutes
}
```

No global mutation. No blank imports. `cmd/arabica/app.go` should explicitly call `arabica.NewApp()` that returns a complete manifest.

---

### 4. App selection is stringly typed and leaks into behavior paths

**Severity: high**

Shared handlers branch on literal names:

- `internal/handlers/feed.go:68` checks `h.app.Name == "oolong"`.
- `internal/handlers/feed.go:135` checks `h.app.Name == "oolong"`.
- `internal/handlers/pages.go:21` checks `h.app.Name == "oolong"`.
- `internal/routing/routing.go:76` checks `cfg.App.Name == "oolong"`.
- `internal/routing/routing.go:122` checks `cfg.App.Name == "arabica"`.

This is a package seam smell: behavior should be owned by app implementations, not selected by string constants inside shared code.

**Code-judo proposal**

Move app-specific behavior behind explicit callbacks/capabilities:

```go
type HomeBehavior interface {
    HomeRedirectOrRender(...)
    Readiness(...)
    SiteCardOptions(...)
}

type StaticPages interface {
    About(...)
    Terms(...)
    ATProtocol(...)
}
```

Then `handlers.Handler` calls capability methods without knowing whether the app is Arabica or Oolong.

---

### 5. Profile handling is Arabica-owned but lives in shared handlers

**Severity: high**

`internal/handlers/profile.go` is structurally Arabica-specific:

- `internal/handlers/profile.go:10-12` imports Arabica entities, components, and pages.
- `internal/handlers/profile.go:24-31` defines `ProfileDataBundle` around beans, roasters, grinders, brewers, brews.
- The shared handler package therefore cannot provide a generic profile feature without compiling Arabica UI/domain types.

This is misplaced ownership. Either profile is a shared platform concept with app-supplied sections/cards, or it is an app-specific feature.

**Code-judo proposal**

Make profile assembly app-owned:

```go
type ProfileRenderer interface {
    FetchProfileSections(ctx, store) ([]ProfileSection, error)
    RenderProfile(...)
}
```

Arabica then owns coffee-specific profile sections. Oolong owns tea-specific sections. Shared code can still resolve actors, moderation, and viewer context.

---

### 6. Notifications are hard-coded to Arabica routes in shared handlers

**Severity: high**

`internal/handlers/notifications.go` imports Arabica entities:

- `internal/handlers/notifications.go:8`

It hard-codes Arabica collection URLs:

- `internal/handlers/notifications.go:93-100`

This is exactly what `internal/entities.Descriptor.URLPath` appears intended to solve, but the shared notification code bypasses the descriptor/app manifest and bakes in Arabica.

**Code-judo proposal**

Delete `collectionURLPath` and `collectionDisplayName` maps from shared notification code. Resolve through the selected app manifest:

```go
d := h.app.DescriptorByNSID(collection)
url := "/" + d.URLPath + "/" + rkey
name := d.Noun
```

If a notification targets a collection outside the current app, render a neutral fallback.

---

### 7. Suggestions use a global all-app registry and path collision rules

**Severity: medium-high**

`internal/handlers/suggestions.go` builds `entityTypeToNSID` from every globally registered descriptor:

- `internal/handlers/suggestions.go:47-51`
  - iterates `entities.All()`
  - maps `d.URLPath` to `d.NSID`

The comment acknowledges collision behavior:

- `internal/handlers/suggestions.go:43-47`

This means a request in one app can be affected by descriptors registered by the other app. The app boundary is global mutable state, not request/app scoped.

**Code-judo proposal**

Build suggestion routing from `h.app.Descriptors`, not `entities.All()`.

```go
func (h *Handler) nsidForSuggestionPath(path string) (string, bool) {
    for _, d := range h.app.Descriptors {
        if d.URLPath == path {
            return d.NSID, true
        }
    }
    return "", false
}
```

This eliminates global collision semantics.

---

### 8. Admin page types are Arabica page types inside shared admin handlers

**Severity: medium-high**

`internal/handlers/admin.go` imports Arabica pages:

- `internal/handlers/admin.go:11`

It uses Arabica page data types throughout shared admin logic:

- `internal/handlers/admin.go:156` uses `coffeepages.EnrichedReport`
- `internal/handlers/admin.go:184` uses `coffeepages.AdminStats`
- `internal/handlers/admin.go:193` returns `coffeepages.AdminProps`
- `internal/handlers/admin.go:241` renders `coffeepages.Admin`
- `internal/handlers/admin.go:253` renders `coffeepages.AdminDashboardBody`

If admin is platform/shared, its view models and pages should live under shared web packages. If it is Arabica-branded, it should live under `internal/arabica`.

**Code-judo proposal**

Either:

1. Move admin UI models/pages to `internal/web/pages/admin`, and keep shared handler ownership.

or:

2. Move admin handler to `internal/arabica/handlers` and add an Oolong admin handler only when needed.

Do not keep shared backend logic typed against Arabica page structs.

---

### 9. Arabica and Oolong have parallel generic CRUD abstractions that should either be shared or intentionally app-owned

**Severity: medium**

There are two app-specific generic CRUD helper layers:

- `internal/arabica/handlers/crud_generic.go:14-27` defines `arabicaValidator` and `arabicaCRUDCreate`.
- `internal/oolong/handlers/crud_generic.go:11-33` defines `oolongValidator` and `oolongCRUDWrite`.

They are similar but not identical:

- Arabica helpers still route through typed `database.Store` methods.
- Oolong helpers use generic `atproto.PutRecord` style paths.

This split may be a migration artifact, but structurally it creates two abstraction systems for the same concept: validate/decode/write/invalidate/respond.

**Code-judo proposal**

Pick one direction:

1. If generic entity storage is the target:
   - promote the Oolong generic write path into shared infrastructure
   - adapt Arabica entities to it
   - delete most typed CRUD plumbing

2. If typed domain stores are the target:
   - give Oolong typed store methods too
   - delete the generic PutRecord-specific helper layer

Do not let both abstractions survive indefinitely. They encode different ownership models.

---

### 10. `cmd/arabica` and `cmd/oolong` duplicate binary bootstrap and rely on side-effect imports

**Severity: medium**

Both binaries duplicate the same shape:

- `cmd/arabica/main.go:29-46`
- `cmd/oolong/main.go:29-46`

Both app builders blank-import web components for descriptor mutation:

- `cmd/arabica/app.go:8-10`
- `cmd/oolong/app.go:8-10`

This keeps composition implicit. The binary should construct a complete app value explicitly, not depend on `init()` hooks from a web package.

**Code-judo proposal**

Create explicit app constructors:

```go
// internal/arabica/app
func New() *domain.App

// internal/oolong/app
func New() *domain.App
```

Those constructors should explicitly assemble:

- entity schemas
- codecs
- renderers
- routes
- static pages
- onboarding behavior
- OAuth scopes

Then both `cmd/*/main.go` can call a shared `server.Run(app)`.

---

### 11. Entity route bundles are a good idea but are only half-applied

**Severity: medium**

There is a promising route bundle abstraction:

- `internal/handlers/entity_routes.go:9-24` defines `EntityRouteBundle`.
- `internal/arabica/handlers/routes.go:8-10` returns Arabica bundles for simple entities.
- `internal/oolong/handlers/routes.go:8-10` returns Oolong bundles.

But `internal/routing/routing.go` still contains explicit app-specific route blocks:

- `internal/routing/routing.go:122` begins Arabica-specific registration.
- `internal/routing/routing.go:127-135` registers brew routes explicitly.
- Oolong-specific routes are also registered directly in the shared router.

**Code-judo proposal**

Promote route bundles to the app manifest and make them complete:

```go
type AppRoutes struct {
    EntityBundles []EntityRouteBundle
    ExtraRoutes func(mux *http.ServeMux, h *handlers.Handler)
}
```

Then shared routing does not know brew vs tea vs cafe. It just invokes the app route registrar.

---

## Package boundary target state

A cleaner package graph should look like:

```text
cmd/arabica
  -> internal/arabica/app
      -> internal/arabica/entities
      -> internal/arabica/handlers
      -> internal/arabica/web/...
  -> internal/atplatform/server

cmd/oolong
  -> internal/oolong/app
      -> internal/oolong/entities
      -> internal/oolong/handlers
      -> internal/oolong/web/...
  -> internal/atplatform/server

internal/atplatform/server
  -> internal/routing or internal/platform/http
  -> internal/handlers/shared
  -> no imports from internal/arabica or internal/oolong

internal/entities
  -> pure descriptor/schema/codec metadata only
  -> no templ
  -> no global mutable app-spanning registry
```

Forbidden dependencies in the target state:

```text
internal/handlers -> internal/arabica
internal/handlers -> internal/oolong
internal/routing  -> internal/arabica
internal/routing  -> internal/oolong
internal/entities -> github.com/a-h/templ
cmd/*             -> blank import web components for side effects
```

## Recommended execution order

1. **Stop new leakage first**
   - Add a convention/test that shared packages cannot import `internal/arabica` or `internal/oolong`.

2. **Extract app route registration**
   - Move app-specific route blocks out of `internal/routing`.

3. **Replace string app-name branching with capabilities**
   - Home behavior, static pages, readiness, site cards, and route sets.

4. **Split descriptor schema from renderers**
   - Remove `templ` from `internal/entities`.
   - Remove descriptor mutation from web component `init()` hooks.

5. **Move profile/admin ownership**
   - Either shared-neutralize the view models/pages or move the handlers into app packages.

6. **Unify CRUD write abstractions**
   - Choose typed store or generic entity store, then remove the duplicate helper family.

## Bottom line

The current codebase is halfway through becoming multi-app, but the package graph still says “Arabica core with Oolong bolted on.” The biggest simplification is not adding another abstraction layer; it is moving ownership back to the app packages and making shared packages truly app-agnostic.

The hard rule should be:

> Shared packages may accept app-provided manifests/capabilities, but must not import Arabica or Oolong packages.

## Data/storage/backend package seams follow-up

Scope reviewed: `internal/atproto`, `internal/database`, `internal/database/sqlitestore`, `internal/firehose`, `internal/feed`, `internal/backup`, `internal/atplatform`.

Review standard: strict structural/code-quality review focused on package seams, ownership, abstraction lines, and simplification opportunities. No source edits made.

## Executive verdict

The backend data layer has too many “almost boundaries” and not enough real ones. The biggest structural problem is that several packages pretend to be narrow abstractions but actually carry app/domain knowledge across seams:

- `internal/atproto` is not just AT Protocol transport; it owns Arabica entity persistence, app-specific like/comment NSIDs, session cache policy, witness-cache read policy, reference resolution, and social operations.
- `internal/firehose` is not just firehose ingestion; `FeedIndex` is simultaneously SQLite schema owner, record index, feed query engine, profile cache, registry persistence, social lookup store, notifications store, witness cache, and feed-item projector.
- `internal/feed` is not just feed orchestration; `FeedItem` is a cross-app god DTO and `feed` defines interfaces implemented by `firehose`, while `firehose` also imports `feed`, creating an adapter seam that mostly exists to paper over package ownership confusion.
- `internal/database.Store` is a legacy all-entities interface whose width leaks into mocks and keeps unrelated concerns coupled.

The fastest code-judo move is not to add more abstraction. It is to delete fake abstraction layers and split god objects along existing concrete responsibilities.

---

## Findings

### 1. Critical: `database.Store` is a mega-interface, not a boundary

Evidence:

- `internal/database/store.go:13` defines `type Store interface`.
- The same interface owns brew, bean, roaster, grinder, brewer, cafe, drink, recipe, like, comment, generic record, and DID operations across `internal/database/store.go:17-?`.
- `internal/database/store_mock.go:11` defines `type MockStore struct`.
- The mock mirrors the interface with dozens of function fields beginning at `internal/database/store_mock.go:12`.

Why this is structurally bad:

`Store` is named as if it is a persistence boundary, but it is actually an app-service surface for every Arabica entity plus social behavior plus generic ATProto record access. Anything accepting `database.Store` implicitly accepts the entire application persistence universe.

This interface also fails the “consumer owns the interface” rule. It is defined centrally and implemented by `AtprotoStore`, so every caller inherits the broadest possible dependency instead of declaring the few methods it needs.

Code-judo proposal:

- Delete or stop exporting the central `database.Store` as the primary seam.
- Move narrow interfaces to consumers:
  - handlers that need beans define `BeanStore`.
  - handlers that need likes define `LikeStore`.
  - backup/export code defines `RecordSource`.
  - reference resolution defines `RecordResolver`.
- Keep concrete implementations concrete until a second implementation exists.
- Delete `MockStore` once call sites use local small interfaces; tests can stub only the methods they need.

Expected simplification:

- Smaller mocks.
- Less accidental coupling.
- Clearer ownership: database package no longer pretends to own the app domain.

---

### 2. Critical: `AtprotoStore` is a god object crossing transport, cache, witness, domain, and social boundaries

Evidence:

- `internal/atproto/store.go:20` defines `type AtprotoStore struct`.
- Fields include transport/client identity plus cache infrastructure:
  - `client *Client` at `internal/atproto/store.go:21`
  - `did syntax.DID` at `internal/atproto/store.go:22`
  - `sessionID string` at `internal/atproto/store.go:23`
  - `cache *SessionCache` at `internal/atproto/store.go:24`
  - `witnessCache WitnessCache` at `internal/atproto/store.go:25`
- App-specific like/comment collection state is also embedded:
  - comment says these are app-specific at `internal/atproto/store.go:27`
  - `likeNSID` and `commentNSID` at `internal/atproto/store.go:31-32`
- Constructors multiply around the same object:
  - `NewAtprotoStore` at `internal/atproto/store.go:37`
  - `NewAtprotoStoreWithWitness` at `internal/atproto/store.go:48`
  - `NewAtprotoStoreForApp` at `internal/atproto/store.go:60`
- It owns social methods directly:
  - `CreateLike` at `internal/atproto/store.go:986`
  - `CreateComment` at `internal/atproto/store.go:1089`

Why this is structurally bad:

This type is named as if it is “ATProto storage,” but it has at least five jobs:

1. authenticated ATProto repo client,
2. typed Arabica entity repository,
3. app-specific social collection selector,
4. session-cache policy coordinator,
5. witness-cache read/write-through coordinator.

Those are not implementation details of one concept. They are separate axes of change. Adding Oolong or new entity types forces the ATProto package to know app semantics it should not own.

Code-judo proposal:

Split by behavior, not by entity count:

- `RepoClient` / `RecordRepository`: raw `GetRecord`, `PutRecord`, `ListRecords`, `DeleteRecord`.
- `CachedRecordRepository`: wraps raw repository with session/witness policy.
- `ArabicaRepository`: typed Arabica entity methods, in an Arabica-owned package.
- `SocialRepository`: like/comment operations parameterized by app collection config.
- Keep `AtprotoStore` only as a thin composition root temporarily, then delete it once handlers depend on smaller interfaces.

The main simplification is to make cache/witness logic decorate raw record access instead of living inside every typed store method.

---

### 3. Critical: `internal/atproto` imports Arabica domain models, so the protocol package is app-owned in disguise

Evidence:

- `internal/atproto/store.go:9` imports `internal/arabica/entities`.
- `internal/atproto/store_arabica_codecs.go:8` imports `internal/arabica/entities`.
- `internal/atproto/store_generic.go:9` imports `internal/arabica/entities`.
- `internal/atproto/store_arabica_codecs.go:17` starts Arabica-specific codecs such as `roasterCodec`.
- `internal/atproto/store_arabica_codecs.go:49-50` wires `PostGet` to `resolveBeanRefs`.
- `internal/atproto/store.go:613` defines `resolveBeanRefs`.
- `internal/atproto/store.go:876` defines `resolveRecipeRefs`.

Why this is structurally bad:

A package called `atproto` should own protocol mechanics: auth client, repo operations, NSID handling, record marshal/unmarshal primitives. It should not know Arabica beans, roasters, recipes, or reference-resolution rules.

This dependency direction makes reuse impossible without dragging Arabica into protocol code. It also makes Oolong support structurally awkward: either `atproto` grows more app imports, or more app-specific exceptions are injected into the same god object.

Code-judo proposal:

- Move `store_arabica_codecs.go` out of `internal/atproto`.
- Put typed Arabica repository/codecs under an Arabica-owned package, for example:
  - `internal/arabica/repository`
  - `internal/arabica/atproto`
- Leave only generic record operations in `internal/atproto`.
- Make reference resolution a domain repository concern, not an ATProto transport concern.

Rule of thumb: if a file mentions `Bean`, `Roaster`, `Recipe`, `Brew`, `Cafe`, or `Drink`, it should not live in `internal/atproto`.

---

### 4. Critical: `FeedIndex` is a database god object with at least seven responsibilities

Evidence:

- `internal/firehose/index.go:75` defines `type FeedIndex struct`.
- It owns app-specific social collection config through `commentNSID` at `internal/firehose/index.go:85`.
- It owns profile cache state:
  - `profileCache` at `internal/firehose/index.go:88`
  - `profileCacheMu` at `internal/firehose/index.go:89`
- It creates many unrelated SQLite tables in one schema owner:
  - `records` at `internal/firehose/index.go:132`
  - `meta` at `internal/firehose/index.go:147`
  - `known_dids` at `internal/firehose/index.go:152`
  - `registered_dids` at `internal/firehose/index.go:153`
  - `backfilled` at `internal/firehose/index.go:157`
  - `profiles` at `internal/firehose/index.go:159`
  - `did_by_handle` at `internal/firehose/index.go:165`
  - `likes` at `internal/firehose/index.go:172`
  - `comments` at `internal/firehose/index.go:180`
  - `notifications` at `internal/firehose/index.go:193`
- It owns record deletion at `internal/firehose/index.go:584`.
- It owns record lookup at `internal/firehose/index.go:756`.
- It owns feed querying at `internal/firehose/index.go:810`.
- It owns profile lookup at `internal/firehose/index.go:1139`.
- It owns profile stats visibility at `internal/firehose/index.go:1728`.
- It owns batch record lookup at `internal/firehose/index.go:2056`.
- It owns comment lookup at `internal/firehose/index.go:2145`.

Why this is structurally bad:

`FeedIndex` has become “the SQLite backend.” The name says feed indexing, but the type owns witness-cache lookup, registry persistence, profile cache, feed projections, notifications, likes/comments, known-DID/backfill state, and moderation-adjacent profile visibility.

This creates package gravity. Every new backend feature finds the existing `FeedIndex` pointer and adds another method. That is the god-object failure mode.

Code-judo proposal:

Keep one SQLite database if desired, but split ownership by repository:

- `RecordIndex`: `records` table, witness cache operations, raw indexed records.
- `FeedProjector` / `FeedQueryStore`: feed-specific SQL and feed item construction.
- `ProfileStore`: `profiles`, `did_by_handle`, profile cache.
- `SocialIndex`: `likes`, `comments`.
- `NotificationStore`: notifications table and queries.
- `RegistryStore`: `registered_dids`, `known_dids`, `backfilled`.

These can share `*sql.DB` and a migrations package. The split does not require multiple DB files.

Immediate judo move:

- Create small structs over the existing `*sql.DB`.
- Move methods without changing SQL.
- Keep `FeedIndex` temporarily as a facade for compatibility.
- Delete the facade once call sites accept the narrower stores.

---

### 5. High: `firehose` imports app domains and `feed`, so ingestion owns projection and presentation DTOs

Evidence:

- `internal/firehose/index.go:15` imports `internal/arabica/entities`.
- `internal/firehose/index.go:18` imports `internal/feed`.
- `internal/firehose/index.go:19` imports `internal/lexicons`.
- `internal/firehose/index.go:20` imports `internal/oolong/entities`.
- `internal/firehose/index.go:1065` comments `recordToFeedItem converts an IndexedRecord to a FeedItem`.
- `internal/firehose/index.go:1068` defines `func (idx *FeedIndex) recordToFeedItem(...)`.
- `internal/firehose/index.go:1049` calls `idx.recordToFeedItem`.

Why this is structurally bad:

The firehose package should ingest and index records. Instead, it knows how to turn Arabica and Oolong domain records into `feed.FeedItem`. That means feed presentation shape drives ingestion package dependencies.

The presence of both Arabica and Oolong imports in `firehose/index.go` is a smell: app-specific record projection is not a firehose concern.

Code-judo proposal:

- Move `recordToFeedItem` out of `firehose`.
- Define a projector registry outside ingestion:
  - input: indexed raw record + resolved referenced records/profiles
  - output: `feed.FeedItem`
- App packages register projectors for their record types.
- `firehose` should expose indexed records; `feed` or app-level feed composition should project them.

Even simpler:

- Rename current `firehose.FeedIndex` package responsibility to `index` or `witness` if it remains a query DB.
- Make firehose consumer depend on `RecordSink`, not on the full feed index.

---

### 6. High: `feed` and `firehose` have a circular conceptual dependency hidden by an adapter

Evidence:

- `internal/feed/service.go:196` defines `type FirehoseIndex interface`.
- That interface requires methods like:
  - `GetRecentFeed` at `internal/feed/service.go:198`
  - `GetFeedWithQuery` at `internal/feed/service.go:199`
- `internal/firehose/adapter.go:9` says `FeedIndexAdapter wraps FeedIndex to implement feed.FirehoseIndex`.
- `internal/firehose/adapter.go:12-13` says the adapter exists to translate `feed.FirehoseFeedQuery` into `firehose.FeedQuery`.
- `internal/firehose/adapter.go:25` proxies `IsReady`.
- `internal/firehose/adapter.go:35-36` proxies `GetFeedWithQuery`.

Why this is structurally bad:

The adapter is not adapting an external system. It is adapting two internal packages that disagree about who owns feed query types. Worse, `firehose/index.go` imports `feed`, while `feed` defines an interface called `FirehoseIndex`. That is a conceptual cycle even if Go import cycles are avoided.

The comment in `adapter.go` admits the problem: “Items flow through unchanged”; the adapter mostly translates duplicate query structs.

Code-judo proposal:

Pick one owner:

Option A — feed owns feed queries and projection:
- `firehose` exposes raw/indexed records.
- `feed` queries through a `RecordQueryStore` and constructs `FeedItem`.
- Delete `FeedIndexAdapter`.

Option B — firehose/index owns feed query results:
- Move `FirehoseFeedQuery` to a neutral package or use `firehose.FeedQuery` directly.
- `feed.Service` depends on an interface with firehose-owned query type.
- Delete duplicate query structs and adapter.

Preferred judo move: Option A. Feed presentation belongs with `feed`, not ingestion.

---

### 7. High: `SessionCache` is an app/entity cache wearing a generic map

Evidence:

- `internal/atproto/cache.go:14-15` says `Records` values are typed slices and accessor methods cast them.
- `internal/atproto/cache.go:19` defines `type UserCache struct`.
- `internal/atproto/cache.go:20` stores `Records map[string]any`.
- `internal/atproto/cache.go:21` stores a single `Timestamp time.Time`.
- `internal/atproto/cache.go:86` shows `SessionCache` has a mutex.
- `internal/atproto/cache.go:128` initializes `newCache.Records = make(map[string]any)`.

Why this is structurally bad:

`map[string]any` is not a clean abstraction; it is erased typing. The comments acknowledge that callers know which typed slices are hiding under each NSID. That makes cache correctness dependent on convention, not structure.

The cache also mixes two policies:

1. per-session TTL cache,
2. dirty collection tracking to avoid stale witness reads.

Those are different concerns. Dirty witness invalidation belongs with a witness-read-through repository, not the session value cache itself.

Code-judo proposal:

- Replace `map[string]any` with raw record cache keyed by collection, or typed repositories each owning their typed cache.
- Move dirty collection handling into a `WitnessReadPolicy` wrapper.
- If keeping generic cache, store `[]json.RawMessage` or typed `RecordEnvelope`, then decode at the repository edge.

Immediate simplification:

- Make the cache unaware of Arabica typed slices.
- Cache raw ATProto records; typed conversion remains in app repositories.

---

### 8. High: `WitnessCache` interface lives in `atproto`, but the witness implementation is firehose/SQLite

Evidence:

- `internal/atproto/witness.go:9` defines `WitnessRecord`.
- `internal/atproto/witness.go:36` defines `type WitnessCache interface`.
- The methods include record indexing operations such as `GetRecord`, `ListRecords`, `UpsertRecord`, and `DeleteRecord` in `internal/atproto/witness.go:36+`.
- `internal/firehose/index.go` owns the `records` table at `internal/firehose/index.go:132`.
- `internal/firehose/index.go:756` defines record lookup on `FeedIndex`.
- `internal/firehose/index.go:2056` defines batch record lookup on `FeedIndex`.

Why this is structurally bad:

`WitnessCache` is not an ATProto concept; it is this application’s local indexed replica/witness. Placing the interface in `atproto` makes the protocol package own the shape of a local SQLite index.

This also encourages `AtprotoStore` to know witness behavior directly, which is why store reads mix session cache, witness cache, and PDS fallback.

Code-judo proposal:

- Move `WitnessRecord` and `WitnessCache` to a neutral package, for example:
  - `internal/records`
  - `internal/witness`
  - `internal/index`
- Let `atproto` expose raw PDS repository only.
- Build a `ReadThroughRepository` outside `atproto` that tries session cache, witness index, then PDS.

The code-judo simplification is composition:

```text
TypedRepository
  -> CachedRecordRepository
    -> WitnessReadThroughRepository
      -> PDSRecordRepository
```

Each layer has one policy.

---

### 9. Medium: `sqlitestore` is a grab-bag for unrelated SQLite adapters

Evidence:

- `internal/database/sqlitestore/moderation.go:14-16` defines `ModerationStore`.
- `internal/database/sqlitestore/moderation.go:27` asserts it implements `moderation.Store`.
- `internal/database/sqlitestore/oauth.go:15-21` defines `OAuthStore`.
- `internal/database/sqlitestore/oauth.go:29` asserts it implements `oauth.ClientAuthStore`.
- `internal/database/sqlitestore/oauth.go:31` starts session methods with `GetSession`.

Why this is structurally weak:

`sqlitestore` groups code by technology instead of ownership. OAuth session persistence and moderation persistence do not share a domain boundary. They only share `*sql.DB`.

This is tolerable while tiny, but it reinforces the same architectural pattern as `FeedIndex`: “SQLite things go here.” That is not a useful package seam.

Code-judo proposal:

- Move SQLite adapters next to the owning domain packages:
  - `internal/moderation/sqlite`
  - `internal/oauth/sqlite` or `internal/atproto/oauth/sqlite`
- Keep shared migration/DB-opening helpers separate if needed.
- Let packages expose their own store interfaces and SQLite implementation.

Do not create a generic `database/sqlite` dump package. Shared technology is not shared ownership.

---

### 10. Medium: `atplatform/server.Run` is a composition root, but it also owns too many lifecycle policies inline

Evidence:

- `internal/atplatform/server/server.go:74` defines `func Run`.
- It resolves app data directory around `internal/atplatform/server/server.go:80`.
- It initializes tracing around `internal/atplatform/server/server.go:95`.
- It builds the DB path at `internal/atplatform/server/server.go:128`.
- It configures firehose index path at `internal/atplatform/server/server.go:136`.
- It constructs `FeedIndex` at `internal/atplatform/server/server.go:144`.
- It imports and wires `backup`, `sqlitestore`, `feed`, `firehose`, and `handlers` at `internal/atplatform/server/server.go:21-26`.

Why this is structurally weak:

`Run` is the composition root, so some wiring is acceptable. The problem is that lifecycle decisions, data-dir policy, tracing singleton behavior, database ownership, firehose wiring, backup wiring, HTTP/router wiring, and shutdown behavior are all in one long function.

This makes app startup hard to test in slices and encourages new infrastructure to be bolted into `Run`.

Code-judo proposal:

Keep `Run`, but factor by lifecycle ownership:

- `openAppDatabase(app, opts)`.
- `buildStores(db, app)`.
- `buildFeedPipeline(db, app, opts)`.
- `buildHTTPServer(app, stores, services)`.
- `startBackgroundWorkers(ctx, workers)`.

Avoid adding an abstraction framework. Just cut the function into named constructors that return concrete structs.

---

### 11. Medium: `backup` package is cleanly isolated, but its source/destination abstraction may be overbuilt for current use

Evidence:

- `internal/backup/backup.go:41` defines `type Source interface`.
- `internal/backup/backup.go:46` defines `type Destination interface`.
- `internal/backup/backup.go:63` stores status in a manager map.
- `internal/backup/backup.go:72` initializes `status`.

Why this is structurally mixed:

Compared to the other packages, `backup` is relatively coherent. It has a clear source/destination/manager model.

The risk is premature abstraction: if there is only one SQLite source and one local destination, Source/Destination are likely test seams rather than domain seams. That is not fatal, but it should not grow into a framework.

Code-judo proposal:

- Keep if tests or multiple destinations use it.
- Otherwise collapse to concrete `SQLiteBackupManager` until a second real implementation exists.
- Do not let backup abstractions leak into `database` or `firehose`.

---

## Cross-cutting deletion/simplification plan

### Step 1: Delete fake feed/firehose adapter seam

Target:

- `internal/firehose/adapter.go`
- duplicate feed/firehose query structs

Move toward one owner for feed query DTOs. The lowest-risk path is to make `feed` own feed queries and projection, while `firehose` exposes indexed records.

### Step 2: Split `FeedIndex` without changing tables

Create narrow structs over the same `*sql.DB`:

```text
RecordIndex
ProfileStore
SocialIndex
NotificationStore
RegistryStore
FeedQueryStore
```

Initially, keep `FeedIndex` as:

```go
type FeedIndex struct {
    Records       *RecordIndex
    Profiles      *ProfileStore
    Social        *SocialIndex
    Notifications *NotificationStore
    Registry      *RegistryStore
    Feed          *FeedQueryStore
}
```

Then move methods one group at a time. No schema migration required.

### Step 3: Move Arabica typed repositories out of `internal/atproto`

Target files:

- `internal/atproto/store_arabica_codecs.go`
- Arabica-specific methods in `internal/atproto/store.go`
- Arabica-specific imports in `internal/atproto/store_generic.go`

Destination should be Arabica-owned, not protocol-owned.

### Step 4: Replace `database.Store` with consumer-owned narrow interfaces

Do not split `database.Store` into `BeanStore`, `BrewStore`, etc. in the database package. That just creates interface confetti.

Instead:

- each handler/service declares the methods it consumes;
- concrete repositories implement them naturally;
- mocks shrink to local test fakes.

### Step 5: Make cache/witness/PDS fallback a repository decorator

Current policy is smeared through `AtprotoStore`.

Desired ownership:

```text
PDSRecordRepository: talks XRPC only
WitnessRepository: talks SQLite indexed records only
SessionRecordCache: TTL cache only
ReadThroughRepository: policy: session -> witness -> PDS
TypedRepository: decode/encode domain records
```

This deletes dirty-cache logic from typed entity methods and makes stale-read policy testable in one place.

---

## Package ownership target state

Recommended target boundaries:

```text
internal/atproto
  OAuth, identity, XRPC client, raw repo operations only.

internal/witness or internal/index
  SQLite indexed record replica, raw record lookup/list/upsert/delete.

internal/arabica/repository
  Arabica typed repositories, codecs, reference resolution.

internal/oolong/repository
  Oolong typed repositories, codecs, reference resolution.

internal/feed
  Feed service, feed query DTOs, feed item projection orchestration.

internal/firehose
  Jetstream consumer/event parsing and dispatch to sinks only.

internal/moderation/sqlite
  SQLite implementation of moderation store.

internal/oauth/sqlite or internal/atproto/oauth/sqlite
  SQLite OAuth session store.

internal/atplatform/server
  Thin composition root and lifecycle orchestration.
```

---

## Bottom line

The current backend works by centralizing state into two god objects:

- `AtprotoStore`
- `FeedIndex`

Most of the “abstractions” around them are compensating for those god objects rather than reducing complexity.

The highest-leverage simplification is:

1. make raw record access the primitive;
2. make cache/witness/PDS fallback a wrapper around raw record access;
3. move typed app repositories out of `atproto`;
4. split `FeedIndex` into small SQLite stores over the same DB;
5. delete the `firehose`/`feed` adapter by choosing one package to own feed query DTOs and projection.

## Cross-Cutting Backend Package Seams Review

Scope reviewed: `internal/middleware`, `internal/web/bff`, `internal/web/assets`, `internal/signup`, `internal/onboarding`, `internal/suggestions`, `internal/metrics`, `internal/tracing`, `internal/logging`, `tests/`.

## Findings

### 1. `internal/suggestions` is a global mutable registry pretending to be a pure service

**Severity: High**

Evidence:

- `internal/suggestions/suggestions.go:38` exposes `var PreferredDIDs = map[string]struct{}{}`.
- `internal/suggestions/suggestions.go:59` stores `var entityConfigs = map[string]FieldConfig{}`.
- `internal/suggestions/suggestions.go:66` mutates that global registry via `Register`.
- `internal/arabica/entities/suggestions.go:15`, `:21`, `:27`, `:33`, `:55` register Arabica configs through package init side effects.
- `internal/oolong/entities/suggestions.go:13`, `:19`, `:25`, `:31` does the same for Oolong.
- `internal/handlers/suggestions.go:43-47` documents the seam failure directly: overlapping app URL paths depend on iteration order.

The package has unclear ownership: entity packages own configs, handlers own app scoping, `suggestions` owns global state, and `handlers` adapts `firehose.FeedIndex` because importing entity packages would create cycles.

**Code-judo proposal:**

Make suggestions app-scoped and registry-owned.

- Move suggestion config into the entity registry/descriptor layer.
- Replace `suggestions.Register` and package globals with an immutable `SuggestionRegistry` built per app.
- Handler receives `SuggestionRegistry` and `RecordSource`.
- Delete global `PreferredDIDs`; make preferred-source scoring explicit config.
- Remove URL-path fallback maps from `internal/handlers/suggestions.go`; resolve by active app registry only.

The simplification is: no init side effects, no global registry, no cross-app path collision, no handler-side entity knowledge.

---

### 2. `internal/metrics` couples HTTP routing, firehose, PDS, moderation, and Prometheus global registration into one package

**Severity: High**

Evidence:

- `internal/metrics/metrics.go:9`, `:23`, `:41`, `:49`, `:62`, `:75`, `:108`, `:131` define package-level Prometheus collectors.
- `internal/metrics/metrics.go:10`, `:15` use `promauto`, which registers against the global Prometheus registry at init time.
- `internal/middleware/logging.go:10` imports `internal/metrics` directly from request logging middleware.
- `internal/metrics/metrics.go:153-155` has HTTP path normalization inside the metrics package.
- `internal/metrics/metrics.go:157`, `:174`, `:177`, `:192` hard-code route-family knowledge such as `/static/*`, `/brews/:id`, `/api/:entity/:id`.
- `internal/metrics/collector.go:13-20` defines `StatsSource` spanning record counts, user counts, collections, and firehose connection state.

This package is not a seam; it is an ambient sink for every subsystem. It knows too much and is globally active too early.

**Code-judo proposal:**

Invert metrics ownership.

- Middleware should depend on a tiny `RequestObserver` interface, not `internal/metrics`.
- Routing should provide the route pattern label; metrics should not rediscover routes from raw paths.
- Firehose/PDS/moderation should expose local observers or typed events.
- Build Prometheus collectors in `cmd`/server composition against an explicit registry.
- Tests should construct isolated registries instead of sharing package-global collectors.

The simpler architecture is: subsystems emit events; the server wires those events to Prometheus.

---

### 3. Backend asset pipeline has two sources of truth: router config and global template registries

**Severity: High**

Evidence:

- `internal/routing/routing.go:35-36` accepts `CSSBundle` and `JSAssets` in router config.
- `internal/routing/routing.go:266-270` serves those configured asset handlers.
- `internal/web/assets/bundle.go:180-182` defines a package-global CSS registry.
- `internal/web/assets/bundle.go:187` registers bundles globally.
- `internal/web/assets/bundle.go:200` resolves template CSS hrefs globally with `HrefFor`.
- `internal/web/assets/js_assets.go:185-187` defines global JS registration state.
- `internal/web/assets/js_assets.go:193` mutates the JS global through `RegisterJS`.
- `internal/web/assets/js_assets.go:65` builds hrefs from package state.
- `internal/web/components/layout_templ.go:52` calls `assets.HrefFor`.
- `internal/web/components/layout_templ.go:388`, `:401`, `:414`, `:427`, `:440`, `:453`, `:466`, `:479` call global JS href helpers.
- `internal/web/assets/bundle_test.go:88-92` mutates and manually cleans global registry state.
- `internal/web/assets/js_assets_test.go:80-85` saves and restores `jsRegistered`.

The router already has explicit asset dependencies, but templates bypass that composition root and reach into global package state. The tests expose the wrong seam by mutating globals directly.

**Code-judo proposal:**

Make assets a request/layout dependency.

- Put CSS and JS hrefs on `components.LayoutData` or an `AssetManifest`.
- Router serves the configured handlers; handlers pass the matching manifest to templates.
- Delete `Register`, `Registered`, `HrefFor`, `RegisterJS`, and `JSHrefFor` globals.
- Tests assert manifest behavior without global cleanup.

This removes hidden app state and makes Arabica/Oolong asset selection explicit.

---

### 4. `internal/middleware` is a grab-bag of unrelated backend concerns

**Severity: Medium-High**

Evidence:

- `internal/middleware/security.go:33` implements security headers.
- `internal/middleware/security.go:81`, `:97`, `:111` define an in-memory rate limiter and cleanup goroutine.
- `internal/middleware/moderation.go:14`, `:38`, `:61` implements moderation/admin authorization.
- `internal/middleware/logging.go:43` implements request logging.
- `internal/middleware/logging.go:10` directly couples logging middleware to metrics.
- `internal/middleware/request_id.go:22` injects zerolog trace context.
- `internal/middleware/logging.go:18` exports `GetClientIP`, while `security_test.go:305` tests it from the security test file.

This package boundary is "anything wrapping HTTP" rather than a cohesive abstraction. It mixes observability, trust/proxy policy, security headers, rate limiting, and domain authorization.

**Code-judo proposal:**

Split by ownership, not by mechanism.

- `internal/httpobs`: request logging, trace IDs, request metrics observer.
- `internal/httpsecurity`: CSP, nonce, HTMX/body-size/rate-limit policies.
- `internal/moderation/httpauth`: admin/moderator authorization middleware.
- Move client IP extraction behind a `TrustedProxyPolicy`.

The simplification is that middleware composition remains in routing/server setup, but package ownership becomes clear.

---

### 5. `internal/web/bff` is not a BFF; it is a mixed bag of view models, formatters, URL sanitizers, and trace helpers

**Severity: Medium**

Evidence:

- `internal/web/bff/helpers.go:16` defines `UserProfile`.
- `internal/web/bff/helpers.go:24`, `:40`, `:57`, `:66`, `:74`, `:101` define formatting helpers.
- `internal/web/bff/helpers.go:108` defines `SafeAvatarURL`.
- `internal/web/bff/helpers.go:154` defines `SafeWebsiteURL`.
- `internal/web/bff/tracing.go:14` defines `Traceparent`.
- Generated templates consume it directly: `internal/web/components/header_templ.go:21`, `internal/web/components/layout_templ.go:21`, `internal/web/components/layout_templ.go:595`.

The package name implies an application-facing backend facade, but the contents are presentation utilities and security-adjacent URL policy. That muddles ownership and encourages templates to import arbitrary backend helpers.

**Code-judo proposal:**

Rename/split by responsibility.

- `internal/web/viewmodel`: `UserProfile` and layout props.
- `internal/web/format`: temp/time/rating formatting.
- `internal/web/safeurl`: avatar and website URL sanitization.
- `internal/web/tracing`: `Traceparent`.

Or simpler: keep one `internal/web/view` package, but make it explicitly presentation-only and move safe URL policy into a named helper type.

---

### 6. `internal/onboarding` is generic by name but Arabica-specific by type

**Severity: Medium**

Evidence:

- `internal/onboarding/readiness.go:18-21` defines `BrewPrerequisiteStore` using `arabica.Bean`, `arabica.Brewer`, and `arabica.Roaster`.
- `internal/onboarding/readiness.go:30` defines `ReadinessStatus` for brew prerequisites.
- `internal/onboarding/readiness.go:45` checks brew readiness.
- `internal/onboarding/readiness_test.go:14-18` builds the test with `database.MockStore`.

The package name suggests app-neutral onboarding, but it is coffee/brew-specific. The test seam also pulls in the broad database mock instead of a tiny local fake.

**Code-judo proposal:**

Either:

- move this to `internal/arabica/onboarding`, or
- generalize it through entity descriptors/prerequisite declarations.

For the current codebase, the judo move is likely the first option: rename/move to reflect actual ownership and avoid pretending this is shared infrastructure.

---

### 7. Test packages expose wrong seams by requiring same-package access and global mutation

**Severity: Medium**

Evidence:

- `internal/web/assets/bundle_test.go:1` uses `package assets`, then mutates `registry` at `:88-92`.
- `internal/web/assets/js_assets_test.go:1` uses `package assets`, then mutates `jsRegistered` at `:80-85`.
- `internal/middleware/security_test.go:1` uses `package middleware`, then constructs `RateLimiter` internals directly at `:79`, `:92`, `:105`, `:118`.
- `internal/middleware/logging_test.go:1` uses `package middleware`, then directly tests private `responseWriter` internals at `:15`, `:17`, `:22`, `:32`, `:34`, `:40`.
- `internal/middleware/logging_test.go:23`, `:41` use `t.Errorf`, violating the repo’s stated testify/assert convention.
- `tests/integration/harness.go:20-27` imports many internals directly, including entities, atproto, feed, firehose, handlers, and routing.
- `tests/integration/harness.go:145` starts a full-stack harness.
- Integration tests then reach into harness internals heavily, e.g. `tests/integration/firehose_test.go:36`, `tests/integration/handle_resolution_test.go:30`, `tests/integration/cache_test.go:52-56`.

Same-package tests are not automatically bad, but here they are compensating for globals and unowned seams. The integration harness is also acting as a mega-composition root for unrelated subsystems.

**Code-judo proposal:**

- Convert asset and middleware tests to external `_test` packages where possible.
- Test `RateLimiter` through public constructor/config, not private maps.
- Test logging through emitted log/metrics observer behavior, not private wrappers.
- Introduce a narrow `internal/testkit` or per-package harnesses:
  - `httptestkit` for HTTP flows.
  - `firehosetestkit` for index/firehose assertions.
  - `pdstestkit` for PDS-backed store tests.
- Keep full-stack integration tests, but stop using one god harness for every seam.

The goal is not fewer tests; it is tests that pressure the right public boundaries.

---

### 8. `internal/tracing` is a very thin wrapper around `internal/atplatform/tracing`

**Severity: Low-Medium**

Evidence:

- `internal/tracing/tracing.go:20-22` only sets the zerolog OTel logger and delegates `Init(ctx, "arabica")`.
- `internal/tracing/tracing.go:26`, `:31`, `:36` expose span helpers with hard-coded tracer naming.
- `internal/tracing/tracing.go:41` wraps span error recording.

This package may be convenient, but it is close to a forwarding layer. It adds another abstraction line without owning much behavior.

**Code-judo proposal:**

Either collapse this into the server composition code or make it own app-level tracing policy explicitly:

- service name,
- logger bridge,
- sampler/exporter config,
- common span naming conventions.

If it remains a wrapper, it should be boring enough to delete.

---

### 9. `internal/signup` is policy, catalog, and environment mode mixed into one untested package

**Severity: Low-Medium**

Evidence:

- `internal/signup/signup.go:6` defines `Provider`.
- `internal/signup/signup.go:20` defines `Category`.
- `internal/signup/signup.go:31` exposes `Categories(devMode bool)`.
- `internal/signup/signup.go:46` builds all categories.
- `internal/signup/signup.go:151` validates allowed PDS URLs.
- There are no `*_test.go` files under `internal/signup`.

The package owns static provider catalog data and policy validation, with dev-mode branching threaded through call sites.

**Code-judo proposal:**

Split static catalog from policy.

- `Catalog`/`ProviderSet` contains providers.
- `AllowPolicy` validates URLs.
- Dev providers are injected by composition, not toggled by a boolean passed everywhere.
- Add focused tests for URL validation and dev/prod catalog differences.

## Cross-cutting simplification theme

The repeated structural problem is hidden global state plus packages named after mechanisms rather than owners:

- `suggestions` has global entity config.
- `metrics` has global Prometheus collectors.
- `assets` has global template href registries.
- `middleware` aggregates unrelated HTTP wrappers.
- `bff` aggregates unrelated presentation helpers.
- tests compensate with same-package access and registry mutation.

The code-judo move is to push ownership upward into explicit app/server composition and push behavior downward into cohesive packages:

1. Build app-specific registries once.
2. Pass immutable manifests/configs to handlers/templates.
3. Make observers interfaces at subsystem boundaries.
4. Delete global registration APIs.
5. Use tests as pressure: if a test must mutate a package global, the seam is wrong.
