# Inline Entity Typeahead Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the select + "+ New" modal pattern in the brew form with a combo input that searches existing entities, shows suggestions from other users, and allows creating new entities inline with just a name.

**Architecture:** Build a reusable Alpine.js `comboSelect` component that wraps a text input with a dropdown. The dropdown shows the user's existing entities (from the dropdown-manager cache), server-side suggestions from other users (from `/api/suggestions`), and a "Create [name]" option. Selecting an existing entity sets a hidden `_rkey` input. Selecting "Create" calls the entity API to create a minimal record, then sets the rkey. The component replaces the current `<select>` + `<button>` pattern in all three brew form entity fields (bean, brewer, grinder).

**Tech Stack:** Alpine.js, HTMX, Go/templ, Tailwind CSS

---

## Context

### Current Flow (painful)
1. User sees a `<select>` dropdown with their existing entities
2. If entity doesn't exist, user clicks "+ New" button
3. Modal opens (fetched via HTMX), user fills multi-field form
4. Modal submits, closes, dropdown refreshes
5. User must re-select the new entity in the dropdown

### New Flow (smooth)
1. User types in a combo input (e.g., "Yirga...")
2. Dropdown appears with three sections:
   - **Your entities** — filtered matches from user's collection
   - **Community** — suggestions from other users (via `/api/suggestions`)
   - **Create** — "Create «Yirgacheffe Natural»" option at bottom
3. Selecting a user entity → sets hidden rkey, shows name in input
4. Selecting a community suggestion → calls entity creation API with suggestion data (name + fields), sets rkey
5. Selecting "Create" → calls entity creation API with just the name, sets rkey
6. All inline, no modal, no page navigation

### Existing Infrastructure
- `dropdown-manager.js` — cached user entities, already loaded on brew form init
- `entity-suggest.js` — server-side suggestion search (`/api/suggestions/{type}`)
- `entity-manager.js` — entity CRUD via API (`POST /api/{type}`)
- `/api/list-all` — returns all user entities (beans, grinders, brewers, etc.)
- `/api/suggestions/{type}?q={query}` — returns community suggestions
- `POST /api/{type}` — creates entity, returns JSON with rkey

### Key Files
- `static/js/combo-select.js` — **NEW** — reusable combo select component
- `static/js/brew-form.js` — wire combo selects into brew form
- `internal/web/pages/brew_form.templ` — replace select+button with combo input HTML
- `internal/web/components/layout.templ` — add script tag for combo-select.js
- `static/css/app.css` — combo select dropdown styling

---

## Task 1: Create the Combo Select Alpine.js Component

**Files:**
- Create: `static/js/combo-select.js`

This is the core reusable component. It manages:
- Text input state and filtering
- Dropdown visibility and keyboard navigation
- Three result sections (user entities, community suggestions, create option)
- Entity selection (sets hidden rkey input)
- Inline entity creation via API

**Step 1: Write the combo-select.js component**

