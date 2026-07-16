# Runtime and Data Flow

Authoritative sources:

- [`internal/atproto/store_generic.go`](../../internal/atproto/store_generic.go)
- [`internal/atproto/cache.go`](../../internal/atproto/cache.go)
- [`internal/atproto/witness.go`](../../internal/atproto/witness.go)
- [`internal/firehose/index.go`](../../internal/firehose/index.go)
- [`internal/firehose/explore_index.go`](../../internal/firehose/explore_index.go)

Related decisions: [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md),
[ADR-002](../adr/ADR-002-layered-local-reads-with-dirty-bypass.md), and
[ADR-003](../adr/ADR-003-local-indexes-are-rebuildable-projections.md).

## Data Authority

| Data | Role | Durability | Owner/source |
|---|---|---|---|
| Arabica/Oolong records | Authoritative user data | PDS repository durability | User's PDS |
| OAuth sessions and requests | Local operational state | SQLite-backed and deployment-local | OAuth/session storage |
| Session cache | Typed collection read optimization | In-memory, two-minute validity | User-scoped session cache |
| Witness records | Local evidence of repository records | SQLite, recoverable through backfill/firehose | Firehose index |
| Feed/social views | Local query projection | SQLite, rebuildable from indexed records/events | Firehose index |
| Explore documents and values | Search/facet projection | SQLite, versioned and rebuildable | Explore index |

Local indexes can improve latency and availability but do not transfer record
ownership away from the PDS.

## Read Paths

There is no single function that always performs three identical steps. The
layers depend on the shape of the read:

- Typed collection reads first use a valid session-cache entry.
- Eligible collection and single-record misses can use witness records from the
  firehose-backed SQLite index.
- Missing, unavailable, or deliberately bypassed witness data falls back to an
  authoritative PDS read.
- Cross-user public record reads may use the witness index or a public PDS
  client according to the owning loader and visibility rules.

### Dirty Collections

A successful local write invalidates session state and marks the affected
collection dirty. While dirty, witness collection results are not trusted
because firehose delivery can lag the completed PDS mutation. An authoritative
collection refresh repopulates session state and clears the dirty marker.

The dirty marker is a consistency guard, not a durable queue or proof that a
firehose event has arrived.

## Write Path

1. Validate and encode the requested record.
2. Create, update, or delete the record on the authenticated user's PDS.
3. After PDS success, update or remove the corresponding witness record when a
   witness implementation is available.
4. Invalidate session state and mark the collection dirty.
5. Let the firehose event authoritatively refresh indexed CID/event state.

Local cache failure after a successful PDS write can reduce freshness or
performance, but it must not redefine whether the record exists.

## Firehose, Backfill, and Derived Views

The Jetstream consumer applies repository commits to `FeedIndex`. Backfill can
also enumerate existing repository records into the same index. Feed, social,
notification, witness, and Explore capabilities share this SQLite-backed
runtime while retaining separate query responsibilities.

Explore has its own schema version and rebuild operation. A rebuild clears
derived Explore rows and re-extracts searchable documents from indexed records.
The UI can continue serving results while explicitly reporting that the index
is catching up or may be stale.

## Current Constraints

- Never make a local projection the only copy of a user record.
- Never require a local index write to succeed before acknowledging a completed
  PDS mutation unless a new decision deliberately changes the failure contract.
- Treat witness and Explore results as potentially lagging.
- Preserve cross-user visibility, ownership, and moderation checks when using
  locally indexed records.
- Test both cache hits and the authoritative fallback path for correctness-
  sensitive changes.
