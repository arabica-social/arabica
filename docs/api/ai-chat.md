# AI-assisted record drafts

AI chat proposes drafts; it never creates or changes a PDS record. Record writes
remain on the existing authenticated bean, brew, grinder and brewer mutation APIs.

## Configuration

By default, load `./ai.json` relative to the server process working directory.
Set `ARABICA_AI_CONFIG` to override that path. Use `ai.example.json` as the
starting point. Replace `base_url` and `model`. Set `api_key` to a literal key
or use `"api_key": "${ARABICA_AI_API_KEY}"` to interpolate an environment variable.
The former `api_key_env` field is replaced by `api_key`. Omit `api_key` or use an
empty string only for an intentionally unauthenticated provider. Never commit a
real key; keep credential-bearing local config files private.

Only `${NAME}` syntax is expanded in the API key, after JSON decoding, once at
startup. Literal text and multiple references may be combined. Bare `$NAME` stays
literal; substituted values are not expanded recursively. Missing/empty variables,
malformed references, whitespace/control characters or keys over 8192 bytes fail
startup without exposing the key. Restart to pick up environment/key changes.
`base_url` is an API root such as `https://api.openai.com/v1`, not the complete
`/chat/completions` endpoint. Plain HTTP is permitted for self-hosted providers.

A missing default `./ai.json` disables the feature. Invalid/unreadable default
configuration, a missing explicit override, or a missing configured key fails startup. Configuration is read once; restart after
changing it. Requests are non-streaming OpenAI-compatible chat completions. The
provider must return the JSON requested in the system instruction; strict
JSON-schema response support or tool execution is not required initially.
Transport supports caller-defined tools, but the app exposes no executable or
record-writing tools. There are no automatic provider retries or fallbacks.

The example defaults to a 30-second timeout, 6 MiB provider request cap,
1 MiB provider response cap and 2048 output tokens. Redirects are not followed.
Provider response bodies and credentials are not returned as API errors.

## GET /api/ai/status

```json
{"enabled": true}
```

Availability only. No provider URL, model, credential or account information.
The homepage entry is shown only to authenticated users when enabled.

## POST /api/ai/chat

Authentication and same-origin protection are required. No client-selected
provider, target URL, system prompt, tool definitions or library catalog accepted.

```json
{
  "session_id": "local-chat-id",
  "request_id": "local-request-id",
  "revision": 2,
  "messages": [{"role": "user", "content": "V60, 15g coffee and 250g water"}],
  "draft": {"kind": "brew", "fields": {}}
}
```

Kinds: `brew`, `bean`, `grinder`, `brewer`. Messages are user/assistant text, with optional user images;
maximum 40 messages, with user messages limited to 4000 characters. Identifiers
are correlation aids, not authentication. The server reads only the authenticated
user's library for grounding. It does not accept arbitrary cross-user references.
Compact labels, bean roaster/open status, saved recipe details and opaque IDs go
to the provider; account DID, handle and the entire brew history are not included
for identification or personalization. Recipe context includes actual dose,
water, ordered pours and saved instructions, never arbitrary fetched URLs.
Recipe instructions are untrusted data, not authority to execute actions.

### Transient images

A user message may include `images`, for example:

```json
{"role":"user","content":"What does this bag say?","images":[{"mime_type":"image/jpeg","data":"<bare base64>"}]}
```

Only JPEG, PNG and WebP are accepted. No URLs, data-URL prefixes, SVG or GIF.
Image-only user messages may have empty `content`; assistant messages cannot carry
images. After reload/switch, a user message may instead carry
`images_unavailable: 1` (integer 1–4), mutually exclusive with `images`. Empty
content is also valid with this marker. It tells the provider those earlier images
are no longer visible, not to reconstruct their contents. Retry of an unanswered
image-bearing turn with unavailable images is blocked by the client; reattach in a
new message. The browser downsizes larger images before sending. Oversized or malformed
images are rejected, not transformed by the server.

Homepage image paste uses the same preprocessing and image-only support as chat.
Send transfers the text and sanitized images through a single-use account-scoped
in-memory handoff (30-second expiry), starting a fresh chat rather than appending
to an older draft. Image handoffs never enter sessionStorage; failed navigation
keeps the homepage attachments for retry. Ordinary text-only handoff compatibility
is unchanged. No upload control is added on Home.

