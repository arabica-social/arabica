# FDR-003: Feedback

**Status:** Active **Last reviewed:** 2026-10-28

## Overview

Feedback gives visitors a lightweight way to tell Arabica's operators about a
rough edge, missing detail, or idea. Operator correspondence never touches a
user's Personal Data Server.

## Behavior

- The homepage right rail and the shared footer link to Feedback. Both
  open the Arabica feedback board on userinput.app in a new tab
  (`https://userinput.app/s/did:plc:chqc2ockzmyvlrasfb66x64a/3mrgh3b4f722p`).
- There is no dedicated in-app feedback page and no feedback form. The board
  URL is app configuration (`AppDefinition.feedbackUrl`), so a sister app can
  point its CTAs at its own board.

## Design Decisions

### 1. Send feedback to the Arabica userinput.app board

**Decision:** Feedback CTAs open the Arabica feedback board on userinput.app in
a new tab instead of collecting notes in an in-app page or a mail draft.

**Why:** Suggestions stay public and visible to other users, and operators can
review them without Arabica storing operator correspondence in a new
server-side store or in user PDSes. Opening a new tab keeps the visitor in
context.

**Tradeoff:** Contributing requires an account on userinput.app, and operators
must watch the board rather than receiving notes in email.

### 2. Keep a concise invitation on Home

**Decision:** Group Feedback and About links in the homepage right rail with
one sentence inviting reports of confusing, broken, or missing behavior. Keep
the footer link plain.

**Why:** Alpha users benefit from knowing what feedback is welcome. A short
invitation serves that purpose without the old panel's extra headline and
labels or separate promotional sections.

**Tradeoff:** The invitation takes more space than a bare link, but remains
secondary to brew logging and community content.

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md)
- **FDRs:** [FDR-005: Home](FDR-005-home.md)
