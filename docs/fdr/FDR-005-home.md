# FDR-005: Home

**Status:** Active
**Last reviewed:** 2026-11-04

## Overview

Home shows community activity and gives returning users a quick way to log a
brew. Visitors can browse the public feed before logging in.

## Behavior

- One masthead replaces the journal hero and separate “Recent activity” heading.
  It has a single Fraunces H1, “Community,” with the primary action alongside.
  Signed-in feed filters and sorting controls sit beneath the title within the
  masthead. The feed follows directly; signed-in users can load more activity.
- Ready accounts see Log Brew without an eyebrow, slogan, introduction, or
  secondary Browse your journal link.
- If required setup is incomplete, one Finish setup action replaces Log Brew.
  App readiness guidance explains the missing prerequisites. There is no second
  setup action.
- Signed-out visitors see Log in, a separate Create an account link, and
  “Track brews, beans, and recipes.” above the public feed. The masthead has no
  account-server explanation. Visitors do not see personal shortcuts,
  suggestions, or feed filters.
- The original masked bean SVG remains behind the masthead at opacity `.045`,
  positioned independently of prose. On desktop it is `20rem` square at
  `top: -.7rem` and `right: 3rem`. At widths up to `719px`, it is `15rem` square
  at `top: auto`, `bottom: -6.5rem`, and `right: 0`, clipped by the existing
  lower divider. Masthead height follows the content; no minimum height or
  extra mobile bottom padding is added.
- The established three-column desktop layout keeps the feed between two rails.
  The left rail contains icon shortcuts to My Coffee, Explore, Recipes, and
  the user's profile when signed in, plus any unfinished entries. The right
  rail groups Feedback and About links with a brief invitation to report
  problems or suggest improvements, followed by popular recipes when any exist; otherwise a compact Recipes section with an Explore-all-recipes link keeps the rail present. Both rails
  follow the feed on narrow screens.
- Unfinished-entry links open the relevant edit page; recipe links open the
  recipe. Individual rail sections beyond the always-present feedback and Recipes sections are omitted when empty, without changing desktop column widths.
- Global navigation remains in the shared header and mobile navigation.
  Feedback, About, and AT Protocol links also remain in the footer.
- Feed brew cards lead with the bean name across the full card width, then the
  roaster and bean facts. The brew's numbers follow as one facts table: aligned
  label-over-value cells (brewer, grinder, dose, water, temperature, time, plus
  espresso or pourover parameters when recorded) that fill the card width in up
  to three columns. The rating sits at the right of the card's action line.
  Home cards include a Pours cell in the same facts table, showing only the
  count instead of individual water amounts and timings; omit it when no pours
  are recorded.
  Explore cards and brew detail pages retain the full pour breakdown. Tasting
  notes close the card.
- Loading, empty, and failed-feed states are distinct.
- Load more appends records once per AT-URI, ignoring overlapping entries
  within or between pages while retaining their existing order.
- When AI is configured, signed-in users also see a compact description input
  between the masthead and feed. It adds no heading or suggestions and does not
  replace Community, manual actions or feed controls. See
  [FDR-007](FDR-007-ai-assisted-creation.md).

## Design Decisions

### 1. Put community activity first

**Decision:** Replace the journal hero and repeated feed heading with one
Community masthead, the primary action, and signed-in feed controls. Keep the
left shortcuts and right rail unchanged.

**Why:** Returning users can reach logging and community activity without
passing a repeated introduction. Journal navigation remains in the shortcuts
and shared navigation.

**Tradeoff:** Ready accounts have no explanatory copy. Visitors still need a
factual introduction and separate login and account-creation actions;
setup-incomplete accounts need readiness guidance and one Finish setup action.
Account and storage explanations remain available through About and AT Protocol,
including the footer links.

### 2. Position artwork independently of copy

**Decision:** Keep the original bean mask and opacity in a new, smaller
composition, using desktop top/right placement and a mobile bottom crop.
Keep the masthead content-driven rather than preserving the old hero's height.

**Why:** The former crop depended on the journal headline and introduction.
Independent placement lets the masthead lose that copy without adding empty
space to support the illustration.

**Tradeoff:** This composition does not reproduce the old hero crop. Review
artwork placement across ready, setup-incomplete, and visitor states at narrow
and wide widths in both themes.

### 3. Align brew facts in a table

**Decision:** Show a brew's numbers as label-over-value cells in a width-filling
grid rather than an inline run of `Label: value` facts, and move the rating to
the action line so the bean name spans the card.

**Why:** Inline facts left each row with a ragged tail, put the dose on a line of
its own, and gave the brewer a whole line. Aligned columns use the card's width,
make values comparable down a column, and shorten the card. The rating moves to
where the profile and My Coffee cards already place it, which also frees the bean
name from a narrow column beside the badge.

**Tradeoff:** A card with four recorded facts falls back to two columns instead
of leaving an orphan cell in a three-column row, so column count varies with the
amount of recorded data.

## Related

- **FDRs:** [FDR-001: Explore](FDR-001-explore.md),
  [FDR-003: Feedback](FDR-003-feedback.md)
- **Guidance:** [PRODUCT.md](../../PRODUCT.md), [DESIGN.md](../../DESIGN.md)