Images remain in browser memory for follow-ups/retries until leaving, reloading or
switching chats. They are excluded from browser chat persistence. The server keeps
no image archive and does not log bytes or filenames; it only validates and forwards
them in the current request. The upstream provider receives them and its own data
retention policy applies. Use a vision-capable model; provider capability is not
automatically probed and unsupported providers fail without retry/fallback.

For image-bearing requests, the provider receives real ordered user/assistant
messages, with inline data URLs in typed `image_url` content parts on the original
user turns. Images are never base64 text inside the JSON grounding context.
Text-only requests retain the existing provider encoding. Images are untrusted
coffee context and confer no Save, tool or library-reference authority. A conversation
containing images (including unavailable-image markers) cannot authorize the
sole-open-bean default, even on later text-only follow-ups; explicit textual
library matches remain eligible, while image-only
reference guesses require review. Image decoding occurs only after request
admission and within the operation deadline.

Fields are a flat partial map of existing mutation input keys. `recipe_rkey` may
select an existing recipe in the authenticated user's own library. Nonempty
`source_ref` and `recipe_owner_did` remain unsupported; no cross-user recipe
lookup, importing or recipe creation is granted. An omitted recipe owner uses
the authenticated owner through the existing brew mutation API. Nested brew
fields use leaf paths such as `espresso_params.pressure`; `pours` is one atomic
array of `{water_amount, time_seconds}`. An absent field means unchanged; null
means explicitly clear. Preserve existing API units and precision. Dose, water,
time, rating and pour values are integer fields; temperature, espresso yield
and pressure support their existing floating-point representation. Temperature
has no persisted C/F unit; chat must not infer a unit conversion.

Response:

```json
{
  "reply": "Which bean did you use?",
  "kind": "brew",
  "patch": {"coffee_amount": 15, "water_amount": 250},
  "choices": [
    {"label": "Ethiopia Gedeb", "patch": {"bean_rkey": "real-own-record-key"}}
  ]
}
```

Optional `brewer_type` on a response or choice is derived from the actual chosen
own-library brewer, not accepted as model-invented metadata. Proposed fields and
references are validated before returning. Choices are data, not HTML/actions.
Clicking one updates a local draft; it does not create or import a library record.
After grounding, choices whose complete patch already equals the effective draft
(request fields overlaid with the response patch) are omitted. Combined labels
identify only differing references; their original validated patch and equipment
metadata remain intact for recipe coupling and revision checks. A preparation-only
change retains its source-context label and still requires selection.

Validated provider choices are normalized by reference authority, not by the
number of fields or buttons. A clear bean, equipment or roaster match can fill
the draft directly, including from a combined choice. Exact full names are not
required: meaningful short names and minor long-word typos can identify a unique
best match against the full retrieved collection. For beans, known open bags
compete before historical closed bags, so a matching open bag can fill even when
closed history repeats its name. Competing best matches remain choices; the
server supplies up to eight distinct open-bean alternatives and reports the full
ambiguity count, not merely the retained provider subset.

A new brew with no selected bean can use the sole open bag when no known bean
name or explicitly named roaster points elsewhere. That default never replaces
an existing bean on an unrelated follow-up. The provider is instructed not to
substitute it for an explicitly unavailable or newly named coffee. A bean can
also match an exactly named roaster if it is the single open bag from that
roaster across the full retrieved library. Legacy missing `closed` remains open;
malformed status is unknown, not open. Closed and unknown-status bags are never
open defaults; an explicitly unique full name retains its existing authority.

Provider-only `request_match`, `request_matches` and `sole_open_default` hints
are computed by the server before projection. Neither model confidence,
assistant prose nor a single suggestion proves a match. Unsupported or foreign
references are still rejected. This is conservative token matching, not an
unrestricted semantic or alias search; insufficient evidence remains reviewable.
Saved recipes also accept distinctive, uniquely matching name shorthand. For
example, “fast 60” can identify “Fast V60”; after a recipe-name word anchors the
match, the numeric equipment shorthand can distinguish it from “Fast AeroPress”.
A bare equipment name or number cannot authorize recipe preparation. An explicit
pour-count phrase (for example “3 pour Orea recipe” or “three-pour Orea”) can
distinguish sources using their declared name count, or stored pours when the name
has no count. Bloom counts, doses and equipment model numbers are not pour-count
clues. Contradicted recipe proposals and their dependent preparation/brewer are
removed before promotion, preserving independent reference choices and validating
the complete provider envelope first. A fully named recipe can still be adapted
rather than being rejected solely because its stored count differs. Competing
recipe-name matches are counted before the 20-recipe provider cap and remain
reviewable; two matching three-pour recipes still require disambiguation. A wholly grounded recipe proposal uses normal owned-source
enrichment and retains recipe/preparation atomicity.
Existing main/current selections and manual preparation are preserved;
contradictory candidate facts remain reviewable rather than being merged in
candidate order. Unresolved choices keep their complete validated patches for
revision guards. Already-selected references are context, not fresh ambiguity.
Public response fields and Save-only write authority are unchanged.

