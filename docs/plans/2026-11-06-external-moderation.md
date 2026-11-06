# External moderation automation

Status: proposed; no runtime behavior changed. Recorded: 2026-11-06.

## Intent and constraints

An independently deployed service in a private repository proactively reviews new
and updated Arabica content. It may hide individual records and block accounts,
with stronger account-level evidence required for blocks. Future inputs include
reports; the same service may later publish AT Protocol labels under an
operator-owned labeler DID.

A block suppresses all public content and interactions within Arabica, including
existing interactions. Unblock restores still-existing content/interactions except
independently hidden records. It does not delete PDS records or ban anyone from AT
Protocol. Arabica owns enforcement, permissions, audit, and human overrides; the
private service owns detection rules, model/provider integration, and thresholds.

Prefer refactoring existing behavior over parallel implementations. No plugin
framework, shared database access, private Go imports, second admin API, broker,
or labeler runtime in Arabica. This proposal does not change persisted AT Protocol
record formats. See [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md),
[ADR-003](../adr/ADR-003-local-indexes-are-rebuildable-projections.md), and
[ADR-007](../adr/ADR-007-single-app-backend-composition.md).

## Prior art and gaps

| Existing owner | Evidence and implication |
|---|---|
| Action routes | `internal/routing/routing.go:172-190` already registers permission-gated hide/unhide/block/unblock. Keep these paths and browser contracts. |
| Permissions | `internal/moderation/service.go:127-146` authorizes user DIDs through configured roles. Add service principals without pretending a bot is a user. |
| Action implementation | `internal/handlers/admin_json.go:62-180` writes state and then separately logs an action; hide/block ignore audit errors. Move this behavior into one shared command path. |
| Suppression state | `internal/moderation/sqlite/store.go:26-48` upserts/deletes hidden records; block/unblock similarly changes its own table. Preserve these independent states. |
| Visibility | `internal/moderation/filter.go:26-70` already expresses hidden URI OR blocked author. Feed and profile use it, but failures degrade to partial/no filtering. |
| Direct reads | `internal/handlers/entity_view_loader.go:102-149` loads own-store/witness/PDS without a moderation gate. Returning `is_record_hidden` is not enforcement. |
| Social data | `internal/handlers/handlers.go:621-645` removes explicitly hidden comments, not blocked authors; `internal/firehose/social_storage.go:30-60,127-140` counts unfiltered interactions. |
| Notifications | `internal/firehose/notification_storage.go:34-99` lists/counts without suppression. Notifications lack an originating interaction URI (`internal/firehose/notifications.go:26-32`). |
| Existing automod | `internal/handlers/report.go:33-98` independently hides records after report thresholds. It must use the same command/override rules, not remain a bypass. |

These are source-inspected gaps, not a claim of an exhaustive route audit.

## Options and recommendation

| Option | Arabica changes | Operational cost / failure behavior | Reversibility / blast radius |
|---|---|---|---|
| **A. Bot consumes AT Protocol events; existing action routes** | Scoped authentication, shared commands, enforcement repair; no content-event endpoint | Private service owns stream reconnect, checkpointing, deduplication, PDS verification and catch-up; outages delay review, not Arabica writes | Two-way integration choice; enforcement changes affect all readers |
| B. Arabica exposes a durable polling feed | All of A's enforcement work plus a transactional change journal, cursors, retention, replay and bootstrap | One ingestion owner; Arabica owns another durable delivery contract and bot lag monitoring | Reversible internally, but published cursor/retention contract becomes costly to change |

**Recommend A initially.** In the context of an external service that may become a
labeler, choose existing AT Protocol ingestion and one shared Arabica action path
over a new Arabica event feed, to minimize duplicated behavior and public API
surface, accepting ingestion/recovery responsibility in the private service.

B's strongest argument is a much simpler bot with Arabica-specific context and
report events in one place. Reconsider it if an actual consumer needs that context
or operating a separate ingestion checkpoint proves burdensome. Do not implement
both now. This changes the earlier tentative preference for an Arabica event feed
in response to the minimal-change requirement.

```text
AT Protocol events / PDS ──> private bot (later: labeler)
                                    |
                         scoped hide/block requests
                                    v
Human moderator ──> existing routes ──> shared moderation commands
Report automod ──────────────────────>            |
                                      existing state + atomic audit
                                                 |
                                      shared visibility queries
```

