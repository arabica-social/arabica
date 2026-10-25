# FDR-004: Smart Autofill

**Status:** Active
**Last reviewed:** 2026-10-25

## Overview

Smart Autofill helps a user start a new Brew by filling empty preparation
fields from their own previous brews of the selected Bean and Recipe. It is a
client-side suggestion derived from cached history; it does not create a new
record or change existing brews.

## Behavior

- Smart Autofill is enabled by default and can be turned off in Settings under
  **Brewing preferences**. The preference follows the user's DID.
- On a new-brew form in Detailed mode, selecting a bean computes suggestions
  from the user's matching brew history. Quick does not apply new history
  suggestions, including after delayed preference, history, or recipe loads.
  Switching to Quick preserves previously applied values and their provenance;
  Clear autofill remains available. Explicit recipe copying also works in Quick.
  With a recipe selected, matching first uses the bean-plus-recipe pair, then falls back to bean-only history when the pair has
  fewer than two brews.
- Autofill only writes into empty fields. Explicit user input and values copied
  from a Recipe always take precedence.
- Suggested fields include equipment and preparation measurements such as the
  Grinder, Brewer, grind setting, temperature, brew time, filter, and relevant
  method-specific parameters. The selected or suggested brewer determines the
  category used for preparation suggestions. Espresso and pour-over parameters
  are restricted to their matching category. Grind settings come only from
  history for the selected or suggested grinder. Tasting notes, rating, and pour
  stages are not autofilled.
- Each autofilled field is marked with its raw support, such as
  “Autofilled · in 12 of 15 brews (80%).” Editing that field removes its
  autofill marker.
- A form-level **Clear autofill** action appears while autofilled fields remain.
  It clears those suggestions without changing fields the user subsequently
  edited. An autofilled grinder or brewer is retained if an edited dependent
  setting or method-specific value now needs that association.

## Design Decisions

### 1. Use the existing client-side history

**Decision:** Compute suggestions in the SPA from the user's existing `/api/data`
brew history cache.

**Why:** The new-brew form already loads this user-scoped data for entity
selection, so suggestions are immediate and require no new API endpoint.

**Tradeoff:** The result follows the cache's freshness window rather than
being a separately queried history view.

### 2. Bias toward recent habits

**Decision:** Score values with exponential recency decay. The newest brew has
weight 1, and each five-brew recency interval halves a brew's weight. The
highest weighted value wins; the UI reports raw occurrence counts.

**Why:** This follows a changing habit without treating one recent experiment
as definitive.

**Tradeoff:** The selected value's weighted score can differ from its displayed
raw percentage.

### 3. Preserve explicit intent

**Decision:** Precedence is user-entered values, then recipe-applied values,
then Smart Autofill. Autofill never overwrites a non-empty field.

**Why:** A user's current action and an explicitly selected reusable Recipe are
stronger signals than historical defaults.

## Related

- **FDRs:** [FDR-002](FDR-002-brew-logging.md)
- **Plan:** [Smart Autofill implementation plan](../plans/2026-07-24-brew-smart-autofill.md)
