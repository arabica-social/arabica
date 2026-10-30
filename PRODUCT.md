# Arabica

Arabica is a coffee journal for home brewers. People use it to remember what
worked, keep track of beans and equipment, and compare notes with other people.
It should be easy to log a cup on a phone and useful to browse on a larger screen.

## Audience and character

Arabica serves casual home brewers and detail-minded hobbyists. Morning logging
on a phone should be quick; browsing brew history or community notes on desktop
can reward a closer look. This is an unhurried hobby companion.

The personality is cozy, social, and inviting, with the warmth of a neighborhood
specialty café. Aim for calm satisfaction after logging a good cup and geeky
delight in the details of a recipe. People should feel pride in their craft and
welcome in the community, without pressure to perform or keep up.

## Priorities

- Make logging and finding previous brews straightforward. Keep the primary
  action visible without surrounding it with a pitch.
- Preserve the character people enjoy: display typography, coffee illustrations,
  and icon navigation. Trim repeated explanations rather than removing those
  details. A short headline can stand on its own.
- Help beginners understand required information without getting in the way of
  experienced users. Keep optional details optional.
- Make community activity easy to browse without pressure, streaks, or other
  gamification.
- Explain privacy, shared-account effects, and data limitations where they
  affect a decision. A shorter page is not worth hiding an important warning.

## Writing

Use the words people use for their coffee: brews, beans, recipes, and equipment.
Use technical terms such as PDS or DID when explaining the protocol, not as
ordinary navigation labels. Explain unfamiliar terms on first use.

Headings identify the content or task. Buttons name the action. Help text adds
information the label cannot convey. Remove text that only repeats a nearby
heading or button. An empty state should explain what can happen next without
inventing an action that is unavailable in that state.

Warmth does not require a café metaphor in every section. Avoid generic
encouragement, ownership slogans, and repeated introductions. A feedback link
can say "Feedback."

## Page priorities

- Home opens with a Community masthead and the primary action alongside it.
  Signed-in feed filters sit beneath the title; the feed follows directly.
  Ready accounts need no introduction, slogan, eyebrow, or secondary journal
  link. Setup-incomplete accounts get one Finish setup action and readiness
  guidance. Visitors get Log in, a separate Create an account link, and
  “Track brews, beans, and recipes.” Keep account-storage explanations on the
  AT Protocol and About pages, not in the masthead. Preserve the bean artwork
  without adding empty space to accommodate it.
- My Coffee is the user's collection. Navigation, search, and logging take
  precedence over summaries of what a collection is. Keep Refresh beside the
  collection navigation, not in a separate equipment-summary or troubleshooting
  rail.
- Forms and setup explain requirements and next actions. Do not repeat the same
  instruction in a page introduction, a card description, and an empty state.
  The general Add page presents independent choices without onboarding's
  numbering or prerequisite badges; setup retains meaningful required and
  optional states.
- About explains the app and how to start. The AT Protocol page owns the deeper
  explanation of accounts and data storage.

## Scope and contracts

The PDS is authoritative. Existing records, visibility boundaries, and shared
account behavior must survive presentation changes. Current feature behavior
belongs in [docs/fdr/](docs/fdr/INDEX.md); protocol and API contracts belong in
[docs/architecture/](docs/architecture/INDEX.md) and [docs/api/](docs/api/README.md).

See [DESIGN.md](DESIGN.md) for visual and accessibility guidance.
