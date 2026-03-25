# Clean Craft UI Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Overhaul Arabica's visual design from gradient-heavy brown-on-brown to clean white cards on warm cream, with CSS custom properties for future dark mode support.

**Architecture:** The overhaul is split into two layers: (1) CSS-only changes that redefine existing component classes — this covers ~80% of the visual change with zero template edits, and (2) targeted template edits for structural changes like removing nested content boxes and adding type indicators. CSS custom properties are introduced from the start so dark mode is a color swap later, not a rewrite.

**Tech Stack:** Tailwind CSS (config + `@apply` in app.css), Templ templates, no new dependencies.

**Key files:**
- `static/css/app.css` — all component class definitions
- `tailwind.config.js` — color palette and theme
- `internal/web/components/layout.templ` — body background, CSS version
- `internal/web/components/header.templ` — nav bar
- `internal/web/components/footer.templ` — footer
- `internal/web/components/shared.templ` — shared components (WelcomeCard, PageHeader, etc.)
- `internal/web/pages/feed.templ` — feed card structure
- `internal/web/components/record_*.templ` — feed content boxes (brew, bean, roaster, grinder, brewer)
- `internal/web/components/entity_tables.templ` — entity list cards
- `internal/web/components/action_bar.templ` — feed card wrappers
- `internal/web/components/profile_brew_card.templ` — profile brew cards

---

## Task 1: CSS Custom Properties Foundation

Introduce CSS custom properties for all semantic colors so that component classes reference tokens instead of hardcoded Tailwind values. This is the dark-mode foundation — changing these variables later switches the entire theme.

**Files:**
- Modify: `static/css/app.css` (add `:root` block at top, before `@tailwind` directives)
- Modify: `tailwind.config.js` (add `cream` color)

**Step 1: Add CSS custom properties to app.css**

Add this block at the very top of `static/css/app.css`, before the `@tailwind` directives but after the `@font-face` declarations:

```css
/* ========================================
   Design Tokens (CSS Custom Properties)
   Light theme (default)
   ======================================== */
:root {
  /* Page */
  --page-bg: #FAF7F5;
  --page-text: #3d2319;

  /* Cards */
  --card-bg: #FFFFFF;
  --card-border: #eaddd7;
  --card-shadow: rgba(61, 35, 25, 0.06);
  --card-shadow-hover: rgba(61, 35, 25, 0.10);

  /* Surfaces (inset areas inside cards) */
  --surface-bg: rgba(250, 247, 245, 0.5);
  --surface-border: #f2e8e5;

  /* Header */
  --header-bg-from: #4a2c2a;
  --header-bg-to: #3d2319;
  --header-border: #7f5539;
  --header-text: #FAF7F5;

  /* Text hierarchy */
  --text-primary: #3d2319;
  --text-secondary: #4a2c2a;
  --text-muted: #7f5539;
  --text-faint: #bfa094;
  --text-placeholder: #d2bab0;

  /* Interactive */
  --btn-primary-bg: #4a2c2a;
  --btn-primary-bg-hover: #3d2319;
  --btn-primary-text: #FAF7F5;
  --btn-secondary-bg: #FFFFFF;
  --btn-secondary-border: #e0cec7;
  --btn-secondary-text: #6b4423;
  --btn-secondary-bg-hover: #FAF7F5;

  /* Forms */
  --input-bg: #FFFFFF;
  --input-border: #e0cec7;
  --input-border-focus: #7f5539;
  --input-ring-focus: rgba(127, 85, 57, 0.15);
  --input-bg-focus: rgba(250, 247, 245, 0.3);

  /* Tables */
  --table-bg: #FFFFFF;
  --table-header-bg: #FAF7F5;
  --table-border: #eaddd7;
  --table-row-hover: #FAF7F5;
  --table-divider: #f2e8e5;

  /* Modals */
  --modal-bg: #FFFFFF;
  --modal-border: #eaddd7;
  --modal-backdrop: rgba(0, 0, 0, 0.4);

  /* Feed type indicators (left border) */
  --type-brew: #6b4423;
  --type-bean: #d97706;
  --type-recipe: #bfa094;
  --type-roaster: #d2bab0;
  --type-grinder: #d2bab0;
  --type-brewer: #d2bab0;

  /* Shadows */
  --shadow-sm: 0 1px 3px var(--card-shadow);
  --shadow-md: 0 4px 12px var(--card-shadow-hover);
  --shadow-lg: 0 10px 25px var(--card-shadow-hover);

  /* Footer */
  --footer-bg: #FAF7F5;
  --footer-border: #eaddd7;
}
```

