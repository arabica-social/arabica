# Glossary

The canonical vocabulary for Arabica's product, AT Protocol integration,
runtime architecture, and frontend transition. Entries are intentionally short:
enough to identify the concept and link to the document or source that owns the
details.

This is also a naming surface. Introduce a human-facing or conceptual name here
before allowing several competing names to spread through code, UI, and plans.
The glossary governs those names; existing NSIDs, persisted fields, API paths,
and other compatibility identifiers change only through an explicit decision.

This is not a tutorial, API reference, file inventory, or changelog. Terms are
ordered by conceptual dependency rather than alphabetically.

## Product

Top-level lexicon record types are included here because they are part of the
project's domain language. Nested schema helpers such as `#pour` or
`#espressoParams` stay in the lexicon unless their names become concepts we
regularly use outside schema implementation.

**Arabica** — The coffee-tracking application in this repository. It shares AT
Protocol and runtime infrastructure with Oolong while owning coffee-specific
records, routes, presentation, and product language.

**Oolong** — The tea-tracking sister application. Oolong shares platform code
with Arabica but has its own app configuration, NSID base, entity set, database,
and route ownership.

**Bean** — An Arabica coffee product or lot (`social.arabica.alpha.bean`) that
can be referenced by brews and may reference a roaster.

**Roaster** — An Arabica coffee-roasting business or producer
(`social.arabica.alpha.roaster`) referenced by beans.

**Grinder** — An Arabica coffee grinder or grinding setup
(`social.arabica.alpha.grinder`) that can be referenced by a brew.

**Brewer** — An Arabica brewing device or method-specific piece of equipment
(`social.arabica.alpha.brewer`) that can be referenced by brews and recipes.

**Recipe** — An Arabica reusable preparation specification
(`social.arabica.alpha.recipe`). A recipe describes intended preparation; a
Brew records an actual preparation.

**Brew** — An observed Arabica coffee-preparation session
(`social.arabica.alpha.brew`), potentially referencing a bean, grinder, brewer,
and recipe.

**Tea** — An Oolong tea product or lot (`social.oolong.alpha.tea`) that may
reference a Tea Vendor and is referenced by Tea Brews.

**Tea Vendor** — An Oolong seller or producer of tea
(`social.oolong.alpha.vendor`) referenced by Tea records.

**Vessel** — An Oolong steeping or serving vessel
(`social.oolong.alpha.vessel`) that can be referenced by a Tea Brew.

**Infuser** — An Oolong infusing device (`social.oolong.alpha.infuser`) that can
be referenced by a Tea Brew.

**Tea Brew** — An observed Oolong tea-preparation session
(`social.oolong.alpha.brew`), commonly called a **Steep** in Oolong's user
interface. It references a Tea and may reference a Vessel or Infuser.

**Tea Cafe** — An Oolong cafe record (`social.oolong.alpha.cafe`) that may
reference a Tea Vendor. The lexicon exists, but the entity is currently deferred
and not enabled in the Oolong app.

**Tea Drink** — An Oolong prepared-drink record (`social.oolong.alpha.drink`)
that must reference a Tea Cafe and may reference a Tea. The lexicon exists, but
the entity is currently deferred and not enabled in the Oolong app.

**Community Feed** — The activity-oriented surface showing what people are
brewing or recording. It is distinct from Explore's reusable-record discovery.

**Explore** — Arabica's authenticated discovery surface for reusable community
records such as beans, roasters, grinders, brewers, and recipes. See
[FDR-001](fdr/FDR-001-explore.md).

**Backlink** — A reverse relationship showing records that refer to the current
record, such as brews that use a bean or recipe.

**Like** — An app-scoped social record (`social.arabica.alpha.like` or
`social.oolong.alpha.like`) expressing approval of another record through a
strong reference.

**Comment** — An app-scoped social record (`social.arabica.alpha.comment` or
`social.oolong.alpha.comment`) attached to another record through a strong
reference, optionally linked to a parent comment for a thread.

