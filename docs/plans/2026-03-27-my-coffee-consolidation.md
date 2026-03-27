# My Coffee Page Consolidation & Home Dashboard

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate My Brews + Manage into a single "My Coffee" page, add an authenticated home dashboard with incomplete record nudges, and expand inline creation modals with progressive disclosure.

**Architecture:** Replace two overlapping pages (`/brews` and `/manage`) with a single `/my-coffee` page that has tabs for Brews, Beans, Roasters, Grinders, Brewers, and Recipes. Add a dashboard section to the authenticated home page that surfaces quick actions and incomplete records. Entity edit modals open directly from the home dashboard via HTMX. Expand inline combo-select creation with optional "more details" fields.

**Tech Stack:** Go + Templ (server-side), HTMX (dynamic loading), Alpine.js (client state), Tailwind CSS (styling)

---

## Phase 1: Consolidate My Brews + Manage → "My Coffee"

### Task 1: Create My Coffee page template

This task creates the new unified page that combines brew list + manage tabs.

**Files:**
- Modify: `internal/web/pages/manage.templ` (rename to my_coffee concept, reuse as-is)
- Create: `internal/web/pages/my_coffee.templ`

**Step 1: Create the My Coffee page template**

Create `internal/web/pages/my_coffee.templ` with 6 tabs: Brews (default), Beans, Roasters, Grinders, Brewers, Recipes. The Brews tab embeds the brew list content. The other 5 tabs reuse the existing manage partial content.

```templ
package pages

import "arabica/internal/web/components"

type MyCoffeeProps struct{}

templ MyCoffee(layout *components.LayoutData, props MyCoffeeProps) {
	@components.Layout(layout, MyCoffeeContent(props))
}

templ MyCoffeeContent(props MyCoffeeProps) {
	<script src="/static/js/manage-page.js?v=0.4.0"></script>
	<div class="page-container-xl" x-data="managePage()">
		<div class="flex items-center gap-3 mb-6">
			<h2 class="text-2xl font-semibold text-brown-900">My Coffee</h2>
			<div class="ml-auto flex items-center gap-2">
				<a href="/brews/new" class="btn-primary shadow-lg hover:shadow-xl">+ New Brew</a>
				@ManageRefreshButton()
			</div>
		</div>
		@MyCoffeeTabs()
		<!-- Brews tab: standalone HTMX loader -->
		<div x-show="tab === 'brews'">
			<div hx-get="/api/brews" hx-trigger="load" hx-swap="innerHTML">
				@BrewListLoadingSkeleton()
			</div>
		</div>
		<!-- Entity tabs: loaded from manage partial -->
		<div id="manage-content" x-show="tab !== 'brews'" hx-get="/api/manage" hx-trigger="load, refreshManage from:body" hx-swap="innerHTML">
			@ManageLoadingSkeleton()
		</div>
	</div>
}

templ MyCoffeeTabs() {
	<div class="mb-6 border-b-2 border-brown-300">
		<nav class="-mb-px flex space-x-8 overflow-x-auto">
			<button
				@click="tab = 'brews'"
				:class="tab === 'brews' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Brews
			</button>
			<button
				@click="tab = 'beans'"
				:class="tab === 'beans' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Beans
			</button>
			<button
				@click="tab = 'roasters'"
				:class="tab === 'roasters' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Roasters
			</button>
			<button
				@click="tab = 'grinders'"
				:class="tab === 'grinders' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Grinders
			</button>
			<button
				@click="tab = 'brewers'"
				:class="tab === 'brewers' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Brewers
			</button>
			<button
				@click="tab = 'recipes'"
				:class="tab === 'recipes' ? 'border-brown-700 text-brown-900' : 'border-transparent text-brown-600 hover:text-brown-800 hover:border-brown-400'"
				class="whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm"
			>
				Recipes
			</button>
		</nav>
	</div>
}
```

