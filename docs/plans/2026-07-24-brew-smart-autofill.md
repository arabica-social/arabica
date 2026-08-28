# Plan: Smart Autofill for New Brews

## Context

Users repeatedly brew the same bean (often with the same recipe). The brew
form makes them re-enter grinder, grind size, temperature, timing, filter, and
other details that are nearly always the same across those brews. Smart
Autofill derives suggestions from the authenticated user's own brew history
when they select a bean (and optionally a recipe) on the new-brew form.

Autofill is on by default, toggled in Settings ("Brewing preferences"),
and only fills fields the user (or an applied recipe) has left empty. Every
autofilled value is visibly marked in the UI with its confidence, e.g.
"Autofilled · in 12 of 15 brews (80%)", and a "Clear autofill" button on the
form removes everything autofill contributed.

The suggestion derivation is a read-only, client-side pass over existing
records: no lexicon or record changes, so the PDS stays authoritative and no
compatibility surface is touched. The only backend change is one new
operational-store user preference (not a PDS record).

## Key Decisions

### 1. Compute suggestions client-side

**Decision:** Derive suggestions in the SPA from the `/api/data` payload the
brew form already loads via `appCache` for EntityCombo. No new endpoint.

**Why:** `/api/data` already returns the user's complete brew list with
hydrated bean/grinder/brewer refs and `recipe_owner_did`, cached for 5
minutes. Suggestions are instant and recompute synchronously as the user
changes selections. A server endpoint would add API surface and a round trip
per selection for no benefit at current data sizes.

**Tradeoff:** Suggestion logic lives in the frontend. If history sizes grow
large or cross-device behavior needs to be shared with another app, the pure
computation can move behind a `GET /api/brews/autofill` endpoint later.

### 2. Recency-biased value selection, honest support label

**Decision:** Score each candidate value with an exponential recency decay
over matching brews: the brew at recency rank `r` (newest = 0) contributes
weight `0.5^(r / H)` with a half-life `H` of 5 brews. Suggest the
highest-scoring value per field; ties break toward the value in the newest
brew. The badge still reports raw counts: "in 12 of 15 brews (80%)".

**Why:** A plain mode encodes "what you usually do", but habits drift. Pure
most-recent encodes only the last brew, including one-off experiments.
Decayed weights favor the recent habit: a run of the same setting in the
last few brews overtakes an older majority, while a single recent deviation
does not. Raw counts keep the label truthful and easy to sanity-check.

**Tradeoff:** The chosen value and its displayed percentage can disagree (a
value shown at 40% can beat one at 60% because it is newer). `H` is a
constant for v1; making it user-tunable is a follow-up.

### 3. Autofill never overwrites anything

**Decision:** Autofill fills only fields that are empty, and only after
recipe apply has run. Editing an autofilled field removes its badge.

**Why:** Explicit intent always wins: user-typed values, then recipe-applied
values, then autofill. This preserves the FDR-002 rule that a recipe fills the
brew, not the other way around.

**Tradeoff:** Changing the bean mid-form will not refresh fields the user
already edited; only still-empty fields update.

### 4. Matching key: bean, refined by recipe

**Decision:** Match history brews on bean rkey (same owner as the selected
bean). When a recipe is also selected, first try the (bean rkey, recipe
rkey + recipe owner DID) pair; if fewer than 2 brews match, fall back to
bean-only matching. The UI states which basis was used.

**Why:** The pair is the strongest signal, but a strict pair requirement
makes the feature useless for new bean/recipe combinations. Owner DID matters
because recipes can be cross-user.

**Tradeoff:** Two tiers add a little logic and UI copy; keep it to two.

### 5. Default on, toggled in Settings

**Decision:** Smart autofill is enabled by default. The toggle lives in the
Settings page's "Brewing preferences" section ("travels with your DID"),
persisted server-side as a `smart_autofill` user preference alongside
`temperature_unit`. The brew form has no toggle; it exposes the "Clear
autofill" button instead (Decision 6).

**Why:** The feature must be optional, but the users who want it want it by
default. A Settings toggle keeps the form uncluttered and makes the
preference follow the user across devices, matching the existing pattern in
`profileprefs.UserPreferences`.

**Tradeoff:** Disabling autofill mid-form is one navigation away; the Clear
autofill button covers the common "don't fill this brew" case inline.

### 6. Clear autofill button on the form

**Decision:** While any field carries an autofilled badge, the form shows a
single "Clear autofill" action near the form header / autofill summary. It
empties every autofilled field the user has not since edited and removes all
badges. It does not submit and does not touch user-entered values.

**Why:** One obvious escape hatch beats per-field clear controls for the
"this suggestion is wrong, start over" case, and keeps field chrome minimal.

