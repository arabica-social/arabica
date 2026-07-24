# Architecture Inventory

Last reviewed: 2026-07-16.

This directory records Arabica's current runtime shape: what exists, where its
authority lies, which package owns it, and which contracts current changes must
preserve. It is deliberately not a history or roadmap.

- Cross-cutting rationale and alternatives live in the
  [Architecture Decision Records](../adr/INDEX.md).
- Current feature behavior and feature-specific rationale live in the
  [Feature Decision Records](../fdr/INDEX.md).
- Canonical definitions live in the [glossary](../GLOSSARY.md).
- HTTP and JSON payload details live in the [API documentation](../api/README.md).
- Proposed migrations and unresolved behavior live in [`docs/plans/`](../plans/).

## System Map

1. A user authenticates through AT Protocol OAuth.
2. App handlers create a user-scoped `AtprotoStore` backed by the authenticated
   session and DID.
3. Writes go to the user's PDS before local caches or indexes are updated.
4. Reads use eligible session and witness data before falling back to the PDS;
   dirty collections bypass potentially stale witness results.
5. The Jetstream firehose and PDS backfill populate a local SQLite witness/feed
   index.
6. Feed, social, notification, and Explore reads derive local views from that
   index while the PDS remains authoritative for application records.
7. Go serves HTTP APIs, mutations, OAuth, assets, head metadata, and the
   embedded SvelteKit SPA shell according to explicit route ownership.

The platform remains app-agnostic by design, while this repository hosts the
Arabica application and its app-owned configuration, records, routes, and
presentation.

## Inventories

| Category | Contents |
|---|---|
| [Runtime and data flow](runtime-and-data-flow.md) | PDS authority, local read layers, writes, firehose ingestion, and rebuild paths |
| [Entities and record contracts](entities-and-record-contracts.md) | App configuration, lexicons, descriptors, record behavior, and relationships |
| [Interfaces and route ownership](interfaces-and-route-ownership.md) | OAuth/XRPC boundaries, HTTP/JSON surfaces, page ownership, and contract locations |
| [Frontend transition](frontend-transition.md) | Embedded SPA architecture and current Arabica cutover state |

## Stability Labels

- **Sturdy** — A current invariant or compatibility boundary. Changes require
  explicit rationale and focused verification.
- **Transitional** — Intentionally supports old and new paths during a migration.
  Do not treat duplication as a permanent target architecture.
- **Squishy** — Locally reversible presentation or implementation detail.
  Accessibility, security, and documented product/design principles are never
  squishy merely because they affect the frontend.

## Inventory Rules

- Link to executable registries and tests instead of reproducing lists that code
  already owns.
- Record current facts, not migration archaeology or plans.
- State whether data is authoritative, cached, or rebuildable.
- Update facts in place rather than appending correction notes.
- If current code and an accepted decision disagree, report the contradiction;
  do not silently invent a new decision while refreshing the inventory.
