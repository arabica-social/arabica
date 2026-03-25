# Option A: "Clean Craft" — Modern Minimal with Warmth

**Vibe:** A well-designed tool for coffee people. Clean surfaces, generous whitespace, subtle depth. Feels professional without losing the coffee identity.

**Core principle:** Remove visual noise so the *data* becomes the design.

## Design Direction

Strip away gradients, heavy shadows, and nested containers. Replace with flat white cards on a warm cream background. Let typography weight and spacing create hierarchy instead of color and depth.

The monospace font stays everywhere — it's the brand. But we size it down (monospace reads ~15% larger than proportional) and use weight contrast more aggressively to create hierarchy.

## Color Changes

| Token | Current | Proposed | Reason |
|-------|---------|----------|--------|
| Page background | `brown-50` (#fdf8f6) | `#FAF7F5` (new `cream`) | Warmer, less pink-tinted |
| Card background | gradient brown-100→200 | `#FFFFFF` flat white | Cards pop via contrast with cream bg |
| Card border | `brown-300` | `brown-200` | Lighter border, less "boxed in" |
| Card shadow | `shadow-xl` | `shadow-sm`, `shadow-md` on hover | Quieter at rest, responsive on interaction |
| Feed card bg | gradient brown-50→100 | `#FFFFFF` flat white | Same as primary cards |
| Table bg | gradient brown-100→200 | `#FFFFFF` with `brown-100` header | Clean, scannable |
| Modal bg | gradient brown-100→200 | `#FFFFFF` | Consistent with card treatment |

Keep the existing brown palette for text, borders, and accents. Keep amber for ratings and badges. The palette itself is good — the problem is overuse of mid-tones as backgrounds.

## Typography Changes

| Element | Current | Proposed |
|---------|---------|----------|
| Page title | `text-3xl font-bold` | `text-2xl font-semibold` |
| Section title | `text-2xl font-bold` | `text-base font-semibold uppercase tracking-wider text-brown-500` |
| Card heading | `text-xl font-bold` | `text-base font-semibold` |
| Body text | `text-base` | `text-sm` |
| Meta/labels | `text-xs` | `text-xs font-medium text-brown-500` |

Rationale: Monospace at `text-base` (16px) feels large. Dropping body to `text-sm` (14px) and headings proportionally gives the same visual weight as a proportional font at standard sizes.

Section titles become small uppercase labels — a common pattern in tools like Linear, Notion, and GitHub that creates clear sections without shouting.

## Card System

### Current
```
.card       = gradient bg + rounded-xl + shadow-xl + border-brown-300
.feed-card  = gradient bg + rounded-lg + shadow-md + border-brown-200
.section-box = bg-brown-50 + rounded-lg + border-brown-200
```

Three card types, all slightly different. Feed cards have a nested `.feed-content-box` inside them creating card-in-card.

### Proposed
```
.card       = bg-white rounded-xl border border-brown-200 shadow-sm
              hover:shadow-md transition-shadow
.card-sm    = bg-white rounded-lg border border-brown-200 shadow-sm
.feed-card  = bg-white rounded-lg border border-brown-200 shadow-sm
              hover:shadow-md transition-shadow
```

Key changes:
- **Kill the nested content box.** Feed card content lives directly in the card. No `.feed-content-box` wrapper.
- **One visual language.** All cards are white, rounded, with thin borders and minimal shadow.
- **Type indicator via left border.** Feed cards get a `border-l-3` colored by record type:
  - Brew: `brown-700`
  - Bean: `amber-600`
  - Recipe: `brown-500`
  - Roaster/Grinder/Brewer: `brown-400`

This replaces the need for emoji or labels to distinguish record types at a glance.

### Section box
```
.section-box = bg-brown-50/50 rounded-lg p-4
               (no border — just the subtle background tint)
```

Used inside detail views for grouping related data. Lighter treatment than a card.

## Button System

### Current (5 variants)
```
.btn-primary   = gradient brown-700→900 + shadow-md
.btn-secondary = bg-brown-300
.btn-tertiary  = gradient brown-500→600
.btn-link      = text underline
.btn-danger    = red text underline
```

### Proposed (3 variants)
```
.btn-primary   = bg-brown-800 text-white rounded-lg
                 hover:bg-brown-900 transition-colors
.btn-secondary = bg-white border border-brown-300 text-brown-700 rounded-lg
                 hover:bg-brown-50 transition-colors
.btn-danger    = text-red-600 hover:text-red-800 underline
```

Drop gradients on buttons. Drop `.btn-tertiary` — audit uses and convert to primary or secondary. The link-style `.btn-link` merges into the general `.link` class.

## Form System

### Current
```
.form-input = border-2 border-brown-300 rounded-lg
              focus:border-brown-600 focus:ring-brown-600
```

### Proposed
```
.form-input = border border-brown-300 rounded-lg bg-white
              focus:border-brown-600 focus:ring-1 focus:ring-brown-600
              focus:bg-brown-50/30
              placeholder:text-brown-400
```

Changes:
- Border from 2px to 1px (less heavy)
- Subtle background tint on focus (instead of just border change)
- Keep the focus `translateY(-1px)` lift — it's a nice touch
- Ring reduced to `ring-1` (thinner, more refined)

## Shadow Depth Scale

| Level | Class | Usage |
|-------|-------|-------|
| 0 | `shadow-none` | Flat surfaces, inline elements |
| 1 | `shadow-sm` | Cards at rest, tables, section boxes |
| 2 | `shadow-md` | Hovered cards, dropdowns, action menus |
| 3 | `shadow-lg` | Modals, popovers, floating UI |

Current uses shadow-sm through shadow-2xl inconsistently. This simplifies to 3 levels with clear rules.

## Navigation

**Keep** the dark brown gradient header — it's a strong anchor that works well.

**Refine:**
- Reduce ALPHA badge prominence (smaller, `text-[10px]`)
- Replace hard `border-b` with `shadow-sm` for softer separation
- Shrink header height slightly (48px instead of ~56px)

## Feed Cards — Detailed Layout

```
┌─ border-l-3 brown-700 ─────────────────┐
│                                          │
│  ○ Display Name · @handle · 2h          │
│  brewed with Ethiopian Sidamo            │
│                                          │
│  ┌─ bg-brown-50/50 ──────────────────┐  │
│  │ Sweet Bloom  ·  V60               │  │
│  │ 15g → 250g  ·  1:16.7  ·  ⭐ 8.5 │  │
│  │                                    │  │
│  │ "Bright citrus, chocolate finish"  │  │
│  └────────────────────────────────────┘  │
│                                          │
│  💬 3    ♡ 12    ↗ Share                 │
└──────────────────────────────────────────┘
```

The inner area uses a section-box (subtle bg tint, no border) instead of the current bordered content box. Action bar has no top border — just spacing.

## Table Styling

### Current
```
.table-container = gradient bg + shadow-md + border
.table-header    = bg-brown-200
.table-body      = bg-brown-100
```

### Proposed
```
.table-container = bg-white rounded-lg border border-brown-200 shadow-sm overflow-hidden
.table-header    = bg-brown-50
.table-body      = bg-white divide-y divide-brown-100
.table-row       = hover:bg-brown-50 transition-colors
```

Clean, standard table styling. Header is barely tinted. Rows divide with thin lines. Hover highlights the row.

## Animations

**Keep:** Staggered feed card entry, modal transitions, like pop/shrink, form focus lift.

**Remove:** Table row stagger (too much motion for data tables — feels gimmicky).

**Add:** Subtle `opacity` transition on card border-left color when filtering feed by type.

## Implementation Phases

### Phase 1: Foundation (CSS-only, no template changes)
1. Update `tailwind.config.js` — add `cream` color
2. Rewrite card/button/form/table classes in `app.css`
3. Update `layout.templ` body background
4. Bump CSS cache version

### Phase 2: Template Cleanup
1. Remove `.feed-content-box` wrappers from feed templates
2. Add `border-l-3` type indicators to feed cards
3. Simplify section titles (uppercase label pattern)
4. Remove `.btn-tertiary` uses

### Phase 3: Detail Polish
1. Refine form layouts
2. Update modal content styling
3. Audit shadow usage across all templates
4. Test responsive behavior

## Tradeoffs

| Pro | Con |
|-----|-----|
| Immediately more professional | Less personality than current |
| Easier to maintain (fewer special cases) | Could feel generic if not careful |
| Better feed scannability at scale | Left-border type system is a new concept to learn |
| Lighter page weight (no gradients) | White cards on cream is a common pattern |
| Clear visual hierarchy | Less "cozy coffee shop" feeling |
| Works well with future dark mode | Requires discipline to not drift back to decoration |

## Risk: Becoming Generic

The biggest risk with clean minimal is looking like every other SaaS tool. Mitigations:
- **Monospace font** is the primary differentiator — keep it everywhere
- **Brown palette** prevents blue/gray sameness
- **Grain overlay** (keep at current opacity) adds tactile quality
- **Left-border accents** give the feed a distinctive pattern
- **Dark nav** provides a strong visual anchor

The identity comes from the font + color combination, not from gradients and shadows.