```js
/**
 * Reusable combo select component for entity selection + inline creation.
 *
 * Usage in templ:
 *   <div x-data="comboSelect({
 *     entityType: 'bean',
 *     apiEndpoint: '/api/beans',
 *     suggestEndpoint: '/api/suggestions/beans',
 *     inputName: 'bean_rkey',
 *     placeholder: 'Search or create a bean...',
 *     formatLabel: (e) => e.name + ' (' + e.origin + ')',
 *     formatCreateData: (name, suggestion) => ({ name, origin: suggestion?.fields?.origin || '' }),
 *   })">
 *     <input type="hidden" :name="inputName" :value="selectedRKey" />
 *     <input type="text" x-model="query" @input.debounce.200ms="search()"
 *            @focus="open()" @keydown.escape="close()" @keydown.arrow-down.prevent="moveDown()"
 *            @keydown.arrow-up.prevent="moveUp()" @keydown.enter.prevent="selectHighlighted()" />
 *     <div x-show="isOpen" class="combo-dropdown">
 *       <!-- results rendered here -->
 *     </div>
 *   </div>
 */

document.addEventListener("alpine:init", () => {
  Alpine.data("comboSelect", (config) => ({
    // Config
    entityType: config.entityType || "",
    apiEndpoint: config.apiEndpoint || "",
    suggestEndpoint: config.suggestEndpoint || "",
    inputName: config.inputName || "",
    placeholder: config.placeholder || "Search...",
    formatLabel: config.formatLabel || ((e) => e.name || e.Name || ""),
    formatCreateData: config.formatCreateData || ((name) => ({ name })),
    required: config.required || false,

    // State
    query: "",
    selectedRKey: "",
    selectedLabel: "",
    isOpen: false,
    highlightIndex: -1,
    isCreating: false,

    // Results
    userResults: [],
    communityResults: [],

    // All items for flat indexing (for keyboard nav)
    get allItems() {
      const items = [];
      for (const r of this.userResults) {
        items.push({ type: "user", entity: r });
      }
      for (const r of this.communityResults) {
        items.push({ type: "community", suggestion: r });
      }
      if (this.query.trim() && !this.exactMatch) {
        items.push({ type: "create", name: this.query.trim() });
      }
      return items;
    },

    // Whether query exactly matches an existing entity
    get exactMatch() {
      const q = this.query.trim().toLowerCase();
      return this.userResults.some(
        (e) => (e.name || e.Name || "").toLowerCase() === q,
      );
    },

    init() {
      // If editing, populate from initial value
      const initial = config.initialValue;
      if (initial) {
        this.selectedRKey = initial.rkey || "";
        this.selectedLabel = this.formatLabel(initial);
        this.query = this.selectedLabel;
      }
    },

    open() {
      this.isOpen = true;
      this.highlightIndex = -1;
      this.search();
    },

    close() {
      // Delay to allow click events on dropdown items
      setTimeout(() => {
        this.isOpen = false;
        // Restore label if user didn't complete selection
        if (this.selectedRKey && this.query !== this.selectedLabel) {
          this.query = this.selectedLabel;
        }
      }, 150);
    },

    async search() {
      const q = this.query.trim().toLowerCase();

      // Filter user's entities from cache
      const entities = this.getUserEntities();
      if (q) {
        this.userResults = entities.filter((e) => {
          const label = this.formatLabel(e).toLowerCase();
          return label.includes(q);
        });
      } else {
        this.userResults = entities.slice(0, 10);
      }

      // Fetch community suggestions
      if (q.length >= 2 && this.suggestEndpoint) {
        try {
          const resp = await fetch(
            `${this.suggestEndpoint}?q=${encodeURIComponent(q)}&limit=5`,
            { credentials: "same-origin" },
          );
          if (resp.ok) {
            const data = await resp.json();
            // Filter out entities the user already has (by name match)
            const userNames = new Set(
              entities.map((e) => (e.name || e.Name || "").toLowerCase()),
            );
            this.communityResults = (data || []).filter(
              (s) => !userNames.has((s.name || "").toLowerCase()),
            );
          }
        } catch (e) {
          console.error("Suggestion fetch failed:", e);
        }
      } else {
        this.communityResults = [];
      }

      this.highlightIndex = -1;
      if (!this.isOpen && this.query) {
        this.isOpen = true;
      }
    },

    getUserEntities() {
      const dm = window.ArabicaCache?.getData?.() || {};
      switch (this.entityType) {
        case "bean":
          return (dm.beans || []).filter((b) => !b.closed && !b.Closed);
        case "brewer":
          return dm.brewers || [];
        case "grinder":
          return dm.grinders || [];
        default:
          return [];
      }
    },

    // Select an existing user entity
    selectEntity(entity) {
      const rkey = entity.rkey || entity.RKey;
      this.selectedRKey = rkey;
      this.selectedLabel = this.formatLabel(entity);
      this.query = this.selectedLabel;
      this.isOpen = false;

      // Dispatch change event for other listeners (e.g., onBrewerChange)
      this.$nextTick(() => {
        this.$dispatch("combo-change", {
          entityType: this.entityType,
          rkey,
          entity,
        });
      });
    },

    // Select a community suggestion — creates the entity first
    async selectSuggestion(suggestion) {
      this.isCreating = true;
      try {
        const data = this.formatCreateData(
          suggestion.name,
          suggestion,
        );
        if (suggestion.source_uri) {
          data.source_ref = suggestion.source_uri;
        }
        const resp = await fetch(this.apiEndpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "same-origin",
          body: JSON.stringify(data),
        });
        if (!resp.ok) throw new Error(`Create failed: ${resp.status}`);
        const created = await resp.json();
        const rkey = created.rkey || created.RKey;

        this.selectedRKey = rkey;
        this.selectedLabel = suggestion.name;
        this.query = suggestion.name;
        this.isOpen = false;

        // Invalidate cache so entity appears in future searches
        if (window.ArabicaCache) {
          window.ArabicaCache.invalidate();
        }

        this.$nextTick(() => {
          this.$dispatch("combo-change", {
            entityType: this.entityType,
            rkey,
          });
        });
      } catch (e) {
        console.error("Failed to create from suggestion:", e);
      } finally {
        this.isCreating = false;
      }
    },

    // Create a brand new entity with just the name
    async createNew() {
      const name = this.query.trim();
      if (!name) return;

      this.isCreating = true;
      try {
        const data = this.formatCreateData(name, null);
        const resp = await fetch(this.apiEndpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "same-origin",
          body: JSON.stringify(data),
        });
        if (!resp.ok) throw new Error(`Create failed: ${resp.status}`);
        const created = await resp.json();
        const rkey = created.rkey || created.RKey;

        this.selectedRKey = rkey;
        this.selectedLabel = name;
        this.query = name;
        this.isOpen = false;

        if (window.ArabicaCache) {
          window.ArabicaCache.invalidate();
        }

        this.$nextTick(() => {
          this.$dispatch("combo-change", {
            entityType: this.entityType,
            rkey,
          });
        });
      } catch (e) {
        console.error("Failed to create entity:", e);
      } finally {
        this.isCreating = false;
      }
    },

    // Keyboard navigation
    moveDown() {
      if (this.highlightIndex < this.allItems.length - 1) {
        this.highlightIndex++;
      }
    },

    moveUp() {
      if (this.highlightIndex > 0) {
        this.highlightIndex--;
      }
    },

    selectHighlighted() {
      const item = this.allItems[this.highlightIndex];
      if (!item) return;
      if (item.type === "user") this.selectEntity(item.entity);
      else if (item.type === "community")
        this.selectSuggestion(item.suggestion);
      else if (item.type === "create") this.createNew();
    },

    // Clear selection
    clear() {
      this.selectedRKey = "";
      this.selectedLabel = "";
      this.query = "";
      this.$dispatch("combo-change", {
        entityType: this.entityType,
        rkey: "",
      });
    },
  }));
});
```

