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
   - Toggle for "for" and "at" in pours view
   - Pull bsky account management stuff in? (i.e. email verification, change password, enable two factor?)

- "Close bag" of coffee
  - Remove it from the beans dropdown when adding a new brew
  - Add a "closed"/"archived" field to the lexicon
  - Maybe allow adding a rating?
  - Question: Should it show up in the profile screen? (maybe change header to current beans? --
    have a different list at bottom of previous beans -- show created date, maybe closed date?)
    - should be below the brewers table

## Far Future Considerations

- Pivot to full svelte-kit?

- Maybe swap from boltdb to sqlite
  - Use the non-cgo library

## Fixes

- Migrate terms and about page text. Add links to about at top of non-authed home page

- Backfill on startup should be cache invalidated if time since last backfill exceeds some amount (set in code/env var maybe?)

- Fix always using celcius for units, use settings (future state) or infer from number (maybe also future state)

- Make rating color nicer, but on white background for selector on new/edit brew page

- Refactor: remove the `SECURE_COOKIES` env var, it should be unecessary
  - For dev, we should know its running in dev mode by checking the root url env var I think?
  - This just adds noise and feels like an antipattern

- Fix styling of manage records page to use rounded tables like everything else
  - Should also use tab selectors the same way as the profile uses
