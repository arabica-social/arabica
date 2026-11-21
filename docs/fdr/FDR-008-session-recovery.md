# FDR-008: Session recovery

**Status:** Active  
**Last reviewed:** 2026-11-21

## Behavior

- **Log In Again** in the session-expired dialog returns the user to the same
  page, including its query and fragment, after they sign in to the same account.
- Unsaved changes in coffee record forms, setup drawers, and comment boxes
  reappear in the form. Arabica does not retry Save or Post.
- If the browser cannot keep a draft, the dialog lets the user dismiss it to
  copy their changes, or proceed without them.

## Design Decisions

### Keep a short-lived draft in the current tab

The browser saves changes when the user chooses to log in again, not on every
keystroke. The snapshot is available for one return within 30 minutes and is
never sent to the server. Signing in to a different account drops the snapshot.
Closing the tab can lose it. Chat continues to use its own account-scoped history.