**Step 2: Commit**

```
feat: add reusable combo-select Alpine.js component for typeahead entity selection
```

---

## Task 2: Add Combo Select Styles

**Files:**
- Modify: `static/css/app.css`

**Step 1: Add CSS for the combo dropdown**

Add in the form styling section (after `.form-select`):

```css
  /* Combo select dropdown */
  .combo-select {
    @apply relative;
  }

  .combo-dropdown {
    @apply absolute z-50 w-full mt-1 rounded-lg overflow-hidden;
    background: var(--card-bg);
    border: 1px solid var(--input-border);
    box-shadow: var(--shadow-lg);
    max-height: 280px;
    overflow-y: auto;
  }

  .combo-section-label {
    @apply text-xs font-medium uppercase tracking-wider px-3 py-1.5;
    color: var(--text-muted);
    background: var(--surface-bg);
  }

  .combo-item {
    @apply px-3 py-2 cursor-pointer text-sm;
    color: var(--text-primary);
  }

  .combo-item:hover,
  .combo-item[data-highlighted="true"] {
    background: var(--surface-bg);
  }

  .combo-item-create {
    @apply px-3 py-2 cursor-pointer text-sm font-medium;
    color: var(--accent-primary);
    border-top: 1px solid var(--surface-border);
  }

  .combo-item-create:hover,
  .combo-item-create[data-highlighted="true"] {
    background: var(--surface-bg);
  }

  .combo-item-sub {
    @apply text-xs;
    color: var(--text-muted);
  }

  .combo-creating {
    @apply px-3 py-2 text-sm text-center;
    color: var(--text-muted);
  }
```

