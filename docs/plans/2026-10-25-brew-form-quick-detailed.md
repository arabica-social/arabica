# Plan: Quick and Detailed Brew Logging

Status: Implemented (2026-10-25). Current behavior is documented in FDR-002,
FDR-004, and the brew/settings API contracts.

Implementation notes:

- Missing-prerequisite creation opens in a separate tab. Returning to the brew
  tab and refreshing the collection retains the draft and recipe-link context.
- Long steeps retain integer seconds in PDS records. Both mutation transports
  now accept nonnegative durations beyond the former one-hour handler limit;
  the lexicon already permits these values.
- Known incompatible method labels are cleared on confirmed category changes;
  untouched legacy records and unknown free-text method descriptions remain
  editable.

Verification:

- `just ci-check`: passed (Go vet/build/unit tests, integration contracts,
  Svelte check, and frontend tests).
- `cd web && pnpm run check` and `pnpm run test`: the repository's current
  equivalents of the `check:svelte` / `test:svelte` commands below.
- `just _e2e-run tests/e2e/brew-form.spec.ts tests/e2e/create-brew.spec.ts`:
  focused Chromium coverage; the full browser suite was not run.

## Purpose

Make a short brew log feel complete while retaining preparation details for
people who want them. The current form presents recipes, equipment,
measurements, and method-specific controls before the user's observations.
Optional fields still require attention even when someone leaves them blank.

This plan follows the product discussion rather than treating `PRODUCT.md` as
constraints for the redesign. It does not remove or rewrite that document.

## Agreed Scope

- Offer Quick and Detailed modes on the same form.
- Remember the last explicitly selected mode for future new brews. Keep the
  default-selection policy easy to change.
- Quick contains an optional recipe first, required bean and brewer, optional
  note, and optional rating. Rating starts unset, not at 5/10.
- Show the Quick recipe selector only when the user has recipes, with an
  exception for a recipe already attached or supplied by a recipe link.
- Detailed adds method-aware preparation fields. Measurements remain optional.
- Show an optional grinder among the main Detailed fields for espresso and
  pour-over. Put it under “More brewing details” for other methods.
- Show grind setting only when a grinder is selected, except when preserving
  existing records that already have an unassociated setting.
- Do not apply history-based Smart Autofill in Quick.
- Keep bean and brewer creation separate from the brew form.

No AI capture, cafe support, producer presentation redesign, composable recipes,
new standalone method selector, or removal of existing record fields is included.

## Current Implementation Evidence

- [`BrewForm.svelte`](../../web/src/lib/components/BrewForm.svelte) owns recipe
  copying, measurement fields, method-specific sections, submission, and
  history-autofill tracking. It initializes rating to `5`, requires a bean,
  allows an absent brewer, and disables inline bean creation.
- [`userPrefs.ts`](../../web/src/lib/stores/userPrefs.ts) loads the existing
  account-level Smart Autofill preference through `/api/settings`.
- [`brewAutofill.ts`](../../web/src/lib/utils/brewAutofill.ts) computes history
  suggestions; its existing precedence and provenance behavior should remain
  unless this plan explicitly changes it.
- [`brews/new/+page.ts`](../../web/src/routes/brews/new/+page.ts) accepts recipe
  and recipe-owner query parameters. Direct recipe links must continue to work.
- The [brew lexicon](../../lexicons/social/arabica/alpha/brew.json) requires
  `beanRef` and `createdAt`, not `brewerRef`. It permits `grindSize` without a
  grinder. Those existing records remain valid compatibility inputs.
- [FDR-002](../fdr/FDR-002-brew-logging.md) describes recipe copying and scaling;
  [FDR-004](../fdr/FDR-004-smart-autofill.md) describes current history autofill.
  They describe shipped behavior, not this proposal.

## Form Behavior

### Quick

Order the controls as follows:

1. Recipe, optional and conditional on availability.
2. Bean, required.
3. Brewer, required for new brews.
4. “How was it?”, an optional note using the existing tasting-notes field.
5. Optional rating, retaining the 1–10 scale with an explicit unset state.
6. Save.

Keep the Quick/Detailed control visible and keyboard accessible. No recipe
warning, equipment setup, or measurements should precede the coffee unless the
recipe control is relevant. Keep any necessary recipe warning within the recipe
interaction rather than as a standing notice on every new brew.

A new Quick brew without a recipe submits only the selected references and
observations. It does not acquire preparation measurements from history.

If preparation values already exist because of a recipe, previous Detailed
entry, or an edited record, show a compact “Brewing details added” summary with
an action to open Detailed. Quick is a presentation choice, not a destructive
conversion to a different record type.

### Detailed field matrix