Key design decisions:
- Reuses existing `ManageRefreshButton()`, `ManageLoadingSkeleton()` from `manage.templ`
- Brews tab uses the same `/api/brews` HTMX endpoint the old brew list used
- Entity tabs use the same `/api/manage` HTMX endpoint the old manage page used
- The `managePage()` Alpine component already handles tab persistence via localStorage — it just needs the default changed to `'brews'`
- `+ New Brew` button is always visible in the header (not tab-dependent)

**Step 2: Update manage-page.js default tab**

In `static/js/manage-page.js`, change the default tab from `'beans'` to `'brews'`:

```js
// Line 8: change default
tab: localStorage.getItem("manageTab") || "brews",
```

**Step 3: Run templ generate and verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/web/pages/my_coffee.templ static/js/manage-page.js
git commit -m "feat: add My Coffee page template combining brews and manage"
```

---

### Task 2: Add handler and route for My Coffee

Wire up the new page to the router, and add redirects from old URLs.

**Files:**
- Modify: `internal/handlers/entities.go` (add HandleMyCoffee, or reuse HandleManage)
- Modify: `internal/routing/routing.go` (add `/my-coffee` route, redirect old routes)

**Step 1: Add HandleMyCoffee handler**

Add to `internal/handlers/entities.go` (right after or in place of `HandleManage`):

```go
// HandleMyCoffee renders the unified My Coffee page (replaces both /brews and /manage)
func (h *Handler) HandleMyCoffee(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	layoutData, _, _ := h.layoutDataFromRequest(r, "My Coffee")

	if err := pages.MyCoffee(layoutData, pages.MyCoffeeProps{}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render my coffee page")
	}
}
```

**Step 2: Update routes in routing.go**

In `internal/routing/routing.go`, replace the old `/brews` and `/manage` page routes:

```go
// Replace:
//   mux.HandleFunc("GET /manage", h.HandleManage)
//   mux.HandleFunc("GET /brews", h.HandleBrewList)
// With:
mux.HandleFunc("GET /my-coffee", h.HandleMyCoffee)

// Add redirects for old URLs
mux.HandleFunc("GET /manage", func(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/my-coffee", http.StatusMovedPermanently)
})
mux.HandleFunc("GET /brews", func(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/my-coffee", http.StatusMovedPermanently)
})
```

Keep ALL existing routes under `/brews/new`, `/brews/{id}`, `/brews/{id}/edit`, `/api/brews`, `/api/manage`, etc. — those are still needed for brew CRUD and HTMX partials.

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/handlers/entities.go internal/routing/routing.go
git commit -m "feat: add /my-coffee route with redirects from /brews and /manage"
```

---

### Task 3: Update navigation header

Replace "My Brews" and "Manage Records" dropdown links with a single "My Coffee" link.

**Files:**
- Modify: `internal/web/components/header.templ`

**Step 1: Update header dropdown**

In `internal/web/components/header.templ`, find the dropdown links section (around lines 74-85) and replace:

```templ
// Replace these two links:
<a href="/brews" class="dropdown-item">
	My Brews
</a>
<a href="/recipes" class="dropdown-item">
	Recipes
</a>
<a href="/manage" class="dropdown-item">
	Manage Records
</a>

// With:
<a href="/my-coffee" class="dropdown-item">
	My Coffee
</a>
<a href="/recipes" class="dropdown-item">
	Recipes
</a>
```

**Step 2: Update welcome card links**

In `internal/web/components/shared.templ`, the `WelcomeAuthenticated` component (around line 226) has links to `/brews/new` and `/brews`. Update the "View All Brews" link:

```templ
// Change href="/brews" to href="/my-coffee"
<a
	href="/my-coffee"
	class="home-action-secondary block text-center py-4 px-6 rounded-xl"
	hx-get="/my-coffee"
	hx-target="main"
	hx-swap="innerHTML show:top"
	hx-select="main > *"
	hx-push-url="true"
>
	<span class="text-lg font-semibold">My Coffee</span>
</a>
```

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/web/components/header.templ internal/web/components/shared.templ
git commit -m "feat: update nav to link to /my-coffee instead of /brews and /manage"
```

---

### Task 4: Add modal container to My Coffee page

The entity edit/create modals need a `#modal-container` div on the page to receive HTMX-loaded dialog HTML. The old manage page got this from the manage partial. The My Coffee page needs it too, specifically for the Brews tab where it didn't exist before.

