# Tools-only AI record context

Status: proposed alternative, not implemented. Recorded 2026-11-12.
Decision size: two-way door; no PDS schema or write-authority change.

Current behavior belongs to [FDR-007](../fdr/FDR-007-ai-assisted-creation.md)
and [the AI API contract](../api/ai-chat.md). This proposal does not enable tools.

## Intent and scope

Today, every chat request includes a bounded own-library catalog and the latest
three brews with dates. The user wants to consider fetching context at the
model's discretion instead, reducing irrelevant record context. Natural requests
such as “same as last time” and “last V60 with these beans, but 16g” must still
work. The model chooses when to ask; the server chooses what it may read.

Success means less total provider input on representative conversations without
worse matching, lost preparation details, unsafe references, or unacceptable
extra latency. Token savings are a hypothesis, not a promised outcome: tool
schemas, results, and repeated conversation prefixes also consume tokens.

Non-goals: writes, web search, arbitrary queries, cross-account lookup, new
record schemas, general-purpose agents, or automatic changes to saved records.
The Save button remains the only record-write entry point.

## Prior art and ownership

- `internal/ai/client.go:14` supports assistant tool calls and tool-result message
  IDs; `:80` accepts caller-defined tool schemas. It executes no tools.
- `internal/arabica/ai/service.go` owns the draft conversation, allowed fields,
  catalog conversion and final response validation. Currently it sends no tools
  and rejects unexpected tool calls.
- `internal/arabica/ai/catalog.go:16` defines a read-only, account-scoped
  `LibraryStore`; `:61` projects a bounded catalog from collection reads.
- `internal/arabica/ai/recent_brews.go` projects three dated brews and validates
  an optional provider-selected source ID for scoped linked-reference grounding.
- `internal/arabica/ai/mentions.go` distinguishes supplied reference IDs from
  grounded permission to select them. A search hit must not erase that distinction.
- `internal/arabica/ai/quota.go:95` currently charges dialogue and images, not
  server-added catalog context. A multi-call loop needs an explicit quota policy.
- [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md) and
  [runtime read paths](../architecture/runtime-and-data-flow.md) govern authority,
  cache eligibility and dirty bypass. Tools must use those paths, not query an
  index as an independent source of truth.

App-owned orchestration belongs in `internal/arabica/ai`; generic transport stays
in `internal/ai`. No HTTP or product dependencies move into protocol/storage.

## Options

| Option | Initial context | Latency and cost | Complexity / operation | Reversibility / blast radius |
|---|---|---|---|---|
| Always include | Bounded library plus three brews | One call; pays for unused records | Existing path; broad provider compatibility | Baseline; AI service only |
| One `search_records` tool with a type enum | Draft, policy, one schema | Potentially smaller schemas; lookup adds calls | One dispatch surface; type-dependent arguments/results can become awkward | Reversible; same authority boundary |
| Separate typed search tools | Draft, policy, several small schemas | More schema tokens; independent lookups may share a round | Clear purpose and validation per type; shared internal mechanics | Reversible; same authority boundary |

All three use the same component topology: browser → authenticated AI service →
configured provider; AI service → authenticated store → eligible caches/PDS.
Only tools-based options add a provider → service lookup request between model
calls. There is no provider → PDS connection.

**Provisional recommendation:** separate typed tools, with shared implementation
helpers rather than duplicated search engines. Brew history has materially
different ordering and filters from named equipment. Keep this choice provisional
until an evaluation compares it with a single typed search tool.

In the context of reducing irrelevant record context while preserving grounded
brew drafting, we would choose typed read-only lookup tools over an always-sent
catalog or one polymorphic tool, to make intent and argument validation explicit,
accepting extra model round trips and schema overhead.

The strongest alternative is one `search_records` tool: named-library searches
largely share a query and limit, so multiple nearly identical schemas may cost
more tokens without helping tool selection. If measurement shows that, consolidate
the named-library tools while keeping brew-history retrieval separate.

## Proposed tools

Names below are model-facing APIs, not new public HTTP endpoints.
All objects reject unknown arguments. No tool accepts DID, owner, URL, NSID,
SQL, arbitrary sort expressions, or provider-selected credentials.

| Tool | Arguments | Result |
|---|---|---|
| `search_beans` | optional query, optional status (`open`, `closed`, `any`), limit | Named candidates with roaster/open status |
| `search_grinders` | optional query, limit | Named candidates with grinder metadata |
| `search_brewers` | optional query, limit | Named candidates with brewer type |
| `search_roasters` | optional query, limit | Named candidates for bean drafts |
| `search_recipes` | optional query, optional supplied brewer ID, limit | Saved preparation, linked brewer and bounded instructions |
| `search_brews` | optional supplied bean/brewer/grinder IDs, optional after/before RFC3339 timestamps, limit | Newest-first dated preparation, linked records |

Named tools default to 5 results, maximum 10; query strings cap at 200 characters.
`search_brews` defaults to 3, maximum 10, and supports an empty argument object
for “last time.” Dates compare as instants, with deterministic tie ordering.
Never invent a missing brew date from record creation order. Undated records must
be distinguishable and cannot silently become the “latest dated brew.”

Every result includes `items`, `has_more`, and an explicit completeness scope.
A bounded result is not proof of uniqueness or nonexistence. Exact totals appear
only when the backing read actually establishes them. Failed reads are typed
errors, not empty search results. No pagination in the first iteration: request a
more specific query instead. A later cursor must be opaque and account/query-bound.

Results expose compact labels, opaque IDs and allowlisted human/API-unit values,
never account identifiers or raw references. Recipe instructions are bounded and
marked when truncated. Brew results contain preparation, not historical ratings
or tasting outcomes; their dates identify the source, not a new draft's date.
Cross-user, malformed or missing links never become selectable own-library IDs.

