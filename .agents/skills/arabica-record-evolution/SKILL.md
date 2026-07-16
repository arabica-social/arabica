---
name: arabica-record-evolution
description: Review or implement Arabica/Oolong lexicon and persisted-record changes safely. Use for NSID, record field, type, unit, validation, reference, codec, or existing-PDS compatibility changes.
---

# Arabica Record Evolution

Evolve AT Protocol record contracts without assuming that Arabica controls or
can migrate every user's PDS. Existing records may outlive the deployment that
created them and can be observed through PDS reads, firehose events, witness
indexes, APIs, and older clients.

## When to Use

Use this skill when a task changes or reviews any of the following:

- a lexicon, NSID, record field, validation rule, union, or reference shape;
- the meaning, precision, scale, or unit of a persisted value;
- record decoding, encoding, or typed model representation;
- how old and new record variants appear in feed, Explore, backlinks, APIs, or
  frontend forms/views;
- OAuth collection scopes caused by adding or removing an enabled record type.

Adding a new entity should also use `arabica-entity-change`; use this skill for
the compatibility portion of that work.

## Read First

1. `docs/adr/ADR-001-pds-records-are-authoritative.md`
2. `docs/architecture/entities-and-record-contracts.md`
3. `docs/road-to-v1.md` when the change is part of pre-v1 schema work
4. The affected lexicon under `lexicons/`
5. The owning app's entity registration, model, validation, and record behavior
6. Focused API docs, FDRs, and plans that describe the affected behavior

Do not treat a dated plan as proof that a proposed schema is current.

## Action Mode

- For an implementation request, make the compatible code, tests, generated
  output, and directly affected documentation changes.
- For a review, audit, or design request, report findings and alternatives
  without editing unless the user asks for changes.
- A breaking record change requires explicit user agreement before
  implementation. Do not infer approval merely because the project is pre-v1.

## Compatibility Classification

Classify every persisted or client-visible change:

- **Additive** — old records remain valid; new data is optional or safely
  ignored by older code.
- **Behavioral** — the wire shape remains readable but interpretation,
  validation, defaults, visibility, ordering, or UI behavior changes.
- **Deprecated** — the old representation remains readable while new writes
  move to a replacement.
- **Breaking** — an existing record can no longer decode or retain its meaning,
  or an older deployment/client cannot safely interpret new records.

Examples of breaking changes include:

- changing an existing field's type or unit in place;
- making an optional field required for reads;
- removing a union variant or collection;
- changing an NSID or reference target;
- reusing a field name with different semantics;
- writing values outside the range or vocabulary older code accepts.

Schema validation alone cannot prove semantic compatibility.

## Workflow

### 1. Establish the current contract

- Read the lexicon and current codec/model code.
- Find fixtures or tests representing existing records.
- Search feed, Explore, API, form, view, export, suggestion, and backlink code
  for assumptions about the affected field.
- Identify whether Oolong, social records, or shared code reuse the same record
  type or helper.

### 2. Analyze temporal coexistence

Answer both directions:

- Can new code read and preserve records written by old code?
- What happens when old code encounters a record written by new code?

Also answer:

- Can the PDS contain both variants indefinitely?
- Is a read-time adapter or dual-read/single-write period needed?
- Can a record be round-tripped without silently discarding unknown or legacy
  information?
- Does changing a referenced record alter backlinks, feed hydration, source
  references, or strong-reference behavior?

### 3. Map affected surfaces

Inspect only applicable layers, but do not stop at the lexicon:

- lexicon schema and validation;
- typed model/request representation;
- record encode/decode behavior;
- app descriptor and record-behavior registration;
- OAuth scopes and enabled collection lists;
- PDS store and session/witness cache behavior;
- firehose ingestion, feed conversion, Explore extraction, and rebuilds;
- references, backlinks, suggestions, exports, and OG presentation;
- JSON contracts and integration snapshots;
- Svelte and legacy forms/views;
- glossary, architecture inventory, ADR/FDR, and public-facing documentation.

### 4. Choose an evolution strategy

Prefer, in order:

1. additive optional fields with tolerant reads;
2. a new field or union variant with old-field read compatibility;
3. dual-read/single-write with an explicit retirement condition;
4. a new collection/NSID when the old and new meanings cannot safely coexist;
5. an intentional breaking change with a compatibility and user-impact plan.

Do not add speculative migration infrastructure when tolerant reads solve the
actual compatibility problem.

### 5. Implement and verify

- Add old-record and new-record fixtures before or with the implementation.
- Test missing optional fields and legacy encodings explicitly.
- Verify create, read, update, feed/index, and JSON behavior as applicable.
- For unit/precision changes, test boundary values and display/edit
  round-tripping rather than only one representative number.
- Verify sister-app isolation and OAuth scope behavior when entity availability
  changes.

### 6. Update project knowledge

- Update `docs/api/` for current contract changes.
- Update `docs/architecture/` when authority, indexing, or record boundaries
  change.
- Write or supersede an ADR for a durable cross-cutting compatibility decision.
- Update an FDR only when implemented user-visible behavior changes.
- Keep unresolved alternatives and migration work in `docs/plans/`.
- Add or update glossary entries when canonical vocabulary changes.

## Verification Guidance

Choose checks according to the changed surfaces:

- focused codec/model/validation Go tests;
- lexicon validation or generation tasks used by the repository;
- `just test` for broad Go coverage;
- `just integration-test` for PDS, auth, cache, feed, and JSON contracts;
- `pnpm run check:svelte` and `pnpm run test:svelte` for frontend consumers;
- targeted `just e2e` coverage when browser forms, route state, or mixed records
  are the failure boundary;
- `just ci-check` before declaring a broad compatibility migration complete.

Report exactly which temporal directions and record variants were tested.

## Pitfalls

- Do not assume “pre-v1” means records can be rewritten centrally.
- Do not equate a lexicon parser accepting a record with preserving its meaning.
- Do not update only create forms; existing-record views and edit round-trips are
  usually the sharper compatibility boundary.
- Do not make the witness or Explore index responsible for migrating PDS data.
- Do not delete legacy read support without evidence that old records are no
  longer expected or an explicit breaking-change decision.