## Pages and Surfaces

These entries name the page-level surfaces we regularly discuss. They define
what a page name means; they do not replace the executable route inventories in
the app handlers or the ownership rules in
[Interfaces and route ownership](architecture/interfaces-and-route-ownership.md).
Self-explanatory utility and information pages such as About, Terms, Settings,
and Notifications are intentionally omitted unless their names become
ambiguous in project conversation.

**Home Page** — The root page at `/`. It combines the app's introductory or
onboarding context, personal shortcuts, and the Home Feed. “Home Page” refers to
the whole page, not only its feed section.

**Home Feed** — The Community Feed as presented in the Home Page's “Community
Activity” section. Authenticated viewers can filter and sort it; the public feed
is an unfiltered view. Use “Community Feed” for the product/data concept and
“Home Feed” when referring specifically to its placement or behavior on `/`.

**Profile Page** — The actor-scoped page at `/profile/{actor}` showing a
person's profile identity and their visible records. A user's own Profile Page
is still the public/profile-oriented surface; collection management belongs on
My Coffee or My Tea.

**My Coffee Page** — Arabica's authenticated personal collection page at
`/my-coffee`, with tabs for the viewer's brews, beans, roasters, grinders,
brewers, and recipes. This is the primary management view for coffee records.

**My Tea Page** — Oolong's authenticated personal collection page at `/my-tea`,
covering the viewer's steeps, teas, vendors, vessels, and infusers.

**Your Brews Page (Brew List Page)** — Arabica's focused brew-journal page at
`/brews`, listing the authenticated viewer's own brews. It is narrower than My
Coffee and distinct from brews shown on a Profile Page or in the Home Feed.

**Explore Page** — The page at `/explore` that presents the Explore feature's
search, facets, sorting, and reusable community records. It is distinct from the
Explore Recipes Page. See [FDR-001](fdr/FDR-001-explore.md).

**Explore Recipes Page (Recipe Explore Page)** — The recipe-specific discovery
page at `/recipes`, focused on searching, inspecting, using, and forking
recipes. It predates and remains separate from the general Explore Page.

**Record View Page** — A shareable actor-scoped detail page for one record,
generally using `/{entity-path}/{actor}/{id}`, such as
`/beans/{actor}/{id}`. “Bean View Page,” “Roaster View Page,” “Grinder View
Page,” “Brewer View Page,” “Recipe View Page,” “Tea View Page,” and equivalent
names refer to the corresponding Record View Page.

**Brew View Page** — The specialized Record View Page for one coffee or tea
brew at `/brews/{actor}/{id}`. It presents the recorded preparation, resolved
references, social context, and owner actions where applicable.

**Backlinks Page** — The page at a supported record's `/backlinks` route showing
other records that refer to that record. It is a relationship view, not the
record's main detail page; not every record type necessarily exposes one.

**Record Form Page** — A full-page create or edit surface for one record type,
such as the Brew Form Page, Bean Form Page, or Recipe Form Page. “New” and
“Edit” variants refer to the same form family with different initial state and
mutation behavior.

**Onboarding Page** — The guided setup page at `/onboarding` that helps a new
user establish the minimum records needed to begin logging brews or steeps.

**Add Records Page** — Arabica's authenticated page at `/add` for adding more
records after or outside the initial onboarding flow. It is not the same as the
guided Onboarding Page; Oolong currently uses its own onboarding and My Tea
flows instead.

## AT Protocol

**PDS (Personal Data Server)** — The authoritative host for a user's repository
and Arabica/Oolong records. Local Arabica databases do not replace PDS
authority. See [ADR-001](adr/ADR-001-pds-records-are-authoritative.md).

**DID (Decentralized Identifier)** — The stable identifier used for an AT
Protocol account and repository. A handle can change while the DID remains the
record owner identity.