Detailed retains Quick's controls. All fields below are optional. Whenever a
grinder appears, show its optional grind-setting input immediately alongside or
below it only after a grinder is selected.

| Brewer category | Main preparation fields | More brewing details |
|---|---|---|
| Espresso | Coffee dose, beverage yield, shot time, grinder | Temperature, pressure, pre-infusion |
| Pour-over | Coffee dose, water, brew time, grinder | Temperature, pours, bloom water/time, drawdown, bypass water, filter |
| Immersion | Coffee dose, water, steep time | Grinder, temperature |
| Moka pot | Coffee dose, water | Grinder, temperature, brew time |
| Cold brew | Coffee dose, water, steep time | Grinder, temperature |
| Cupping | Coffee dose, water, steep time | Grinder, temperature |
| Other / unknown | Coffee dose, water, brew time | Grinder, temperature |

The selected saved brewer determines category. Normalize known legacy category
labels consistently; unrecognized categories use the generic fallback. Do not
require users to register equipment and then separately classify the same brew.

Keep persisted units and precision unchanged. Choose usable time controls for
long steeps, converting explicitly to the existing seconds representation if the
UI uses minutes or hours. Do not silently reinterpret water or temperature
units as part of this layout change.

Existing preparation data outside the selected category must remain reachable
when editing. The matrix governs normal entry, not which old values survive.

### Recipe behavior

- Recipe selection is optional in both modes. In Quick, hide the empty selector
  when the user has no recipes; do not flash an empty selector while loading.
- An attached recipe or a valid recipe-link parameter keeps the recipe control
  and summary available even if the user's own recipe list is empty. Preserve
  cross-user recipe ownership when fetching, matching, and submitting.
- Choosing a recipe explicitly copies its supported preparation values. This
  is allowed in Quick and is distinct from inferred history autofill.
- Show a compact summary of copied values and “Edit brewing details”, which
  opens Detailed. Do not expand every technical field automatically.
- A valid brewer supplied by the recipe satisfies the brewer requirement.
  Users can override it without changing the source recipe.
- Retain existing scaling, pour, and bloom behavior. More composable recipes
  that supply grinder or temperature are deferred.
- Recipe fetch failures must not erase the draft or allow stale responses from
  an earlier selection to overwrite a newer one.

### Grinder and grind setting

Use the visible label “Grind setting”; keep the existing `grind_size` /
`grindSize` contract. The value remains optional and can still be descriptive.

Changing or removing a selected grinder clears its old setting. If the setting
was explicitly entered or comes from a saved record, confirm before making the
change; cancellation retains both the grinder and setting. Unedited history
suggestions may be cleared without a destructive-change confirmation.

An old record with a setting but no grinder keeps that setting visible,
editable, and round-trippable. Do not demand a newly invented association.
If the user subsequently associates a grinder, let them review whether the
legacy setting applies rather than silently treating it as calibrated data.

### Mode, autofill, and draft state

- New users start in Quick; returning users start in their remembered mode.
- Resolve the initial mode through one small policy function. Keep persistence
  separate from field rendering; do not introduce a general preference engine.
- Proposed persistence: extend the existing DID-scoped operational preference
  mechanism. A missing, invalid, or unavailable preference falls back to Quick.
  No Settings-page mode selector is needed initially.
- An explicit mode change on a new-brew form updates the preference. Editing a
  record does not update it. Failure to persist a preference must not block
  logging or reset the in-progress form.
- On edit, start in Detailed when preparation values exist; otherwise start in
  Quick. Automatically opened disclosures do not change the remembered mode.
- Switching modes preserves draft values and any existing autofill provenance.
  Switching to Quick stops further history suggestions; already applied values
  remain summarized and reviewable, with Clear autofill still available when
  applicable. This avoids deleting a draft just because its layout changed.
- History autofill runs only for new brews in Detailed when the existing
  preference permits it. Preserve recipe/user precedence and Clear autofill.
  Gate asynchronous preference/history/recipe completions against the current
  mode so late responses cannot autofill a Quick draft.
- Method-specific suggestions must match the selected brewer category. A
  suggested grind setting must come from the selected/suggested grinder's
  history, not an independently selected setting from another grinder.
- A method change preserves common values. If it would remove incompatible
  method-specific values, ask before clearing them and allow cancellation.
  Do not silently submit stale, newly incompatible fields. Legacy mixed records
  remain editable without cleanup unless the user explicitly requests it.

## Prerequisites and Compatibility

New brews require an existing bean and brewer. If either is missing, show the
missing prerequisite with a link to its separate creation form, then return to
brew logging. Do not add nested entity forms. Preserve recipe-link context
across this round trip, and do not mistake a failed data load for an empty
collection. If someone leaves an in-progress draft, preserve it or warn before
navigation rather than silently losing it.

