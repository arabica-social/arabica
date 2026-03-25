# Option B: "Roasted" — Bold, Dark, Editorial

**Vibe:** Specialty coffee packaging meets editorial magazine. Dark surfaces with warm highlights. The app *feels* like coffee — like opening a bag of fresh beans.

**Core principle:** High contrast and bold typography make data dramatic.

## Design Direction

Invert the current model. Instead of brown-tinted light backgrounds, go dark. Deep espresso-brown as the base with cream/white cards floating above. The dark ground creates natural depth without shadows. Typography goes bigger and bolder — every page has a clear headline moment.

The monospace font becomes a statement rather than a quirk. At large sizes against dark backgrounds, monospace reads as intentional and editorial.

## Color System

### New Dark Palette

| Token | Hex | Usage |
|-------|-----|-------|
| `espresso-950` | `#0F0A08` | Deepest background (page bg) |
| `espresso-900` | `#1C1210` | Card backgrounds, primary surface |
| `espresso-850` | `#241A16` | Elevated surfaces, hover states |
| `espresso-800` | `#2E211B` | Borders, dividers |
| `espresso-700` | `#3D2D24` | Secondary borders, subtle elements |
| `cream-50` | `#FAF7F5` | Primary text on dark |
| `cream-100` | `#F2E8E0` | Secondary text on dark |
| `cream-200` | `#E0CEC4` | Muted text, labels |
| `cream-300` | `#C4A898` | Placeholder text, disabled |
| `amber-400` | `#FBBF24` | Primary accent — ratings, highlights |
| `ember-500` | `#C4553A` | Secondary accent — actions, CTAs |
| `ember-600` | `#A3412D` | Hover state for ember accent |

### Usage Rules

- **Dark surfaces layered:** Page (`950`) → Section (`900`) → Card (`850`) creates depth without shadows
- **Warm borders:** `espresso-800` borders, never gray
- **Text contrast:** `cream-50` for headings (≥7:1 ratio), `cream-100` for body (≥4.5:1), `cream-200` for meta
- **Accent restraint:** Amber for data (ratings, stats). Ember for interactive (buttons, links). Never both in the same element.
- **No gradients on surfaces.** Flat dark colors. Gradients only on accent elements (primary button, hero treatments).

## Typography Changes

| Element | Current | Proposed |
|---------|---------|----------|
| Page title | `text-3xl font-bold` | `text-3xl font-semibold text-cream-50 tracking-tight` |
| Section title | `text-2xl font-bold` | `text-lg font-semibold text-amber-400 uppercase tracking-widest` |
| Card heading | `text-xl font-bold` | `text-lg font-semibold text-cream-50` |
| Body text | `text-base` | `text-sm text-cream-100` |
| Meta/labels | `text-xs` | `text-xs font-medium text-cream-300 uppercase tracking-wider` |
| Data values | same as body | `text-sm font-medium text-cream-50 tabular-nums` |

Key difference from Option A: **Section titles use amber accent** as a color label, giving each section a warm highlight. Data values get their own treatment — medium weight, tabular numbers — because in a coffee tracking app, the numbers *are* the content.

## Card System

### Proposed
```
.card       = bg-espresso-900 rounded-xl border border-espresso-800
              (no shadow — depth comes from surface layering)
.card-hover = hover:bg-espresso-850 hover:border-espresso-700 transition-colors
.feed-card  = bg-espresso-900 rounded-lg border border-espresso-800
              hover:bg-espresso-850 transition-colors
```

**No shadows at all on cards.** Dark themes get depth from layered surface colors (Material Design 3 dark theme pattern). Shadows on dark backgrounds look muddy.

**Content areas** inside cards use a slightly lighter surface:
```
.card-inset = bg-espresso-850 rounded-lg p-3
              (no border — just the shade difference)
```

This replaces both `.section-box` and `.feed-content-box` — a single "recessed area" concept.

### Type indicator
Instead of left-border (which can get lost on dark), use a **small colored dot** before the action text:
- Brew: `amber-400` dot
- Bean: `cream-200` dot
- Recipe: `ember-500` dot

Or a subtle top-border accent (2px) on the card — visible but not dominant.

## Button System

### Proposed
```
.btn-primary   = bg-gradient-to-r from-ember-500 to-ember-600 text-cream-50 rounded-lg
                 hover:from-ember-600 hover:to-ember-600 transition-all
.btn-secondary = bg-espresso-850 border border-espresso-700 text-cream-100 rounded-lg
                 hover:bg-espresso-800 hover:border-espresso-700 transition-colors
.btn-danger    = text-red-400 hover:text-red-300 underline
```

Primary button gets the one gradient in the system — the warm ember accent. This makes CTAs unmistakable against the dark background. Secondary buttons are ghost-style (slightly lighter than the surface).

## Form System

```
.form-input = bg-espresso-850 border border-espresso-700 rounded-lg
              text-cream-50 placeholder:text-cream-300
              focus:border-amber-400 focus:ring-1 focus:ring-amber-400
```

Dark inputs with amber focus ring. The focus state is dramatic and clear — amber on dark brown is high contrast without being harsh.

**Form labels:** `text-cream-200 text-xs font-medium uppercase tracking-wider` — small, quiet, functional.

## Navigation

### Proposed
The current dark header is already close to the right direction. Refinements:

```
Header bg: espresso-950 (deepest dark, matches page)
           with a subtle bottom border in espresso-800
```

Since the page is now dark too, the header blends seamlessly. It's separated by the border, not a background change. This creates a more immersive, app-like feel.

