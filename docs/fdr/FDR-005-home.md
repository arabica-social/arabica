# FDR-005: Home

**Status:** Active
**Last reviewed:** 2026-10-28

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
  problems or suggest improvements, followed by any popular recipes. Both rails
  follow the feed on narrow screens.
- Unfinished-entry links open the relevant edit page; recipe links open the
  recipe. Empty data sections are omitted without changing desktop column widths.
- Global navigation remains in the shared header and mobile navigation.
  Feedback, About, and AT Protocol links also remain in the footer.
- Loading, empty, and failed-feed states are distinct.

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

## Related

- **FDRs:** [FDR-001: Explore](FDR-001-explore.md),
  [FDR-003: Feedback](FDR-003-feedback.md)
- **Guidance:** [PRODUCT.md](../../PRODUCT.md), [DESIGN.md](../../DESIGN.md)
