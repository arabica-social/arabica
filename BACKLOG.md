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
  - Use the non-cgo library?
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

## Notes

- Popup menu for feed card extras should be centered on the button
  - Maybe use a different background color (maybe the button color?)

- Add a copy AT URI to extras popup

- Firehose maybe not backfilling likes

- TODO: add OpenGraph embeds (mainly for brews; beans and roasters can come later)

- Fix opengraph to show handle, record type and date?
  - Then show brewer and bean?
  - Add an image of some kind as well
