# HTMX + Alpine.js Integration Pattern

## Problem: "Alpine Expression Error: [variable] is not defined"

When HTMX swaps in content containing Alpine.js directives (like `x-show`, `x-if`, `@click`), Alpine may not automatically process the new DOM elements, resulting in console errors like:

```
Alpine Expression Error: activeTab is not defined
Expression: "activeTab === 'brews'"
```

## Root Cause

HTMX loads and swaps content into the DOM after Alpine has already initialized. The new elements contain Alpine directives that reference variables in a parent Alpine component's scope, but Alpine doesn't automatically bind these new elements to the existing component.

## Solution

Use HTMX's `hx-on::after-swap` event to manually tell Alpine to initialize the new DOM tree:

```html
<div id="content" 
     hx-get="/api/data" 
     hx-trigger="load" 
     hx-swap="innerHTML" 
     hx-on::after-swap="Alpine.initTree($el)">
</div>
```

### Key Points

- `hx-on::after-swap` - HTMX event that fires after content swap completes
- `Alpine.initTree($el)` - Tells Alpine to process all directives in the swapped element
- `$el` - HTMX provides this as the target element that received the swap

## Common Scenario

**Parent template** (defines Alpine scope):
```html
<div x-data="{ activeTab: 'brews' }">
  <!-- Static content with tab buttons -->
  <button @click="activeTab = 'brews'">Brews</button>
  
  <!-- HTMX loads dynamic content here -->
  <div id="content" 
       hx-get="/api/tabs" 
       hx-trigger="load"
       hx-swap="innerHTML"
       hx-on::after-swap="Alpine.initTree($el)">
  </div>
</div>
```

**Loaded partial** (uses parent scope):
```html
<div x-show="activeTab === 'brews'">
  <!-- Brew content -->
</div>
<div x-show="activeTab === 'beans'">
  <!-- Bean content -->
</div>
```

Without `Alpine.initTree($el)`, the `x-show` directives won't be bound to the parent's `activeTab` variable.

## Alternative: Alpine Morph Plugin

For more complex scenarios with nested Alpine components, use the Alpine Morph plugin:

```html
<script src="https://cdn.jsdelivr.net/npm/@alpinejs/morph@3.x.x/dist/cdn.min.js"></script>
<div hx-swap="morph"></div>
```

This preserves Alpine state during swaps but requires the plugin.

## When to Use

Apply this pattern whenever:
1. HTMX loads content containing Alpine directives
2. The loaded content references variables from a parent Alpine component
3. You see "Expression Error: [variable] is not defined" in console
4. Alpine directives in HTMX-loaded content don't work (no reactivity, clicks ignored, etc.)
