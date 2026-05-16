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

- [ ] Cafe and drink lexicons
  - [ ] Drink could be an extension of brew for cafes and _maybe_ for home for
        milk/other drinks?

- [ ] Replace placeholder design assets
  - [ ] Make a new logo
  - [ ] Other graphical assets (hero, banner, probably other stuff)
  - [ ] Maybe replace the icon used for bean variety and tea leaf (it sucks)

- [ ] Improve the about and atproto pages

## Breaking Changes

Brew lexicon changes:

- `coffeeAmount` goes from gram precision to decigrams (breaking)
  - Stored numbers increase by 10x
- `waterAmount` is removed
  - Moved to method params

Recipe lexicon changes:

- TBD but probably gets completely overhauled
