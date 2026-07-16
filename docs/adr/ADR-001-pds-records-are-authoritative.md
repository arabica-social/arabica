# ADR-001: PDS Records Are Authoritative

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Arabica is built on AT Protocol. Beans, brews, gear, recipes, and social records
belong to users and can live on independently operated Personal Data Servers.
Arabica also maintains local record indexes and caches to make reads, feeds, and
discovery practical. Separate local operational stores own deployment state such
as OAuth sessions; they are not copies of user application records.

The system needs one unambiguous answer for whether a user record exists and
which copy is authoritative when local state is missing, stale, or unavailable.

## Decision Drivers

- Preserve AT Protocol's user-owned data model.
- Avoid making one Arabica deployment a required custodian of user records.
- Support federation and records created or read outside the local process.
- Keep local indexes disposable and recoverable.
- Avoid a dual-write contract whose partial failures can create two sources of
  truth.

## Decision

The user's PDS is authoritative for Arabica and Oolong application records.

Record creates, updates, and deletes are completed against the authenticated
user's PDS before local cache or witness maintenance. Local record indexes and
in-memory record caches may accelerate reads and power derived views, but they
do not own the record or decide its durable existence.

When local evidence is missing or intentionally bypassed, the application falls
back to the appropriate PDS read. Public cross-user reads still apply visibility,
moderation, and ownership behavior regardless of the data source used to locate
the record.

## Alternatives Considered

### Make Arabica's SQLite database authoritative

This would simplify local queries and transactional coordination, but it would
turn Arabica into a centralized record host and conflict with the AT Protocol
ownership model.

### Treat PDS and local SQLite as co-authoritative

Dual authority could improve local availability, but every partial write would
require conflict resolution and recovery semantics. It would also make it
unclear which copy external AT Protocol clients should trust.

### Store records only on the PDS with no local indexing

This preserves clean authority but makes feed, discovery, cross-user reference
resolution, and repeated reads depend on expensive distributed fan-out.

## Consequences

### Positive

- Users retain control of their durable records.
- Local caches and indexes can be cleared or rebuilt without data loss.
- External AT Protocol clients and Arabica share the same record authority.
- A local cache failure after a successful PDS mutation does not invalidate the
  completed user write.

### Negative

- PDS availability and latency remain part of authoritative read/write paths.
- Local projections can lag and require explicit stale-state behavior.
- Arabica cannot assume it can centrally migrate every existing user record.
- Schema evolution must tolerate old and new records coexisting across PDSs.

## Current Constraints

- Never introduce a local-only copy of an application record as the durable
  source of truth without a superseding ADR.
- Do not acknowledge a record mutation before the PDS accepts it.
- Treat existing PDS records as compatibility inputs during lexicon evolution.
- Keep cross-user mutations scoped to the authenticated user's repository.

## Related

- Architecture inventory: [Runtime and data flow](../architecture/runtime-and-data-flow.md)
- ADRs: [ADR-002](ADR-002-layered-local-reads-with-dirty-bypass.md), [ADR-003](ADR-003-local-indexes-are-rebuildable-projections.md)
