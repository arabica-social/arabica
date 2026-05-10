# Oolong: Tea Lexicons (Design Spec)

**Date:** 2026-05-10
**Status:** Approved (design phase). Implementation plan to follow.
**Scope:** Define AT Protocol lexicons for a tea-tracking sister app to Arabica.

## 1. Goals & non-goals

### Goals

- Define a coherent set of `social.oolong.alpha.*` lexicons that cover the four
  major tea preparation styles (gong fu / flash steep, matcha, long steep, milk
  tea) without forcing every brew into a single rigid shape.
- Capture tea-specific provenance (category, subStyle, processing chain,
  cultivar, farm, harvest year, vendor) at a granularity that is useful for
  enthusiasts but does not block ordinary users from adding a tea quickly.
- Mirror the architectural patterns already established in Arabica
  (per-entity `sourceRef`, AT-URI references, witness cache + session cache,
  per-entity descriptors) so oolong shares infrastructure rather than forking
  it.
- Leave room — via reserved AT-URI ref slots and a deliberately-named record
  set — for later evolution: vendor-published catalogs, farm/cultivar records,
  shared cross-app lexicons, and lens-based projections (panproto) into more
  granular per-tradition schemas.

### Non-goals

- Building any UI, handlers, or store implementations. This spec defines
  lexicons only; implementation is a follow-up plan.
- Defining a final, stable `social.oolong.*` (non-alpha) namespace. The `alpha`
  segment matches Arabica's convention and lets the schema evolve.
- Deciding the migration mechanics for the duplicated `cafe`, `comment`,
  `like`, `drink` records into a future shared namespace. Documented as a
  hook; not solved here.
- Implementing vendor catalogs, cultivar records, farm records, or lens
  transformations. These are documented as future hooks but not part of v1.

## 2. Architecture

### Sister-app model

Oolong is a **sister app** to Arabica, not an extension of it. The Arabica
codebase is already organized to accommodate this: `internal/entities/arabica/`
is one of potentially several entity packages, and the existing `nsid.go`
comment explicitly anticipates a sibling like "oolong, etc." with its own NSID
base.

Oolong lives in the same Go binary and reuses Arabica's infrastructure (PDS
client, witness cache, session cache, firehose pipeline, Templ rendering, Templ
component patterns) but presents itself as a separate vertical with its own
routes, OAuth scope, and UI surface.

### NSID base

```
social.oolong.alpha
```

The reversed-domain convention follows AT Protocol; `oolong.social` becomes
`social.oolong`. The `alpha` segment matches Arabica and signals the schema is
not yet stable.

### Naming choice: `tea`, not `lot`

The bean-analogue record is named `tea`. An earlier draft considered `lot` (a
generic term used in both coffee and tea trade) so that a future shared
lexicon could keep the name across both apps. That hedge was rejected in
favor of clarity for the UI: users adding a tea see "Add a new tea," not
"Add a new lot." If a shared cross-app lexicon emerges later, an NSID rename
is acceptable.

The other entity names — `vendor`, `brewer`, `recipe`, `brew`, `cafe`,
`drink`, `comment`, `like` — are already generic enough to survive a future
consolidation.

## 3. Lexicon inventory (v1)

