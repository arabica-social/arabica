# AI decisions via a multimodal decision model

Status: proposed future option, not started. Current behavior stays as documented
in [FDR-007](../fdr/FDR-007-ai-assisted-creation.md) and
[AI chat API](../api/ai-chat.md); it records no current behavior change.

## Context

The assistant currently calls one OpenAI-compatible chat-completion provider with
a long system policy and reads a JSON envelope back (`in_scope`, `reply`, `kind`,
`patch`, `choices`). The server validates every field and reference, and the
provider is the only source of proposal text.

An alternative shape exists: a **multimodal decision model**. Instead of asking a
model to write a JSON object, the server sends a `state` plus a schema of typed
questions and receives a probability for every option it offered:
`noul` (yes/no), `choice` (2–255 named options) and `score` (2–10 ordered levels,
lowest first). Up to 64 questions per call. Answers come back keyed by question
id; there is no free-form model text in the response.

Cloudflare's Clef family is the concrete example (announced 2026-10-01):

| | `clef` | `clef-flash` |
|---|---|---|
| Parameters | 27B, post-trained from Qwen3.8-27B | 9B |
| Context | 65,536 tokens | 24,576 tokens |
| Billing | $0.24 per M input tokens, no output tokens | $0.038 per M input tokens |
| Inputs | text, JSON `state`, up to 4 images (PNG/JPEG/WebP, data URL or `{content_type, base64}`, no remote URLs) | same |
| Weights | Apache-2.0, `Cloudflare/clef` on Hugging Face | `Cloudflare/clef-flash` |

Nothing in this option requires Clef specifically; the decision contract is the
proposal, and any provider that answers typed questions with probabilities fits.

## What would change

- `internal/ai` gains a decision transport beside the existing chat transport (or
  the chat transport is replaced), selected by operator configuration.
- The response contract becomes typed decisions plus server-authored copy. Field
  rules and reference authority stay exactly where they are today: the server owns
  allowed fields, library IDs and validation; the decision model only picks among
  options the server supplied.
- `reply` becomes a template rendered from decisions (`ask.bean`, `matched.bean`,
  `refuse.out_of_scope`) instead of model prose. Templates are new work: roughly
  one string per decision outcome, and they must read naturally.
- Library projection maps naturally onto `choice` questions, since the provider
  already receives opaque IDs (`bean-1`, `brewer-2`) rather than record keys.
- Images already validate to a shape Clef accepts (at most four, data URLs,
  PNG/JPEG/WebP, bounded pixels), so the image path is close to unchanged.

## Draft mapping sketch

| Current envelope | Decision contract |
|---|---|
| `in_scope` | `noul`: is the latest turn a request to record or refine a brew, bean, grinder or brewer? |
| `kind` | `choice`: brew, bean, grinder, brewer, or none |
| `bean_rkey` / `brewer_rkey` / `grinder_rkey` / `roaster_rkey` / `recipe_rkey` | `choice` over the supplied own-library IDs plus an explicit "none of these" |
| equipment plausibility, sole-open-bag default, preparation consistency | `noul` questions over the same evidence the prompt sees today |
| confidence thresholds | server-side probability cutoffs, recorded per decision for tuning |

## Open questions

- **Typed values have no question type.** `choice` covers enums and bounded option
  sets, and `score` returns a rubric level, so free-form numbers (dose, water,
  time, temperature) and open text (`name`, `origin`, `tasting_notes`) do not map
  directly. Either quantize them into option sets with follow-up questions, or keep
  a small extraction step for those fields only. This is the decision that
  determines whether the chat provider can be removed at all.
- **Copy quality.** Templated replies lose the current "asks a natural clarifying
  question" texture. Worth prototyping on the brew flow before committing.
- **Question and option budget.** 64 questions × 255 options per call is generous
  but not free; library projection may need to shrink from today's 100 per kind.
- **Hosted or self-hosted.** Hosted keeps operations simple and billing
  input-only; self-hosting the 9B keeps library and image context inside the
  deployment. Clef's weights being Apache-2.0 makes either path possible.
- **Provider maturity.** The family is new, and its published context limits are
  inconsistent between the changelog and the per-model reference. Recheck before
  committing.
- **Image quality.** Bag-label OCR and label-derived facts need the same evaluation
  as text decisions.

## Why consider it

- Response probability per fixed option is something the server can validate
  exactly, so the validated surface is smaller than free-form JSON plus a repair
  path.
- Cost is deterministic and input-only, which suits the existing daily character
  allowance and makes per-request cost predictable.
- One contract covers text and images.
- Open weights leave a self-hosting option open without a contract change.

## Evaluation before adoption

1. Add the decision seam without wiring it to a route.
2. Build a labelled set from the existing service fixtures and real session
   utterances: expected scope, kind, references and refusals.
3. Prototype one flow end to end (brew, including one image case) behind
   configuration.
4. Compare decision accuracy, template readability and p50/p95 latency against the
   current provider before changing the default.

## Non-goals

- No change to PDS schemas, mutation APIs, save authority or record validation.
- No new write path: the Save button remains the only write.
- No UI redesign in this document; the chat surface keeps its current layout.