The integration is a two-way door. A deployed label vocabulary/trust policy and
account-wide enforcement have broader consequences; specify those before rollout.

## Content-event contract: reuse AT Protocol, not a new endpoint

The private service subscribes to the exact currently supported Arabica
collections, including comments and likes, rather than just feedable entities or
a broad suffix match. Use the running app's collection configuration as the source
of truth. It reviews textual/media content where applicable; interactions also
provide account-level evidence. Record deletes invalidate pending evidence.

The bot persists its own stream checkpoint and decision IDs, deduplicates replay,
and checks the latest record/CID through trusted PDS resolution before acting.
Jetstream is a discovery transport, not proof that a relay-supplied record is
current or authentic. Treat text, media and report contents as untrusted input;
credentials and action authority never enter a model prompt or tool sandbox.

A reconnect cursor is not an unlimited replay guarantee. Before enabling actions,
the private implementation must demonstrate recovery inside its source's replay
window and an explicit catch-up/backfill procedure outside it. Limit v1 coverage
to the configured monitored collections/accounts; expose lag and gaps rather than
claiming global or gap-free review. Historical corpus review is an explicit run,
not an accidental consequence of reconnecting.

Review is asynchronous: records can be visible before a decision. Pre-publication
quarantine is a different feature, not an implied guarantee here.

## Action contract: extend the existing routes

Keep `POST /_mod/hide` and `POST /_mod/block`, plus existing human reversal routes.
Accept a JSON representation alongside the existing form representation; both
normalize into the same command. Preserve existing form keys and success fields
from [the admin API](../api/admin.md). Add machine-readable error codes and an
action ID/outcome without requiring a new bot-specific handler implementation.

| Field | Hide | Block |
|---|---|---|
| Target | Existing `uri` | Existing `did` |
| Explanation | Existing `reason`, plus stable `reason_code` | Same, with an account-level rationale |
| Evidence | `observed_cid` for that URI | Bounded list of `{uri, cid}` belonging to the target DID |
| Provenance | `decision_id`, `rule_version` | Same |

Require the machine fields for service principals; keep existing human forms
working. Version the classification vocabulary in documentation, not a second
state store. Store classification/provenance in the existing audit details; do
not automatically turn every local classification into an AT Protocol label.

`internal/moderation` owns command invariants and actor/permission semantics;
`internal/moderation/sqlite` owns transactional persistence. Handlers decode
requests and format responses, and protocol clients remain independent of HTTP
and moderation composition. Extend these owners rather than adding a parallel
application service or bot-specific store.

The shared command path must:

1. Authenticate a principal and check its explicit action permission. A service
   principal uses a stable ID such as `service:arabica-reviewer`, not a fabricated
   DID or membership in the human moderator list. Do not grant incidental AI,
   settings, PDS-write, purge/export, or admin-dashboard access.
2. Check `(principal, decision_id)` before performing fresh work. Identical retry
   returns the original action/outcome; a changed payload with the same ID is
   rejected. Add indexed request identity/fingerprint fields to the existing
   audit table rather than a second receipt/event system. Record accepted no-op
   outcomes too, without overwriting the original suppressor's attribution.
3. Validate canonical target syntax, collection allowlist, reason/evidence size,
   and evidence ownership. Resolve current record versions through the existing
   trusted PDS client, with bounded timeouts/concurrency; unavailable or changed
   evidence does not authorize a fresh action. A repeat violation may justify a
   block; a clearly abusive account need not wait for an arbitrary hide count.
4. In one SQLite transaction, recheck applicable human holds, local known version
   changes, permission/config generation and action budget; change suppression
   state and insert its audit/receipt. Audit failure rolls back the action.
5. Return the action ID and `applied`/`unchanged` outcome. Use 401/403 for auth,
   409 for stale evidence, protected targets or conflicting retries, 429 for a
   reached budget, and 503 when authoritative verification/state is unavailable.

A PDS fetch cannot be atomic with the local transaction: document the remaining
remote-write race. The check means "verified immediately before acceptance," not
"cannot change afterward." A hide remains URI-scoped across edits until unhidden,
as today; CID-scoped label semantics are a separate future policy. In particular,
do not trust `records.cid` alone: `UpdateWitnessRecord` changes a body without
changing its CID (`internal/firehose/index.go:478-482`). No review event should
pair that body with the old CID.

