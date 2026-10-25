# Product Rethink: Topics to Refine

Status: Open exploration, not an implementation roadmap.

Arabica's proposed direction puts more emphasis on community, sharing beans
and roasters, and acknowledging the producers whose work makes each cup
possible. Participation should require less setup and less technical knowledge.
AI-assisted capture and cafe support are part of this exploration.

Use this document to track unresolved product questions. Existing `PRODUCT.md`
is not a constraint for this exercise. Do not treat the possibilities below as
approved features or changes to persisted record contracts.

## Brew Logging: Plan Drafted

- [x] Outline Quick/Detailed logging, method-aware fields, and optional grinder
  selection with dependent grind settings.
- [ ] Implement and verify the separate
  [Quick/Detailed brew-form plan](2026-10-25-brew-form-quick-detailed.md).

That plan covers logging simplification, not the broader topics below.

## 1. Sharing Beans and Roasters

- [ ] Decide what the main contribution is: a personal coffee experience, a
  recommendation, or something saved for later.
- [ ] Define a useful experience for someone who never logs a brew.
- [ ] Distinguish sharing a bag from sharing a roaster and from recording a cup.
- [ ] Decide how recommendations can lead to saves, purchases, or conversations
  without requiring purchase tracking or adding pressure to buy.

Useful exercise: describe how someone shares a bag they love without recording
how they brewed it.

## 2. AI-Assisted Capture

- [ ] Choose the first capture task and input: bag photos, cafe/menu photos,
  pasted links, or natural-language brew notes.
- [ ] Design the path from upload to review to save, including the manual path.
- [ ] Define which facts AI may extract, which relationships it may suggest,
  and what requires explicit user confirmation.
- [ ] Decide the minimum useful draft and how to handle unknown information
  without making the user complete another long form.
- [ ] Define how to distinguish packaging claims, producer-authored material,
  and the user's own observations. AI must not invent personal impressions or
  unsupported producer attribution.
- [ ] Explore matching existing beans, roasters, cafes, or producers without
  silently creating duplicates or merging distinct entities.
- [ ] Decide what happens to original images, including storage, publication,
  processing-provider disclosure, and user control over retention.
- [ ] Set expectations for latency, cost, correction, and extraction failure.
- [ ] Test the first proposed flow against representative real inputs before
  choosing an implementation architecture.

A separate photo-assisted “Add a bag” flow is a candidate. Inline bean creation
inside the brew form is not part of the agreed logging plan.

## 3. Cafes

- [ ] Define the supported actions: recommend a cafe, share a coffee served
  there, or record where a bag was purchased.
- [ ] Decide whether sharing a cafe visit requires a drink or brew record.
- [ ] Allow for visitors who do not know the beans, roaster, or producer.
- [ ] Clarify cafe identity, including individual locations versus a business
  with multiple locations, before designing references.
- [ ] Translate agreed behavior into lexicon requirements using the repository's
  entity-change and record-evolution workflows.

Cafe support is desired, but its product behavior and record contracts remain
undecided. Existing plans are background, not evidence of shipped support.

## 4. Producer Recognition

- [ ] Decide where attribution appears: coffee cards, bag creation, shared
  posts, detail pages, and discovery.
- [ ] Decide whether producers begin as attributed information on a coffee or
  need separate identities and pages.
- [ ] Account for individual producers, farms, cooperatives, and multiple
  contributors without assuming one coffee has one farmer.
- [ ] Define source links and correction behavior for producer information.
- [ ] Explore discovering coffees from the same producer across roasters.
- [ ] Make recognition possible without demanding supply-chain research from
  the person sharing or presenting marketing claims as verified facts.

Mindfulness here means acknowledging producers' importance. It does not imply
an ethical purchasing score, certification claim, or requirement to judge other
people's coffee choices.

## 5. Community

- [ ] Choose the exchanges to encourage: recommendations, shared discoveries,
  questions, and responses such as “I tried this because of you.”
- [ ] Decide whose activity matters first: friends, people with similar tastes,
  local drinkers, roasters, or producers.
- [ ] Explore community value without constant posting, numerical competition,
  rare-bag collecting, or pressure to purchase new coffee.
- [ ] Define value for someone whose friends have not joined yet.
- [ ] Consider audience controls and moderation alongside sharing behavior.

Useful exercise: describe a satisfying week of use for someone who drinks the
same coffee every morning and rarely posts.

## 6. Getting Started Beyond Brew Logging

- [ ] Define what a newcomer can do before adding beans or equipment.
- [ ] Compare entry paths: browse recommendations, save a coffee, share a cafe,
  or add the bag currently at home.
- [ ] Decide when sign-in and setup are necessary rather than front-loading
  every prerequisite.
- [ ] Test first use with a casual coffee drinker and a brewing enthusiast.
- [ ] Keep the distinction between prerequisites for a brew log and
  prerequisites for participating in Arabica.

## How to Use This List

Start with AI-assisted capture. Record decisions and remaining questions here
as the discussion progresses; create a focused plan once an experience is
clear enough to implement. Update FDRs, API docs, or product principles only
when their corresponding behavior or decisions become current.