**Tradeoff:** A user who wants to clear exactly one field edits it directly
(editing already removes that field's badge).

## Suggested Fields

| Field | Notes |
|---|---|
| `grinder_rkey` | Label resolved from hydrated `grinder_obj`. |
| `brewer_rkey` | Only when no recipe applied (recipes set the brewer). |
| `method` | Only when empty. |
| `grind_size` | |
| `temperature` | |
| `time_seconds` | |
| `water_amount`, `coffee_amount` | Only when no recipe applied; recipes own dose/water. |
| `pourover_filter` | |
| `pourover` bloom/drawdown/bypass, `espresso` yield/pressure/pre-infusion | Only fields a recipe did not set. |

Never suggested: `tasting_notes`, `rating`, `pours`, `tasting`-style free
text. These are per-brew observations, not habits.

## Implementation

### Phase 1: Pure suggestion logic

**`web/src/lib/utils/brewAutofill.ts`** (new) — framework-free functions:

```ts
type AutofillSuggestion = {
  value: string;          // normalized form-usable value
  label?: string;         // display label for ref fields (grinder/brewer)
  rkey?: string;          // for ref fields
  matches: number;        // brews with this value
  total: number;          // matching brews overall
};
type AutofillResult = {
  basis: "bean+recipe" | "bean";
  totalBrews: number;
  fields: Partial<Record<AutofillField, AutofillSuggestion>>;
};
computeAutofill(brews: Brew[], key: { beanRKey: string; recipeRKey?: string; recipeOwnerDID?: string }): AutofillResult | null;
```

- Bucket matching brews by the tiered key from Decision 4.
- Sort matching brews newest first; for each suggested field, accumulate
  decayed weight per distinct non-empty value (Decision 2) and return the
  winner with raw counts for the badge.
- Group `pourover_params`/`espresso_params` sub-fields as individual fields.
- Skip everything when fewer than 2 matching brews.

**`web/src/lib/utils/brewAutofill.test.ts`** (new) — vitest unit tests:
pair vs bean fallback, recency weighting (a recent minority beats an older
majority; a single recent deviation does not), tie-breaking by recency,
percentage math, empty/1-brew guards, cross-user recipe owner matching,
ignore of never-suggested fields.

### Phase 2: Preference plumbing (Go + Settings UI)

- **`internal/profileprefs/profileprefs.go`** — add `SmartAutofill` to
  `UserPreferences` as a string enum (`"on"`/`"off"`, default `"on"`),
  following the `TemperatureUnit` pattern (`IsValid`, defaults in
  `WithDefaults`). Use a string, not a bool: JSON absence would decode as
  `false` and silently flip the default to off.
- **`internal/handlers/settings_json.go`** — parse `smart_autofill` in
  `HandleSettingsPreferencesJSON`.
- **`web/src/lib/types/api.ts`** — extend the `UserPreferences` type.
- **`web/src/routes/settings/+page.svelte`** — add a row in "Brewing
  preferences" (a checkbox labelled e.g. "Smart autofill brew details"),
  saved through the existing preferences form POST.
- **`web/src/lib/types/entity_view.ts`** — add `recipe_owner_did?: string` to
  `Brew` (the backend already serializes it).
- **`web/src/lib/stores/userPrefs.ts`** (new, small) — fetch `/api/settings`
  once per session and expose `smart_autofill`; default to on when the fetch
  fails or the key is absent. The brew form reads the preference from here
  instead of the form owning it.

### Phase 3: BrewForm integration

**`web/src/lib/components/BrewForm.svelte`**:

1. On mount, `appCache.getData()` already loads the full dataset; keep a typed
   reference to `data.brews`.
2. Read the `smart_autofill` preference from the `userPrefs` store (default
   on). No toggle is rendered on the form.
3. Recompute suggestions when `beanRKey` or `recipeRKeyValue`/`recipeOwner`
   changes (including after `applyRecipe` finishes, since recipe apply fills
   dose/water/brewer/bloom).
4. Apply pass: for each suggestion whose target field is still empty, set the
   field and record it in an `autofilled: Map<AutofillField, AutofillSuggestion>`
   state. Ref fields also set their label state so EntityCombo displays them.
5. User edits (input events) remove that field from `autofilled`.
6. "Clear autofill" button (Decision 6): rendered only while
   `autofilled.size > 0`; empties unedited autofilled fields, clears badges,
   and hides itself.
7. Edit form (`isEdit`) never autofills; the form is already populated.

**`web/src/lib/components/BrewFormField.svelte`** — accept an optional badge
prop (or use a wrapper): a small informational tag such as
`Autofilled · in 12 of 15 brews (80%)`, visually matching `StampTag.svelte`
patterns from DESIGN.md. Editing the field removes its badge; the Clear
autofill button lives at form level (Decision 6), not on the badge.

### Phase 4: Documentation

- **`docs/GLOSSARY.md`** — add a "Smart Autofill" entry naming the feature.
- **`docs/fdr/`** — new FDR (brew logging's sibling, e.g. FDR-004
  "Smart Autofill") documenting the behavior: optionality, precedence
  (user > recipe > autofill), matching tiers, and the confidence label.
  FDR-002 stays scoped to recipe copying; cross-reference the precedence rule.
- Update this plan or delete it once shipped (plans are not current-state docs).

### Phase 5: Verify

```bash
go test ./internal/profileprefs/... ./internal/handlers/...
pnpm run check:svelte
pnpm run test:svelte
```

Note: the `/api/settings` JSON response gains a `smart_autofill` key; update
the settings API docs (`docs/api/settings.md`) and any settings response
snapshot/contract tests alongside the handler change.

Manual checks:
- New brew with a bean brewed 3+ times: empty fields fill, badges show counts.
- Select a recipe on top: recipe values win; autofill fills only leftovers.
- Edit an autofilled field: badge disappears; value persists.
- Toggle off in Settings: new brews show no autofill; toggle persists across
  devices for the same DID.
- Clear autofill button: appears once autofill fills something, empties only
  unedited autofilled fields, hides after clearing.
- Recency: a setting used in the last ~5 brews is suggested over an older,
  more frequent setting; a one-off latest brew does not flip the suggestion.
- Bean with 1 or 0 prior brews: no autofill, no badges.
- Edit page: no autofill behavior.
- Cross-user recipe: history brews made with the same forked/foreign recipe
  match only when owner DID matches.

Optional later: a Playwright spec covering the badge lifecycle.

## Out of Scope (Follow-ups)

- Server-side suggestion endpoint (`GET /api/brews/autofill`) if history grows
  or another app wants the logic.
- Tunable recency half-life (v1 fixes H = 5 brews).
- Suggesting `pours` patterns.
- Autofill on the edit form for newly-empty fields.
- Any persisted "habit" records — this stays a derived read.
