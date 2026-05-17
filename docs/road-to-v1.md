# V1 Release Checklist and Changes

## Release Checklist

- [ ] Add optional links to bean and gear lexicons
- [ ] Switch precision of coffee amount in brew to decigrams (and any other
      places, I think this is the only one though) (BREAKING)

- [ ] Improve and add method-specific params to brew lexicon and form
  - [ ] Move water amount to method params

- [ ] Improve recipe system (probably BREAKING)
  - [ ] Brainstorm ways to make recipes more modular
  - [ ] Investigate partial recipes, allow using multiple in one brew.
    - User could have one for their brewer and pours, one for bean and grinder
      and grind size, and anything in between.
  - [ ] Recipes should support non-pourover methods better
  - [ ] Link to another user's recipe in brew view page is broken if "Use this
        recipe in brew" was used from another user's recipe.

- [ ] Cafe and drink lexicons
  - [ ] Drink could be an extension of brew for cafes and _maybe_ for home for
        milk/other drinks?

- [ ] Replace placeholder design assets
  - [ ] Make a new logo
  - [ ] Other graphical assets (hero, banner, probably other stuff)
  - [ ] Maybe replace the icon used for bean variety and tea leaf (it sucks)

- [ ] Improve the about and atproto pages

- [ ] Drop boltdb
  - We don't really need it, join requests should live in a pds sidecar service,
    and boltdb doesn't really earn its keep for storing only oauth sessions
  - Make the OAuth session db change in the V1 move to avoid session disruption
    during alpha (since full release will require reauth anyway)

- [ ] Explore page
  - Find things other people rated high that you might like based on your
    history

- [ ] Follows (social graph), alternative following feed
  - Should this use the Bluesky soical graph? maybe? maybe allow importing it?
  - Custom feeds would be cool to support, probably not something for v1 though

- [ ] Make it so "missing roaster" warning can be dismissed
  - Beans shouldn't need a roaster (see "random blend of 4 beans blend"), but it
    should be strongly encouraged to have one. This could probably be solved in
    a better way by just removing the roaster/bean creation from the brew form
    and adding an onboarding flow or warnings that prevent brew creation when a
    user has no beans.

- [ ] Comments tweaks
  - [ ] Make threads look nicer
  - [ ] Allow threads more than 3 messages deep
  - [ ] Make comments collapsible

## Breaking Changes

Brew lexicon changes:

- `coffeeAmount` goes from gram precision to decigrams (breaking)
  - Stored numbers increase by 10x
- `waterAmount` is removed
  - Moved to method params

Recipe lexicon changes:

- TBD but probably gets completely overhauled (quite possibly before v1 comes
  out)