**Files:**
- Modify: `internal/web/pages/my_coffee.templ`

**Step 1: Add modal container**

Add a `#modal-container` div at the end of the page content (inside the `page-container-xl` div but after all tabs):

```templ
<!-- Modal container for HTMX-loaded dialogs -->
<div id="modal-container"></div>
```

This is the target for all `hx-get="/api/modals/..."` requests that load entity edit/create dialogs.

**Step 2: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 3: Commit**

```bash
git add internal/web/pages/my_coffee.templ
git commit -m "feat: add modal container to My Coffee page for entity dialogs"
```

---

## Phase 2: Authenticated Home Dashboard

### Task 5: Define incomplete records data model

Before building the UI, define how to detect incomplete records. An entity is "incomplete" when key fields are empty.

**Files:**
- Modify: `internal/models/models.go` (add IsIncomplete methods)

**Step 1: Add IsIncomplete methods to models**

Add methods to each entity type. These define what "incomplete" means per entity:

```go
// IsIncomplete returns true if the bean is missing key fields beyond name/origin.
func (b *Bean) IsIncomplete() bool {
	return b.RoasterRKey == "" || b.RoastLevel == "" || b.Process == ""
}

// MissingFields returns a human-readable list of missing fields.
func (b *Bean) MissingFields() []string {
	var missing []string
	if b.RoasterRKey == "" {
		missing = append(missing, "roaster")
	}
	if b.RoastLevel == "" {
		missing = append(missing, "roast level")
	}
	if b.Process == "" {
		missing = append(missing, "process")
	}
	return missing
}

// IsIncomplete returns true if the grinder is missing its type.
func (g *Grinder) IsIncomplete() bool {
	return g.GrinderType == ""
}

// MissingFields returns a human-readable list of missing fields.
func (g *Grinder) MissingFields() []string {
	var missing []string
	if g.GrinderType == "" {
		missing = append(missing, "grinder type")
	}
	return missing
}

// IsIncomplete returns true if the brewer is missing its type.
func (b *Brewer) IsIncomplete() bool {
	return b.BrewerType == ""
}

// MissingFields returns a human-readable list of missing fields.
func (b *Brewer) MissingFields() []string {
	var missing []string
	if b.BrewerType == "" {
		missing = append(missing, "brewer type")
	}
	return missing
}
```

Note: Roasters don't get IsIncomplete — name is the only required field, and location/website are truly optional.

**Step 2: Write tests**

