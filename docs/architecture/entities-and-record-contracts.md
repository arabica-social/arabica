# Entities and Record Contracts

Authoritative sources:

- [`internal/arabica/app/config.go`](../../internal/arabica/app/config.go)
- [`internal/entities/entities.go`](../../internal/entities/entities.go)
- [`internal/entities/record_behavior.go`](../../internal/entities/record_behavior.go)
- [`internal/arabica/entities/`](../../internal/arabica/entities/)
- [`lexicons/`](../../lexicons/)

Related decisions: [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md),
[ADR-007](../adr/ADR-007-single-app-backend-composition.md),
and [ADR-005](../adr/ADR-005-separate-entity-descriptors-from-record-behavior.md).

## App Boundary

`arabicaapp.App` groups Arabica collection identities, route metadata, branding
and OAuth scope helpers. It is not a product plugin and does not decorate record
stores. HTTP handlers construct the authenticated Arabica store directly.

| Concern | Owner |
|---|---|
| Protocol clients and cached PDS record access | `internal/atproto` and `internal/records` |
| Coffee models, codecs and record behavior | `internal/arabica/entities` and `internal/arabica/store` |
| HTTP request behavior and route composition | `internal/handlers` and `internal/routing` |
| Collection identities, routes and branding | `internal/arabica/app` |
| Persisted record schema | `lexicons/` and compatibility behavior in record code |

## Descriptor and Behavior Registries

An entity descriptor contains identity metadata only:

- application record type
- collection NSID
- display name

Record behavior is registered separately and can provide:

- raw-record decoding
- reference field declarations
- reference hydration

Routing, edit URLs, Svelte rendering, and feed-card presentation do not
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
- Collection lookups, OAuth scopes and routes must retain Arabica identities.
- Explore constructs its coffee registry directly. Suggestions own a static
  coffee field/deduplication configuration; neither uses per-app factories.
- New entity capabilities are discovered from current registries and sibling
  implementations rather than from a fixed file checklist. Use the
  [`arabica-entity-change`](../../.agents/skills/arabica-entity-change/SKILL.md)
  skill.
