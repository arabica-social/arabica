# FDR-007: AI-assisted record creation

**Status:** Experimental, disabled unless configured
**Last reviewed:** 2026-10-31

## Overview

A task-specific conversation turns descriptions into editable brew, bean, grinder
or brewer drafts. It is an optional alternative entry path, not a replacement for
manual forms or a general-purpose assistant. Only the Save button writes records.

## Behavior

- AI chat is limited to DIDs granted access by a moderator; admins and moderators
  have access automatically. The Moderator page manages explicit grants in a
  deployment-local database table. Revoking an explicit grant cannot remove staff
  access. Denied users see no homepage chat composer and receive 404 for direct
  chat page loads and chat submissions. Page denials use the normal app error
  screen and navigation with HTTP 404, not an access-specific message. Manual
  logging remains available. Approved users with AI disabled see a horizontally centered,
  themed unavailable card matching the missing-page typography and top spacing, with Add manually
  and Back to Home actions.
- A configured, authorized Home Page offers one compact textarea, "Describe
  a brew or bean", with the placeholder "Describe a brew or bean..." and an inset
  Send button. No heading, example suggestions,
  explanatory panel or decorative labels surround this input. Community title,
  manual logging/setup action, navigation and feed filters remain available.
- Enter sends from either composer; Shift+Enter inserts a newline. IME
  composition never submits a partial message.
- Sending opens /chat immediately with visible Opening chat… feedback; duplicate
  submissions are disabled during navigation. Home → Chat progressively enhances
  with a short shared-composer view transition, skipped for reduced motion or
  unsupported browsers. Navigation failure retains the original input.
  Conversation and draft remain on /chat. Ordinary manual forms remain available,
  including when AI is disabled or fails.
- While a request is actually pending, show a visible Working on your draft…
  status and animated spinner beside the conversation; reduced motion keeps a
  static indicator. Do not simulate progress or claim unobserved processing stages.
  Stop, failure and completion remove it. Errors preserve the draft and offer Retry.
  Reloading an interrupted or failed request also offers Retry for its unanswered
  user message; retrying does not append the message again.
- Chat supports JPEG, PNG and WebP attachments through a file picker or paste,
  with removable thumbnails and image-only messages. Mixed text/image paste keeps
  the text; ordinary text paste retains native editing behavior. Up to four images
  remain in memory across the current chat's turns and retries. The browser always
  reencodes images without source metadata, shrinking the longest edge to at most
  2048px and compressing to at most 1 MiB each. Source files above 20 MiB or
  40 megapixels are rejected before decoding. Failed preparation retains any
  existing unsent attachments; excess attachments are refused, never silently dropped.
- Images go inline through the authenticated chat API to the configured
  vision-capable provider, never to the PDS, a file store or browser storage.
  Provider retention is governed by the provider, not Arabica. Leaving/reloading,
  switching chats/accounts, starting a new chat or reconciling another tab's state
  releases image data, as does a successful Save. Only message image counts survive locally. Restored history
  shows unavailable images; retries of an image message require reattachment and
  a new message. Follow-ups explicitly mark missing historical images to the model.
  The homepage bar also accepts image paste, including image-only and mixed
  text/image paste, with removable previews but no extra upload button. Send
  transfers sanitized images to a fresh chat through a single-use account-scoped
  in-memory handoff, never sessionStorage. Failed navigation retains the homepage
  input and images; unused handoffs expire after 30 seconds. Leaving/reloading
  Home or switching accounts clears its unsent images and invalidates late decodes.
  Only Save writes records, never attachments.
  A conversation containing visual context does not authorize the sole-open-bag
  default, including on later text-only follow-ups: visible coffee names
  are not lexical proof, so proposed library references remain reviewable unless
  grounded in explicit user text.
- The server grounds references against the viewer's own library. Uncertain
  matches appear as clickable record choices; choosing one changes the draft,
  never creates/imports a record. Hover and keyboard focus visibly highlight a
  choice. Grounded context is not presented as another alternative: complete no-op
  choices are omitted, and combined labels name only unresolved references while
  preserving their full guarded patches. Uniquely named references can fill the
  draft even when the provider bundled them into a choice. Known beans and
  grinders do not require a redundant click. Grounded recipes use owned source
  defaults; genuinely unresolved references or contradictory preparation remain
  explicit choices with complete revision-guarded patches. Assistant prose and a
  lone suggestion alone grant no selection authority. Choosing a match
  dismisses its resolved alternatives, not unrelated ambiguity groups.
  No arbitrary web search or tool execution.
- Prefer a clear match over another selection step. Meaningful short names and
  minor long-word typos can fill beans, equipment and roasters directly, including
  references bundled into a model choice. For example, “Mango” can match the open
  “El Mango Natural” bag, and “ZP6” can match “1ZPresso ZP6”. Name qualifiers can
  distinguish competing bags; genuinely competing best matches remain choices.