**Step 2: Rebuild CSS**

```bash
just style
```

**Step 3: Commit**

```
feat: add combo-select dropdown CSS styles
```

---

## Task 3: Add Script Tag and Verify Cache API

**Files:**
- Modify: `internal/web/components/layout.templ`
- Modify: `static/js/dropdown-manager.js` (if needed)

The combo-select component reads from `window.ArabicaCache.getData()`. We need
to verify this API exists in the cache layer, and add the script tag.

**Step 1: Check dropdown-manager.js for getData**

The combo-select needs `window.ArabicaCache.getData()` to return the cached
entity data. Read `static/js/dropdown-manager.js` and find how the cache stores
data. If there's no `getData()` method, add one that returns the current cached
data object `{ beans, grinders, brewers, roasters, recipes }`.

Likely the cache already stores data internally. Add a `getData()` method if
missing:

```js
getData() {
  return this._data || {};
},
```

**Step 2: Add script tag in layout.templ**

Add after the entity-suggest.js script and before brew-form.js:

```html
<script src="/static/js/combo-select.js?v=0.1.0"></script>
```

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```
feat: add combo-select script to layout and ensure cache getData API
```

---

## Task 4: Replace Bean Select with Combo Input

**Files:**
- Modify: `internal/web/pages/brew_form.templ`
- Modify: `static/js/brew-form.js`

This is the first entity conversion. Bean is the most important because it's
required.

**Step 1: Replace BeanSelectField in brew_form.templ**

Replace the current `BeanSelectField` component. The new version uses the
combo-select pattern:

```templ
// BeanSelectField renders the bean combo-select with typeahead + inline creation
templ BeanSelectField(props BrewFormProps) {
	<div
		class="combo-select"
		x-data="comboSelect({
			entityType: 'bean',
			apiEndpoint: '/api/beans',
			suggestEndpoint: '/api/suggestions/beans',
			inputName: 'bean_rkey',
			placeholder: 'Search or create a bean...',
			required: true,
			formatLabel: (e) => {
				const name = e.name || e.Name || '';
				const origin = e.origin || e.Origin || '';
				const roast = e.roast_level || e.RoastLevel || '';
				if (origin && roast) return name + ' (' + origin + ' - ' + roast + ')';
				if (origin) return name + ' (' + origin + ')';
				return name;
			},
			formatCreateData: (name, suggestion) => {
				const data = { name };
				if (suggestion && suggestion.fields) {
					if (suggestion.fields.origin) data.origin = suggestion.fields.origin;
					if (suggestion.fields.roastLevel) data.roast_level = suggestion.fields.roastLevel;
					if (suggestion.fields.process) data.process = suggestion.fields.process;
				}
				return data;
			},
		})"
		if props.Brew != nil && props.Brew.BeanRKey != "" {
			x-init={ fmt.Sprintf("selectedRKey = '%s'; query = '%s'; selectedLabel = query", props.Brew.BeanRKey, getBeanLabel(props)) }
		}
	>
		<label class="form-label">
			Coffee Bean
			<span class="text-red-500">*</span>
		</label>
		<input type="hidden" :name="inputName" :value="selectedRKey" x-ref="rkey"/>
		<div class="relative">
			<input
				type="text"
				x-model="query"
				@input.debounce.200ms="search()"
				@focus="open()"
				@blur="close()"
				@keydown.escape.prevent="close()"
				@keydown.arrow-down.prevent="moveDown()"
				@keydown.arrow-up.prevent="moveUp()"
				@keydown.enter.prevent="selectHighlighted()"
				:placeholder="placeholder"
				class="w-full form-input-lg"
				autocomplete="off"
			/>
			<!-- Clear button -->
			<button
				type="button"
				x-show="selectedRKey"
				@click="clear()"
				class="absolute right-2 top-1/2 -translate-y-1/2 text-brown-400 hover:text-brown-600"
				x-cloak
			>
				@components.IconX()
			</button>
		</div>
		<!-- Dropdown -->
		<div x-show="isOpen && (allItems.length > 0 || query.trim())" x-cloak class="combo-dropdown" @mousedown.prevent>
			<!-- Creating indicator -->
			<div x-show="isCreating" x-cloak class="combo-creating">Creating...</div>
			<template x-if="!isCreating">
				<div>
					<!-- User's entities -->
					<template x-if="userResults.length > 0">
						<div>
							<div class="combo-section-label">Your beans</div>
							<template x-for="(entity, i) in userResults" :key="entity.rkey || entity.RKey">
								<div
									class="combo-item"
									:data-highlighted="highlightIndex === i"
									@click="selectEntity(entity)"
									@mouseenter="highlightIndex = i"
								>
									<span x-text="formatLabel(entity)"></span>
								</div>
							</template>
						</div>
					</template>
					<!-- Community suggestions -->
					<template x-if="communityResults.length > 0">
						<div>
							<div class="combo-section-label">Community</div>
							<template x-for="(s, j) in communityResults" :key="s.source_uri || j">
								<div
									class="combo-item"
									:data-highlighted="highlightIndex === userResults.length + j"
									@click="selectSuggestion(s)"
									@mouseenter="highlightIndex = userResults.length + j"
								>
									<div x-text="s.name"></div>
									<div class="combo-item-sub">
										<span x-show="s.fields?.origin" x-text="s.fields?.origin"></span>
										<span x-show="s.fields?.origin && s.fields?.roastLevel"> · </span>
										<span x-show="s.fields?.roastLevel" x-text="s.fields?.roastLevel"></span>
										<span x-show="s.count > 1" x-text="' · ' + s.count + ' users'"></span>
									</div>
								</div>
							</template>
						</div>
					</template>
					<!-- Create option -->
					<template x-if="query.trim() && !exactMatch">
						<div
							class="combo-item-create"
							:data-highlighted="highlightIndex === userResults.length + communityResults.length"
							@click="createNew()"
							@mouseenter="highlightIndex = userResults.length + communityResults.length"
						>
							Create "<span x-text="query.trim()"></span>"
						</div>
					</template>
					<!-- Empty state -->
					<template x-if="allItems.length === 0 && query.trim()">
						<div class="combo-creating">No matches found</div>
					</template>
				</div>
			</template>
		</div>
	</div>
}
```

