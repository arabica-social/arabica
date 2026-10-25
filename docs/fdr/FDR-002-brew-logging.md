# FDR-002: Brew Logging

**Status:** Active
**Last reviewed:** 2026-10-25

## Overview

Brew logging records an individual coffee session while keeping a selected
recipe reusable and intact. A brew copies the recipe's relevant measurements at
creation time, then records the specific cup that was made.

## Behavior

- A user may select a recipe to fill its coffee, water, pour, brewer, and
  brewing category details into a new brew.
- The selected recipe remains unchanged when a user changes the brew.
- The selected-recipe summary includes an inline adjustment control. A user can
  set coffee dose, water, or brew ratio; the other amount updates to match.
- When the selected recipe has pours, quick adjustments scale their water
  amounts to the adjusted total.
- For pour-over recipes, the brew's bloom water and bloom time mirror the
  first pour; bloom water stays in sync as quick adjustments rescale the pours.
- Quick shows an optional recipe when available, a required bean and brewer,
  optional tasting notes, and an optional 1–10 rating. Rating
  starts unset. “Add rating” reveals a 1–10 slider starting at 5; “Remove
  rating” hides it and clears the value, including on edit. Bean and brewer
  creation opens in separate tabs so the draft and recipe-link context stay in
  place.
- New users start in Quick. Explicit mode selections on new brews update a
  DID-scoped operational preference; missing or unavailable preferences use
  Quick. Edits start in Detailed when preparation data exists, otherwise Quick,
  without changing the preference.
- Switching modes preserves the draft. Quick summarizes preparation values
  already copied from a recipe, entered in Detailed, or loaded from a record.
  “Edit brewing details” opens Detailed without changing the remembered mode.
- The form omits decorative header copy and the explanatory sidebar. The
  coffee-to-water ratio appears with those measurements in Detailed.
- Detailed uses the saved brewer's category. Espresso puts coffee dose, beverage
  yield, shot time, and grinder first. Pour-over puts coffee, water, brew time,
  and grinder first. Immersion, cold brew, and cupping put coffee, water, and
  steep time first. Moka pot puts coffee and water first; other categories put
  coffee, water, and brew time first. Remaining optional fields appear under
  “More brewing details”. Long steeps can be entered in minutes or hours and
  are converted to the existing seconds representation.
- Grind setting appears after choosing a grinder. Changing or removing a grinder
  clears its old setting, with confirmation for user-entered or saved settings.
  Legacy settings without a grinder remain editable without a new association.
- Changing brewer category asks before clearing incompatible espresso or
  pour-over parameters and pours. Cancellation retains the previous selection
  and values. Common measurements survive. Mixed legacy values stay reachable
  and survive unrelated edits until the user changes category.
- Recipe copying works in either mode and never changes the recipe. Foreign
  recipe ownership is retained; only a matching saved brewer in the user's own
  collection can satisfy the brewer requirement. Later recipe responses cannot
  overwrite a newer recipe selection or explicit brewer choice.
- Brewer is required only for creation. Existing records without a brewer,
  hydrated references, or rating remain editable. No lexicon or unit changes
  accompany this layout.

## Design Decisions

### 1. Adjust the copied brew, not the recipe

**Decision:** Recipe adjustments affect only the draft brew.

**Why:** A recipe is a reusable starting point. Scaling it for one cup should
not silently alter the reference used by later brews.

**Tradeoff:** The adjusted amounts are recorded on the brew rather than creating
a new reusable recipe variant.

### 2. Keep pour stages proportional

**Decision:** Changing the quick-adjustment measurements scales existing pour
water amounts to the new water total.

**Why:** A recipe's staged pours are part of its brewing method. Scaling only
the total water would leave the draft internally inconsistent.

**Tradeoff:** A user who wants a different pour pattern must edit the pours
after adjusting the recipe.

### 3. Derive bloom water and time from the first pour

**Decision:** For pour-over recipes, the brew's bloom water mirrors the first
pour's water and its bloom time mirrors the first pour's time. Bloom water
updates again as quick adjustments rescale the pours; bloom time is set once on
recipe apply (pour times are not rescaled by quick adjustment).

**Why:** The bloom is conventionally the first pour of a pour-over recipe, so
asking the user to re-enter it duplicates the first pour and risks drift when
the recipe is scaled. Deriving it keeps the brew self-consistent.

**Tradeoff:** A user who wants a bloom that differs from the first pour must
edit the bloom water (or time) after adjusting the recipe; the next
quick-adjust pass resets bloom water to the first pour.

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md)
- **FDRs:** [FDR-004: Smart Autofill](FDR-004-smart-autofill.md) (autofill remains lower precedence than recipe copying)
