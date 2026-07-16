# ADR-003: Local Indexes Are Rebuildable Projections

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Community feeds, social counts, notifications, witness reads, and faceted Explore
queries need local query structures that cannot be produced efficiently through
PDS fan-out on every request. Arabica therefore stores firehose-observed records
and derived data in SQLite.

These local stores can lag, be deleted, or change schema. They need a recovery
model that does not turn derived rows into irreplaceable user data.

## Decision Drivers

- Keep the PDS as record authority.
- Support fast feed, social, witness, and Explore queries.
- Recover from index loss or schema changes.
- Make stale or rebuilding state explicit rather than silently healthy.
- Avoid coupling user writes to every derived view.

## Decision

Treat the local firehose/feed index and its derived Explore index as rebuildable
projections.

The firehose index is populated through ongoing Jetstream events, PDS backfill,
and successful local write-through maintenance. Explore derives searchable
documents, facets, clustering, ratings, and popularity data from indexed
records. Explore schema versions may clear and rebuild those derived rows.

Neither index is an authoritative mutation target for application records.
Read paths and user interfaces may expose explicit stale, dirty, rebuilding, or
unavailable states when local projection freshness is uncertain.

## Alternatives Considered

### Query users' PDSs directly for every feed or discovery request

This avoids local projection drift but requires unbounded distributed fan-out
and cannot efficiently provide global sorting, social aggregation, or facets.

### Make the SQLite index an authoritative mirror

This would make queries straightforward but create a second source of truth and
require durable conflict resolution with independently changing PDS records.

### Require every derived index update in the user write path

Synchronous fan-out could improve freshness but would make successful PDS writes
depend on unrelated local projection schemas and availability.

## Consequences

### Positive

- Feed and Explore queries are local and indexable.
- Projection schemas can evolve through rebuilds rather than user-data
  migrations.
- Index loss does not imply loss of authoritative application records.
- Derived views can be updated independently from PDS mutation availability.

### Negative

- Results can lag the PDS or firehose.
- Backfill and rebuild operations require readiness and progress handling.
- Some social or reference-derived values may be temporarily inconsistent.
- Tests must verify rebuild, deletion, and incremental update behavior.

## Current Constraints

- Do not add application-record writes whose only durable destination is a local
  projection.
- Keep a recovery path from PDS backfill and/or ongoing firehose events.
- Treat projection absence and errors as explicit states, not authoritative
  record deletion.
- Preserve moderation and ownership behavior when serving projected records.

## Related

- Architecture inventory: [Runtime and data flow](../architecture/runtime-and-data-flow.md)
- ADRs: [ADR-001](ADR-001-pds-records-are-authoritative.md), [ADR-002](ADR-002-layered-local-reads-with-dirty-bypass.md)
- FDRs: [FDR-001](../fdr/FDR-001-explore.md)
