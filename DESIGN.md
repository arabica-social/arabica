---
name: Arabica
description: A cozy, federated coffee journal for tracking brews, beans, equipment, and community notes.
colors:
  espresso-ink: "#4a2c2a"
  espresso-deep: "#3d2319"
  paper-cream: "#FAF7F5"
  paper-blush: "#f2e8e5"
  warm-border: "#eaddd7"
  roaster-amber: "#fbbf24"
  amber-deep: "#92400e"
  pinboard-cork: "#C4A882"
  sticky-paper: "#FFFDF5"
  brewer-green: "#5b6e4e"
  bean-copper: "#b45309"
  grinder-stone: "#78716c"
  danger-red: "#b91c1c"
  success-green: "#15803d"
typography:
  display:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "2rem to 3.4rem"
    fontWeight: 700
    lineHeight: 0.98
    letterSpacing: "-0.055em"
  headline:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.33
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.4
  body:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  prose:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "1.0625rem"
    fontWeight: 400
    lineHeight: 1.75
  label:
    fontFamily: "Iosevka Patrick, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "0.75rem"
    fontWeight: 600
    lineHeight: 1.33
    letterSpacing: "0.1em"
rounded:
  xs: "2px"
  sm: "4px"
  md: "8px"
  lg: "12px"
  xl: "16px"
  hero: "21.6px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
  xxl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.espresso-ink}"
    textColor: "{colors.paper-cream}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
  button-primary-hover:
    backgroundColor: "{colors.espresso-deep}"
    textColor: "{colors.paper-cream}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
  button-secondary:
    backgroundColor: "{colors.sticky-paper}"
    textColor: "{colors.bean-copper}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
  card:
    backgroundColor: "{colors.sticky-paper}"
    textColor: "{colors.espresso-deep}"
    rounded: "{rounded.lg}"
    padding: "24px"
  input:
    backgroundColor: "{colors.sticky-paper}"
    textColor: "{colors.espresso-deep}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  filter-pill:
    backgroundColor: "{colors.paper-cream}"
    textColor: "{colors.bean-copper}"
    rounded: "{rounded.full}"
    padding: "4px 12px"
---

# Design System: Arabica

## 1. Overview

**Creative North Star: "Neighborhood Brew Journal"**

Arabica should feel like a well-used coffee notebook left on the counter at a neighborhood cafe. It is social, but not performative. It is technical, but never cold. The interface supports the ritual of brewing: slow enough to feel intentional, structured enough to keep data useful, warm enough to invite return visits.

The visual system blends community cafe atmosphere with analog lab notes. Product surfaces use familiar app patterns so people can log a brew, edit a bean, or scan a feed without learning invented controls. Craft details carry the personality: kraft texture, corkboard feed surfaces, sticky-note cards, entity-tinted washes, dashed rubber-stamp progress, and coffee-toned neutrals.

Arabica explicitly rejects generic SaaS dashboards, corporate productivity tools, overly techy developer aesthetics, and social media feed clones. Data should feel warm, not cold. The feed should feel like overhearing conversations at a cafe, not scrolling a timeline.

**Key Characteristics:**
- Cozy, social, inviting, and grounded in coffee ritual.
- Familiar product UI with craft details, not novelty controls.
- Warm brown and cream neutrals with restrained amber and entity accents.
- Tactile texture and modest lift, not glass, neon, or glossy polish.
- Light and dark themes both supported, with the same coffee-shop warmth.

## 2. Colors

The palette is espresso, paper, cork, and roast labels: warm neutrals first, then amber and entity colors used with intent.

### Primary
- **Espresso Ink**: The core brand brown used for primary actions, headers, strong text, and the deepest UI anchors. It should feel like coffee stain and ink, not black.
- **Deep Espresso**: The darker companion for hover states, header depth, and dark text emphasis.

### Secondary
- **Roaster Amber**: The warm alert and badge color. Use it for alpha labels, warnings, and small moments of attention. It should read as roasted warmth, not generic yellow.
- **Bean Copper**: The bean and roaster accent family. Use for entity pills, type badges, and coffee-specific highlights.
- **Brewer Green**: The brewer accent. Use for equipment-oriented completion, station cards, and gentle success-adjacent cues when the entity is the focus.

### Tertiary
- **Pinboard Cork**: The community feed board color. It frames social content as pinned notes, not a timeline.
- **Grinder Stone**: A muted equipment neutral for grinder identity and low-chroma type accents.

