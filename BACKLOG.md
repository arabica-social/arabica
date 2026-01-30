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

## Refactor

- Move all frontend code into a web subdir in internal (bff and components dirs)
  - Components and pages, live in separate dirs (web/components, web/pages)

- Back button in view should take user back to their previous page (not sure how to handle this exactly though)
  - Back button is over-engineered, switch to using `history.Back`