“Tools only” means no unsolicited library/history dump. The current draft's
selected records remain as small context anchors, hydrated by the server. Empty
query searches support browsing when no name is known. Determining a sole open
bag requires explicit full-set metadata, not inference from one returned hit.

## Bounded execution and reference lifetime

1. Authenticate, validate the request and draft revision, and acquire the existing
   per-account admission slot. Hold it for the whole turn, not each model call.
2. Build policy, dialogue, current draft anchors and server-defined tool schemas.
3. Accept either the existing final draft envelope or a validated batch of tool
   calls. Require unique call IDs, allowlisted names and strictly typed arguments.
4. Run read-only lookups against the authenticated store, reusing collection reads
   within this turn. Return each result with its exact tool-call ID.
5. Append the assistant call and tool results, then ask the model again. Validate
   the eventual final envelope with the existing field/reference pipeline.

Proposed initial limits: 2 lookup rounds, at most 4 calls per round and 6 total,
3 provider calls including the final answer, a shared 45-second wall deadline,
32 KiB per tool result and 96 KiB cumulative result data. Reject a provider batch
that exceeds the budget before executing any of it. At the round limit, omit tool
schemas and require a final response; further tool calls fail with a controlled
error. Never silently omit requested tool results or truncate structured JSON.

Opaque IDs need a per-turn registry, stable across lookup rounds, mapping to
validated own records. Do not renumber IDs when a later search returns more hits.
Register linked records returned in brew/recipe results too. Across turns, rebuild
anchors from validated draft keys; never trust an assistant's old ID as authority.
A filtered search hit permits consideration, not automatic selection. Preserve
full-library ambiguity checks, closed-bean safeguards and current manual values;
explicit historical source selection must authorize only that source's validated
linked records. “Use the last brew” must not grant blanket access to any result.

Dates and copied settings stay data, not instructions. Explicit user overrides
win over historical preparation. Reusing history must not import old ratings,
tasting notes, record identity or timestamps into the new record.

## Failure, quotas and operations

- No automatic retries or switch back to sending the whole catalog on failure.
  An unsupported provider is an explicit configuration/capability failure; retain
  the draft and manual editing. Tool support varies across compatible providers.
- Invalid calls fail closed. Bounded, safe lookup failures can be reported to the
  model as typed unavailable results; never report “no matches” after a timeout.
- Cancel all remaining work when the user stops/leaves. Reuse frontend stale
  response, manual-edit revision, selection confirmation and save-attempt guards.
- Before rollout, define allowance accounting for every provider round. Proposed:
  charge user dialogue/images as today for every resend, plus tool-result text
  on each resend; keep static policy/schema overhead deployment-funded. This is
  a quota-contract change requiring API docs/tests, not an incidental loop detail.
- Record content-free per-call usage, latency, failures, lookup kind/count, and
  budget exhaustion. Correlate rounds under one chat request; the request log
  currently records one row per upstream observation, so audit its identity and
  aggregation semantics before enabling multi-call turns. Do not log arguments,
  results, account-labelled metrics or prompts.
- Bounding returned context does not bound collection-read cost. The current
  `FetchAllRecords` seam can materialize history; a cached O(N) scan remains O(N).
  Profile this path before adding indexes; preserve PDS fallback/dirty bypass.

## Risks and counterarguments

- **Operations:** a slow store plus multiple provider calls can exhaust the turn
  deadline. Enforce one shared budget and expose real pending state, not invented
  progress stages.
- **Security:** prompt injection in record text can induce more lookups, not new
  authority. Strict schemas, own-store execution and fixed budgets constrain it.
- **Maintainability:** per-type tools could duplicate code. Share projection,
  matching, validation and limits internally; keep only contracts type-specific.
- **Product:** a model may forget to search or pick the wrong time window. Evaluate
  natural requests and ask when ambiguous; do not paper over this with fabricated
  matches or an ever-growing phrase detector.

The current catalog is a safety mechanism, not just prompt bulk. Removing it
also removes full-set uniqueness and sole-open-bag evidence unless those facts
are deliberately represented by lookup metadata. Public mutation contracts and
persisted records must remain compatible throughout rollout.

## Implementation and verification plan

1. Extract reusable projections from `catalog.go` and recent-brew context without
   changing current output; keep existing fixtures passing.
2. Add app-owned schemas, dispatcher and turn-local ID registry under
   `internal/arabica/ai`; add wire round-trip tests in `internal/ai/client_test.go`.
3. Add the bounded loop in `service.go`, retaining final-envelope validation,
   admission, cancellation, image handling and revision semantics. Update quota
   and request-log tests before enabling it.
4. Test end-to-end with a fake provider that requests tools then returns a draft:
   latest brew with a dose override; several type lookups in one round; ambiguous
   names; closed beans; a linked record outside normal catalog limits; empty and
   failed results; foreign refs; unknown tools/arguments; duplicate call IDs;
   repeated calls; round/byte/deadline exhaustion; canceled and stale requests.
5. Compare always-include, one search tool and typed tools on the same text/image
   conversations, with small and large libraries. Measure all-round input/output
   tokens, tool success, final-field/reference correctness, clarification rate,
   p50/p95 latency and quota consumption. Report omitted/failed cases, not only
   successful turns. Set adoption thresholds before running the comparison.
6. Only after approval, introduce explicit configuration and update FDR-007,
   API and architecture docs for the selected behavior. Keep always-include as
   an operator-selectable rollback during evaluation, never a silent retry.

Open decisions: measured benefit of per-type schemas; provider capability
configuration; allowance migration; user-visible source attribution; whether
large histories warrant a bounded indexed read seam before deployment.