Add to `internal/models/models_test.go` (create if it doesn't exist):

```go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBeanIsIncomplete(t *testing.T) {
	// Complete bean
	complete := &Bean{Name: "Test", Origin: "Ethiopia", RoasterRKey: "abc", RoastLevel: "Light", Process: "Washed"}
	assert.False(t, complete.IsIncomplete())
	assert.Empty(t, complete.MissingFields())

	// Incomplete bean — missing roaster
	incomplete := &Bean{Name: "Test", Origin: "Ethiopia", RoastLevel: "Light", Process: "Washed"}
	assert.True(t, incomplete.IsIncomplete())
	assert.Contains(t, incomplete.MissingFields(), "roaster")

	// Stub bean — name only
	stub := &Bean{Name: "Test"}
	assert.True(t, stub.IsIncomplete())
	assert.Len(t, stub.MissingFields(), 3)
}

func TestGrinderIsIncomplete(t *testing.T) {
	complete := &Grinder{Name: "Test", GrinderType: "Hand"}
	assert.False(t, complete.IsIncomplete())

	incomplete := &Grinder{Name: "Test"}
	assert.True(t, incomplete.IsIncomplete())
	assert.Contains(t, incomplete.MissingFields(), "grinder type")
}

func TestBrewerIsIncomplete(t *testing.T) {
	complete := &Brewer{Name: "V60", BrewerType: "pourover"}
	assert.False(t, complete.IsIncomplete())

	incomplete := &Brewer{Name: "V60"}
	assert.True(t, incomplete.IsIncomplete())
	assert.Contains(t, incomplete.MissingFields(), "brewer type")
}
```

**Step 3: Run tests**

```bash
go test ./internal/models/... -v
```

**Step 4: Commit**

```bash
git add internal/models/models.go internal/models/models_test.go
git commit -m "feat: add IsIncomplete and MissingFields methods to entity models"
```

---

### Task 6: Add incomplete records API endpoint

Create an HTMX partial that returns incomplete record items for the home dashboard. This keeps the home page handler lightweight — the dashboard section loads async.

**Files:**
- Modify: `internal/handlers/entities.go` (add handler)
- Modify: `internal/routing/routing.go` (add route)
- Create: `internal/web/components/incomplete_records.templ` (new component)

**Step 1: Create the incomplete records component**

Create `internal/web/components/incomplete_records.templ`:

```templ
package components

import (
	"arabica/internal/models"
	"fmt"
	"strings"
)

// IncompleteRecord represents a single entity that needs attention
type IncompleteRecord struct {
	EntityType    string   // "bean", "grinder", "brewer"
	RKey          string
	Name          string
	MissingFields []string
}

type IncompleteRecordsProps struct {
	Records []IncompleteRecord
}

templ IncompleteRecords(props IncompleteRecordsProps) {
	if len(props.Records) > 0 {
		<div class="card p-4 sm:p-6 mb-6">
			<div class="flex items-center gap-2 mb-3">
				<svg class="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"></path>
				</svg>
				<h3 class="text-lg font-semibold text-brown-900">
					{ fmt.Sprintf("%d", len(props.Records)) } { incompleteNoun(len(props.Records)) } need details
				</h3>
			</div>
			<div class="space-y-2">
				for _, rec := range props.Records {
					<div class="flex items-center justify-between p-3 rounded-lg" style="background: var(--surface-bg); border: 1px solid var(--surface-border);">
						<div>
							<span class="font-medium text-brown-900">{ rec.Name }</span>
							<span class="text-sm text-brown-600 ml-2">
								missing { strings.Join(rec.MissingFields, ", ") }
							</span>
						</div>
						<button
							hx-get={ fmt.Sprintf("/api/modals/%s/%s", rec.EntityType, rec.RKey) }
							hx-target="#modal-container"
							hx-swap="innerHTML"
							class="text-sm font-medium text-brown-700 hover:text-brown-900 cursor-pointer"
						>
							Complete
						</button>
					</div>
				}
			</div>
			if len(props.Records) > 3 {
				<a href="/my-coffee" class="block text-center text-sm text-brown-600 hover:text-brown-800 mt-3">
					View all in My Coffee
				</a>
			}
		</div>
		<!-- Modal container for HTMX-loaded dialogs -->
		<div id="modal-container" hx-on::after-request="if(event.detail.successful && event.target.closest('dialog')) { htmx.ajax('GET', '/api/incomplete-records', {target: '#incomplete-records-section', swap: 'innerHTML'}); }"></div>
	}
}

func incompleteNoun(count int) string {
	if count == 1 {
		return "record"
	}
	return "records"
}

// CollectIncompleteRecords scans all entities and returns incomplete ones (max limit).
func CollectIncompleteRecords(beans []*models.Bean, grinders []*models.Grinder, brewers []*models.Brewer, limit int) []IncompleteRecord {
	var records []IncompleteRecord

	for _, b := range beans {
		if b.IsIncomplete() && !b.Closed {
			records = append(records, IncompleteRecord{
				EntityType:    "bean",
				RKey:          b.RKey,
				Name:          b.Name,
				MissingFields: b.MissingFields(),
			})
		}
	}
	for _, g := range grinders {
		if g.IsIncomplete() {
			records = append(records, IncompleteRecord{
				EntityType:    "grinder",
				RKey:          g.RKey,
				Name:          g.Name,
				MissingFields: g.MissingFields(),
			})
		}
	}
	for _, b := range brewers {
		if b.IsIncomplete() {
			records = append(records, IncompleteRecord{
				EntityType:    "brewer",
				RKey:          b.RKey,
				Name:          b.Name,
				MissingFields: b.MissingFields(),
			})
		}
	}

	if limit > 0 && len(records) > limit {
		return records[:limit]
	}
	return records
}
```

**Step 2: Add the HTMX partial handler**

Add to `internal/handlers/entities.go`:

```go
// HandleIncompleteRecordsPartial returns HTML fragment for incomplete records section.
func (h *Handler) HandleIncompleteRecordsPartial(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		// Return empty — not an error, just no content for unauthenticated
		return
	}

	ctx := r.Context()
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var grinders []*models.Grinder
	var brewers []*models.Brewer

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Msg("Failed to fetch data for incomplete records")
		return
	}

	records := components.CollectIncompleteRecords(beans, grinders, brewers, 5)

	if err := components.IncompleteRecords(components.IncompleteRecordsProps{
		Records: records,
	}).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render incomplete records")
	}
}
```

**Step 3: Add the route**

In `internal/routing/routing.go`, add with the other HTMX partials:

```go
mux.Handle("GET /api/incomplete-records", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleIncompleteRecordsPartial)))
```

**Step 4: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 5: Commit**

```bash
git add internal/web/components/incomplete_records.templ internal/handlers/entities.go internal/routing/routing.go
git commit -m "feat: add incomplete records API endpoint and component"
```

---

### Task 7: Add dashboard section to home page

Add the dashboard section above the community feed for authenticated users.

**Files:**
- Modify: `internal/web/pages/home.templ`
- Modify: `internal/web/components/shared.templ` (update WelcomeAuthenticated)

**Step 1: Update home page content**

In `internal/web/pages/home.templ`, add a dashboard section between the welcome card and the feed for authenticated users:

```templ
templ HomeContent(props HomeProps) {
	<div class="page-container-lg">
		@components.WelcomeCard(components.WelcomeCardProps{
			IsAuthenticated: props.IsAuthenticated,
			UserDID:         props.UserDID,
		})
		if props.IsAuthenticated {
			<!-- Incomplete records loaded async -->
			<div id="incomplete-records-section" hx-get="/api/incomplete-records" hx-trigger="load, refreshManage from:body" hx-swap="innerHTML">
			</div>
		}
		if !props.IsAuthenticated {
			@components.AboutInfoCard()
		}
		@CommunityFeedSection(props.IsAuthenticated)
		if props.IsAuthenticated {
			@components.AboutInfoCard()
		}
	</div>
}
```

Key: The `hx-trigger` includes `refreshManage from:body` so that when a user completes a record via the edit modal (which triggers `refreshManage`), the incomplete records section auto-refreshes.

**Step 2: Update WelcomeAuthenticated quick actions**

In `internal/web/components/shared.templ`, update the `WelcomeAuthenticated` component to have three action buttons instead of two:

```templ
templ WelcomeAuthenticated(userDID string) {
	<div class="mb-6">
		<p class="text-sm text-brown-700">
			Logged in as: <span class="font-mono text-brown-900 font-semibold">{ userDID }</span>
			<a href="/atproto" class="text-brown-700 hover:text-brown-900 transition-colors">(What is this?)</a>
		</p>
	</div>
	<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
		<a
			href="/brews/new"
			class="home-action-primary block text-center py-4 px-6 rounded-xl"
		>
			<span class="text-lg font-semibold">Log Brew</span>
		</a>
		<a
			href="/my-coffee"
			class="home-action-secondary block text-center py-4 px-6 rounded-xl"
		>
			<span class="text-lg font-semibold">My Coffee</span>
		</a>
		<a
			href={ templ.SafeURL("/profile/" + userDID) }
			class="home-action-secondary block text-center py-4 px-6 rounded-xl"
		>
			<span class="text-lg font-semibold">Profile</span>
		</a>
	</div>
}
```

Note: Remove the `hx-get`/`hx-target`/`hx-swap`/`hx-select`/`hx-push-url` attributes from the action links. They cause issues when navigating away from the home page — standard `<a href>` links are simpler and work correctly.

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/web/pages/home.templ internal/web/components/shared.templ
git commit -m "feat: add dashboard with incomplete records nudge to home page"
```

---

### Task 8: Handle modal refresh on home page

When a user clicks "Complete" on the home dashboard and saves in the modal, the incomplete records section should refresh. The modal's `hx-on::after-request` triggers `refreshManage` on the body. Since the home page's incomplete records section listens for `refreshManage from:body` (added in Task 7), this should work automatically.

However, the `#modal-container` div needs to exist on the home page. It's part of the `IncompleteRecords` component (added in Task 6), but only renders when there ARE incomplete records.

**Files:**
- Modify: `internal/web/pages/home.templ`

**Step 1: Add fallback modal container**

Add a `#modal-container` div to the home page that always exists (the one inside `IncompleteRecords` will overwrite it when loaded):

```templ
if props.IsAuthenticated {
	<!-- Incomplete records loaded async -->
	<div id="incomplete-records-section" hx-get="/api/incomplete-records" hx-trigger="load, refreshManage from:body" hx-swap="innerHTML">
	</div>
	<!-- Modal container for entity edit dialogs opened from dashboard -->
	<div id="modal-container"></div>
}
```

Wait — there's a problem. The IncompleteRecords component already includes a `#modal-container`. If the home page also has one, there will be duplicate IDs. Solution: remove the `#modal-container` from the IncompleteRecords component and keep it only in the pages that use the component (home page, my coffee page).

**Step 2: Update IncompleteRecords component**

In `internal/web/components/incomplete_records.templ`, remove the `#modal-container` div from inside the component. The parent page is responsible for providing it.

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/web/pages/home.templ internal/web/components/incomplete_records.templ
git commit -m "fix: ensure modal container exists on home page for entity edit dialogs"
```

---

## Phase 3: Expandable Inline Creation in Brew Form

### Task 9: Add "more details" toggle to combo-select create flow

When the combo-select's `createNew()` fires (user types a name that doesn't match, clicks "Create [name]"), instead of immediately POSTing with just the name, show an expandable section with extra fields.

This is a JS-only change to the combo-select component. The approach: when the user clicks "Create [name]", instead of immediately calling the API, set a `showCreateDetails` flag that reveals additional fields inline in the dropdown. A "Save" button in that expanded section performs the actual POST with all the data.

**Files:**
- Modify: `static/js/combo-select.js` (add create-with-details flow)
- Modify: `internal/web/pages/brew_form.templ` (add extra field config to comboSelectInit)

**Step 1: Add expandable create fields to combo-select**

In `static/js/combo-select.js`, add new state and methods:

```js
// Add to the Alpine.data("comboSelect") return object:

// New state for inline creation with details
showCreateForm: false,
createFormData: {},

// Modified createNew — shows inline form instead of immediately creating
createNewWithDetails() {
  const name = this.query.trim();
  if (!name) return;

  // Initialize form data based on entity type
  this.createFormData = { name };
  if (this.extraFields) {
    for (const field of this.extraFields) {
      this.createFormData[field.name] = "";
    }
  }
  this.showCreateForm = true;
  this.isOpen = false;
},

// Submit the create form with all details
async submitCreateForm() {
  const data = { ...this.createFormData };
  this.isCreating = true;
  try {
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
    this.selectedLabel = data.name;
    this.query = data.name;
    this.showCreateForm = false;

    if (window.ArabicaCache) {
      window.ArabicaCache.invalidateCache();
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

cancelCreateForm() {
  this.showCreateForm = false;
  this.createFormData = {};
},
```

The `extraFields` config is provided per entity type (see Step 2).

**Step 2: Add extra field config to brew form combo-select init**

In `internal/web/pages/brew_form.templ`, update the `comboSelectInit` function to include extra fields config per entity type. Add to the config object:

For beans:
```js
extraFields: [
  { name: 'roast_level', label: 'Roast Level', type: 'select', options: ['Light', 'Medium-Light', 'Medium', 'Medium-Dark', 'Dark'] },
  { name: 'process', label: 'Process', type: 'text', placeholder: 'e.g. Washed, Natural, Honey' },
  { name: 'variety', label: 'Variety', type: 'text', placeholder: 'e.g. SL28, Typica, Gesha' },
]
```

For grinders:
```js
extraFields: [
  { name: 'grinder_type', label: 'Type', type: 'select', options: ['Hand', 'Electric', 'Portable Electric'] },
  { name: 'burr_type', label: 'Burr Type', type: 'select', options: ['Conical', 'Flat', 'Blade'] },
]
```

For brewers:
```js
extraFields: [
  { name: 'brewer_type', label: 'Type', type: 'select', options: ['pourover', 'espresso', 'immersion', 'mokapot', 'coldbrew', 'cupping', 'other'] },
]
```

**Step 3: Add create form template to combo-select markup**

In `internal/web/pages/brew_form.templ`, update the `comboSelectInput` template to include a create form section that shows when `showCreateForm` is true:

```templ
<!-- After the dropdown list, add: -->
<div x-show="showCreateForm" x-transition class="mt-2 p-3 rounded-lg" style="background: var(--surface-bg); border: 1px solid var(--surface-border);">
	<p class="text-sm font-medium text-brown-900 mb-2">
		Creating: <span x-text="createFormData.name" class="font-semibold"></span>
	</p>
	<template x-if="extraFields && extraFields.length > 0">
		<div class="space-y-2">
			<template x-for="field in extraFields" :key="field.name">
				<div>
					<template x-if="field.type === 'select'">
						<select
							:name="field.name"
							x-model="createFormData[field.name]"
							class="w-full form-input text-sm"
						>
							<option value="" x-text="field.label + ' (optional)'"></option>
							<template x-for="opt in field.options" :key="opt">
								<option :value="opt" x-text="opt"></option>
							</template>
						</select>
					</template>
					<template x-if="field.type === 'text'">
						<input
							type="text"
							:placeholder="field.placeholder || field.label"
							x-model="createFormData[field.name]"
							class="w-full form-input text-sm"
						/>
					</template>
				</div>
			</template>
			<div class="flex gap-2 mt-2">
				<button
					type="button"
					@click="submitCreateForm()"
					class="flex-1 btn-primary text-sm py-1.5"
					:disabled="isCreating"
				>
					<span x-show="!isCreating">Save</span>
					<span x-show="isCreating">Saving...</span>
				</button>
				<button
					type="button"
					@click="cancelCreateForm()"
					class="flex-1 btn-secondary text-sm py-1.5"
				>
					Cancel
				</button>
			</div>
		</div>
	</template>
</div>
```

**Step 4: Update createNew to use createNewWithDetails when extraFields exist**

In the combo-select dropdown, change the "Create [name]" button to call `createNewWithDetails()` when extra fields are configured, and `createNew()` when not:

```js
// In the allItems getter, the "create" type still appears.
// In selectHighlighted, change:
else if (item.type === "create") {
  if (this.extraFields && this.extraFields.length > 0) {
    this.createNewWithDetails();
  } else {
    this.createNew();
  }
}
```

**Step 5: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 6: Commit**

```bash
git add static/js/combo-select.js internal/web/pages/brew_form.templ
git commit -m "feat: add expandable details to inline entity creation in brew form"
```

---

## Phase 4: Post-Save Nudge (Optional)

### Task 10: Show toast after brew save if entities are incomplete

After successfully creating a brew, check if the referenced bean/grinder/brewer is incomplete and show a toast notification with a "Complete" link that opens the edit modal.

**Files:**
- Modify: `internal/handlers/brew.go` (add incomplete check to HandleBrewCreate response)
- Modify: `static/js/brew-form.js` (handle toast response)

**Step 1: Add incomplete info to brew create response**

In `internal/handlers/brew.go`, after successfully creating a brew, check if the referenced entities are incomplete. Add a JSON field to the response:

```go
// After successful brew creation, check for incomplete entities
type brewCreateResponse struct {
	RKey       string                  `json:"rkey"`
	Incomplete []incompleteEntityInfo  `json:"incomplete,omitempty"`
}

type incompleteEntityInfo struct {
	EntityType    string   `json:"entity_type"`
	RKey          string   `json:"rkey"`
	Name          string   `json:"name"`
	MissingFields []string `json:"missing_fields"`
}
```

After creating the brew, fetch the referenced bean/grinder/brewer and check:

```go
var incomplete []incompleteEntityInfo

if req.BeanRKey != "" {
	if bean, err := store.GetBean(ctx, req.BeanRKey); err == nil && bean != nil && bean.IsIncomplete() {
		incomplete = append(incomplete, incompleteEntityInfo{
			EntityType:    "bean",
			RKey:          bean.RKey,
			Name:          bean.Name,
			MissingFields: bean.MissingFields(),
		})
	}
}
// Similarly for grinder and brewer...

resp := brewCreateResponse{RKey: rkey, Incomplete: incomplete}
```

**Step 2: Show toast in brew form JS**

In `static/js/brew-form.js`, after a successful brew save, check the response for incomplete entities and show a toast:

```js
// After successful save:
if (data.incomplete && data.incomplete.length > 0) {
  const item = data.incomplete[0];
  const msg = `${item.name} is missing ${item.missing_fields.join(", ")}`;
  showToast(msg, `/api/modals/${item.entity_type}/${item.rkey}`);
}
```

Toast implementation: a simple fixed-position div at the bottom of the screen with auto-dismiss after 8 seconds, and a "Complete" button that fetches the edit modal.

**Step 3: Verify build**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```bash
git add internal/handlers/brew.go static/js/brew-form.js
git commit -m "feat: show toast nudge after brew save if entities are incomplete"
```

---

## Bump JS/CSS Versions

### Task 11: Bump script versions for cache busting

After all changes, bump the version query params on JS files to bust Cloudflare and service worker caches.

**Files:**
- Modify: `internal/web/components/layout.templ` (bump version for combo-select.js, brew-form.js, manage-page.js)
- Modify: `internal/web/pages/my_coffee.templ` (set version for manage-page.js)

**Step 1: Update versions**

Find all `?v=` query strings on the modified JS files and increment them.

**Step 2: Commit**

```bash
git add internal/web/components/layout.templ internal/web/pages/my_coffee.templ
git commit -m "chore: bump JS versions for cache busting"
```

---

## Verification Checklist

After all tasks:

1. `go vet ./...` passes
2. `go build ./...` passes
3. `go test ./...` passes
4. Visiting `/brews` redirects to `/my-coffee`
5. Visiting `/manage` redirects to `/my-coffee`
6. `/my-coffee` shows Brews tab by default with brew list
7. Switching to Beans/Grinders/Brewers/Roasters/Recipes tabs works
8. Entity create/edit modals work from My Coffee page
9. Home page shows incomplete records section when records exist
10. Clicking "Complete" on home page opens edit modal
11. After saving in modal, incomplete records section refreshes
12. Header dropdown shows "My Coffee" instead of "My Brews" and "Manage Records"
13. Inline creation in brew form shows expandable details section