Add the helper function:

```go
func getBeanLabel(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.Bean != nil {
		return formatBeanLabel(*props.Brew.Bean)
	}
	return ""
}
```

**Step 2: Update brew-form.js to listen for combo-change events**

In the brew form Alpine component's `init()`, add a listener for brewer
combo changes (to update `brewerCategory`):

```js
// Listen for combo-select changes
this.$el.addEventListener('combo-change', (e) => {
  if (e.detail.entityType === 'brewer') {
    const brewerType = e.detail.entity?.brewer_type || e.detail.entity?.BrewerType || '';
    this.brewerCategory = this.normalizeBrewerCategory(brewerType);
    if (this.brewerCategory === 'pourover') {
      this.showPours = true;
    }
  }
});
```

**Step 3: Run templ generate and verify**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```
feat: replace bean select+modal with inline combo-select typeahead
```

---

## Task 5: Replace Brewer Select with Combo Input

**Files:**
- Modify: `internal/web/pages/brew_form.templ`

**Step 1: Replace BrewerSelectField**

Same pattern as bean, but with brewer-specific config:

```templ
templ BrewerSelectField(props BrewFormProps) {
	<div
		class="combo-select"
		x-data="comboSelect({
			entityType: 'brewer',
			apiEndpoint: '/api/brewers',
			suggestEndpoint: '/api/suggestions/brewers',
			inputName: 'brewer_rkey',
			placeholder: 'Search or create a brew method...',
			formatLabel: (e) => e.name || e.Name || '',
			formatCreateData: (name, suggestion) => {
				const data = { name };
				if (suggestion && suggestion.fields) {
					if (suggestion.fields.brewerType) data.brewer_type = suggestion.fields.brewerType;
				}
				return data;
			},
		})"
		if props.Brew != nil && props.Brew.BrewerRKey != "" {
			x-init={ fmt.Sprintf("selectedRKey = '%s'; query = '%s'; selectedLabel = query", props.Brew.BrewerRKey, getBrewerLabel(props)) }
		}
	>
		<label class="form-label">Brew Method</label>
		<input type="hidden" :name="inputName" :value="selectedRKey"/>
		<!-- Same dropdown structure as bean, but with "Your brewers" / "Community" sections -->
		<!-- ... (identical HTML structure, just different section labels) -->
	</div>
}
```