**Step 2: Add `cream` color to tailwind.config.js**

Add a `cream` color to the colors object in `tailwind.config.js`:

```js
cream: {
  50: "#FAF7F5",
},
```

This lets templates use `bg-cream-50` for the page background if needed.

**Step 3: Verify build**

Run: `just style && go vet ./...`
Expected: Clean build, no errors. No visual changes yet (properties defined but not consumed).

**Step 4: Commit**

```bash
git add static/css/app.css tailwind.config.js
git commit -m "feat: add CSS custom properties foundation for theme support"
```

---

## Task 2: Redefine Core Component Classes

Rewrite the component class definitions in `app.css` to use the CSS custom properties and implement the Clean Craft visual style. This single file change transforms the entire app's appearance.

**Files:**
- Modify: `static/css/app.css` (rewrite `@layer components` block)

**Step 1: Rewrite card classes**

Replace the existing card definitions:

```css
/* Cards and Containers */
.card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  @apply rounded-xl;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 200ms ease;
}

.card:hover {
  box-shadow: var(--shadow-md);
}

.card-inner {
  @apply p-6;
}

.card-sm {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  @apply rounded-lg;
  box-shadow: var(--shadow-sm);
}

/* Section box for lighter content areas */
.section-box {
  background: var(--surface-bg);
  @apply rounded-lg p-4;
}
```

**Step 2: Rewrite button classes**

```css
/* Buttons */
.btn {
  @apply inline-flex items-center justify-center px-4 py-2 rounded-lg font-medium transition-colors cursor-pointer;
}

.btn-primary {
  @apply btn text-white;
  background: var(--btn-primary-bg);
}

.btn-primary:hover {
  background: var(--btn-primary-bg-hover);
}

.btn-secondary {
  @apply btn;
  background: var(--btn-secondary-bg);
  color: var(--btn-secondary-text);
  border: 1px solid var(--btn-secondary-border);
}

.btn-secondary:hover {
  background: var(--btn-secondary-bg-hover);
}

.btn-tertiary {
  @apply btn text-white;
  background: var(--btn-primary-bg);
}

.btn-tertiary:hover {
  background: var(--btn-primary-bg-hover);
}

.btn-link {
  color: var(--text-muted);
  @apply font-medium underline transition-colors cursor-pointer;
}

.btn-link:hover {
  color: var(--text-primary);
}

.btn-danger {
  @apply text-red-600 hover:text-red-800 font-medium underline transition-colors cursor-pointer;
}
```

Note: `.btn-tertiary` is redefined to match `.btn-primary` (no more gradient). It's used in 1 place (`shared.templ:206`). We keep the class to avoid template churn but visually unify it.

**Step 3: Rewrite form classes**

```css
/* Forms */
.form-label {
  @apply block text-sm font-medium mb-2;
  color: var(--text-primary);
}

.form-input {
  @apply rounded-lg shadow-sm text-base py-2 px-3;
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  color: var(--text-primary);
  transition: border-color 150ms ease, box-shadow 150ms ease, background-color 150ms ease;
}

.form-input:focus {
  border-color: var(--input-border-focus);
  box-shadow: 0 0 0 2px var(--input-ring-focus);
  background: var(--input-bg-focus);
  outline: none;
}

.form-input::placeholder {
  color: var(--text-placeholder);
}

.form-input-lg {
  @apply form-input py-3 px-4;
}

.form-select {
  @apply form-input truncate max-w-full min-w-0;
}

.form-textarea {
  @apply form-input min-h-[100px];
}
```

**Step 4: Rewrite table classes**

```css
/* Tables */
.table-container {
  background: var(--table-bg);
  border: 1px solid var(--table-border);
  @apply rounded-lg overflow-hidden;
  box-shadow: var(--shadow-sm);
}

.table {
  @apply min-w-full;
  border-collapse: collapse;
}

.table-header {
  background: var(--table-header-bg);
  border-bottom: 1px solid var(--table-border);
}

.table-th {
  @apply px-6 py-3 text-left text-xs font-medium uppercase tracking-wider;
  color: var(--text-muted);
}

.table-body {
  background: var(--table-bg);
}

.table-body tr {
  border-bottom: 1px solid var(--table-divider);
}

.table-body tr:last-child {
  border-bottom: none;
}

.table-row {
  transition: background-color 150ms ease;
}

.table-row:hover {
  background: var(--table-row-hover);
}

.table-td {
  @apply px-6 py-4 whitespace-nowrap text-sm;
  color: var(--text-secondary);
}
```

