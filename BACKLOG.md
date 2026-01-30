## Description

This file includes the backlog of features and fixes that need to be done.
Each should be addressed one at a time, and the item should be removed after implementation has been finished and verified.

---

## Features

1. LARGE: complete record styling refactor that changes from table-style to more mobile-friendly style
   - Likely a more "post-style" version that is closer to bsky posts
   - To be done later down the line
   - setting to use legacy table view

2. Settings menu (mostly tbd)
   - Private mode -- don't show in community feed (records are still public via pds api though)
   - Dev mode -- show did, copy did in profiles (remove "logged in as <did>" from home page)
   - Toggle for table view vs future post-style view

- Maybe add loading bars to page loads? (above the header perhaps?)
  - A separate nicer pretty loading bar would also be nice on the brews page?

## Far Future Considerations

- Maybe swap from boltdb to sqlite
  - Use the non-cgo library

## Fixes

- Feed item "view details" button should go away, the "new brew" in "addded a new brew" should take to view page instead (underline this text)

- Manage brews page truncates notes, but variables wrap to considerably more lines, probably don't need to truncate notes.

- Loading on htmx could probably be snappier by using a loading bar, and waiting until everything is loaded
  - Alternative could be using transitionary animations between skeleton and full loads
  - Do we even need skeleton loading with SSR? (I think yes here because of PDS data fetches -- maybe not if we kept a copy of the data)

- Headers in skeletons need to exactly match headers in final table
  - Refreshing profile should show either full skeleton with headers, or use the correct headers for the current tab
    (It currently shows the brew header for all tabs)

- Modals still don't fade out on save/cancel like I want them to fade out

- Add styling to mail link and back to home button on terms page

- Take revision pass on text in about and terms

- Need to cache profile pictures to a database to avoid reloading them frequently
  - This may already be done to some extent

- Hitting back from `/brews/new` page to home page shows welcome box, then it transitions in, which looks bad
  - Seems to be triggering the transition twice
  - BIG: Refactoring the transition code to be less brittle would probably be good, but I'm unsure of how to do it

- Back button on view page does nothing, no logs

- Adding a record from a modal in the brews page does not currently add to the list until refreshed
  - An older version of the code (probably on main) adds to the list and auto-selects, which is desired behavior

- Tables flash a bit on load, could more smoothly load in? (maybe delay page until the skeleton is rendered at least?)
  - I think the profile page does this relatively well, with the profile banner and stats

## Refactor

- Move all frontend code into a web subdir in internal (bff and components dirs)
  - Components and pages, live in separate dirs (web/components, web/pages)

- Need to think about if it is worth having manage, profile, and brew list as separate pages.
  - Manage and profile could probably be merged? (or brew list and manage)

- Profile tables should all either have edit/delete buttons or none should
- Also, add buttons below each table for that record type would probably be nice

- Maybe having a way of nesting modals, so a roaster can be created from within the bean modal?
  - Maybe have a transition that moves the bean modal to the left, and opens a roaster modal to the right

## Templ Code Review Findings

### ✅ COMPLETED FIXES

1. **Fixed unsafe avatar indexing** - header.templ now uses `bff.SafeAvatarURL` safely (was already fixed)
2. **Fixed buggy getRowsAttr function** - forms.templ now uses `strconv.Itoa(rows)` (was already fixed)
3. **Extracted dropdown constants** - models/options.go with RoastLevels, GrinderTypes, BurrTypes (was already done)
4. **Eliminated empty state duplication** - All empty states now use `EmptyState` component from shared.templ
   - Updated: brew_list_table.templ, profile_partial.templ (4 locations), manage_partial.templ (4 locations)
5. **Profile.templ roast levels** - Now uses `models.RoastLevels` instead of hardcoded values
6. **Avatar Rendering Logic Consolidated** - Created single `Avatar` component in shared.templ
   - Replaced duplicated avatar logic in: feed.templ, header.templ, profile.templ
   - Three size variants (sm, md, lg) with consistent safety checks using bff.SafeAvatarURL
   - Reduced code duplication by ~40 lines
7. **Script Loading Duplication Fixed** - Removed duplicate entity-manager.js loads from manage.templ and profile.templ
   - Script is loaded once in layout.templ, removed redundant loads from child templates
8. **Form Input Styling Standardized** - Profile modals now use form-input/form-textarea CSS classes
   - BeanFormModalProfile and RoasterFormModal updated to use modal-backdrop, modal-content, modal-title classes
   - Replaced inline button styling with btn-primary/btn-secondary classes
   - Consistent styling across all modals (entity_modals.templ, dialog_modals.templ, profile.templ)

### CRITICAL: Duplication Issues

1. **Three Different Modal Systems** - MUST CONSOLIDATE
   - `entity_modals.templ` uses Alpine.js x-show modals
   - `dialog_modals.templ` uses native HTML5 `<dialog>` elements
   - `profile.templ` reimplements inline Alpine modals (BeanFormModalProfile, RoasterFormModal)
   - **Action:** Pick ONE approach (recommend native `<dialog>`) and migrate all modals to it
   - **Impact:** High - causes confusion and maintenance burden

2. **Loading Skeleton Inconsistencies**
   - Different column counts: `brew_list.templ` (6 cols), `profile.templ` (5 cols), `manage.templ` (5 cols)
   - Headers don't match actual table headers (brew skeleton shows for all profile tabs)
   - **Action:** Match skeleton headers to actual content, make skeletons tab-aware
   - **Impact:** Low - UX issue (already noted in Fixes section)

### Broken/Unsafe Code

All items in this section have been fixed (see completed fixes #1, #2, and verified getProfileIdentifier)

### Underutilized Components

1. **Form Components from `forms.templ` Rarely Used**
   - Nice reusable components defined but only used in `brew_form.templ`
   - Modals and other forms use inline HTML instead
   - **Action:** Refactor all forms to use these components

2. **Card Component Underused**
   - `card.templ` defines Card component but many places use inline card classes
   - **Action:** Replace inline card HTML with Card component

3. **Button Components Underused**
   - `buttons.templ` defines PrimaryButton/SecondaryButton but rarely used
   - **Action:** Replace inline button classes with components

### Refactoring Recommendations

1. **Extract Common Data to Shared Package**
   - Create constants/functions for: roast levels, grinder types, burr types
   - Import from single source of truth

2. **Consolidate Modal Implementation**
   - Migrate all modals to native `<dialog>` (modern standard, better a11y)
   - Remove Alpine.js modal code
   - Remove inline modal implementations

3. **Standardize Form Field Classes**
   - Use `form-input`, `form-select`, `form-textarea` classes everywhere
   - Remove inline `rounded-lg border-2 border-brown-300` patterns

4. **Create Avatar Component**
   - Single component with all safety checks and fallback logic
   - Props: avatarURL, displayName, size
   - Replace all avatar rendering with this component

5. **Extract Table Cell Styling**
   - Common patterns like `px-6 py-4`, `px-4 py-4` repeated everywhere
   - Create CSS utility classes or extract to table components

6. **Fix Loading Skeletons**
   - Make skeletons match their content structure
   - Create tab-specific skeletons or dynamic skeleton based on active tab
   - Headers should exactly match final table headers
