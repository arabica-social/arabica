# Arabica Design Context

## Design Context

### Users
Mixed audience ranging from casual home brewers to serious hobbyists. The core user cares about their coffee craft — they track beans, dial in recipes, and want to see what the community is brewing. They use Arabica on phones (morning brew logging) and desktop (exploring data, browsing the feed). Context is unhurried — this isn't a work tool, it's a hobby companion.

### Brand Personality
**Cozy, social, inviting** — like a neighborhood specialty cafe where regulars share what they're drinking. Not clinical or corporate. The warmth comes from the subject matter (coffee, ritual, craft) and the community layer (AT Protocol federation, shared feed, likes/comments).

### Emotional Goals
- **Calm satisfaction**: The quiet pleasure of a well-made cup — unhurried, grounded, warm
- **Geeky delight**: Data is fun, details are rewarding — dialing in extraction is part of the joy
- **Community belonging**: Seeing what others brew, sharing discoveries, social warmth
- **Craft pride**: Your brewing journey documented — mastery over your process

### Aesthetic Direction
**Visual references:**
- Specialty coffee bag packaging (Counter Culture, Onyx, George Howell) — craft labels, earthy tones, confident typography, tactile feel
- Analog journals — Moleskine notebooks, handwritten brew logs, the texture of paper and ink

**Anti-references:**
- Generic SaaS dashboards, corporate productivity tools
- Overly techy/developer aesthetic — data should feel warm, not cold
- Social media feed clones (Instagram/Twitter layouts with no personality)

**Theme:** Light + dark (both supported, OS preference + manual toggle). The warm brown palette (#4a2c2a family) and cream backgrounds (#FAF7F5) are core to the identity.

**Typography:** Iosevka Patrick (custom monospace Nerd Font) is the established body/UI font and well-liked by users. Open to pairing with a warmer display font for headings to soften the technical feel while keeping Iosevka's character for body text and data.

### Design Principles

1. **Warmth over precision** — Brown paper, not graph paper. Even data-heavy views should feel handcrafted, not clinical. Tint everything toward the coffee palette.

2. **Quiet confidence** — Like a well-designed coffee bag: strong typography, restrained color, no need to shout. Let the content (brews, beans, community) be the star.

3. **Tactile texture** — Evoke the analog: the weight of a ceramic mug, the grain of a kraft label, the soft edge of a journal page. Avoid flat, sterile surfaces.

4. **Community as atmosphere** — The social feed should feel like overhearing conversations at a cafe, not scrolling a timeline. Presence over performance.

5. **Respect the ritual** — Coffee brewing is meditative. The interface should match that pace — no urgency, no pressure, no gamification. Every interaction should feel intentional.

### Existing Design System
- **Colors:** Brown 50-900 scale + amber accents, CSS custom properties for all surfaces
- **Font:** Iosevka Patrick (400/500/600 weights), monospace throughout
- **Components:** Cards, tables, combo-selects, feed cards, filter pills, modals, badges
- **Shadows:** 3-tier system (sm/md/lg) with warm-tinted rgba
- **Animations:** Quick (100-400ms), fade+slide patterns, like-pop interaction
- **Theme:** Full light/dark with CSS variables, OS preference detection

### Established Patterns

Reusable patterns from shipped work. Reach for these before inventing — they carry the brand identity and have been validated.

#### Per-entity color tinting
Use the `--type-{entity}` and `--type-{entity}-tint` tokens (brewer / roaster / bean / grinder / brew / recipe / tea / vendor / cafe / drink) as accent colors when an entity is the focus of a surface. The fill color is for borders, badges, and active states; the tint is for translucent wash backgrounds. Each entity carries its own identity — surfaces that group multiple entity types should let each one carry its own accent rather than picking a single page-wide color.

#### Tinted-wash cards
For entity-centric cards (stations, hero panels, focused tiles), apply the entity's `--type-{x}-tint` as a `::after` overlay at ~0.45 opacity, position card content with `z-index: 1`, and use a large faint number or letter watermark in the corner via `::before` at ~0.07 opacity using the entity's full tint color. Hover lifts the card 2px and shifts the border toward the entity tint via `color-mix()`.

#### Rubber-stamp progress
For multi-step prereq flows, use circular "stamps" with `border-style: dashed` (todo) → `solid` filled (done) → solid outlined with an outer pulse ring (current). Stamps carry slight per-element rotation (`-2deg`, `+1.5deg`, `-1deg`) and a dotted inner ring via `::after` for the postage-stamp imprint feel. Connect with stitched dashed lines that flip solid when the preceding stamp is done. Done state triggers a one-shot `stamp-press` scale animation (scale 1.4 → 0.94 → 1, 350ms ease-out).

#### Action-cue empty states
Empty states never say "nothing here" alone. They name the next action: `Nothing here yet — <strong>add the bag you're on now</strong>`. Tone matches the entity — concrete and warm, not abstract ("add a thing").

#### Contextual encouragement copy
Status panels name exactly what's left, not just a count. "Add a roaster and a bean to unlock your first brew." beats "2 items remaining." The encouragement scales: 3 remaining → "Start anywhere", 2 → name both, 1 → "Just a {x} left — you're almost there."

#### Container card with kraft texture
Page-level containers that need warmth get `background-image: var(--texture-kraft)` with `background-blend-mode: multiply` on top of `--card-bg`. A 4px multi-color top edge using a hard-stop `linear-gradient` between entity colors signals scope at a glance (e.g. brewer green / roaster amber / bean orange across three steps).

#### "Required / Optional / Done" tag triad
Tags in the top-right of a card section, sized 0.625rem with 0.16em tracking, uppercase. `required` uses the entity's solid tint with cream text. `optional` is a transparent outline in `--text-faint`. `done` is `--brand-green-100` background with an inline check glyph. Always position right-aligned in the section header via `margin-left: auto`.

#### State-aware CTA panel
A panel that changes both copy and visual treatment based on completion state. Not-ready: dashed border on `--surface-bg`, calm informative copy with a large numeric count. Ready: solid border, warm diagonal gradient mixing two entity tints via `color-mix(in oklch, ...)`, larger headline, and the primary CTA with a hover-glide arrow (`translateX(3px)` on hover).

#### Stagger entrance
Page sections fade in with translateY(8px) → 0 over ~380ms with `cubic-bezier(0.16, 1, 0.3, 1)`. Each sibling gets +80ms delay. Always paired with `@media (prefers-reduced-motion: reduce)` killing the animation. Hero gets a smaller translateY(-6px) for differentiation.

#### Inline SVG glyphs over emoji
Check marks, arrows, and small decorative icons are inline `<svg>` with `stroke="currentColor"` so they inherit the surrounding text color and adapt to themes. No emoji in primary UI — they break the analog/craft aesthetic.

#### "Add another" vs "Add a thing"
CTA buttons inside entity sections change copy based on count: empty → "Add a {noun}", non-empty → "Add another". Reduces visual noise (no repeated noun) once the user has context.

### CSS Architecture Notes
- New component CSS files go in `internal/web/assets/css/components/` with a `NN-name.css` prefix that preserves cascade ordering (see existing 01–21 files).
- `components/*.css` is glob-embedded, so new files auto-include without registration.
- Use `oklch()` and `color-mix(in oklch, ...)` for new colors. Existing tokens are hex but new computations should be perceptually uniform.
- The `--texture-kraft`, `--texture-dotgrid`, and `--texture-cork` tokens swap between light and dark themes automatically.
