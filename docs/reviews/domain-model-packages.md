# Domain/model package review

## Scope

Reviewed current backend code in:

- `internal/arabica/entities`
- `internal/oolong/entities`
- `internal/entities`
- `internal/lexicons`
- `internal/suggestions`
- `internal/matching`
- `internal/moderation`
- `internal/ogcard`

## Strongest structural findings

### 1. Entity metadata is split across too many registries

Evidence:

- `internal/entities/entities.go:18` defines `Descriptor`.
- `internal/entities/entities.go:95` registers descriptors globally.
- `internal/lexicons/record_type.go:37` has separate record-type parsing logic.
- `internal/arabica/entities/register.go:27` and
  `internal/oolong/entities/register.go:18` register app descriptors.
- `internal/suggestions/suggestions.go:59` has a separate suggestion registry.
- `internal/arabica/entities/suggestions.go:15` and
  `internal/oolong/entities/suggestions.go:13` register suggestion configs
  separately.

Impact:

- Adding a new entity requires touching multiple parallel sources of truth.
- Display names, NSIDs, URL paths, suggestion behavior, and record-type parsing
  can drift.

Code-judo move:

- Make the entity descriptor the canonical metadata source for display name,
  NSID, route path, suggestion/search fields, and app ownership.
- Keep `lexicons.RecordType` as the enum layer, but avoid duplicating human
  display metadata there.
- Attach suggestion config to entity descriptors, or derive it from descriptor
  fields when possible.

Risk: medium-high maintainability risk. Current behavior can work, but entity
expansion remains fragile.

Approval bar: acceptable for small incremental fixes, but this should be
simplified before another broad entity expansion.

---

### 2. The shared entity registry weakens type boundaries with `any` callbacks

Evidence:

- `internal/entities/entities.go:33` uses
  `GetField func(entity any, field string)`.
- `internal/entities/entities.go:39` uses
  `RecordToModel func(record map[string]any, uri string) (any, error)`.
- `internal/arabica/entities/fields.go:7` starts runtime type assertions for
  field access.
- `internal/oolong/entities/fields.go:5` repeats the same pattern.

Impact:

- Type mistakes become runtime failures or silent false returns.
- String field names are duplicated and unchecked.
- The registry is useful, but it currently acts like a loosely typed service
  locator.

Code-judo move:

- Keep the untyped boundary only at the cross-app registry edge.
- Use typed per-entity descriptor builders internally, then erase to the shared
  descriptor once.
- Replace ad-hoc string field switches with declarative typed field descriptors
  where possible.

Risk: medium. Not immediately broken, but it makes future refactors and entity
additions more error-prone.

Approval bar: functional, but below the maintainability bar for a growing domain
model.

---

### 3. Arabica record conversion is too monolithic and repetitive

Evidence:

- `internal/arabica/entities/records.go:137` defines `BrewToRecord`.
- `internal/arabica/entities/records.go:241` defines `RecordToBrew`.
- `internal/arabica/entities/records.go:361` defines `BeanToRecord`.
- `internal/arabica/entities/records.go:403` defines `RecordToBean`.
- Repeated `time.Parse(time.RFC3339, ...)` appears throughout the same file,
  including around `records.go:89`, `:266`, `:427`, `:515`, `:628`, `:713`,
  `:795`, and `:882`.

Impact:

- One file is carrying too much domain conversion logic.
- Adding fields encourages copy/paste changes.
- Common parsing/defaulting rules are repeated instead of centralized.

Code-judo move:

- Split Arabica converters by entity, matching the Oolong pattern such as
  `records_tea.go` and `records_vendor.go`.
- Add small shared helpers for required strings, optional strings, RFC3339
  timestamps, AT-URI refs, and scaled numeric values.
- Keep entity-specific mapping explicit, but delete repetitive parse scaffolding.

Risk: medium-high. This is exactly the kind of file that becomes dangerous once
more entities or fields land.

Approval bar: needs refactor soon; this is structural, not cosmetic.

---

### 4. Suggestions search loads and ranks whole collections in memory

Evidence:

- `internal/suggestions/suggestions.go:153` defines `Search`.
- `internal/suggestions/suggestions.go:173` loads records via
  `ListRecordsByCollectionOldest`.
- `internal/suggestions/suggestions.go:230` performs substring matching in
  process.
- `internal/suggestions/suggestions.go:239` deduplicates after loading.
- `internal/suggestions/suggestions.go:273` sorts results after building them.

Impact:

- Suggestion latency and memory cost scale with full collection size, not the
  requested limit.
- Dedup/ranking logic is locked to in-memory scans even though the firehose
  index is the natural place for bounded candidate search.

Code-judo move:

- Introduce a source query method that accepts collection, normalized query,
  searchable fields, and limit.
- Let the SQLite/index layer return bounded candidates.
- Keep final app-specific enrichment/dedup in Go only where it actually buys
  clarity.

Risk: high scalability risk as indexed community data grows.

Approval bar: acceptable for small data; not acceptable as production-scalable
discovery infrastructure.

---

### 5. Arabica/Oolong validation and common social shapes are duplicated

Evidence:

- `internal/arabica/entities/models.go:11` defines common length limits.
- `internal/oolong/entities/models.go:7` defines similar common length limits.
- `internal/arabica/entities/models.go:465` and
  `internal/oolong/entities/models_social.go:9` define social records
  separately.
- `internal/oolong/entities/models_cafe.go:5` and
  `internal/oolong/entities/models_drink.go:5` mirror categories that also
  exist in Arabica models.

Impact:

- Validation behavior can diverge silently between apps.
- Error constants and length limits are policy-level rules but live in separate
  app packages.

Code-judo move:

- Do not necessarily share full structs where domain language differs.
- Extract common validation helpers/errors/length policy into a small internal
  package.
- Keep app-specific record structs explicit.

Risk: medium. Current duplication is manageable, but the two apps are already
drifting toward parallel implementations.

Approval bar: fine for the initial split; refactor common validation before
adding more app domains.

---

### 6. `ogcard` is Arabica-entity coupled despite partial multi-app support

Evidence:

- `internal/ogcard/card.go:30` embeds an Oolong logo.
- `internal/ogcard/card.go:372` branches on app name for Oolong logo loading.
- `internal/ogcard/entities.go:22`, `:107`, `:150`, `:201`, and `:248` define
  Arabica entity cards only.
- `internal/ogcard/brew.go:166` draws only `*arabica.Brew`.
- No tests exist under `internal/ogcard`.

Impact:

- Site cards are multi-app, but entity cards are not structurally ready for
  Oolong.
- Adding Oolong cards will likely duplicate hand-positioned drawing functions.
- Lack of tests makes image layout regressions easy.

Code-judo move:

- Extract a small card composition layer: title, subtitle/details, metric rows,
  description block, accent/type label.
- Keep per-entity functions as thin adapters that populate this structure.
- Add golden-light tests around text truncation/wrapping and successful PNG
  generation.

Risk: medium. Not a runtime correctness blocker, but this will become expensive
when Oolong entity OG cards are required.

Approval bar: acceptable for Arabica-only entity cards; not ready as a shared
multi-app OG-card package.

## Overall approval bar

These packages are generally coherent and many are test-backed, but the domain
layer is accumulating parallel registries, runtime-typed dispatch, and repeated
conversion/validation logic. Small fixes are fine. A broad entity expansion
should first simplify the descriptor/metadata boundary and record conversion
structure.