- Match beans through their actual roaster relationship and open/closed status,
  not just names. Prefer a matching open bag over repeated closed history. A
  single open bag from an explicitly named roaster can fill automatically. A new
  brew without a selected bean can also use its sole open bag when no known
  coffee or explicitly named roaster points elsewhere; this default cannot
  replace a selected bean on an unrelated follow-up. Do not substitute it for a
  newly named or unavailable coffee. Ambiguity is checked across the full
  retrieved library, not the bounded provider subset; show up to eight named
  alternatives with the actual count. Legacy missing closed status is open;
  malformed status is unknown. Closed/unknown bags are never open defaults.
- Read the user's saved recipes: actual dose, water, ordered pours, linked brewer
  and saved instructions. Use known details rather than asking the user to
  retype them; ask only for missing or unrepresentable values. Unique exact or
  distinctive shorthand recipe names can fill a draft, including its authoritative
  own linked brewer and stored preparation. For example, “fast 60” can identify
  “Fast V60” without another confirmation button. Bare equipment names or numbers
  cannot choose a recipe on their own. “3 pour Orea recipe” distinguishes a saved
  three-pour recipe from a four-pour recipe, including names such as “3-Pour
  (1 bloom) Orea 01”. Use declared name counts, or actual stored pours when the
  name has no count; do not confuse bloom counts, doses or brewer model numbers
  with pour counts. Do not offer contradicted alternatives or require another
  click for that clear match. Two matching three-pour recipes remain genuine
  ambiguity, with their preparation details behind an explicit choice. Never invent a recipe from its name or
  silently round fractional values to fit the current integer brew API.
  Recipe notes remain grounding context, not brew tasting notes or an instruction
  panel in chat. The standard non-chat editor retains its own-recipe selector
  and source details; manual selection keeps the current brewer when its source
  ownership cannot be proven by the API. Chat changes recipes conversationally.
  No recipe creation, cross-user search/import, new PDS shape or unit migration.
- On Community, show the chat prompt above the masthead separator, with feed
  filters and sorting below it. Keep the non-chat layout unchanged when AI entry
  is unavailable.
- Show a useful draft rather than an initially empty form: bean name+origin;
  equipment name; brew bean+brewer or one reference plus two substantive
  preparation/result fields. The conversation remains the primary editing path.
  Useful drafts appear as read-only cards with actual record names, supplied
  measurements, results and ordered relative pour times, plus Save. Derived ratios
  are display-only; placeholders and missing values never become preview facts.
  Recipe instructions remain distinct from tasting notes. Brew cards always show
  an editable Rating (1–10) beside Save, outside Edit details: blank is optional,
  no rating is invented, and clearing removes it. There is no Add rating step or
  duplicate rating editor. Rating edits use the same local draft/history and
  guarded edit path; only Save writes the chosen score. A saved draft retains its
  preview and record link without editable controls.
- Edit details is an optional, initially collapsed native disclosure. Add details
  manually supports completion before a useful draft exists. Once revealed, the
  draft card stays available even if fields are cleared. Disclosure state resets
  between chats; opening, closing and changing views never edit the draft.
  Record type and Undo live under Chat options rather than above every draft.
- Inputs remain real controlled fields. Edit details is a flat correction form
  for bean/brewer/grinder, dose, water or espresso yield, grind setting, temperature,
  category-appropriate time and tasting notes. It has no nested More brewing
  details, Quick/Detailed modes, recipe selector/instruction panel or pour editor.
  Existing advanced fields and pours remain draft facts, visible in the preview
  and unchanged by basic edits or Save; use conversation for advanced changes.
  The standard non-chat editor remains complete. Ghost text is examples, not
  default values; the always-visible optional Rating starts blank, never guessed.
  Method is not an editor input; legacy draft values remain compatible. Missing
  optional values stay absent. Existing units and required-field policies remain.
  Save can be submitted without opening the editor. Invalid hidden controls open
  their ancestor disclosures before native validation focuses the relevant field;
  a blocked submission performs no write.
- Direct edits after a request starts win over that request's proposals. Later
  explicit chat instructions see the current draft and can revise those values.
  Grinder/category changes that clear entered details require confirmation.
  Stale match choices require renewed confirmation before replacing newer picks.
- Undo last AI change does not erase subsequent manual edits. Stop invalidates a
  request before aborting it, so a late response cannot revise the draft.
- Save is unavailable while an AI update is pending. Saving freezes the submitted
  draft; success shows a record link, failure keeps the draft. Unknown save
  outcomes (network failure, 5xx or reload during a write) require checking My
  Coffee before another submission. Live chat works without storage, but a new
  save requires writable device storage and browser Web Locks support. Tabs
  coordinate save admission against the complete stored draft before any write;
  an already saved, missing or changed draft cannot be submitted from a stale tab.
  Successful save markers remain on this device, including after chat deletion,
  so stale snapshots cannot authorize another write. These are browser-local
  safeguards, not server idempotency; clearing site storage removes them.
  Saying "save it" is not an alternative write authority in this iteration.
