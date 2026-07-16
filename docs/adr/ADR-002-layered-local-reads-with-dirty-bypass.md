# ADR-002: Layer Local Reads with Dirty-Collection Bypass

**Status:** Accepted
**Recorded:** 2026-07-16
**Decision period:** Retrospective

## Context

Always reading a user's PDS would preserve authority but add avoidable latency
and remote load. Arabica has both short-lived per-session state and a local
firehose-backed witness index. The witness can lag immediately after a local PDS
write, so treating every indexed result as current would violate read-your-writes
expectations.

Different read shapes also have different useful cache layers: typed collection
reads benefit from session caching, while individual and cross-user record reads
often begin with witness lookup.

## Decision Drivers

- Reduce repeated PDS reads without weakening PDS authority.
- Provide useful read-your-writes behavior after local mutations.
- Reuse firehose/backfill data for low-latency reads.
- Keep cache invalidation understandable and user-scoped.
- Avoid pretending every read goes through one identical generic pipeline.

## Decision

Use layered local reads according to the read shape:

- typed collection reads use a valid per-session cache when available;
- eligible misses use the local witness cache;
- missing, unavailable, or bypassed witness data falls back to the PDS.

After a successful local PDS mutation, invalidate session state and mark the
affected collection dirty. While dirty, collection reads bypass witness results
that may predate the mutation. A successful authoritative collection refresh
repopulates session state and clears the dirty marker.

Successful writes may update or remove witness rows immediately to reduce lag,
but the next firehose event remains responsible for authoritative indexed event
and CID state.

## Alternatives Considered

### Read from the PDS every time

This is simple and authoritative but wastes existing local evidence, increases
latency, and makes common pages dependent on repeated remote calls.

### Trust witness results immediately after every write

Write-through updates reduce lag but cannot guarantee that every derived row,
CID, reference, or firehose-side effect is current. A stale collection could be
presented as authoritative.

### Force every read through one universal three-step cache abstraction

This produces a neat diagram but obscures meaningful differences between typed
collection caching, single-record lookup, cross-user reads, and derived queries.

## Consequences

### Positive

- Repeated reads are usually local and fast.
- Dirty tracking prevents a common stale-read window after local mutations.
- PDS fallback preserves correctness when local state is absent.
- The witness index is reused across store, feed, and discovery needs.

### Negative

- Cache behavior is distributed across session, store, and witness code.
- Dirty state is conservative and can temporarily give up a usable witness hit.
- Tests must cover cache hits, dirty bypass, and PDS fallback rather than only
  one happy path.
- Local and firehose write timing remains eventually consistent.

## Current Constraints

- Dirty collections must not use witness collection results as current state.
- Dirty markers are consistency guards, not durable firehose acknowledgements.
- A local cache miss or error must retain a viable authoritative fallback where
  the operation permits one.
- Cache keys and entries remain scoped so one user's data cannot satisfy another
  user's private read.

## Related

- Architecture inventory: [Runtime and data flow](../architecture/runtime-and-data-flow.md)
- ADRs: [ADR-001](ADR-001-pds-records-are-authoritative.md), [ADR-003](ADR-003-local-indexes-are-rebuildable-projections.md)
