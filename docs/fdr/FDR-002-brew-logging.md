# FDR-002: Brew Logging

**Status:** Active
**Last reviewed:** 2026-07-21

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
- Users can still edit the remaining brew details, such as bean, equipment,
  temperature, timing, notes, and rating.

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
- **FDRs:** None
