# FDR-002: Brew Logging

**Status:** Active
**Last reviewed:** 2026-07-18

## Overview

Brew logging records an individual coffee session while keeping a selected
recipe reusable and intact. A brew copies the recipe's relevant measurements at
creation time, then records the specific cup that was made.

## Behavior

- A user may select a recipe to fill its coffee, water, pour, and brewing
  category details into a new brew.
- The selected recipe remains unchanged when a user changes the brew.
- The selected-recipe summary includes an inline adjustment control. A user can
  set coffee dose, water, or brew ratio; the other amount updates to match.
- When the selected recipe has pours, quick adjustments scale their water
  amounts to the adjusted total.
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

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md)
- **FDRs:** None