**Step 5: Rewrite modal classes**

```css
/* Modals */
.modal-backdrop {
  @apply fixed inset-0 flex items-center justify-center z-50 p-4;
  background: var(--modal-backdrop);
  backdrop-filter: blur(4px);
}

.modal-content {
  background: var(--modal-bg);
  border: 1px solid var(--modal-border);
  @apply rounded-xl p-6 max-w-md w-full max-h-[90vh] overflow-y-auto;
  box-shadow: var(--shadow-lg);
}

.modal-title {
  @apply text-xl font-semibold mb-4;
  color: var(--text-primary);
}

/* Native Dialog Element */
.modal-dialog {
  @apply p-0 bg-transparent border-none shadow-none max-w-md w-full;
}

.modal-dialog::backdrop {
  background: var(--modal-backdrop);
  backdrop-filter: blur(4px);
}

/* Dialog content wrapper (nested inside dialog) */
.modal-dialog .modal-content {
  background: var(--modal-bg);
  border: 1px solid var(--modal-border);
  @apply rounded-xl p-6 w-full max-h-[90vh] overflow-y-auto;
  box-shadow: var(--shadow-lg);
}
```

**Step 6: Rewrite feed component classes**

```css
/* Feed Components */
.feed-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  @apply rounded-lg p-3 sm:p-4 transition-shadow;
  box-shadow: var(--shadow-sm);
}

.feed-card:hover {
  box-shadow: var(--shadow-md);
}

.feed-content-box {
  background: var(--surface-bg);
  @apply rounded-lg p-3 sm:p-4;
}

.feed-content-box-sm {
  background: var(--surface-bg);
  @apply rounded-lg p-2 sm:p-3;
}
```

Note: We keep `.feed-content-box` and `.feed-content-box-sm` but restyle them as subtle surface tints (no border, no backdrop-blur). This way existing templates work immediately. Task 4 removes the wrapper elements from templates where possible.

**Step 7: Rewrite remaining component classes**

Update avatar, text utility, badge, link, action, dropdown, and comment classes. The key changes are:

