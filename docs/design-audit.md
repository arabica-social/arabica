# Design Audit — April 2026

## What's Already Working

- **Coffee-tinted shadows** (`rgba(61,35,25,...)` instead of black) — subtle, distinctive
- **Like-pop animation** — multi-point spring scale, genuinely delightful
- **Feed card stagger** — nth-child delays create alive-feeling lists
- **Dark mode** — thoughtful (amber focus rings, warm undertones preserved, not lazy inversion)
- **Custom Iosevka Patrick font** — gives personality that system fonts can't
- **Filter pill hover/active pattern** — outline previews type color, fills on select
- **Brown palette** — custom, not Tailwind defaults; follows coffee color progression
- **CSS variable architecture** — 100+ design tokens, robust theme switching

## Core Problem

Arabica has strong *tokens* (colors, font, shadows) but **generic *patterns***. The
brown palette is distinctive, but it's applied to layouts that could be any admin
dashboard: same-sized cards in grids, identical icon+label metadata rows repeated
30+ times, flat tab navigation, uniform spacing. The "craft coffee journal" feeling
lives in the color tokens but not in the structure or typography.

## High-Impact Opportunities

### 1. Typography — Display Font Pairing

Iosevka does everything: headings, body, labels, data. No typographic *contrast*.
A warm display face for headings paired with Iosevka for body/data would separate
"content you read" from "data you scan."

Brand words: cozy, textured, handcrafted. Think: the typeface on a specialty coffee
bag label — confident, slightly imperfect, warm. Not a geometric sans. Something
with character like Bitter (earthy slab serif), Vollkorn (warm old-style), or
Zilla Slab (humanist slab — sturdy, approachable, uncommon).

### 2. Entity Cards Need Visual Identity

Bean, Roaster, Grinder, Brewer, Recipe cards are structurally identical — same
layout, same icon+label grid, same spacing. They should feel like different kinds
of objects in a collection:

- **Bean cards**: roast-level color indicator, origin as visual badge
- **Brewer cards**: method icon (V60, espresso portafilter, AeroPress)
- **Recipe cards**: different proportions, recipe-card feel
- **Roaster cards**: location as primary visual element

### 3. Feed Cards — Replace the Left-Stripe

The 3px `border-left` colored stripe is the most recognizable Arabica pattern, but
it's also one of the most generic dashboard conventions. The type-color system
(brew=#6b4423, bean=#b45309, etc.) is good — needs a less templated expression:

- Small colored dot or pill badge with type name
- Subtle background tint (bean cards get faint amber wash)
- Type icon in card header instead of a border

### 4. Spacing Rhythm

Almost everything is `p-4` or `p-6`. Pages feel evenly padded rather than composed.
Brew view should breathe differently than a management table. Rating deserves more
space. Section transitions need varied gaps — tight within, generous between.

### 5. Welcome/Landing Experience

Unauthenticated visitors see a login form and bullet points. No hero moment, no
imagery, no personality. First impression should feel like picking up a beautiful
bag of coffee, not opening a SaaS tool.

### 6. Empty States & Loading

Skeletons are generic pulse animations. Empty states are plain text. Missed
personality moments — "No beans yet. Time to visit your local roaster."

### 7. Form Modals Are Interchangeable

All 5 entity creation dialogs identical. Fields could be grouped semantically.
Required vs optional have no visual distinction. Forms work but feel like database
admin, not a coffee journal.

## Medium-Impact Opportunities

- **Buttons too uniform** — every button is `btn-primary`, no hierarchy
- **Section headers** use `uppercase tracking-wider` as visual crutch
- **Rounded corners everywhere** — no sharp edges for texture contrast
- **No micro-interactions** beyond like-pop — card hovers only change shadow

## Suggested Priority

1. Display font pairing (foundational — touches every page)
2. Feed card + entity card visual identity (most-seen components)
3. Welcome/landing redesign (first impression)
4. Spacing rhythm on key pages (brew view, feed)
