# FDR-003: Feedback

**Status:** Active **Last reviewed:** 2026-07-19

## Overview

Feedback gives visitors a lightweight way to tell Arabica's operators about a
rough edge, missing detail, or idea without putting operator correspondence in
their Personal Data Server.

## Behavior

- The home page includes a reusable feedback prompt that links to `/feedback`.
  The same component can be placed on other SvelteKit pages.
- The feedback page collects a required short title, an optional email address
  or AT Protocol handle, and an optional longer description.
- Submitting opens a plain-text email draft addressed to `mail@arabica.systems`.
  Arabica does not persist the form content or write it to the user's PDS.

## Design Decisions

### 1. Send feedback through the visitor's email client

**Decision:** The form uses a `mailto:` submission rather than an in-app
feedback inbox.

**Why:** Feedback is operator correspondence, not user-owned coffee data. A mail
draft makes the delivery path clear and avoids collecting contact details in a
new server-side store.

**Tradeoff:** A visitor needs an available email client to send the message. The
form cannot offer delivery confirmation inside Arabica.

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md)
- **FDRs:** None