### Neutral
- **Paper Cream**: The page background and default cream surface. It is the main environmental color.
- **Paper Blush**: The softer divider and surface layer for table headers, hover rows, and quiet panels.
- **Warm Border**: The default card and container stroke. It keeps surfaces tactile without heavy outlines.
- **Sticky Paper**: The feed card and journal-card base. Use where content should feel like a note or label.

### Named Rules

**The Brown Paper Rule.** Tint every neutral toward coffee. New surfaces should start from Paper Cream, Sticky Paper, Paper Blush, or Espresso Ink, never from sterile gray.

**The Accent Has a Job Rule.** Amber is for attention and state. Entity colors are for entity identity. Do not use saturated color as decoration without a semantic role.

**The Corkboard Rule.** Social feed surfaces may be more tactile than utility pages. Cork texture and sticky-note treatment belong in community context, not every settings panel.

## 3. Typography

**Display Font:** Iosevka Patrick, with ui-monospace and system monospace fallbacks.
**Body Font:** Iosevka Patrick, with ui-monospace and system monospace fallbacks.
**Label/Mono Font:** Iosevka Patrick.

**Character:** Arabica is mono-forward by design. Iosevka Patrick gives the system a logged, measured, notebook quality while the warm palette prevents it from becoming developer-cold.

### Hierarchy
- **Display** (700, 2rem to 3.4rem, 0.98 line-height): Use for static-page heroes and rare large product moments. Tight tracking is allowed when the heading needs packaging-label confidence.
- **Headline** (600, 1.5rem, 2rem line-height): Use for page titles, record titles, and primary section names.
- **Title** (600, 1.25rem, 1.75rem line-height): Use for modal titles, card headings, and focused content modules.
- **Body** (400, 1rem, 1.5 line-height): Use for product UI, forms, lists, and table content.
- **Prose** (400, 1.0625rem, 1.75 line-height): Use for about, AT Protocol, terms, onboarding explanation, and other long reading surfaces. Keep prose line length to 65 to 75 characters.
- **Label** (600, 0.75rem, 0.1em tracking, uppercase): Use for section labels, filter metadata, field groups, and stamp-like status text.

### Named Rules

**The Notebook Mono Rule.** Iosevka Patrick is the default voice. Do not introduce a second typeface unless a page genuinely needs a softer display voice and the choice is documented.

**The Label Restraint Rule.** Uppercase tracking is for labels and tags, not whole paragraphs or button-heavy regions.

## 4. Elevation

Arabica uses tactile but restrained elevation. Borders and tonal layers establish the default structure. Warm shadows appear for cards, dropdowns, modals, hover lift, and feed notes. Texture can also create depth, especially kraft paper and corkboard surfaces.

### Shadow Vocabulary
- **Low Surface** (`box-shadow: 0 1px 3px var(--card-shadow)`): Default cards and small containers.
- **Raised Surface** (`box-shadow: 0 4px 12px var(--card-shadow-hover)`): Hovered cards, dropdowns, and focused panels.
- **Floating Surface** (`box-shadow: 0 10px 25px var(--card-shadow-hover)`): Modals, combo dropdowns, and high-priority overlays.
- **Sticky Note** (`box-shadow: 1px 2px 4px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(0, 0, 0, 0.06)`): Feed cards on corkboard surfaces.

### Named Rules

**The Border First Rule.** Use a warm 1px border before reaching for a large shadow. The interface should feel printed and tactile, not floating in space.

**The Lift on Intent Rule.** A 2px hover lift is enough. Elevation should confirm interaction, not perform choreography.

## 5. Components

Components should be familiar product UI with craft details. Controls need predictable states, generous touch targets, and consistent shapes. Personality belongs in texture, entity tinting, copy, and small state cues.

### Buttons
- **Shape:** Gently rounded rectangles (8px radius).
- **Primary:** Espresso Ink background with Paper Cream text, medium weight, 8px vertical and 16px horizontal padding.
- **Hover / Focus:** Hover deepens to Deep Espresso. Focus uses the global 2px input-focus outline. Motion is 150ms and state-based.
- **Secondary:** Sticky Paper or card surface with Warm Border and Bean Copper text. Use for alternative actions, sponsor links, and lower-priority navigation.
- **Danger / Link:** Destructive actions use the red semantic token as text first. Avoid heavy filled danger buttons unless the action is both destructive and primary in the moment.