- Avatar rings: keep as-is (they're fine)
- Text utilities: reference CSS variables
- Badges: keep as-is (amber accent works)
- Links: reference CSS variables
- Action buttons: remove brown-100 background, use transparent with hover
- Dropdowns: use card-bg variable
- Comments: reference variables for borders/backgrounds

For text utilities:
```css
/* Text Utilities */
.text-helper {
  @apply text-sm mt-1;
  color: var(--text-muted);
}

.text-meta {
  @apply text-xs;
  color: var(--text-muted);
}

.text-meta-sm {
  @apply text-sm;
  color: var(--text-muted);
}

.text-label {
  color: var(--text-muted);
}
```

For action buttons and bars:
```css
/* Action Bar */
.action-bar {
  @apply flex items-center gap-2 mt-3 pt-3;
  border-top: 1px solid var(--surface-border);
}

.brew-view-actions .action-bar {
  @apply mt-0 pt-0 border-t-0;
}

.comment-item .action-bar {
  @apply mt-1 border-t-0 gap-1 rounded-lg px-1.5 py-1 inline-flex items-center;
  background: var(--surface-bg);
}

.action-btn {
  @apply inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors cursor-pointer min-h-[44px];
  color: var(--text-muted);
  background: transparent;
}

.action-btn:hover {
  background: var(--surface-bg);
  color: var(--text-secondary);
}
```

For dropdowns:
```css
.action-menu {
  @apply absolute left-1/2 -translate-x-1/2 w-36 rounded-lg py-1 z-50;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow-md);
}

.dropdown-menu {
  @apply absolute right-0 mt-2 w-48 rounded-lg py-1 z-50;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow-md);
}

.dropdown-item {
  @apply block px-4 py-2 text-sm transition-colors;
  color: var(--text-muted);
}

.dropdown-item:hover {
  background: var(--surface-bg);
}
```

For like/share/comment buttons, suggestions:
```css
.like-btn {
  @apply inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors min-h-[44px];
}

.like-btn-liked {
  @apply like-btn text-red-600;
  background: transparent;
  animation: like-pop 400ms ease-out;
}

.like-btn-liked:hover {
  background: var(--surface-bg);
}

.like-btn-unliked {
  @apply like-btn;
  color: var(--text-muted);
  background: transparent;
  animation: like-shrink 200ms ease-out;
}

.like-btn-unliked:hover {
  background: var(--surface-bg);
}

.share-btn {
  @apply inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors min-h-[44px];
  color: var(--text-muted);
  background: transparent;
}

.share-btn:hover {
  background: var(--surface-bg);
}

.comment-btn {
  @apply inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors min-h-[44px];
  color: var(--text-muted);
  background: transparent;
}

.comment-btn:hover {
  background: var(--surface-bg);
}
```

For suggestions dropdown:
```css
.suggestions-dropdown {
  @apply absolute z-50 left-0 right-0 mt-1 rounded-lg max-h-48 overflow-y-auto;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow-md);
}

.suggestions-item {
  @apply w-full text-left px-3 py-2 flex items-center gap-2 transition-colors cursor-pointer last:border-b-0;
  border-bottom: 1px solid var(--surface-border);
}

.suggestions-item:hover {
  background: var(--surface-bg);
}
```

For comments:
```css
.comment-section {
  @apply mt-8 pt-6;
  border-top: 2px solid var(--card-border);
}

.comment-login-prompt {
  @apply flex items-center gap-3 rounded-lg p-4 mb-5 border border-dashed;
  background: var(--surface-bg);
  border-color: var(--card-border);
}

.comment-compose {
  @apply rounded-lg p-4 mb-5 flex flex-col gap-2;
  background: var(--surface-bg);
  border: 1px solid var(--card-border);
}

.comment-textarea {
  @apply w-full rounded-lg px-3 py-2.5 text-base resize-none transition-colors focus:ring-0 focus:outline-none;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  color: var(--text-primary);
}

.comment-textarea::placeholder {
  color: var(--text-placeholder);
}

.comment-textarea:focus {
  border-color: var(--input-border-focus);
}

.comment-item {
  @apply relative rounded-lg p-3 transition-colors;
}

.comment-item:hover {
  background: var(--surface-bg);
}

.comment-thread-line {
  @apply absolute left-0 top-3 bottom-3 w-0.5 rounded-full;
  background: var(--card-border);
}

.comment-reply-btn {
  @apply inline-flex items-center gap-1 transition-colors text-xs font-medium;
  color: var(--text-placeholder);
}

.comment-reply-btn:hover {
  color: var(--text-muted);
}

.comment-delete-btn {
  @apply transition-colors;
  color: var(--text-placeholder);
}

.comment-delete-btn:hover {
  color: var(--text-muted);
}

.comment-reply-form {
  @apply flex flex-col gap-2 rounded-lg p-3;
  background: var(--surface-bg);
  border: 1px solid var(--card-border);
}
```

**Step 8: Rebuild CSS and verify build**

Run: `just style && go vet ./... && go build ./...`
Expected: Clean build.

**Step 9: Commit**

```bash
git add static/css/app.css
git commit -m "feat: redefine component classes with Clean Craft styling and CSS variables"
```

---

## Task 3: Update Layout, Header, and Footer

Update the structural templates to use the new color system.

**Files:**
- Modify: `internal/web/components/layout.templ`
- Modify: `internal/web/components/header.templ`
- Modify: `internal/web/components/footer.templ`

**Step 1: Update layout.templ**

Change the `<html>` tag's inline background:
```
style="background-color: #fdf8f6;"  →  style="background-color: #FAF7F5;"
```

Change the `<body>` tag:
```
class="bg-brown-50 min-h-full flex flex-col"
style="background-color: #fdf8f6;"
```
to:
```
class="min-h-full flex flex-col"
style="background-color: var(--page-bg); color: var(--page-text);"
```

Bump the CSS version:
```
output.css?v=0.6.1  →  output.css?v=0.7.0
```

**Step 2: Update header.templ**

Change the nav element from:
```
class="sticky top-0 z-50 bg-gradient-to-br from-brown-800 to-brown-900 text-white shadow-xl border-b-2 border-brown-600"
```
to:
```
class="sticky top-0 z-50 text-white"
style="background: linear-gradient(135deg, var(--header-bg-from), var(--header-bg-to)); border-bottom: 1px solid var(--header-border);"
```

Remove `shadow-xl` from the nav — the border provides sufficient separation. Add `box-shadow: var(--shadow-sm);` to the style attribute if a subtle shadow is wanted.

Reduce padding in the container div:
```
class="container mx-auto px-4 py-4"  →  class="container mx-auto px-4 py-3"
```

Make the ALPHA badge smaller:
```
class="text-xs bg-amber-400 text-brown-900 px-2 py-1 rounded-md font-semibold shadow-sm"
```
to:
```
class="text-[10px] bg-amber-400 text-brown-900 px-1.5 py-0.5 rounded font-semibold"
```

**Step 3: Update footer.templ**

Change the footer from:
```
class="mt-auto border-t border-brown-200 bg-brown-50"
```
to:
```
class="mt-auto"
style="background: var(--footer-bg); border-top: 1px solid var(--footer-border);"
```

**Step 4: Regenerate templ, rebuild CSS, verify**

Run: `templ generate && just style && go vet ./... && go build ./...`
Expected: Clean build.

**Step 5: Commit**

```bash
git add internal/web/components/layout.templ internal/web/components/header.templ internal/web/components/footer.templ
git commit -m "feat: update layout, header, footer for Clean Craft theme"
```

---

## Task 4: Add Feed Card Type Indicators

Add colored left borders to feed cards to distinguish record types (brew, bean, recipe, etc.) at a glance.

**Files:**
- Modify: `static/css/app.css` (add type indicator classes)
- Modify: `internal/web/components/action_bar.templ` (add type class to feed card wrapper)
- Modify: `internal/web/pages/feed.templ` (add type class where feed cards are rendered)

**Step 1: Add type indicator CSS classes**

Add to `app.css` after the `.feed-card` definition:

```css
/* Feed card type indicators */
.feed-card-brew {
  border-left: 3px solid var(--type-brew);
}

.feed-card-bean {
  border-left: 3px solid var(--type-bean);
}

.feed-card-recipe {
  border-left: 3px solid var(--type-recipe);
}

.feed-card-roaster {
  border-left: 3px solid var(--type-roaster);
}

.feed-card-grinder {
  border-left: 3px solid var(--type-grinder);
}

.feed-card-brewer {
  border-left: 3px solid var(--type-brewer);
}
```

**Step 2: Identify where feed cards are rendered with type context**

Read the following files to understand how the feed card type is available in the template context:
- `internal/web/components/action_bar.templ` — the `FeedCard` component that wraps all feed items
- `internal/web/pages/feed.templ` — where feed items are rendered

The feed card wrapper likely receives a type string (e.g., from `FeedItem.Collection` or similar). Add the appropriate `feed-card-{type}` class based on this value.

**Important:** Read the actual template code to determine exact prop names and conditional logic. The plan cannot specify exact line numbers because the template structure may vary. The key pattern is:

```go
// In the feed card wrapper component, add the type class:
class={ templ.Classes(
    "feed-card",
    templ.KV("feed-card-brew", props.Type == "brew"),
    templ.KV("feed-card-bean", props.Type == "bean"),
    // ... etc
) }
```

**Step 3: Rebuild and verify**

Run: `templ generate && just style && go vet ./... && go build ./...`

**Step 4: Commit**

```bash
git add static/css/app.css internal/web/components/action_bar.templ internal/web/pages/feed.templ
git commit -m "feat: add colored left-border type indicators to feed cards"
```

---

## Task 5: Clean Up Template Inline Styles

Several templates use inline Tailwind gradient classes and shadow overrides that bypass the component classes. These need updating to match the new system.

**Files:**
- Modify: `internal/web/components/shared.templ`
- Modify: `internal/web/pages/about.templ`
- Modify: `internal/web/pages/atproto.templ`

**Step 1: Update shared.templ**

In `WelcomeCard`: change `class="card p-8 mb-8"` — the `card` class now handles styling. Keep `p-8 mb-8`.

In `WelcomeAuthenticated`: remove the inline `shadow-lg hover:shadow-xl` from button links. The `.btn-primary` and `.btn-tertiary` classes handle it now. Example:
```
class="btn-primary block text-center py-4 px-6 rounded-xl shadow-lg hover:shadow-xl"
```
becomes:
```
class="btn-primary block text-center py-4 px-6 rounded-xl"
```

In `EmptyState`: remove the inline `shadow-lg hover:shadow-xl` from the action link.

In `PageHeader`: change the action button default from `"btn-primary shadow-lg hover:shadow-xl"` to just `"btn-primary"`.

In `AboutInfoCard`: change from inline gradient classes:
```
class="bg-gradient-to-br from-amber-50 to-brown-100 rounded-xl p-6 border-2 border-brown-300 shadow-lg mb-6"
```
to:
```
class="card p-6 mb-6"
```
(or keep as a special card with amber tint if desired — read the actual usage context first)

**Step 2: Update about.templ and atproto.templ**

These pages use extensive inline gradient classes for feature sections. Read each file and replace:
- `bg-gradient-to-br from-brown-100 to-brown-200` → `card` class or inline `background: var(--card-bg);`
- `shadow-xl` / `shadow-lg` → remove (cards get shadow from class)
- `border-2 border-brown-300` → `border border-brown-200` or let card class handle it

Be careful with these pages — they have custom layouts. Don't break the structure, just update the color/shadow treatment.

**Step 3: Rebuild and verify**

Run: `templ generate && just style && go vet ./... && go build ./...`

**Step 4: Commit**

```bash
git add internal/web/components/shared.templ internal/web/pages/about.templ internal/web/pages/atproto.templ
git commit -m "refactor: remove inline gradient/shadow overrides from templates"
```

---

## Task 6: Typography Refinements

Downsize the typography scale and update section title treatment.

**Files:**
- Modify: `static/css/app.css` (update typography classes)
- Modify: `internal/web/components/shared.templ` (PageHeader title size)

**Step 1: Update typography classes in app.css**

```css
/* Typography */
.section-title {
  @apply text-xs font-semibold uppercase tracking-widest mb-4;
  color: var(--text-faint);
}

.page-title {
  @apply text-2xl font-semibold;
  color: var(--text-primary);
}
```

**Step 2: Update PageHeader in shared.templ**

Change the heading from `text-3xl font-bold` to use the `.page-title` class:
```
<h2 class="text-3xl font-bold text-brown-900">{ props.Title }</h2>
```
becomes:
```
<h2 class="page-title">{ props.Title }</h2>
```

**Step 3: Search for other `text-3xl` usages**

Grep for `text-3xl` across templ files. Update each to `text-2xl font-semibold` or use `.page-title` class. Key locations:
- `manage.templ` — page title
- `notifications.templ` — page title
- `recipe_explore.templ` — page title
- `brew_form.templ` — page title
- `shared.templ` — WelcomeCard title

**Step 4: Rebuild and verify**

Run: `templ generate && just style && go vet ./... && go build ./...`

**Step 5: Commit**

```bash
git add static/css/app.css internal/web/components/shared.templ [other modified templ files]
git commit -m "feat: refine typography scale — smaller titles, uppercase section labels"
```

---

## Task 7: Remove Table Row Stagger Animation

The stagger animation on table rows feels gimmicky for data tables. Remove it while keeping feed card stagger.

**Files:**
- Modify: `static/css/app.css` (remove table row animation rules)

**Step 1: Remove table row stagger CSS**

Delete these rules from app.css (around lines 556-577):

```css
/* Table rows slide in with stagger effect (dynamic content) */
.table-body tr {
  animation: fade-in-slide-up 300ms ease-out backwards;
}

.table-body tr:nth-child(1) { animation-delay: 0ms; }
.table-body tr:nth-child(2) { animation-delay: 30ms; }
.table-body tr:nth-child(3) { animation-delay: 60ms; }
.table-body tr:nth-child(4) { animation-delay: 90ms; }
.table-body tr:nth-child(5) { animation-delay: 120ms; }
.table-body tr:nth-child(n + 6) { animation-delay: 150ms; }
```

**Step 2: Rebuild**

Run: `just style`

**Step 3: Commit**

```bash
git add static/css/app.css
git commit -m "refactor: remove table row stagger animation"
```

---

## Task 8: Update Form Input Focus Behavior

Remove the `translateY(-1px)` focus lift on form elements. Clean Craft uses a subtle background tint change on focus instead, which is already handled by the new `.form-input` definition.

**Files:**
- Modify: `static/css/app.css` (remove focus transform rules)

**Step 1: Remove focus transform**

Delete these rules (around lines 647-660):

```css
.form-input:focus,
.form-select:focus,
.form-textarea:focus {
  transform: translateY(-1px);
}
```

Also update the transition rule to remove `transform`:
```css
.form-input,
.form-select,
.form-textarea {
  transition:
    border-color 100ms ease,
    box-shadow 100ms ease,
    transform 50ms ease;
}
```
Change to:
```css
.form-input,
.form-select,
.form-textarea {
  transition:
    border-color 150ms ease,
    box-shadow 150ms ease,
    background-color 150ms ease;
}
```

Note: If the new `.form-input` definition in Task 2 already includes its own transition, this separate rule may be redundant. Check whether it's still needed after Task 2 is applied. If the Task 2 definition already has transition on the class itself, delete this separate rule entirely.

**Step 2: Rebuild**

Run: `just style`

**Step 3: Commit**

```bash
git add static/css/app.css
git commit -m "refactor: replace form focus lift with background tint transition"
```

---

## Task 9: Visual QA and Polish

Manual review pass to catch inconsistencies.

**Files:** Various — depends on findings.

**Step 1: Run the dev server**

Run: `go run cmd/server/main.go`

**Step 2: Visual checklist**

Check each page and verify:

- [ ] **Home page:** WelcomeCard renders as white card on cream background. No gradient. Login form inputs have 1px borders.
- [ ] **Feed:** Feed cards are white with subtle shadow. Type indicators show colored left borders. Action buttons are transparent (no background) until hover.
- [ ] **Brew form:** All inputs have 1px borders. Focus shows tint change + ring. No translateY lift.
- [ ] **Brew view:** Detail fields use section-box with subtle tint. Card is white.
- [ ] **Manage page:** Tables are white with light header. No gradient backgrounds.
- [ ] **Profile:** Stats cards are white. Tab content loads properly.
- [ ] **Recipe explore:** Cards are white. Detail panel matches.
- [ ] **Modals:** White background, subtle border, no gradient.
- [ ] **Header:** Slightly shorter, ALPHA badge smaller. Shadow subtle or absent.
- [ ] **Footer:** Clean, matches cream background.
- [ ] **Mobile:** Check all of the above at < 640px width.

**Step 3: Fix any issues found**

Address visual inconsistencies discovered during QA. Common issues to watch for:
- Templates with hardcoded `bg-brown-100` or `bg-brown-50` that should now be `bg-white` or use variables
- Inline `shadow-*` classes that override the component class shadow
- `border-2` on inputs that weren't caught in the component class rewrite (inline overrides in templates)
- Text color classes that should be updated (`text-brown-800` → just inherit from parent or use variable)

**Step 4: Commit fixes**

```bash
git add -A
git commit -m "fix: visual QA polish for Clean Craft overhaul"
```

---

## Task 10: Bump CSS Version and Final Build Check

**Files:**
- Modify: `internal/web/components/layout.templ` (verify CSS version bumped)

**Step 1: Verify CSS version**

The version should already be `0.7.0` from Task 3. Confirm it's correct.

**Step 2: Full build and vet**

Run: `templ generate && just style && go vet ./... && go build ./... && go test ./...`

Expected: All pass.

**Step 3: Final commit if needed**

If any last fixes were made:
```bash
git add -A
git commit -m "chore: final Clean Craft overhaul build verification"
```

---

## Future Work (Not in This Plan)

These are noted for later and should NOT be done in this implementation:

1. **Dark mode (Option B):** Add `@media (prefers-color-scheme: dark)` block redefining all CSS variables with espresso/cream values. Also add a manual toggle. All the structural work is done — this is purely a color variable swap.

2. **SVG icon system:** Replace emoji icons (📍🔥🌱⚖️🏭) with SVG icons from Lucide or Phosphor. Separate task, requires icon selection and template updates.

3. **Feed content box removal:** The `.feed-content-box` wrappers are restyled but still exist in templates. A future cleanup can remove them entirely and let content sit directly in the feed card, but this is optional since the restyled version (subtle tint, no border) already looks clean.