A saved recipe selection brings validated stored amounts, pours and its linked
own brewer as one proposal. PDS recipe doses/water are decoded from persisted
tenths into existing public API units; pour water and relative start times are
not rescaled or reinterpreted as bloom duration. Fractional recipe amounts remain
faithful context but are not silently rounded into the integer brew fields.
Recipe instructions can supply explicit preparation facts; absent facts still
require clarification. Unchanged recipe references do not reset manual values.
An ambiguous recipe and its preparation fields remain behind the same choice,
even if the provider proposed those fields separately. Ordinary provider choices
remain reference-only. A choice selecting a validated own recipe may also carry
allowlisted preparation facts from saved instructions or explicit user text;
ratings, tasting results and unrelated edits remain disallowed in choices. The
server additionally attaches validated structured recipe defaults.

The native editor exposes own recipes and their stored details. Its legacy data
view does not prove ownership of a recipe's linked brewer, so manual recipe
selection preserves the current brewer rather than trusting an rkey collision;
copying pours requires known compatible categories. The server-side chat catalog
joins full authoritative own AT-URIs before attaching a brewer. Existing manual
record routes are unchanged.
The client renders text as text, not executable HTML.

A provisional empty draft may acquire a different kind. A nonempty draft cannot
silently change kind. Unrelated requests should receive a short redirect to
recording coffee, not general-purpose assistant service. Prompt restrictions
alone are not a guarantee against unrelated text; bounded operations, request
limits and a no-write server boundary constrain actual capabilities.

## Client reconciliation and persistence

Each request captures a local session ID, request ID and revision. Field-level
revisions prevent a response overwriting newer edits. Destructive dependency
changes require confirmation of the current revision. Stopped, discarded and
saved drafts reject late responses. Only Save invokes a record mutation.

Conversation/draft persistence is local to the browser and scoped to account
DID, not a server-side chat archive. Reload does not resume pending requests;
active chats ending in an unanswered user message offer Retry without duplicating
that message. Browser tabs serialize envelope mutations with Web Locks and compare
the complete expected stored session, not just its field revision. Stale writes
cannot replace newer work or terminal sessions. External storage changes invalidate
pending AI work before the latest stored state is shown.
Draft storage is schema-versioned and capped at 256 KiB/chat and 2 MiB/account,
with up to 20 separately switchable chats; quota/corruption/limit failures must
be visible without deleting a live unsaved draft. No cross-device synchronization
is provided. A newer manual recipe selection supersedes all older preparation
proposals; stale recipe choices require a current-revision confirmation.

## Request bounds

- HTTP body: 6 MiB (6291456 bytes); composed text/library context: 196608 bytes,
  checked separately without image bytes.
- At most four images across the entire request conversation; each at most 1 MiB
  decoded, longest edge 2048 pixels and at most 2048×2048 pixels. Actual MIME,
  dimensions and full decodability are validated before forwarding.
- Provider request default: 6 MiB; configurable up to 8 MiB. A lower explicit
  `max_request_bytes` is respected and returns HTTP 413 with
  `ai_provider_request_too_large`; remove images or increase the operator limit.
- Up to 40 messages, each up to 4000 characters; full accepted conversation kept.
- Up to 100 each of beans, roasters, brewers and grinders, plus 20 saved recipes
  in provider context, retaining selected references and preferring relevant/open
  matches. Recipe instructions are capped at 2000 bytes and pours at 100.
  This is a subset, not proof other records do not exist. Current store reads
  still materialize each collection before projection; roaster/brewer identity
  indexes support authoritative joins. Only the provider projection is bounded.
