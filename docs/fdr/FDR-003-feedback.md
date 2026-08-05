# FDR-003: Feedback

**Status:** Active **Last reviewed:** 2026-08-05

## Overview

Feedback gives visitors a lightweight way to tell Arabica's operators about a
rough edge, missing detail, or idea. Operator correspondence never touches a
user's Personal Data Server.

## Behavior

- The home page's "Around the café" rail includes a reusable feedback prompt,
  and the footer links to Feedback. Both open the Arabica feedback board on
  userinput.app in a new tab
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

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md)
- **FDRs:** None
