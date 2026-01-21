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

## Fixes

- Adding new gear (grinders, etc.) from profile page redirects to brews on profile page after (should stay on current page)
- Brews on profile page are stored in chronological order, should be reverse chronological (newest first)

- [Future work]: adjust timing of caching in feed, maybe use firehose and a sqlite database since we are only storing a few anyway
  - Goal: reduce pings to server when idling