Note: since the HTML dropdown structure is identical across all three entity
types, the implementing engineer should extract the shared dropdown markup into
the templ component itself. The only differences are:
- Section label text ("Your beans" vs "Your brewers" vs "Your grinders")
- Subtitle fields shown in community results

To avoid duplicating the dropdown HTML three times, create a shared templ
component `ComboSelectDropdown` that accepts a section label prop, or simply
include the dropdown markup inline in each field (it's ~30 lines and the
differences are minor enough that extraction isn't required).

Add helper:

```go
func getBrewerLabel(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.BrewerObj != nil {
		return props.Brew.BrewerObj.Name
	}
	return ""
}
```

**Step 2: Commit**

```
feat: replace brewer select+modal with inline combo-select typeahead
```

---

## Task 6: Replace Grinder Select with Combo Input

**Files:**
- Modify: `internal/web/pages/brew_form.templ`

**Step 1: Replace GrinderSelectField**

Same pattern, grinder-specific:

```templ
templ GrinderSelectField(props BrewFormProps) {
	<div
		class="combo-select"
		x-data="comboSelect({
			entityType: 'grinder',
			apiEndpoint: '/api/grinders',
			suggestEndpoint: '/api/suggestions/grinders',
			inputName: 'grinder_rkey',
			placeholder: 'Search or create a grinder...',
			formatLabel: (e) => e.name || e.Name || '',
			formatCreateData: (name, suggestion) => {
				const data = { name };
				if (suggestion && suggestion.fields) {
					if (suggestion.fields.grinderType) data.grinder_type = suggestion.fields.grinderType;
					if (suggestion.fields.burrType) data.burr_type = suggestion.fields.burrType;
				}
				return data;
			},
		})"
		if props.Brew != nil && props.Brew.GrinderRKey != "" {
			x-init={ fmt.Sprintf("selectedRKey = '%s'; query = '%s'; selectedLabel = query", props.Brew.GrinderRKey, getGrinderLabel(props)) }
		}
	>
		<label class="form-label">Grinder</label>
		<input type="hidden" :name="inputName" :value="selectedRKey"/>
		<!-- Same dropdown structure -->
	</div>
}
```

Add helper:

```go
func getGrinderLabel(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.GrinderObj != nil {
		return props.Brew.GrinderObj.Name
	}
	return ""
}
```

**Step 2: Commit**

```
feat: replace grinder select+modal with inline combo-select typeahead
```

---

## Task 7: Update Recipe Autofill to Work with Combo Selects

**Files:**
- Modify: `static/js/brew-form.js`

The recipe autofill currently calls `this.setFormField(form, 'brewer_rkey',
...)` which sets `<select>` values. With combo selects, we need to dispatch
events or directly update the combo component state.

**Step 1: Update applyRecipe**

Replace the `setFormField` calls for `bean_rkey`, `brewer_rkey` with direct
Alpine component communication. The simplest approach: dispatch custom events
that the combo-select listens for.

In `combo-select.js`, add to `init()`:

```js
// Listen for external set events (e.g., from recipe autofill)
this.$el.addEventListener('combo-set', (e) => {
  if (e.detail.rkey) {
    this.selectedRKey = e.detail.rkey;
    this.selectedLabel = e.detail.label || '';
    this.query = this.selectedLabel;
  }
});
```

In `brew-form.js`, update `applyRecipe`:

```js
// Instead of: this.setFormField(form, 'brewer_rkey', recipe.brewer_rkey || '');
// Do:
const brewerCombo = form.querySelector('[x-data*="entityType: \'brewer\'"]');
if (brewerCombo) {
  brewerCombo.dispatchEvent(new CustomEvent('combo-set', {
    detail: { rkey: recipe.brewer_rkey || '', label: brewerName },
    bubbles: false,
  }));
}
```

**Step 2: Commit**

```
feat: update recipe autofill to work with combo-select components
```

---

## Task 8: Update Entity Creation API for Minimal Records

**Files:**
- Modify: `internal/handlers/entities.go`

Currently `HandleBeanCreate` requires both `name` and `origin`. For inline
creation with just a name, we need to relax the bean validation so only `name`
is required. Check if `origin` is enforced in the handler or model validation.

**Step 1: Check and relax bean creation validation**

In `internal/models/models.go`, `CreateBeanRequest.Validate()` requires `name`
but does NOT require `origin` (only the handler may check). Verify that the
handler doesn't reject beans without an origin.

If the handler has additional validation beyond `Validate()`, relax it to allow
name-only beans.

**Step 2: Verify grinder and brewer only require name**

Check that `CreateGrinderRequest.Validate()` and `CreateBrewerRequest.Validate()`
only require `name`. They should already be fine.

**Step 3: Commit (if changes needed)**

```
fix: allow creating entities with just a name for inline typeahead creation
```

---

## Task 9: Clean Up Removed Code

**Files:**
- Modify: `internal/web/pages/brew_form.templ`
- Modify: `static/js/brew-form.js`

**Step 1: Remove entity manager initialization from brew-form.js**

The `initEntityManagers()` method and related `beanManager`, `grinderManager`,
`brewerManager` properties are no longer needed since we're not using modals.
Remove:
- `initEntityManagers()` method
- `beanManager`, `grinderManager`, `brewerManager` properties
- `saveBean()`, `saveGrinder()`, `saveBrewer()` delegates
- `showBeanForm`, `showGrinderForm`, `showBrewerForm` getters/setters
- `editingBean`, `editingGrinder`, `editingBrewer` getters
- `beanForm`, `grinderForm`, `brewerForm` getters/setters

Keep: `dropdownManager` (still used for recipe data and brewer type lookup).

**Step 2: Remove "+ New" modal buttons from brew_form.templ**

These were part of the old select fields and should already be gone after Tasks
4-6. Verify no remnants exist.

**Step 3: Remove HTMX modal loading comment**

Remove the `<!-- Entity modals now loaded via HTMX into #modal-container -->`
comment from `BrewFormContent`.

**Step 4: Run tests**

```bash
go test ./...
```

**Step 5: Commit**

```
refactor: remove entity modal code from brew form (replaced by combo-select)
```

---

## Task 10: Rebuild and Final Verification

**Files:**
- None new

**Step 1: Rebuild CSS**

```bash
just style
```

**Step 2: Run full test suite**

```bash
templ generate
go vet ./...
go test ./...
```

**Step 3: Bump cache versions**

In `layout.templ`, bump CSS and JS versions as needed.

**Step 4: Commit**

```
chore: rebuild CSS and bump cache versions for combo-select
```

---

## Summary

| Task | What | Key files |
|---|---|---|
| 1 | Combo select Alpine.js component | `static/js/combo-select.js` (new) |
| 2 | Dropdown CSS styles | `static/css/app.css` |
| 3 | Script tag + cache API | `layout.templ`, `dropdown-manager.js` |
| 4 | Bean combo input | `brew_form.templ`, `brew-form.js` |
| 5 | Brewer combo input | `brew_form.templ` |
| 6 | Grinder combo input | `brew_form.templ` |
| 7 | Recipe autofill integration | `brew-form.js`, `combo-select.js` |
| 8 | Relax entity creation validation | `handlers/entities.go` |
| 9 | Clean up removed modal code | `brew-form.js`, `brew_form.templ` |
| 10 | Final rebuild + verification | CSS, tests, versions |

### What's NOT in scope
- Recipe select (stays as-is — recipes are a different interaction pattern)
- Manage page entity forms (keep modal pattern, different context)
- Profile page entity forms (keep inline Alpine forms, different context)
- The mode chooser removal and section collapsing (separate work items)
