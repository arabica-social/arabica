## Description

This file includes the backlog of features and fixes that need to be done.
Each should be addressed one at a time, and the item should be removed after implementation has been finished and verified.

---

## Far Future Considerations

- Pivot to full svelte-kit?

- Maybe swap from boltdb to sqlite
  - Use the non-cgo library
  - Is there a compelling reason to do this?
  - Might be good as a sort of witness-cache type thing

- The profile, manage, and brews list pages all function in a similar fashion,
  should one or more of them be consolidated?
  - Manage + brews list together probably makes sense

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

- Add my custom iosevka font as default font

- Improve caching of profile pictures, tangled.sh apparently does this really well
  - Something with a cloudflare cdn
  - Might be able to just save to the db when backfilling a profile's records
  - NOTE: requires research into existing solustions (whatever tangled does is probably good)

## Fixes

- Migrate terms page text. Add links to about at top of non-authed home page

- Backfill on startup should be cache invalidated if time since last backfill exceeds some amount (set in code/env var maybe?)

- Make rating color nicer, but on white background for selector on new/edit brew page

- Profile page should show more details, and allow brew entries to take up more vertical space

- Show "view" button on brews in profile page (same as on brews list page)

- Navigating to "my profile" while on another user's profile, the url changes but the page does not change

- Clicking "back to brews" on brew view page returns user to their brews list, even if the brew belonged to another user