- Chat has no persistent history sidebar or mobile Chats panel. For now, existing
  stored drafts can be accessed through Chat options → Previous drafts; the
  long-term history/resume interaction remains deferred. Conversations remain
  separate, never inline with each other. New chat preserves the old
  conversation, draft and unresolved-save markers; Delete explicitly removes one
  local chat, never its saved coffee records. Switching stops the old AI request.
  Drafts, the selected chat and unsent composer text survive reload, scoped to the
  account DID. Tabs serialize storage updates and reject stale snapshots, including
  same-revision changes to messages or unsent input. When another tab changes or
  deletes the current chat, invalidate pending AI work and show the latest stored
  state with a notice rather than overwrite it. Existing single-chat storage loads
  without losing its contents.
  Keep up to 20 chats, 256 KiB per chat and 2 MiB per account; refuse overflow
  rather than silently evicting history. No server archive or cross-device sync.
  Live-only chats require a destructive confirmation before starting over.
- Chat uses the viewport below the site header, with one scrollable lane for the
  conversation and editable draft and a composer anchored above mobile navigation.
  There is no arbitrary-height transcript nested above a separate form. Controls
  and thin scrollbars follow the active theme; keyboard users can focus the lane
  and scroll it without moving the page. A compact action bar sits above one
  centered reading column, without a sidebar gutter. The floating composer uses the exact same
  accessible small circular arrow button and theme-primary colors as Community.
  Its visible 28px disc retains a 44px touch target. Home and Chat inputs have no
  resize grip: one initial line grows for content and wrapping, reflows with width
  changes, and shrinks after clearing or restoring shorter text. Empty chats
  center the composer with Add details manually available, without decorative
  welcome copy. Draft arrival uses a short fade/settle animation; reduced motion
  disables both this animation and spinner movement.
- Bounded storage/turn limits surface errors instead of silently deleting an
  unsaved draft. Account changes clear old in-memory library and pending work.
- Each account has a daily character allowance for assistant requests: 10,000
  characters per UTC day by default, configurable with `ARABICA_AI_DAILY_CHARS`
  and disabled with `0`. Admins and explicitly listed DIDs
  (`ARABICA_AI_UNLIMITED_DIDS`) are exempt; moderators are not. A request is
  charged the message characters it sends, including re-sent history, plus 1000
  characters per image. Requests refused before the provider call are not
  charged. Exhaustion returns 429 `ai_daily_limit` with `Retry-After` to the next
  UTC midnight; the client keeps the draft and shows the server's message, so
  starting a new chat or waiting resets the practical cost without losing work.

## Decisions

### Separate proposal from authority

Models propose allowlisted fields. Application code validates references and
values, and existing mutation handlers remain authoritative for record writes.
There is no AI-specific PDS schema and no model-accessible save tool.

### Compose controlled editors rather than embed full-page forms

Full-page forms own private state, navigation and save lifecycles. Chat needs one
canonical draft shared with conversation requests. Small controlled editors use
existing native controls/theme patterns without taking ownership from that draft.
There is no new runtime UI dependency solely for message appearance.

### Bound cost per account, not just per minute

The admission limiter caps concurrency and request rate, but a granted account
could still spend indefinitely. Provider spend scales with the characters sent,
so the allowance meters characters rather than requests, is keyed by account and
UTC day in the deployment-local database, and exempts staff and an operator
list. Counting in the database keeps the bound across restarts and replicas; a
missing table refuses requests instead of serving them unmetered.

### Keep deployment optional and provider-neutral

Server-side JSON selects an OpenAI-compatible chat-completions provider and model;
keys are server-only strings, optionally using `${VAR}` environment interpolation. The initial app requests a constrained
JSON response without requiring provider tool-calling or strict-schema support.
Provider transport is app-agnostic; coffee fields/policy stay app-owned.

### Minimize context without inventing privacy restrictions

Public records may be sent to the configured provider. Send compact relevant
library data and the bounded conversation/current draft, not automatic usernames,
account profile or whole brew history. Preserve user-authored descriptions rather
than silently redacting their meaning. Provider grounding is capped; do not claim
an omitted library record does not exist. Prompt policy limits intended use;
validation, caps and read-only capabilities enforce the operational boundary.

## Deferred

History search, streaming, conversational save confirmation,
recipe/roaster drafts, URL ingestion, and editing/deleting saved records.

## Related

- [API and configuration](../api/ai-chat.md)
- [Implementation plan and prototype references](../plans/2026-10-31-ai-chat.md)
- [Home](FDR-005-home.md), [Brew logging](FDR-002-brew-logging.md),
  [Smart Autofill](FDR-004-smart-autofill.md)