Requiring a brewer is a behavioral change to creation, not permission to make
`brewerRef` mandatory in the lexicon or decoder. Proposed enforcement is in both
the new-brew UI and create API. Existing records without brewers must still
load and support unrelated edits; update validation must preserve that legacy
absence. A new brew created from an old recipe still needs a brewer selected.

Use the [record-evolution workflow](../../.agents/skills/arabica-record-evolution/SKILL.md)
when implementing validation changes. Test old-record editing and new-record
reads, including older-client compatibility with unchanged wire shapes. No PDS
migration, NSID change, numeric-unit change, or schema tightening is planned.

Optional rating must round-trip absence through the request/model/codec. Do not
substitute 5 or write an invalid zero rating. Verify that explicitly removing an
existing rating works through the update API's actual replacement semantics.

## Implementation Sequence

### 1. Establish compatibility and form policy

- Inspect actual create/update request validation and codecs before changing
  required fields or rating handling. Add fixtures for absent brewer, absent
  rating, unassociated grind setting, and mixed legacy method parameters.
- Add a small field-visibility/category policy and initial-mode policy with unit
  tests. Keep these app-specific; do not move coffee behavior into shared
  entity descriptors.
- Implement the preference read/write path with account isolation, fallback,
  and failure tests. Confirm partial preference updates do not reset unrelated
  settings or overwrite a newer explicit mode choice.

### 2. Ship the Quick/Detailed interaction

- Refactor `BrewForm.svelte` around shared draft state and conditional sections.
- Add Quick controls, unset rating, conditional recipe selection, and the
  attached-details summary. Ensure both modes serialize the same draft safely.
- Gate history autofill by mode and retain recipe application in Quick.
- Add the bean/brewer prerequisite route flow and creation validation, preserving
  existing-record updates. Keep session-expiry and save-error behavior intact.

### 3. Apply method-aware detail and dependent fields

- Implement the matrix and More brewing details disclosure.
- Add grinder-dependent setting visibility and safe grinder/method transitions.
- Make autofill category- and grinder-consistent. Preserve explicit user and
  recipe values; do not turn this into an unrelated scoring-algorithm rewrite.
- Exercise recipe copying/scaling, cross-user recipe links, and legacy edits
  through the actual form and mutation endpoints.

### 4. Update documentation and verify

Update FDR-002 and FDR-004 only when behavior ships. Update the relevant brew and
settings API docs with contract tests for required-on-create brewer behavior,
optional ratings, and the mode preference. Check the actual handlers rather
than copying stale transport descriptions from existing API documentation.
Keep current-state documentation unchanged while this is only a plan.

## Acceptance Checks

- A first-time user with a bean and brewer can save in Quick without opening a
  technical section, choosing a recipe, or supplying a rating.
- Missing prerequisites lead to separate creation and back to logging; loading
  failures and authentication expiry remain distinguishable from empty data.
- Recipe availability, recipe-link exceptions, copied-value summaries, brewer
  overrides, and owner-aware saves work in both modes.
- Every category follows the matrix. Other/unknown types remain usable.
- Grinder selection reveals setting; cancellation and confirmation of a grinder
  change preserve or clear the correct values. Legacy unassociated settings
  survive edits.
- Quick never starts history autofill, including after delayed loads. Detailed
  honors the existing preference. Switching modes loses no values and does not
  hide the fact that additional preparation values will be saved.
- Autofill never pairs one grinder with another grinder's setting or adds
  parameters from a different method.
- Remembered mode survives reload, remains scoped to the account, and is not
  changed by edit-page initialization. Preference failures do not block saving.
- Existing brews without a brewer, rating, or hydrated reference remain
  viewable and editable. Mixed historical parameters are not silently dropped.
- Method changes cannot silently lose entered values or carry incompatible
  values into a new method. Recipe and mode changes cannot bypass this rule.
- Keyboard users can switch modes, expand details, clear ratings, resolve
  confirmations, and submit. Mobile layout remains usable; focus survives
  conditional-section changes and validation errors identify their controls.

Run `pnpm run check:svelte` and `pnpm run test:svelte`, extending the existing
`web/tests/pages/brew-form*.test.ts`, new/edit route tests, and
`web/src/lib/utils/brewAutofill.test.ts`. Add focused browser coverage in
`web/tests/e2e/brew-form.spec.ts` and `web/tests/e2e/create-brew.spec.ts`.
For preference and validation changes, run targeted Go tests and
`just integration-test`, including brew and settings JSON contracts. Broaden to
`just ci-check` after the integrated change. Record which checks actually ran.
