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

- Loading on htmx could probably be snappier by using a loading bar, and waiting until everything is loaded
  - Alternative could be using transitionary animations between skeleton and full loads
  - Do we even need skeleton loading with SSR? (I think yes here because of PDS data fetches -- maybe not if we kept a copy of the data)

- Headers in skeletons need to exactly match headers in final table
  - Refreshing profile should show either full skeleton with headers, or use the correct headers for the current tab
    (It currently shows the brew header for all tabs)

- Add styling to mail link and back to home button on terms page

- Take revision pass on text in about and terms

- Need to cache profile pictures to a database to avoid reloading them frequently
  - This may already be done to some extent

- Tables flash a bit on load, could more smoothly load in? (maybe delay page until the skeleton is rendered at least?)
  - I think the profile page does this relatively well, with the profile banner and stats

## Refactor

- Need to think about if it is worth having manage, profile, and brew list as separate pages.
  - Manage and profile could probably be merged? (or brew list and manage)

- Profile tables should all either have edit/delete buttons or none should
- Also, add buttons below each table for that record type would probably be nice

- Maybe having a way of nesting modals, so a roaster can be created from within the bean modal?
  - Maybe have a transition that moves the bean modal to the left, and opens a roaster modal to the right