**Handle** — A human-readable account name that resolves to a DID. Handles are
not stable record ownership keys.

**Repository** — The signed collection of records hosted for one DID on a PDS.

**Collection** — The records in a repository sharing one NSID, such as
`social.arabica.alpha.bean`.

**NSID (Namespaced Identifier)** — The stable identifier for an AT Protocol
record collection or XRPC method. Arabica and Oolong use separate NSID bases.

**AT-URI** — A record identifier containing its repository DID, collection
NSID, and record key: `at://did/collection/rkey`.

**RKey (Record Key)** — The key identifying one record within a repository
collection.

**TID (Timestamp Identifier)** — The timestamp-derived RKey format used when
Arabica creates records.

**Lexicon** — An AT Protocol schema defining a record or XRPC contract. Arabica
lexicons are compatibility contracts for records that may already exist on
users' PDSs.

**Strong reference** — The `com.atproto.repo.strongRef` lexicon type containing
both an AT-URI and CID, used when the referenced version matters, including
likes and comments.

## Architecture

**App** — Runtime configuration for Arabica or Oolong, including its name, NSID
base, enabled entity descriptors, routes, branding, and record-store behavior.
The current source is `internal/atplatform/domain/app.go`.

**Record type** — The application-level category used to dispatch record
identity and behavior across shared infrastructure.

**Entity descriptor** — Lightweight identity metadata for a record type: its
record type, NSID, and display name. See
[ADR-005](adr/ADR-005-separate-entity-descriptors-from-record-behavior.md).

**Record behavior** — Record-specific decoding, field access, display-title
extraction, and reference hydration registered separately from an entity
descriptor.

**Session cache** — A short-lived, per-session in-memory cache for typed
collection reads.

**Witness cache** — The `AtprotoStore` view of locally indexed records used to
serve reads without contacting a PDS. It is backed by the firehose index and is
not authoritative.

**Witness index** — The local SQLite record index populated by firehose events,
PDS backfill, and successful write-through updates. The implementation type is
`firehose.FeedIndex`; “witness” names its role as evidence of remote records.

**Dirty collection** — A collection recently written locally whose witness
results may lag the PDS. Dirty collections bypass witness reads until an
authoritative collection read refreshes session state.

**PDS fallback** — An authoritative read from the user's PDS after an eligible
local cache or witness lookup misses or is bypassed.

**Firehose** — The stream of AT Protocol repository events consumed to keep
local indexes up to date.

**Feed index** — Feed and social query capabilities built over the local
firehose-backed SQLite index. It shares storage with the witness index but names
the feed-oriented use of that data.

**Explore index** — A searchable, faceted projection derived from records in
the firehose index. It can be cleared and rebuilt. See
[ADR-003](adr/ADR-003-local-indexes-are-rebuildable-projections.md).

**Rebuildable projection** — Derived local state that can be discarded and
reconstructed from authoritative or recoverable inputs. It must not become a
write target or source of record truth.

**Source reference** — Explicit provenance from a copied or forked record to
its source. Explore uses it for duplicate clustering and presents product copy
such as “Used by N” rather than exposing the field name.

## Frontend Transition

**SPA shell** — The embedded SvelteKit `index.html` served by Go with
server-controlled head, session, security, and Open Graph data injected before
the client application starts.

**SPA-owned route** — A page route explicitly assigned to SvelteKit after its
direct-load implementation and JSON dependencies exist. Arabica's source of
truth is `Routes.SPAOwnedRoutes()`.

**Legacy route** — A page route still rendered by the Templ/HTMX stack during
the incremental SPA migration.

**JSON contract** — A documented response shape consumed by the SPA and tested
at the HTTP boundary. Current contracts live under [`docs/api/`](api/README.md).

**Svelte island** — A Svelte component mounted inside a legacy server-rendered
page. Islands remain transitional while full page routes move to SvelteKit.
