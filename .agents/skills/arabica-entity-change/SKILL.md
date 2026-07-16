---
name: arabica-entity-change
description: Add or change an Arabica/Oolong entity by discovering its required capabilities from current registries and sibling implementations. Use for new record types or changes to CRUD, feed, Explore, references, backlinks, routes, forms, or presentation.
---

# Arabica Entity Change

Add or change entity capabilities without relying on a fixed file checklist.
The repository's registries, app configuration, routes, and tests are the
current sources of truth; package reorganizations should not make this workflow
stale.

## When to Use

Use this skill for:

- a new Arabica or Oolong record type;
- enabling or disabling an entity for one app;
- adding CRUD, feed, Explore, reference, backlink, suggestion, OG, API, or page
  capabilities to an existing entity;
- reviewing whether an entity is fully wired for its intended behavior.

Use `arabica-record-evolution` alongside this skill whenever a persisted record
contract is introduced or changed.

## Read First

1. `docs/architecture/entities-and-record-contracts.md`
2. `docs/adr/ADR-004-shared-platform-and-app-owned-package-boundary.md`
3. `docs/adr/ADR-005-separate-entity-descriptors-from-record-behavior.md`
4. The owning app's `domain.App` construction, entity registrations, routes,
   and store behavior
5. One sibling entity with the closest capability set

Read the relevant FDR or plan when the task changes user-visible behavior. Do
not infer feature requirements solely from another entity's implementation.

## Action Mode

- Implementation requests authorize coordinated code, test, generated-output,
  API-doc, and directly affected knowledge updates.
- Audits and reviews are report-only unless the user asks to apply fixes.
- Ask before adding a capability whose product behavior is not established,
  especially feed visibility, Explore indexing, social actions, or cross-user
  reuse.

## Workflow

### 1. Define the capability set

Before editing, classify what the entity actually needs:

| Capability | Questions |
|---|---|
| Identity/schema | Which app owns it? What NSID, record type, display name, and lexicon apply? |
| Model/codec | How is the record validated, encoded, decoded, titled, and keyed? |
| References | Which direct AT-URI or strong-reference fields exist? How are they hydrated? |
| CRUD | Is it create/read/update/delete, read-only, or derived? |
| Session/witness | Does normal app-store caching handle it, or is custom behavior required? |
| Feed | Is creating the record an activity users should see? What references must feed cards resolve? |
| Explore | Is it a reusable discovery object? Which explicit facets and source relationships apply? |
| Backlinks | Which records can refer to it, and where should reverse relationships appear? |
| Suggestions | Does a form need existing-record suggestions or cached options? |
| API | Which list, detail, mutation, and auxiliary JSON contracts are required? |
| Pages/forms | Which SPA routes, legacy fallbacks, dialogs, and validation states are required? |
| Presentation | Which record cards, OG images, type colors, labels, and actions are required? |
| Social | Can it be liked, commented on, shared, moderated, or deleted by its owner? |

Record the intended capability set in the task notes. “Same as Bean” is not
sufficient unless every capability is intentionally the same.

### 2. Discover current extension points

Inspect current code instead of trusting historical file lists:

- `domain.App` and the owning app constructor for enabled descriptors and routes;
- `internal/entities` for descriptor and record-behavior contracts;
- the owning app's entity registration package;
- app route registration and generic entity route bundles;
- record store, firehose/feed, Explore, suggestion, backlink, and OG registries;
- Svelte route/component registries and remaining legacy surfaces;
- architecture and integration tests that enforce app isolation or contracts.

Choose a sibling entity by capability similarity:

- a simple independent entity for basic CRUD;
- an entity with references for hydration/backlinks;
- a reusable entity already present in Explore;
- an activity entity already present in the feed.

### 3. Preserve app and registry boundaries

- Keep app-specific models, codecs, presentation, and route behavior in the
  owning app package.
- Supply shared behavior through existing app configuration or narrow
  interfaces; do not add imports from shared packages into
  `internal/arabica` or `internal/oolong`.
- Keep descriptors limited to identity metadata.
- Register behavior separately from identity.
- Ensure global registration cannot leak the entity into the sister app's
  OAuth scopes, routes, feed, Explore index, or database.

### 4. Implement by capability

Only touch layers justified by the capability set. For each layer:

- follow the current sibling pattern;
- keep generic mechanics shared and product policy app-owned;
- update generated code when the source contract requires it;
- add focused tests at the layer where omission would fail;
- update API documentation for current JSON behavior.

If the entity has a lexicon or persisted-record change, pause and apply the
`arabica-record-evolution` compatibility analysis before choosing the write
shape.

### 5. Verify end-to-end registration

Check that:

- the owning app includes the descriptor and the sister app does not;
- NSIDs and OAuth scopes are app-correct;
- duplicate descriptor or behavior registration fails loudly;
- create/read/update/delete work for the intended operations;
- references resolve and backlinks appear where intended;
- feed and Explore inclusion match the product decision rather than occurring
  through global registration side effects;
- JSON list/detail/mutation contracts are covered;
- direct-load SPA routes and error/auth states work when pages are added;
- legacy fallbacks remain only where still needed;
- deletion invalidates session/witness/derived views as applicable.

### 6. Update project knowledge

- Add genuinely new canonical terms to `docs/GLOSSARY.md`.
- Update `docs/architecture/entities-and-record-contracts.md` only when the
  extension model or boundaries change; do not manually duplicate the registry.
- Write an ADR only for a durable cross-cutting choice with real alternatives.
- Create or update an FDR only for implemented user-visible feature behavior.
- Keep unresolved modeling choices and future capabilities in `docs/plans/`.

## Verification Guidance

Use the narrowest useful checks, then broaden according to capability spread:

- descriptor, registry, codec, and validation unit tests;
- architecture seam tests for shared/app boundaries;
- focused handler and route-registration tests;
- `just test` for broad Go registration and generation coverage;
- `just integration-test` for PDS CRUD, cache, cross-user, firehose, and JSON
  behavior;
- `pnpm run check:svelte` and `pnpm run test:svelte` for forms, pages, and
  shared components;
- targeted `just e2e` or `just e2e-oolong` for browser-critical flows;
- `just ci-check` for a completed cross-layer entity addition.

Report capabilities deliberately omitted as well as those implemented.

## Pitfalls

- Do not copy every file touched by an older entity; architecture may have
  consolidated or removed that extension point.
- Do not put routes, rendering callbacks, or form policy on the descriptor for
  convenience.
- Do not register an entity globally and assume app scoping will happen later.
- Do not make every record feedable or discoverable by default.
- Do not add frontend UI before the JSON and existing-record contracts are
  understood.
- Do not describe a partially wired entity as complete without verifying its
  declared capability set.
