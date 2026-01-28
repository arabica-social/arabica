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

- Consider fully separating API backend from frontend service
  - Currently using HTMX header checks to prevent direct browser access to internal API endpoints
  - If adding mobile apps, third-party API consumers, or microservices architecture, revisit this
  - For now, monolithic approach is appropriate for HTMX-based web app with decentralized storage

- Maybe swap from boltdb to sqlite
  - Use the non-cgo library

## Fixes

- Homepage still shows cached feed items on homepage when not authed. should show a cached version of firehose (last 5 entries, cache last 20) from the server.
  This fetch should not try to backfill anything

- Feed database in prod seems to be showing outdated data -- not sure why, local dev seems to show most recent.

- View button for somebody else's brew leads to an invalid page. need to show the same view brew page but w/o the edit and delete buttons.
- Back button in view should take user back to their previous page (not sure how to handle this exactly though)

- Header should probably always be attached to the top of the screen?

- Feed item "view details" button should go away, the "new brew" in "addded a new brew" should take to view page instead (underline this text)

- Manage brews page truncates notes, but variables wrap to considerably more lines, probably don't need to truncate notes.

- Loading on htmx could probably be snappier by using a loading bar, and waiting until everything is loaded
  - Alternative could be using transitionary animations between skeleton and full loads
  - Do we even need skeleton loading with SSR? (I think yes here because of PDS data fetches -- maybe not if we kept a copy of the data)

- I think we should be using htmx more than alpine, we are leaning too much on alpine and not enough on htmx
  - I think for transitions we should be able to swap to htmx from alpine?
  - I may be completely off base on this though