### Chips
- **Style:** Filter pills are rounded capsules with 12px horizontal padding, 4px vertical padding, 0.75rem type, and a 1px warm border.
- **State:** Inactive pills are quiet and bordered. Active pills fill with Espresso Ink or the relevant entity color. Entity hover should shift border and text toward that entity color.

### Cards / Containers
- **Corner Style:** Standard cards use 12px radius. Static-page heroes use larger 21.6px radius. Feed notes use sharper 4px or 8px radii to feel like paper scraps.
- **Background:** Product cards use card surfaces and Paper Cream layers. Entity cards may use `--type-{entity}-tint` as a translucent wash.
- **Shadow Strategy:** Low Surface at rest, Raised Surface on hover, Floating Surface for overlays.
- **Border:** Default 1px Warm Border. Do not use colored side stripes.
- **Internal Padding:** 16px for compact cards, 24px for major containers, 32px for spacious static-page heroes.

### Inputs / Fields
- **Style:** 8px radius, card or input background, 1px warm border, 12px horizontal padding.
- **Focus:** Border shifts to the focus token and gets a 2px warm ring. Background warms slightly.
- **Error / Disabled:** Error uses semantic red for border and text. Disabled fields should mute text and preserve shape rather than disappearing.

### Navigation
- **Style:** Navigation inherits the warm text hierarchy. Links are muted by default and strengthen on hover.
- **Active State:** Active tabs use Espresso Ink border or fill depending on context. Profile and manage tabs should remain visually connected to their content region.
- **Mobile Treatment:** Preserve 44px touch targets. Dense pills may opt out only when they are not primary navigation.

### Combo Select
- **Style:** Dropdowns use card background, 8px radius, input border, and Floating Surface shadow. Section labels use uppercase label treatment on a quiet surface background.
- **State:** Hovered or highlighted rows use the warm surface layer. Creation rows are medium weight and separated by a 1px warm divider.

### Feed Cards
- **Style:** Feed cards are sticky notes on a Pinboard Cork board. They use warm paper backgrounds, entity-tinted washes, slight rotation on desktop, and modest hover lift.
- **State:** Hover removes the casual tilt and lifts the note 2px. Keep feed cards legible first, charming second.

### Modals and Menus
- **Style:** Modals use the card background, 12px radius, 24px padding, and Floating Surface shadow on a solid backdrop.
- **State:** Dialog animation is brief: fade and scale over 100 to 150ms. Dropdown menus scale from 0.95 to 1 and fade in over 100ms.

## 6. Do's and Don'ts

### Do:
- **Do** use Paper Cream as the environmental base and Espresso Ink as the main anchor.
- **Do** keep product flows familiar: standard buttons, inputs, dropdowns, tabs, and modals with predictable states.
- **Do** use entity colors for entity identity: bean, brew, brewer, grinder, roaster, recipe, tea, vendor, cafe, and drink.
- **Do** add texture where it supports the metaphor: kraft containers, cork feed boards, and sticky notes.
- **Do** keep motion short and state-based: 100 to 250ms for most UI, 300 to 380ms only for page or feed entrances.
- **Do** write empty states that name the next action, such as "add the bag you're on now".
- **Do** preserve dark mode warmth. Dark surfaces should be coffee-black and brown, not neutral charcoal.

### Don't:
- **Don't** make Arabica look like generic SaaS dashboards or corporate productivity tools.
- **Don't** use an overly techy/developer aesthetic. Data should feel warm, not cold.
- **Don't** make the feed feel like Instagram, Twitter, or any social media feed clone with no personality.
- **Don't** use colored `border-left` or `border-right` accents greater than 1px on cards, callouts, alerts, or list items.
- **Don't** use gradient text, decorative glassmorphism, neon-on-black, or hero-metric SaaS templates.
- **Don't** wrap everything in identical card grids. Use cards when they clarify grouping, not as the default answer.
- **Don't** add new pure black or pure white neutrals. Existing legacy white card tokens should not be the model for new surfaces.
- **Don't** use emoji as primary UI glyphs. Use inline SVG with `stroke="currentColor"`.
- **Don't** animate layout properties or use bounce/elastic easings. Motion should feel like a hand placing a note, not an app trying to impress.
