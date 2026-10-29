# Arabica interface guidance

Preserve Arabica's warm palette and established typography. Prioritize readable
coffee notes, clear controls, and space for the content. This document is a
design reference, not a template that every page must reproduce.

## Visual direction

Take cues from specialty coffee packaging and analog brew journals: confident
type, earthy colors, and tactile surfaces. Familiar controls keep logging easy;
craft details give the journal and community feed their character. Use those
details where they suit the content, without decorating every control.

Avoid generic SaaS dashboards and corporate productivity styling, cold developer
aesthetics, and personality-free social feed clones. Data-heavy views should
still feel like part of a coffee journal. Warmth and community matter alongside
clarity; reducing copy is not a reason to strip away the visual identity.

## Colors and typography

- Use the existing theme tokens for surfaces, text, borders, and interaction
  states. Brown and cream are the base palette in light mode; dark mode uses
  the corresponding warm dark surfaces. Do not hard-code light-only colors.
- Iosevka Patrick is the reading and UI font, including forms, tables, feed
  content, and explanatory pages. Fraunces is for short display moments such
  as page and record titles, not data or compact labels. Do not add another
  typeface.
- Use entity colors to identify entity types, and semantic colors for warnings
  and status. Decorative color should not compete with those meanings.
- Keep body copy legible and paragraphs to a comfortable reading width, roughly
  65–75 characters. Small uppercase labels are optional and should add context
  that the heading does not already provide.

The token definitions in `internal/web/assets/css/tokens.css` are the source
of truth for actual values.

## Layout

Use headings and panels to clarify content. Keep the visual details people
like without surrounding each one with explanatory copy. Empty space does not
need another paragraph.

- Start with a page title, necessary context, and the primary action. Home uses
  one Fraunces H1, “Community,” with Log Brew alongside for ready accounts.
  Signed-in feed filters sit beneath it within the masthead, and the feed
  follows directly without a separate “Recent activity” heading. Omit the
  returning-user eyebrow, slogan, introduction, and secondary journal link.
  Keep setup guidance and visitor introduction only where needed for those
  states, as specified in [FDR-005](docs/fdr/FDR-005-home.md).
- Home retains the original masked bean SVG at opacity `.045`, positioned
  independently of prose. On desktop it is `20rem` square at `top: -.7rem` and
  `right: 3rem`. At widths up to `719px`, it is `15rem` square at `top: auto`,
  `bottom: -6.5rem`, and `right: 0`, clipped at the existing masthead divider.
  Keep masthead height content-driven; do not add a minimum height or extra
  padding to accommodate the artwork. Check all account states at narrow and
  wide widths.
- Home's icon navigation provides useful shortcuts; keep Feedback and About
  easy to find. Omit empty data sections without making the remaining content
  harder to reach.
- Use cards for meaningful groups or independent entries. Prose usually needs
  headings and spacing, not nested boxes or feature grids.
- Keep borders and shadows modest. Avoid decorative gradients, glass effects,
  and dashboard-style metric tiles without a user need. Keep the homepage bean
  illustration faint and behind the content.
- Do not require numbered sections, tilted elements, progress stamps, entrance
  animations, or extra eyebrow headings. Existing examples are not mandates
  for new work.

## Established patterns

Reuse these where they clarify the content or state. They are options, not a
checklist for every page.

- Entity-focused cards can use `--type-{entity}` for borders, badges, or active
  states and `--type-{entity}-tint` for a restrained wash. In mixed collections,
  keep each entity's identity rather than applying one accent to the whole
  page. Color must not be the only way to identify the type.
- Kraft texture can warm a page-level container; cork and paper-note surfaces
  give the community feed a distinct setting. Use the existing `--texture-kraft`
  and `--texture-cork` tokens and check readability in both themes. Utility pages
  do not need the same treatment.
- Setup progress should distinguish required, optional, current, and completed
  states with text and clear visual cues. Existing dashed and filled stamp
  treatments can help. Name the missing prerequisites and change the primary
  action when it becomes available; do not substitute a decorative step count
  for useful guidance.
- Empty states name a concrete next action when one is available, such as
  "Add a bean." If the viewer cannot add records, explain that state without
  offering an unavailable action. "Add another" can shorten a repeated action
  once the entity is clear from context.

## Controls and content

Reuse established buttons, inputs, dialogs, and page components in
`web/src/lib/components/`. Their names may reflect older visual treatments;
reuse their behavior without adding decorative framing to match the name.

- Buttons name their action. Use one primary page action; make other actions
  visually secondary. Repeated navigation is justified only by a real access
  need, such as mobile navigation.
- Preserve required/optional indicators, validation messages, and explanations
  of privacy or cross-app effects. Keep them next to the relevant control.
- Empty, loading, and error states must be distinct. Do not describe a failed
  request as an empty collection.
- Use icons consistently, with text or accessible names for controls. Do not
  introduce emoji as replacements for the established UI icons.
- Write direct copy as described in [PRODUCT.md](PRODUCT.md). Do not add slogans
  or encouragement merely to fill a layout.

## Accessibility and responsive behavior

- Use semantic headings and landmarks, labeled controls, visible keyboard
  focus, and usable contrast in both themes.
- Keep primary navigation and action targets at least 44px tall. Do not rely
  on hover or color alone to convey state.
- Keep reading and focus order sensible when columns collapse. Mobile users
  should reach the main task before optional supporting content.
- Allow text to wrap and controls to reflow at narrow widths and browser zoom.
- Motion should explain state changes, not decorate page loading. Respect
  `prefers-reduced-motion` wherever motion exists.

## Implementation and checks

Component-specific styles belong with the Svelte component. Shared styles live
in `internal/web/assets/css/components/`; numbered files preserve cascade order.
Do not add another token source or generated design specification.

After layout changes, check signed-in and signed-out states where applicable,
empty and populated content, narrow and wide screens, and both themes. Run
Svelte checks and relevant tests; inspect rendered pages for hierarchy and
layout issues that unit tests cannot catch. Update visual baselines only after
reviewing the new rendering.
