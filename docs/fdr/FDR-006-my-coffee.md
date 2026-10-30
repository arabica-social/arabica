# FDR-006: My Coffee

**Status:** Active
**Last reviewed:** 2026-10-30

## Behavior

My Coffee is the authenticated user's collection at `/my-coffee`. Brews,
beans, roasters, grinders, brewers, and recipes are selected through collection
navigation with record counts. The selected collection is remembered locally.
Desktop uses navigation beside the records; narrow screens put the navigation
above them. There is no separate equipment-summary sidebar.

Add and Log brew remain page actions. Empty collections offer their relevant
creation action, including logging a first brew. These actions complement the
global navigation rather than introducing another summary of the collection.

Refresh sits beside the collection navigation. It reloads records from the
user's account without switching collection, disables while pending, and reports
success or failure. A failed refresh retains the currently displayed records.
It remains available without a permanent troubleshooting panel.

## Rationale

Collection navigation already exposes beans and equipment counts. Repeating
those in a right rail takes space from records, and separates tools from the
collection on phones. A compact Refresh control preserves access to an explicit
reload without surrounding every visit with troubleshooting copy.

No record formats, API paths, storage authority, or visibility rules change.
