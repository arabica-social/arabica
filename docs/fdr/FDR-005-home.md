# FDR-005: Home

**Status:** Active
**Last reviewed:** 2026-10-28

## Overview

Home shows community activity and gives returning users a quick way to log a
brew. Visitors can browse the public feed before logging in.

## Behavior

- Signed-out visitors see a short introduction, login and account-creation
  actions, and the public feed. A brief server-storage explanation links to
  AT Protocol. Visitors do not see personal shortcuts, suggestions, or feed
  filters.
- Signed-in users retain the original journal header: its small journal label,
  “Good coffee deserves a record” display headline, brew/community introduction,
  Log Brew action with an arrow, and Browse your journal link. If required setup
  is incomplete, one Finish setup action replaces Log Brew and the introduction
  explains the prerequisites. There is no second setup action.
- The original masked bean artwork remains faint behind the right side of the
  header, beside the headline and actions, cropped by the lower separator.
  Its original size and bottom offset are retained. Header height follows the
  content; no minimum height or extra mobile bottom padding is added.
- Signed-in users can filter and sort the feed and load more activity.
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

### 1. Trim repetition without removing useful guidance

**Decision:** Keep the established journal header composition and one setup
action, icon navigation, and concise feedback guidance. Group the rail's
supporting links rather than giving each a separate promotional section.

**Why:** The old setup actions and surrounding pitches repeated each other.
The feedback invitation explains what users can report, and the visitor storage
line explains something the generic journal introduction does not.

**Tradeoff:** The homepage retains some supporting copy. Each line must add
information rather than restating an action label.

### 2. Restore the original header composition

**Decision:** Retain the desktop column proportions, original header alignment,
bean mask, size, opacity, and `bottom: -7.5rem` crop. Preserve the journal label,
ready-account introduction, and secondary library link as part of that layout.

**Why:** Moving the bean below the headline and adding minimum header height
made it more cropped while adding unwanted empty space. The preferred design
shows the bean beside the headline and actions. Restoring offsets alone while
removing the original header content also clips the bean's top.

**Tradeoff:** The header retains more copy than a minimal title and button.
Browser tests pin the original crop, content-driven height, and bottom-aligned
columns. Header snapshots cover visitor, setup-incomplete, and ready accounts
in both themes, including the wide ready-account composition from main.

## Related

- **FDRs:** [FDR-001: Explore](FDR-001-explore.md),
  [FDR-003: Feedback](FDR-003-feedback.md)
- **Guidance:** [PRODUCT.md](../../PRODUCT.md), [DESIGN.md](../../DESIGN.md)