Use high-entropy bearer credentials over TLS, server-side credential digests,
constant-time verification, log redaction, rotation and revocation. Extend the
existing moderation config/auth seam; no token-management UI or general OAuth
server. Accept these credentials only on explicitly allowed moderation routes.
Invalid bearer credentials must not fall back to a human cookie. Retain browser
cross-origin protection; do not globally exempt moderation paths for bots.

Separate per-principal hide/block budgets and a server-side automation disable
switch apply regardless of model recommendations. Start in observe-only mode in
the private service, then enable actions. A lower configurable block budget is a
blast-radius control, not a substitute for the private service's stronger block
threshold. Maintain budgets across restarts; atomically account for accepted
state-changing actions, not duplicate deliveries.

## Human overrides and reversible enforcement

Retain existing hidden-record and blacklist tables. Add one small per-subject guard
table holding revision and automation hold; do not duplicate suppressed state.
Unhide/unblock leaves a durable hold even after the existing suppression row is
deleted. Automated commands, including report automod, cannot clear it. Releasing
a hold is an explicit human control on the existing moderation surface, with an
audit entry. Migrate existing manual suppression conservatively as protected;
backfill reversal holds from the latest applicable unhide/unblock audit entry per
subject where history is available. Flag audit gaps rather than claiming earlier
human reversals can always be reconstructed.

A record hold prevents automatic re-hiding of that record. An account hold prevents
automatic re-blocking of that account. Neither silently exempts the other scope:
unblocking permits later record-specific hiding; unhiding a record does not
whitelist its account against an independently justified block. Make this scope
visible to moderators. Holds/revisions are durable operational state, not a cache.

**Continue ingesting raw records, likes, comments and notification history while
blocked.** Suppress them in reads and aggregates; never skip or purge them just
because of a block. Otherwise unblock cannot restore interactions received while
blocked. Continue processing genuine PDS deletes and updates, so unblock never
resurrects deleted or superseded data. Restoration is subject to existing retention
and upstream deletion; it is not an archive guarantee.

Reject new interactions by blocked actors, and new interactions targeting
suppressed content, before PDS writes through Arabica. Permit ownership-checked
deletes/undo operations and account recovery/settings; do not introduce a blanket
"blocked users cannot authenticate" middleware. Entity creation can remain a PDS
write whose public content is suppressed. External clients can still write to the
PDS, but those records/interactions must not participate in Arabica's public views.

## Enforcement checklist and shared seams

| Surface | Required refactoring / acceptance check |
|---|---|
| Feed, Explore, profile and library lists | Reuse the existing filter; cover every entity, including non-brew profile lists. Apply visibility before pagination, totals, facets and ranking, not only to final cards. |
| Direct entity and backlinks | Gate canonical owner DID/URI in `EntityViewLoader`, before loading/hydrating. Make backlink source queries and totals visibility-aware; cover nested references too. Missing/suppressed referenced content must not expose names or summaries. |
| Social comments, likes and counts | Refactor existing `socialIndexStorage` reads to shared visible-row queries/views; single and batch counts must agree. Replace per-comment hide checks with the same URI/author policy; preserve unblocked replies without leaking suppressed parent text. |
| Notifications and unread counts | Apply the same suppression before LIMIT and COUNT, including actor, subject owner and subject visibility. Persist originating interaction URI for new notifications so an independently hidden comment/like can suppress its own notification. Backfill resolvable old rows; conservatively suppress ambiguous rows tied to hidden interactions rather than leak text. Keep raw history. |
| Explore popularity/ratings/derived statistics | Refresh or invalidate affected derived statistics on moderation changes. Never serve stale suppressed contributions while a recomputation is pending; omit affected metrics or compute from visible sources. Audit reference-derived ratings as well as likes/comments. |
| OG images and SPA head metadata | Use the same visibility gate in image and metadata loaders, including references. `WriteOGImage` currently emits a 24-hour public cache (`entity_views.go:281-285`); require revalidation for moderated resources and purge caches under our control. Previously cached third-party previews cannot be recalled. |
| Mutations | Share a subject/actor participation check across like/comment/report creation and other Arabica interaction writes. Do not reject external ingestion: retain raw events, but use visibility-aware queries for every public projection. |
| Moderator evidence / private own-record management | Keep explicit authorized access to raw evidence and owner-scoped record management; normal public detail/profile paths do not gain an implicit moderator bypass. Do not grant raw dashboard access to the bot. |
| Failures | Change enabled moderation checks from silent fail-open to explicit unavailable responses for affected public reads/writes. Preserve intentionally unconfigured deployments; distinguish "disabled" from "database unavailable." |

