# Onboarding Flow — Get Started Card

**Status:** Design approved 2026-05-17
**Related:** `docs/road-to-v1.md` — "Make it so 'missing roaster' warning can be dismissed" item

## Motivation

Today, `/brews/new` lets users create beans, brewers, grinders, and roasters
inline via the combo-select "+ Create new" affordance. This conflates two
distinct flows (logging a brew vs. seeding the user's gear/pantry) and pushes
all setup into the brew form. The road-to-v1 doc calls for ripping out the
inline creation and replacing it with a dedicated onboarding surface that
warns or blocks brew creation until the user has the minimum prerequisites.

## Goals

- Make a new user's first 30 seconds clear: see what they need, add it, log a
  brew.
- Stop using the brew form as a back-door entity-creation surface.
- Keep beans-without-roaster legal (e.g. "random blend of 4 beans").
- Reuse the existing entity-creation modals rather than building a new wizard.

## Non-Goals

- A multi-step wizard with forward/back navigation.
- Persisting "user dismissed onboarding" state.
- Tutorial coachmarks or guided product tours beyond the static card.
- Onboarding for recipes, cafes, drinks (only brew prereqs are in scope).
- Touching `/my-coffee` empty states (deferred).

## Prereq Rule

A user is **ready to brew** when:

- They own **≥1 brewer**, AND
- They own **≥1 bean**.

Grinder and roaster never gate anything. Roaster on a bean stays optional so
the "random blend" case keeps working.

Readiness is derived from PDS state on every render. There is no
"onboarding completed" flag. Deleting all your brewers puts you back into
onboarding — this falls out of the derived model and is correct.

## Surfaces

### Home page (`/`)

- New `GetStartedCard` component rendered at the **top** of `HomeContent`,
  above `WelcomeCard`. Only renders when the authed user is not ready.
- `WelcomeCard`'s "Log Brew" action button becomes a visually-disabled link
  reading **"Log Brew — finish setup ↑"** and anchors to `#get-started`
  instead of `/brews/new`. Other action grid buttons unchanged.

### `/brews/new`

- Server handler checks readiness. If not ready:
  `http.Redirect(w, r, "/#get-started", http.StatusFound)`.
- The brew form template itself has no readiness branching.

### Brew form combo-selects

- `ComboSelectConfig` gains an `AllowCreate bool` field (default true).
- Brew form passes `AllowCreate: false` for bean, brewer, grinder, and
  roaster combo-selects.
- `static/js/combo-select.js` honors the flag and hides the "+ Create new"
  dropdown item when false.
- All other combo-select usages (bean modal's roaster picker, etc.) keep the
  current behavior — only the brew form is locked down.

## `GetStartedCard` Layout

One card containing four stacked sub-sections, in this order:

1. **Brewers** *(required)*
2. **Beans** *(required)*
3. **Roasters** *(optional)*
4. **Grinders** *(optional)*

Rationale: required-first so the user sees what unblocks them. Roaster comes
before grinder because beans reference roasters; creating a roaster before
the bean lets the user pick it from the bean modal's combo-select. Grinder
is independent of everything else and goes last.

Each sub-section renders:

- Heading + `(required)` or `(optional)` badge. Required uses the warm primary
  accent; optional uses muted/secondary styling.
- List of the user's existing items of that type (label only — name).
  Empty list shows muted text "None yet."
- `[+ Add a <entity>]` button that opens the existing dialog modal via the
  same `hx-get` → `#modal-container` pattern the dashboard already uses.

**Card footer (always present):**

- Status line: `✓ brewer  ✗ bean — add 1 of each to start logging brews`
  (checks reflect current readiness flags).
- When ready, the line becomes **"Ready to brew!"** and a prominent
  `[Log your first brew →]` button appears linking to `/brews/new`.
- **Not dismissible.** When prereqs are missing, the homepage "Log Brew"
  CTA is broken anyway; hiding the card would only confuse.

## Data Flow

### Readiness helper

New method on `AtprotoStore` (`internal/atproto/store.go` or a thin sibling
file):

```go
type ReadinessStatus struct {
    HasBrewer bool
    HasBean   bool
}

func (s ReadinessStatus) Ready() bool { return s.HasBrewer && s.HasBean }

func (s *AtprotoStore) BrewReadiness(ctx context.Context) (ReadinessStatus, error)
```

Implementation: query witness cache first (`ListBrewers` / `ListBeans` with
`limit=1`), fall back to PDS only when witness cache reports empty and
session cache has neither collection populated. We only need existence, not
contents — keep it cheap.

### Call sites

1. **`HomeHandler`** populates a new `HomeProps.Readiness ReadinessStatus`
   field. Template uses it to decide whether to render `GetStartedCard` and
   whether to disable the Log Brew action.
2. **`BrewNewHandler`** (`GET /brews/new`) checks `Ready()` first; redirects
   to `/#get-started` with 302 if false.

### HTMX refresh wiring

- Existing entity-create modals already fire a `refreshManage` HX-Trigger
  event on success (used today by `#incomplete-records-section`).
- `GetStartedCard` listens on the same event:
  `hx-trigger="refreshManage from:body"`,
  `hx-get="/api/get-started-card"`,
  `hx-swap="outerHTML"`.
- New endpoint `GET /api/get-started-card` re-runs readiness and returns
  the partial. When readiness transitions to `Ready()`, the partial includes
  the "Log your first brew →" button. (The page does not auto-navigate; the
  user clicks through.)

### Per-app variants (arabica vs oolong)

- `GetStartedCard` accepts an `AppName string` field, mirroring the existing
  `welcomeAuthenticatedArabica` / `welcomeAuthenticatedOolong` split.
- Arabica: brewer, bean, roaster, grinder labels as above.
- Oolong: equivalent entities (steeper, tea, vendor, etc. — exact mapping
  determined by the existing oolong entity model at implementation time).
- **Arabica ships in the first commit; oolong variant lands as a follow-up
  commit.**

## File-Level Changes

This is a guide for the implementation plan, not an exhaustive list.

- `internal/atproto/store.go` (or new `readiness.go`) — `BrewReadiness` method
  and `ReadinessStatus` type.
- `internal/handlers/` — home handler populates readiness; new
  `/api/get-started-card` handler; brew-new handler adds redirect.
- `internal/routing/routing.go` — register `/api/get-started-card`.
- `internal/web/pages/home.templ` — render `GetStartedCard` above
  `WelcomeCard` when not ready; pass readiness through `HomeProps`.
- `internal/web/components/shared.templ` — update `welcomeAuthenticatedArabica`
  (and oolong variant in follow-up) so the Log Brew button reflects disabled
  state when not ready.
- `internal/web/components/get_started_card.templ` — new component.
- `internal/web/components/combo_select.templ` — add `AllowCreate` field to
  `ComboSelectConfig`; thread through to the petite-vue directive data.
- `internal/web/assets/js/combo-select.js` — honor `allowCreate` flag.
- Brew form templ (wherever combo-selects are wired for brew creation) —
  pass `AllowCreate: false` for bean/brewer/grinder/roaster.

## Testing

Per project conventions: testify/assert, no `if t.Error()`.

- **Unit (`atproto` package):** `BrewReadiness` returns correct flags across
  four cases: none, brewer-only, bean-only, both. Use existing test store
  fixtures.
- **Handler:** `BrewNewHandler` returns 302 to `/#get-started` when not ready,
  renders the form when ready.
- **Handler:** `HomeHandler` includes `GetStartedCard` in the response when
  not ready, omits it when ready.
- **Templ smoke:** `GetStartedCard` renders correct required/optional badges
  and arabica labels.
- **Manual end-to-end:** fresh account → home shows card with Log Brew
  disabled → add brewer via modal → card refreshes via HTMX → add bean →
  "Ready to brew!" appears → click through to `/brews/new` succeeds.

## Rollout

- **Commit 1:** arabica implementation — readiness helper, home card,
  /brews/new redirect, combo-select `AllowCreate` plumbing, brew form
  lockdown, tests.
- **Commit 2:** oolong variant of `GetStartedCard` and corresponding home /
  brew-new wiring for the oolong app.

## Open Questions

None at design freeze. Decisions to revisit if implementation surfaces them:

- Exact oolong entity mapping (resolved during commit 2).
- Whether `AllowCreate: false` should be the new default for combo-select
  (currently keeping default `true` to minimize ripple).