**ALPHA badge:** `bg-amber-400 text-espresso-950` (inverted — amber background, dark text). Small, punchy.

**User dropdown:** `bg-espresso-900 border border-espresso-800` — matches card styling.

## Feed Cards — Detailed Layout

```
┌─ bg-espresso-900 border-espresso-800 ───┐
│                                          │
│  ○ Display Name · @handle · 2h          │
│  ● brewed with Ethiopian Sidamo          │  ← amber dot for brew type
│                                          │
│  ┌─ bg-espresso-850 ─────────────────┐  │
│  │ Sweet Bloom  ·  V60               │  │  ← cream-100 text
│  │ 15g → 250g  ·  1:16.7            │  │  ← cream-50 data values
│  │                                    │  │
│  │ ⭐ 8.5                            │  │  ← amber badge
│  │                                    │  │
│  │ "Bright citrus, chocolate finish"  │  │  ← cream-200 italic
│  └────────────────────────────────────┘  │
│                                          │
│  💬 3    ♡ 12    ↗ Share                 │  ← cream-300, hover cream-50
└──────────────────────────────────────────┘
```

The inset area (`.card-inset`) provides visual grouping without borders. On dark backgrounds, even a small shade difference reads clearly.

## Table Styling

```
.table-container = bg-espresso-900 rounded-lg border border-espresso-800 overflow-hidden
.table-header    = bg-espresso-850 border-b border-espresso-800
.table-th        = text-cream-300 text-xs font-medium uppercase tracking-wider
.table-body      = divide-y divide-espresso-800
.table-row       = hover:bg-espresso-850 transition-colors
.table-td        = text-cream-100 text-sm
```

Clean, dark table. Header row is barely differentiated. Dividers between rows. On hover, rows lighten slightly.

## Animations

**Keep:** Staggered feed card entry, modal transitions, like pop/shrink.

**Modify:**
- Feed card entry: Use `opacity` + `translateY(6px)` (shorter travel on dark — movement reads more clearly against dark backgrounds)
- Modal backdrop: `bg-black/60` (needs to be darker since the page is already dark)

**Add:**
- Subtle `glow` on amber accent elements: `box-shadow: 0 0 20px rgba(251,191,36,0.1)` — very subtle warm halo
- Card hover: border transitions from `espresso-800` to `espresso-700` (warm reveal)

**Remove:** Table row stagger, form focus lift (feels odd on dark).

## Texture & Atmosphere

**Grain overlay:** Increase from `0.025` to `0.04` opacity. Grain reads better on dark backgrounds and adds significant tactile quality. This is one of the biggest differentiators — the paper-grain-on-dark effect feels like a coffee bag or craft packaging.

**Optional:** Subtle warm vignette on the page body:
```css
body::after {
  content: "";
  position: fixed;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(ellipse at center, transparent 50%, rgba(15,10,8,0.3) 100%);
}
```

This darkens the edges slightly, creating a cozy, focused feel. Can be skipped if it feels heavy.

## Implementation Phases

### Phase 1: Foundation
1. Extend `tailwind.config.js` with `espresso` and `cream` color scales
2. Rewrite `app.css` component classes for dark surfaces
3. Update `layout.templ` body background + text colors
4. Update `header.templ` to match dark theme
5. Bump CSS cache version

### Phase 2: Template Updates
1. Update all page templates — swap brown-* text utilities to cream-*
2. Replace `.feed-content-box` with `.card-inset`
3. Update form styling (dark inputs, amber focus)
4. Update modal styling

### Phase 3: Polish
1. Audit contrast ratios (WCAG AA minimum)
2. Add subtle glow effects on accent elements
3. Tune grain overlay opacity
4. Test all states (hover, focus, active, disabled) on dark

### Phase 4: Accessibility Audit
Dark themes have higher risk of contrast failures. Must verify:
- All text meets WCAG AA (4.5:1 for body, 3:1 for large text)
- Focus indicators are visible
- Disabled states are distinguishable
- Form validation errors are readable

## Tradeoffs

| Pro | Con |
|-----|-----|
| Extremely distinctive — memorable identity | Harder to implement correctly (contrast, accessibility) |
| Dark mode is practical (morning/evening brew logging) | More CSS to maintain (dark needs different strategies) |
| High contrast makes data pop | Polarizing — some users hate dark UIs |
| Grain texture reads beautifully on dark | Heavier visual treatment, more code for atmosphere |
| Feels premium, like specialty coffee packaging | Template changes are more extensive (every text color) |
| Monospace font becomes a bold statement | No easy "light mode" toggle without a full second theme |
| Natural depth from surface layering (no shadows needed) | Photos/avatars need extra treatment to not look jarring |

## Risk: Too Dark / Oppressive

The biggest risk is the UI feeling heavy or hard to read. Mitigations:
- **Cream text, not white.** Pure white (#fff) on dark brown is harsh. Warm cream (#FAF7F5) reduces eye strain.
- **Layered surfaces** prevent "black void" feeling — there's always subtle differentiation.
- **Amber accents** add warmth and break up the dark expanse.
- **Generous spacing** — dark UIs need more whitespace (darkspace?) to breathe than light ones.
- **Grain overlay** prevents "screen" feeling, adds organic quality.

## Risk: Light Mode Demand

If users request light mode later, you'd need to:
1. Define all component colors via CSS custom properties (not Tailwind classes directly)
2. Create a parallel set of light-theme values
3. Use `prefers-color-scheme` or a toggle

This is significant work. If you think light mode will be needed within 6 months, Option A is a safer starting point (and can *add* dark mode later more easily than B can add light mode).
