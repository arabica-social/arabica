## Description

This file includes the backlog of features and fixes that need to be done.
Each should be addressed one at a time, and the item should be removed after implementation has been finished and verified.

---

## Far Future Considerations

- Pivot to full svelte-kit?

- Maybe swap from boltdb to sqlite
  - Use the non-cgo library
  - Is there a compelling reason to do this?
  - Might be good as a sort of witness-cache type thing (record refs to avoid hitting PDS's as often?)
  - Probably not worth unless we keep a copy of all (or all recent) network data

- The profile, manage, and brews list pages all function in a similar fashion,
  should one or more of them be consolidated?
  - Manage + brews list together probably makes sense

- IMPORTANT: If this platform gains any traction, we will need some form of content moderation
  - Due to the nature of arabica, this will only really need to be text based (text and hyperlinks)
  - Malicious link scanning may be reasonable, not sure about deeper text analysis
  - Need to do more research into security
  - Need admin tooling at the app level that will allow deleting records (may not be possible),
    removing from appview, blacklisting users (and maybe IPs?), possibly more
  - Having accounts with admin rights may be an approach to this (configured with flags at startup time?)
    @arabica.social, @pdewey.com, maybe others? (need trusted users in other time zones probably)
  - Add on some piece to the TOS that mentions I reserve the right to de-list content from the platform
  - Continue limiting firehose posts to users who have been previously authenticated (keep a permanent record of "trusted" users)
    - By logging in users agree to TOS -- can create records to be displayed on the appview ("signal" records)
      Attestation signature from appview (or pds -- use key from pds) was source of record being created
  - This is a pretty important consideration going forward, lots to consider

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

- Opengraph metadata in brew entry page, to allow rich embeds in bluesky
  - All pages should have opengraph metadat, but view brew, profile, and home/feed are probably the most important

- Maybe move water amount below pours in form, sum pours if they are entered first.
  - Would need to not override if water amount is entered after pours
    (maybe update after leaving pour input?).

## Fixes

- Migrate terms page text. Add links to about at top of non-authed home page

- Backfill on startup should be cache invalidated if time since last backfill exceeds some amount (set in code/env var maybe?)

- Make rating color nicer, but on white background for selector on new/edit brew page

- Profile page should show more details, and allow brew entries to take up more vertical space

- Show "view" button on brews in profile page (same as on brews list page)

- The "back" button behaves kind of strangely
  - Goes back to brews list after clicking on view bean in feed,
    takes to profile for other users' brews.

- Update terms page to be more clear about the public nature of all data
  - Link to about page and terms at the top of the unauthed feed