- Provider content: 32768 bytes; assistant reply: 2000 characters; up to 8 choices.
- At most 8 active requests globally, 1/account, 10 admissions/minute/account.
  Rate-state map capped at 1024 accounts; expired inactive entries are pruned.
- Whole operation deadline: 45 seconds, including library reads. Provider timeout
  remains separately configured (30 seconds by default). No queues or retries.

## Save outcomes

A local in-flight registry preserves saves on route/account return. Web Locks
coordinate independently loaded tabs: canonical-session validation, attempt-marker
creation and POST initiation share the account lock, and a per-chat lock remains
held until completion. Before POST, the complete stored session must match the
submitted active draft with no pending AI request or previous save marker. Missing,
changed, terminal or unverifiable sessions cannot authorize a write. Without usable
device storage or Web Locks support, live chat can continue but Save fails safely.

A hard reload, network failure, or HTTP 5xx may leave an unknown record-write
outcome. Save remains blocked until the user checks My Coffee and explicitly
confirms the record was not created. Known 4xx rejection permits retry. A known
successful write whose local persistence fails shows its saved-record link; it
must not offer a "not saved" retry. Successful attempt markers remain as local
tombstones even after deleting the chat, preventing reappearing stale snapshots
from authorizing duplicate writes. Explicit unknown-outcome recovery cannot clear
an in-flight save in another tab or a known successful outcome. These protections
are browser-local, not server idempotency; clearing site storage removes them.
Late completion must not overwrite any newer stored session state.

## Failure behavior

Errors use the standard `{error, code}` envelope with an appropriate HTTP status.
Authentication, unavailable configuration, invalid/oversized requests, rate or
concurrency limits, unavailable library reads, cancellation and provider failures
must leave the client draft intact. The chat server performs no mutation in any case.

| HTTP | Code |
|---|---|
| 401 | `authentication_required` |
| 408 | `request_cancelled` (caller cancellation or request deadline) |
| 503 | `ai_disabled` |
| 415 | `invalid_content_type` |
| 413 | `request_too_large` |
| 400 | `invalid_request` |
| 503 | `ai_library_unavailable` |
| 429 | `ai_rate_limited` (`Retry-After: 60`) |
| 502 | `ai_provider_error` |


## Provider diagnostics

Caller cancellation or an expired incoming request returns 408 where the
connection is still writable and does not emit a provider-failure error log. An
upstream provider timeout while the incoming request is live remains a 502.

A 502 also emits an `AI chat provider failure` server log event correlated by
`trace_id`. The public API error remains generic. Logs include only fixed
`stage`/`reason`/`provider_error` categories, numeric `upstream_status`, allowlisted
`finish_reason`, response byte/tool-call counts, and an allowlisted invalid field
name when applicable. They never include credentials, URLs, request headers,
prompts, library values, raw model output or upstream error bodies.

Examples:

- `stage=request provider_error=upstream_http upstream_status=401`: upstream
  authentication rejection; 429 indicates upstream rate limiting.
- `provider_error=tls_protocol_mismatch`: HTTPS received plaintext HTTP; check
  configured base URL and HTTP(S) proxy schemes. TLS certificate, DNS, refused
  connections, timeout and cancellation have distinct categories.
- `stage=response reason=incomplete_completion finish_reason=length`: the
  completion hit its token budget rather than returning a finished draft.
- `reason=invalid_json_envelope`, `missing_choices`, `missing_in_scope` or
  `invalid_draft_fields`: transport succeeded but the draft contract was rejected.

Restart the Go server after code/config changes before reproducing a failure.


## Browser-local chat history

Chat history is not a new HTTP API. The browser keeps up to 20 conversations per
account DID, with a 256 KiB per-chat cap and 2 MiB account cap, in a versioned
localStorage envelope. Schema 2 adds a selected-session ID and bounded collection;
schema 1 single-chat records load compatibly, with no mutation during reads.
New chat preserves the previous draft; explicit deletion removes only the selected
local conversation, not PDS records. The desktop sidebar/mobile Chats control
switches conversations, retaining manual edits and unsent input. An older Save
completing after a switch or route remount can update only its originating chat,
never select it or overwrite another draft. Corrupt/quota failures retain existing
storage, and storage-full creation requires explicitly deleting a chat first.