| NSID                              | Role                                                       |
| --------------------------------- | ---------------------------------------------------------- |
| `social.oolong.alpha.tea`         | A tea (analogue of Arabica's `bean`)                       |
| `social.oolong.alpha.brew`        | A brewing session, polymorphic via `methodParams` union    |
| `social.oolong.alpha.brewer`      | A brewing vessel (gaiwan, kyusu, gongfu pot, matcha bowl)  |
| `social.oolong.alpha.recipe`      | A reusable brewing template                                |
| `social.oolong.alpha.vendor`      | A seller / blender / importer (analogue of `roaster`)      |
| `social.oolong.alpha.cafe`        | A cafe or teahouse                                         |
| `social.oolong.alpha.drink`       | A tea ordered at a cafe                                    |
| `social.oolong.alpha.comment`     | A comment on any record (mirrors arabica's `comment`)      |
| `social.oolong.alpha.like`        | A like on any record (mirrors arabica's `like`)            |

> **Note on `cafe` and `drink`:** Arabica's CLAUDE.md lists `cafe` and `drink`
> as part of its lexicon set, but no JSON exists for them in `lexicons/` and no
> NSID constants exist in `internal/entities/arabica/nsid.go`. They are
> planned-but-unimplemented in arabica. This spec defines concrete shapes for
> oolong's `cafe` and `drink` from scratch, which the arabica side can later
> mirror or fork as needed.

### Notably absent

- **No grinder.** Tea is not ground at point of brewing. Matcha arrives
  pre-powdered.
- **No farm or cultivar records.** Both are fields on `tea` for v1, with
  `farmRef` and `cultivarRef` AT-URI slots reserved for a future where third
  parties publish their own farm/cultivar records.

## 4. Record specifications

### 4.1 `social.oolong.alpha.tea`

The tea record. Captures provenance, classification, and processing.

```
required: name, createdAt
properties:
  name              string (≤200)
  category          string knownValues:
                      green | yellow | white | oolong | red | dark
                      | flavored | blend | other
  subStyle          string (≤200)
                      # examples: "Yue Guang Bai", "Baozhong", "Wuyi",
                      # "Raw Puerh", "Ripe Puerh", "Hei Cha"
  origin            string (≤200)        # region or country
  cultivar          string (≤200)        # plant variety
  cultivarRef       at-uri               # reserved for future cultivar records
  farm              string (≤200)
  farmRef           at-uri               # reserved for future farm records
  harvestYear       integer
  processing        array of #processingStep
  description       string (≤1000)
  vendorRef         at-uri               # → social.oolong.alpha.vendor
  rating            integer (1-10)
  closed            boolean              # bag/pouch finished
  createdAt         datetime
  sourceRef         at-uri               # forked/copied from
```

#### `#processingStep`

A single step in the tea's processing chain. Steps are ordered (array order is
significant). The structured array is preferred over a single `process` string
because tea processing is genuinely multi-step (e.g., shaded → steamed →
rolled, or oxidized → roasted → aged), and the array supports arbitrary
combinations without the lexicon needing to enumerate every cross product.

```
properties:
  step              string knownValues:
                      steamed | pan-fired | rolled | oxidized | shaded
                      | roasted | aged | fermented | compressed
                      | scented | smoked | blended | flavored | other
  detail            string (≤300)
                      # examples: "20 days under kanreisha", "stone-pressed cake",
                      # "1995 vintage", "jasmine, 5 scenting cycles"
```

### 4.2 `social.oolong.alpha.brew`

The brewing session record. Top-level fields are common across all four styles;
style-specific parameters live inside the `methodParams` union.

```
required: teaRef, style, createdAt
properties:
  teaRef            at-uri (required)    # → social.oolong.alpha.tea
  style             string knownValues:
                      gongfu | matcha | longSteep | milkTea
  brewerRef         at-uri               # → social.oolong.alpha.brewer
  recipeRef         at-uri               # → social.oolong.alpha.recipe
  temperature       integer
                      # tenths of °C, e.g. 850 = 85.0°C
  leafGrams         integer
                      # tenths of g, e.g. 50 = 5.0g
  vesselMl          integer              # vessel volume / total water
  timeSeconds       integer
                      # primarily for longSteep and milkTea; ignored by matcha;
                      # gongfu records per-steep times inside #gongfuParams
  tastingNotes      string (≤2000)       # overall impression
  rating            integer (1-10)       # overall rating
  methodParams      union {
                      #gongfuParams
                      | #matchaParams
                      | #milkTeaParams
                    }                    # optional; absent for longSteep
  createdAt         datetime
```

#### Why `methodParams` is a union

Earlier drafts modeled style-specific parameters as separate optional ref
fields (e.g., `gongfu`, `matcha`, `milkTea` siblings on the record). The
union shape is preferred because:

1. Exactly one method can apply to a given brew; the union enforces that
   semantically.
2. AT Protocol's `union` type is the idiomatic discriminator for polymorphic
   payloads, and the runtime `$type` field tells consumers exactly which shape
   to parse.
3. Adding new styles later (e.g., `#westernParams` if longSteep ever earns
   structure) is a backward-compatible union expansion rather than a
   record-shape change.

#### Why `style` remains a top-level enum despite `methodParams`

The union's `$type` discriminates the params block, but `style` is the
canonical, queryable axis for filtering ("show me my gongfu sessions") and is
required even when `methodParams` is absent (the `longSteep` case). Keeping
both is intentional: `style` is the user's intent label; `methodParams.$type`
is the structural signal. They will agree in practice.

#### `#gongfuParams`

```
properties:
  rinse             boolean              # was the leaf rinsed first
  rinseSeconds      integer
  steeps            array of #steep
  totalSteeps       integer
                      # absolute count, in case not every steep was logged
```

#### `#steep`

```
required: number, timeSeconds
properties:
  number            integer (≥ 1)        # 1-indexed steep number
  timeSeconds       integer
  temperature       integer              # tenths of °C, override of brew.temperature
  tastingNotes      string (≤500)        # per-steep flavor notes
  rating            integer (1-10)       # per-steep rating
```

Per-steep tasting notes coexist with the top-level `brew.tastingNotes`. The
top-level field is "overall impression"; the per-steep field is the detail.
UIs may render either, both, or neither depending on context.

#### `#matchaParams`

```
properties:
  preparation       string knownValues: usucha | koicha | iced | other
  sieved            boolean
  whiskType         string (≤200)
                      # examples: "chasen 80-prong", "chasen 120-prong",
                      # "milk frother", "electric whisk"
  waterMl           integer              # finer than top-level vesselMl
```

#### `#milkTeaParams`

```
properties:
  preparation       string (≤300)
                      # examples: "stovetop simmer", "shaken iced",
                      # "tea bag steep then mix"
  ingredients       array of #ingredient
  iced              boolean
```

#### `#ingredient`

```
required: name
properties:
  name              string (≤200)
  amount            integer              # tenths of unit
  unit              string knownValues:
                      g | ml | tsp | tbsp | cup | pcs | other
  notes             string (≤200)        # examples: "Madhava brand", "Demerara"
```

Structured ingredients (rather than a single string) enable later interop —
shareable masala chai recipes, blend ratios, etc. — and underpin a richer
ingredients UI.

#### Why `longSteep` has no params block

Western brewing, grandpa style, cold brew, and Japanese medium steep all
collapse into the base `brew` record with `style: "longSteep"` and
`methodParams` absent. Everything those styles need is captured by the base
record's top-level fields: `temperature`, `timeSeconds`, `leafGrams`,
`vesselMl`, `tastingNotes`, `rating`.

The base `brew` shape carries `timeSeconds` at the top level
(see field list above) specifically so longSteep does not need its own
params block. The same field is meaningful for milkTea, ignored by matcha,
and unused by gongfu (which records per-steep times inside
`#gongfuParams.steeps[].timeSeconds`). Carrying one optional integer at the
top level is preferred over creating `#longSteepParams` solely to hold it.

If a future style genuinely needs more structure than this single field
provides, `#longSteepParams` (or `#westernParams`, `#coldBrewParams`, etc.)
can be added to the union without changing the existing record shape.

### 4.3 `social.oolong.alpha.brewer`

A physical brewing vessel.

```
required: name, createdAt
properties:
  name              string (≤200)
                      # e.g. "Yixing zisha", "120ml gaiwan", "Kyusu — Tokoname"
  style             string knownValues:
                      gaiwan | kyusu | teapot | matcha-bowl
                      | infuser | thermos | other
  capacityMl        integer
  material          string (≤200)
                      # examples: "porcelain", "yixing clay", "glass",
                      # "cast iron", "stoneware"
  description       string (≤1000)
  createdAt         datetime
  sourceRef         at-uri
```

The user explicitly preferred including a brewer record (over deriving brew
style from a freeform brewing-method string) because vessel size, shape, and
material genuinely affect the brew.

### 4.4 `social.oolong.alpha.recipe`

Reusable brewing template. Mirrors Arabica's `recipe` shape, with the
`methodParams` union pattern matching `brew`.

```
required: name, createdAt
properties:
  name              string (≤200)
  brewerRef         at-uri               # specific brewer
  style             string knownValues:
                      gongfu | matcha | longSteep | milkTea
  teaRef            at-uri
                      # optional — recipes may be tea-specific or generic
  temperature       integer              # tenths of °C
  timeSeconds       integer              # primarily for longSteep
  leafGrams         integer              # tenths of g
  vesselMl          integer
  methodParams      union {
                      #gongfuParams
                      | #matchaParams
                      | #milkTeaParams
                    }                    # optional
  notes             string (≤2000)
  createdAt         datetime
  sourceRef         at-uri
```

The `#gongfuParams`, `#matchaParams`, `#milkTeaParams`, `#steep`, and
`#ingredient` definitions are shared with `brew` — recipe templates carry the
same structures because a recipe is essentially a "brew without a
specific session timestamp."

> **Implementation note:** in the lexicon JSON, these definitions will be
> duplicated under both `brew` and `recipe` (lexicon refs are scoped to a
> single document). Tools that consume both will need to recognize the
> equivalence; the entity package can share the Go types.

### 4.5 `social.oolong.alpha.vendor`

A seller, blender, or importer of tea. Mirrors `roaster`.

```
required: name, createdAt
properties:
  name              string (≤200)
                      # examples: "Spirit Tea", "Tezumi", "Yunnan Sourcing"
  location          string (≤200)
  website           uri (≤500)
  description       string (≤1000)
  createdAt         datetime
  sourceRef         at-uri
```

Vendor is intentionally broader than `roaster`: it spans single-origin
sellers, blenders, and importers without requiring any of them to
self-classify.

### 4.6 `social.oolong.alpha.cafe`

A cafe or teahouse. Defined fresh in this spec because Arabica's `cafe` is
documented but unimplemented; this shape is a candidate for arabica to mirror
later.

```
required: name, createdAt
properties:
  name              string (≤200)
  location          string (≤200)
                      # human-readable, e.g. "Brooklyn, NY", "Tokyo"
  address           string (≤500)        # street address (optional)
  website           uri (≤500)
  description       string (≤1000)
  vendorRef         at-uri               # → vendor, if the cafe is also a vendor
  createdAt         datetime
  sourceRef         at-uri
```

`vendorRef` (rather than coffee's `roasterRef`) lets a cafe be linked to its
parent vendor when applicable (e.g., a vendor-operated tasting room).

### 4.7 `social.oolong.alpha.drink`

A tea ordered at a cafe.

```
required: cafeRef, createdAt
properties:
  cafeRef           at-uri (required)    # → social.oolong.alpha.cafe
  teaRef            at-uri               # → social.oolong.alpha.tea, optional
  name              string (≤200)
                      # menu item, e.g. "Iced hojicha latte", "Jasmine pearls"
  style             string knownValues:
                      gongfu | matcha | longSteep | milkTea | other
  description       string (≤500)
  rating            integer (1-10)
  tastingNotes      string (≤2000)
  priceUsdCents     integer
                      # optional price in US cents; future revision may
                      # generalize to currency-aware
  createdAt         datetime
  sourceRef         at-uri
```

`teaRef` is optional because most cafe drinks are not labeled with a specific
single-origin tea the user has tracked; the menu-item name in `name` is the
primary identifier.

### 4.8 `social.oolong.alpha.comment`

Mirrors `social.arabica.alpha.comment`. A comment on any record. Uses a
`strongRef` (which is NSID-agnostic) so a tea-app comment can target an
arabica record by AT-URI, and vice versa.

```
required: subject, text, createdAt
properties:
  subject           ref → com.atproto.repo.strongRef
  parent            ref → com.atproto.repo.strongRef
                      # optional, for threading; points to the parent comment
  text              string (maxLength 1000, maxGraphemes 300)
                      # matches arabica's comment limits
  createdAt         datetime
```

### 4.9 `social.oolong.alpha.like`

Mirrors `social.arabica.alpha.like`. A like on any record, NSID-agnostic via
`strongRef`.

```
required: subject, createdAt
properties:
  subject           ref → com.atproto.repo.strongRef
  createdAt         datetime
```

## 5. Cross-cutting patterns

### `sourceRef`

Every record carries an optional `sourceRef` (AT-URI). When a user forks
another user's tea, brewer, recipe, etc., this points back to the original.
Same convention as Arabica.

### AT-URI references only

No embedded entities. Every cross-record link is an AT-URI reference;
resolution flows through the witness cache → session cache → PDS fallback
machinery already built for Arabica. Adding a new entity package
(`internal/entities/oolong/`) is the implementation pattern; the cache
layers do not need oolong-specific logic beyond knowing which collections
exist.

### OAuth scope

Oolong gets its own OAuth scope listing all 9 collections. Authorization is
per-app; a user who authorizes Arabica does not implicitly authorize Oolong.
This matches the AT Protocol convention of per-app scopes.

### Per-app entity package

Oolong's Go entity package lives at `internal/entities/oolong/` with the same
structure as `internal/entities/arabica/`:

- `nsid.go` — NSID constants
- `models.go` — typed Go models for each record
- `records.go` — `XToRecord` / `RecordToX` conversions
- `register.go` — descriptor registration
- `fields.go`, `options.go`, `recipe_filter.go`, `resolve.go` — as needed

The shared `internal/entities/entities.go` registry already supports multiple
entity packages.

## 6. Future hooks

### Vendor catalogs

Vendors are encouraged to publish their own `social.oolong.alpha.vendor` and
`social.oolong.alpha.tea` records to their DIDs. End-users reference those
via `sourceRef` (and copy the data into their own PDS for offline
availability). The `name` + AT-URI dedup story is identical to Arabica's
roaster suggestions.

### Cultivar / farm records

`tea.cultivarRef` and `tea.farmRef` slots exist now but are unused. When the
ecosystem develops to where dedicated cultivar and farm records make sense,
those become full record types — likely under a shared namespace (e.g.,
`agriculture.atmosphere.alpha.cultivar`) rather than oolong's own.

### Lenses / panproto

Out of scope for v1. The `category` enum + freeform `subStyle` + structured
`processing` array is intentionally wide enough to be a source for narrower
per-tradition lexicons later. A panproto lens could project `tea` records
into, say, a Japanese-tea-specific lexicon by reading `processing` for
`shaded` steps, `cultivar` for known Japanese cultivars, etc.

### Shared cross-app lexicons

`cafe`, `drink`, `comment`, `like` are explicit candidates for migration to a
shared namespace once a migration story exists. Until then they live as
duplicates under both `social.arabica.alpha.*` and `social.oolong.alpha.*`.

### Atmosphere interop

The user's longer-term vision includes interop with `Beacon Bits` (location
beacons), `Bluesky` (general social), and community-event lexicons. None of
those are in v1 scope, but the per-record `sourceRef` convention and the
NSID-agnostic strongRef in `like` and `comment` already accommodate it: a
user can like or comment on a Beacon Bits location, a Bluesky post, or a
community event from within oolong's UI without lexicon changes.

## 7. Out of scope (explicitly deferred)

- Implementation of any handler, store, witness cache, OAuth scope, route, or
  Templ component. These are the subject of a follow-up implementation plan.
- Migration of duplicated lexicons (`cafe`, `comment`, `like`, `drink`) into
  a shared namespace.
- Promotion of `cultivar` and `farm` from string fields to record types.
- Vendor-published catalog ergonomics (suggestions UI, dedup heuristics).
- Lens transformations / panproto integration.
- Any changes to existing `social.arabica.alpha.*` lexicons.

## 8. Acceptance criteria

This spec is acceptable when:

1. The 9 lexicon JSONs can be authored from this document without further
   ambiguity.
2. Each record's required fields, optional fields, and reference targets are
   unambiguous.
3. The `methodParams` union members are fully specified, including the
   nested `#steep` and `#ingredient` shapes.
4. The intent of the borderline modeling decisions (per-steep notes,
   `style` + `methodParams` redundancy, the `tea` record name, no grinder,
   no farm/cultivar records, top-level `timeSeconds` rather than a
   `#longSteepParams` block) is documented with rationale a future
   maintainer can read.

The follow-up implementation plan will translate this spec into code.
