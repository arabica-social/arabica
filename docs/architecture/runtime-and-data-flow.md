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
| Arabica records | Authoritative user data | PDS repository durability | User's PDS |
| OAuth sessions and requests | Local operational state | SQLite-backed and deployment-local | OAuth/session storage |
| AI access grants | Authoritative local operational state | SQLite-backed, deployment-local; not reconstructible from PDS | `internal/arabica/ai` (`AccessStore`), `ai_access_grants` in the shared operational/index DB |
| AI daily usage | Authoritative local operational state | SQLite-backed, deployment-local; reset by UTC day | `internal/arabica/ai` (`DailyUsageStore`), `ai_daily_chat_usage` in the shared operational/index DB |
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

### Identity Reconciliation

Jetstream identity events are untrusted refresh hints for known or registered
DIDs. The profile watcher changes handle ownership only when the trusted AppView
profile and a fresh handle resolution agree. Failed verification preserves
existing profiles and handle mappings.

A single background worker retains at most 64 pending DID/event-kind pairs.
Changed identity hints replace older pending identity hints; account hints are
retained separately. Identical hints do not reset the retry budget. Identity
verification failures retry after 1, 5, 30, and 120 seconds without blocking
other ready work. Completed or exhausted identical hints have a 30-second
cooldown. Cancellation stops pending retries. Queue overflow drops new hints;
retry exhaustion logs a warning and retains existing data. This is bounded,
best-effort reconciliation, not a durable delivery guarantee: a later event or
explicit profile refresh may still be needed after prolonged authority lag,
overflow, or process restart.

Explore has its own schema version and rebuild operation. A rebuild clears
derived Explore rows and re-extracts searchable documents from indexed records.
The UI can continue serving results while explicitly reporting that the index
is catching up or may be stale.

## Container Packaging

The portable `deploy/Dockerfile` and the Nix `container` output package the same Go/SPA
application. Containers run as uid/gid 10001 and keep persistent SQLite state in
`/var/lib/arabica`. Packaging does not split background workers from HTTP serving
or make the runtime multi-replica. Each independent deployment needs its own data
volume and OAuth origin. See [container operations](../containers.md).

## Current Constraints

- Never make a local projection the only copy of a user record.
- Never require a local index write to succeed before acknowledging a completed
  PDS mutation unless a new decision deliberately changes the failure contract.
- Treat witness and Explore results as potentially lagging.
- Explore rebuild and dependency-refresh scans release their selection cursor
  before indexing individual records, so nested work cannot exhaust the shared
  SQLite connection pool.
- Preserve cross-user visibility, ownership, and moderation checks when using
  locally indexed records.
- Test both cache hits and the authoritative fallback path for correctness-
  sensitive changes.


## Transient AI image context

Chat attachment preprocessing belongs to the SPA (`web/src/lib/ai/images.ts`);
`ChatWorkspace.svelte` keeps sanitized images in a bounded, per-mounted-chat memory
map, separate from the canonical draft and persistence envelope. Home image paste
uses the same preprocessor; `homeImages.ts` transfers sanitized attachments to chat
in a single-use, account-scoped memory slot expiring after 30 seconds. No images
are serialized for route navigation. Only attachment
counts are part of saved message metadata. Chat/account changes, route teardown,
pagehide and external-state reconciliation release the map and invalidate pending
preprocessing. Images cannot authorize record writes.

The app-owned assistant service validates inline images and supplies typed
multimodal context to the app-agnostic `internal/ai` provider transport. There are
no attachment records, blob uploads, remote image fetches, server archives or disk
files in this flow. Provider retention is outside Arabica's control. For request
bounds and the missing-image contract, see [AI chat API](../api/ai-chat.md).

## Optional AI access gate

`AccessStore` in `internal/arabica/ai` owns the dedicated, bounded
`ai_access_grants` table, initialized by server composition on the existing SQLite
connection. These are operational
permissions, not a record projection: preserve the table when rebuilding indexes
and include it in operational database backups. No PDS record schema changes.

The HTTP adapter in `internal/handlers/ai_access.go` combines explicit grants with
implicit `moderation.Service.IsModerator` access (including admins). It gates the
SPA chat route and chat API before content decoding/provider work, while status
returns per-caller configured availability. Missing storage or read errors never
permit an explicit user. Grants are read by primary key per gated request, so
revocation takes effect without a cache invalidation or server restart. Moderator
management is authenticated and mutations have same-origin protection.

`DailyUsageStore` in the same package owns the `ai_daily_chat_usage` table on the
same connection:
one charged-character count per account and UTC day. The drafting service consults
it immediately before each provider call and charges the request's message
characters plus a fixed rate per image; admins and an operator-configured DID list
are exempt. Rows are keyed by account and day and purged once that day passes, so
the table stays small. Missing storage or a failed write refuses the request
rather than allowing unmetered provider spend.

Access enforcement remains separate from drafting and record mutations, even
though operational storage and drafting share the AI package; removing the temporary rollout restriction does not require changing
provider, draft reconciliation or PDS mutation behavior.