Keep writes on existing base tables and public reads on shared visibility-aware
queries. Named SQLite views for visible records/interactions are preferable to
copying blacklist predicates into every count/list query. Keep the existing
in-memory filter for non-SQL/PDS reads and test parity with the SQL predicate; no
policy DSL or generic repository layer. Raw storage must remain usable for witness
recovery and authorized evidence, not become silently filtered everywhere.

This is the unavoidable larger part of the work. Adding only bot credentials and
calling the current block route would automate an incomplete exclusion contract.

## Implementation slices and verification

1. **Unify commands first.** Move existing hide/unhide/block/unblock and report
   auto-hide onto one command/store transaction. Add audit-backed idempotency,
   guards/holds and migration fixtures. Preserve current routes and form clients.
2. **Complete enforcement.** Refactor shared query/load/participation seams above,
   including counts, errors and cache behavior. Prove block/unblock restoration
   end-to-end before giving a service principal write access.
3. **Enable one external client.** Add scoped bearer authentication, persisted
   budgets and operator disable controls to the same paths. Publish the machine
   payload contract; validate with a hermetic client, then observe-only deployment.
   Build ingestion/detection in the private repo, not in this change.

Each slice should stand alone and pass its tests; do not retain old command bodies
as a second implementation after migrating their callers. Update current API,
architecture, glossary and feature documentation only as behavior actually ships.
This document is not an accepted ADR or a description of deployed behavior.

Required implementation tests: existing browser forms still work; bot hide/block
permissions cannot reach unrelated routes; revoked credentials fail immediately;
identical retries and concurrent requests produce one receipt/action; changed
retry payloads fail; audit failure rolls back; human reversals beat late/replayed
bot and report-automod actions; restart does not reset holds/budgets; stale or
unverifiable PDS evidence cannot trigger a fresh block. Migration tests preserve
existing suppression, attribution and valid audit history.

Integration fixtures must cover an account blocked after it has content, likes,
comments and notifications, new external activity while blocked, paginated results
and totals, direct-load/PDS fallback, references/backlinks, OG/head requests,
moderation-store failure and unblock. Independently hidden records stay hidden;
other still-existing interactions return; deleted upstream records do not. These
are required future tests, not assertions established by this proposal.

Before implementing scoped credentials, rerun targeted Go tests for moderation,
middleware, handlers, firehose and feed, then the integration suite and relevant
moderation/profile/entity browser checks. Baseline actually run for this proposal:
`go test ./internal/moderation/... ./internal/middleware -count=1` — all three
packages passed. No new runtime behavior or cross-route enforcement was tested.

## Deferred work and risks

- Reports: add narrowly scoped report reads with reporter privacy; do not expose
  the entire dashboard as a shortcut. Report investigations call the same commands.
- Labeler: publish signed assertions from the private service, with explicit issuer
  trust and classification-to-action policy in Arabica. Existing internal labels
  are not signed AT Protocol labels. Choose one enforcement adapter per decision;
  do not apply direct decisions and their published labels twice. A retraction
  must not clear a human hold or an independently justified local suppression.
- Operational risk: a compromised bot can act within its scopes. Separate block
  budget, revocation, human holds and complete audit limit damage; they cannot
  establish classification accuracy. A private repo is not a security boundary.
- Storage: the external service owns bounded stream backlog/evidence retention.
  Arabica retains existing audit as operational history; add an explicit archive
  policy before pruning request identities or override history. Index new receipt
  and guard lookups; no scan of the full audit on each command. No new unbounded
  content journal or in-process queue is introduced by option A.
- Open implementation details: initial action budgets, supported reason codes,
  private-service catch-up limits, and treatment of historical ambiguous
  notifications. Resolve with fixtures/operator settings before production writes.
