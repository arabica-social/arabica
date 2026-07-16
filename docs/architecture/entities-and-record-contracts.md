# Entities and Record Contracts

Authoritative sources:

- [`internal/atplatform/domain/app.go`](../../internal/atplatform/domain/app.go)
- [`internal/entities/entities.go`](../../internal/entities/entities.go)
- [`internal/entities/record_behavior.go`](../../internal/entities/record_behavior.go)
- [`internal/arabica/entities/`](../../internal/arabica/entities/)
- [`internal/oolong/entities/`](../../internal/oolong/entities/)
- [`lexicons/`](../../lexicons/)

Related decisions: [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md),
[ADR-004](../adr/ADR-004-shared-platform-and-app-owned-package-boundary.md),
and [ADR-005](../adr/ADR-005-separate-entity-descriptors-from-record-behavior.md).

## App Boundary

`domain.App` is the runtime seam between shared platform infrastructure and an
individual application. It provides the app name, NSID base, enabled entity
descriptors, entity routes, branding, and record-store behavior.

Shared packages discover an app's supported collections and routes through this
configuration. They must not import Arabica or Oolong packages to ask app-
specific questions.

| Concern | Owner |
|---|---|
| Shared AT Protocol, OAuth, routing, cache, firehose, feed, and record mechanics | Shared platform packages |
| Coffee record registrations and behavior | `internal/arabica` |
| Tea record registrations and behavior | `internal/oolong` |
| Per-app enabled descriptors, routes, branding, and NSID base | `domain.App` construction |
| Persisted record schema | `lexicons/` and compatibility behavior in app record code |

## Descriptor and Behavior Registries

An entity descriptor contains identity metadata only:

- application record type
- collection NSID
- display name

Record behavior is registered separately and can provide:

- raw-record decoding
- form-field extraction
- record-key and display-title access
- reference field declarations
- reference hydration

Routing, edit URLs, Templ/Svelte rendering, and feed-card presentation do not
belong on the descriptor. Architecture tests enforce this boundary.

## Lexicons and Existing Records

Lexicons define AT Protocol record contracts, but deployed code must also cope
with records already written under earlier schemas. Unlike a centralized
database migration, Arabica cannot assume it can rewrite every user's PDS.

Record evolution therefore considers:

- old records read by new code
- new records encountered by older deployments or clients
- field absence and changed semantic meaning
- reference compatibility
- numeric unit or precision changes
- firehose, witness, feed, Explore, and UI decoding

Use the
[`arabica-record-evolution`](../../.agents/skills/arabica-record-evolution/SKILL.md)
skill for these changes.

## Relationships

Records identify each other with AT-URIs. Some relationships also use strong
references when a URI and CID are required. Record behavior declares direct
reference fields that shared feed/index code should fetch before app-owned
hydration runs.

Backlinks are reverse queries over these relationships. Source references are
provenance relationships used by copied or forked records and by Explore
clustering; they are not inferred from similar names.

## Current Constraints

- App-specific record behavior remains in app-owned packages.
- Shared registries must not acquire Templ or Svelte presentation ownership.
- Global registration must not leak sister-app entities into app-scoped routes,
  OAuth scopes, feed queries, or databases.
- New entity capabilities are discovered from current registries and sibling
  implementations rather than from a fixed file checklist. Use the
  [`arabica-entity-change`](../../.agents/skills/arabica-entity-change/SKILL.md)
  skill.
