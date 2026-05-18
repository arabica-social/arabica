# Onboarding Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Block `/brews/new` for users without a bean + brewer, and replace inline entity creation on the brew form with a homepage "Get Started" card that opens the existing entity modals.

**Architecture:**
- New `ReadinessStatus` derived on every render from PDS state (no persistence). Computed in shared `internal/onboarding` package via the `database.Store` interface.
- `HomeHandler` populates `HomeProps.Readiness`. Home template renders a `GetStartedCard` slot above `WelcomeCard` when not ready; otherwise the slot stays empty. The card itself is loaded/refreshed via HTMX (`hx-trigger="load, refreshManage from:body"`) so adding entities through existing modals re-renders it without a full page reload.
- `HandleBrewNew` 302-redirects to `/#get-started` when not ready.
- Brew form combo-selects switch to a `ComboSelectConfigNoCreate` variant that hides the inline "+ Create" affordance; other forms (e.g. bean modal's roaster picker) keep current behavior.
- **Arabica only this commit.** Oolong variant lands in a follow-up commit per the spec.

**Tech Stack:** Go 1.21+, templ, HTMX, petite-vue (Alpine-like), testify/assert.

**Spec:** `docs/superpowers/specs/2026-05-17-onboarding-flow-design.md`

---

## File Map

**Create:**
- `internal/onboarding/readiness.go` — `ReadinessStatus` type + `CheckBrewReadiness` function.
- `internal/onboarding/readiness_test.go` — unit tests against `database.MockStore`.
- `internal/arabica/web/components/get_started_card.templ` — the card component.
- `internal/arabica/handlers/onboarding.go` — `GET /api/get-started-card` handler.
- `internal/arabica/handlers/onboarding_test.go` — handler unit tests.

**Modify:**
- `internal/web/components/combo_select.templ` — add `ComboSelectConfigNoCreate` helper; thread `AllowCreate` field through JSON config; add `&& allowCreate` guard on the dropdown "Create" item.
- `internal/web/assets/js/combo-select.js` — read `allowCreate` flag (default true); short-circuit `createNew()` when false.
- `internal/arabica/web/pages/brew_form.templ` — switch bean/grinder/brewer combos to `ComboSelectConfigNoCreate`.
- `internal/arabica/handlers/brew.go` — readiness redirect in `HandleBrewNew`.
- `internal/web/pages/home.templ` — `HomeProps.Readiness *onboarding.ReadinessStatus`; render `GetStartedCard` slot when not ready.
- `internal/web/components/shared.templ` — `welcomeAuthenticatedArabica` accepts readiness and renders "Log Brew" as a disabled link when not ready.
- `internal/handlers/feed.go` — populate `HomeProps.Readiness` from the store (arabica only this commit; oolong/unauth = nil).
- `internal/routing/routing.go` — register `/api/get-started-card`.

---

## Task 1: Readiness package

**Files:**
- Create: `internal/onboarding/readiness.go`
- Test: `internal/onboarding/readiness_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/onboarding/readiness_test.go`:

```go
package onboarding

import (
	"context"
	"errors"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/database"

	"github.com/stretchr/testify/assert"
)

func TestCheckBrewReadiness_None(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.False(t, got.HasBean)
	assert.False(t, got.HasBrewer)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_BrewerOnly(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "a"}}, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBrewer)
	assert.False(t, got.HasBean)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_BeanOnly(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.False(t, got.HasBrewer)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_Both(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.True(t, got.HasBrewer)
	assert.True(t, got.Ready())
}

func TestCheckBrewReadiness_BeanError(t *testing.T) {
	want := errors.New("boom")
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, want },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	_, err := CheckBrewReadiness(context.Background(), store)

	assert.ErrorIs(t, err, want)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/onboarding/... -v`
Expected: FAIL with "package not found" / "undefined: CheckBrewReadiness".

- [ ] **Step 3: Implement the package**

Create `internal/onboarding/readiness.go`:

```go
// Package onboarding holds shared helpers for the new-user onboarding flow.
//
// Currently it derives "is this user ready to log a brew?" from the user's
// PDS state. There is no persistence: deleting all beans puts the user back
// into onboarding, which is the correct behavior.
package onboarding

import (
	"context"
	"fmt"

	"tangled.org/arabica.social/arabica/internal/database"
)

// ReadinessStatus reports which brew-prerequisite collections the user owns.
type ReadinessStatus struct {
	HasBean   bool
	HasBrewer bool
}

// Ready returns true when the user owns at least one bean and one brewer —
// the minimum required to log a brew.
func (s ReadinessStatus) Ready() bool {
	return s.HasBean && s.HasBrewer
}

// CheckBrewReadiness derives the user's readiness from the store. It calls
// ListBeans / ListBrewers; the AtprotoStore implementation uses its caches,
// so this is cheap on repeat calls within a request.
func CheckBrewReadiness(ctx context.Context, store database.Store) (ReadinessStatus, error) {
	beans, err := store.ListBeans(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list beans: %w", err)
	}
	brewers, err := store.ListBrewers(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list brewers: %w", err)
	}
	return ReadinessStatus{
		HasBean:   len(beans) > 0,
		HasBrewer: len(brewers) > 0,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/onboarding/... -v`
Expected: all 5 tests PASS.

- [ ] **Step 5: Verify build**

Run: `go vet ./... && go build ./...`
Expected: clean exit.

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat(arabica): add brew-readiness check for onboarding"
```

(Project prefers `jj`; if unavailable use `git add internal/onboarding && git commit -m "feat(arabica): add brew-readiness check for onboarding"`.)

---

## Task 2: Combo-select `AllowCreate` flag

**Files:**
- Modify: `internal/web/components/combo_select.templ`
- Modify: `internal/web/assets/js/combo-select.js`

Adds a `ComboSelectConfigNoCreate` helper that emits `allowCreate: false` in the v-scope JSON, plus JS-side gating that hides the dropdown "Create" item and short-circuits `createNew()`. Other callsites remain on `ComboSelectConfig` (default `allowCreate: true`).

- [ ] **Step 1: Add the no-create helper in combo_select.templ**

In `internal/web/components/combo_select.templ`, find the existing `ComboSelectConfig` function (around line 20) and add a sibling helper directly below it:

```go
// ComboSelectConfigNoCreate is like ComboSelectConfig but with inline entity
// creation disabled. Used on forms (e.g. /brews/new) where prerequisite
// entities must already exist; the user is sent through the onboarding
// flow to create them.
func ComboSelectConfigNoCreate(entityType, apiEndpoint, suggestEndpoint, inputName, placeholder string, required bool) string {
	cfg := struct {
		EntityType      string `json:"entityType"`
		APIEndpoint     string `json:"apiEndpoint"`
		SuggestEndpoint string `json:"suggestEndpoint"`
		InputName       string `json:"inputName"`
		Placeholder     string `json:"placeholder"`
		Required        bool   `json:"required"`
		AllowCreate     bool   `json:"allowCreate"`
	}{
		EntityType:      entityType,
		APIEndpoint:     apiEndpoint,
		SuggestEndpoint: suggestEndpoint,
		InputName:       inputName,
		Placeholder:     placeholder,
		Required:        required,
		AllowCreate:     false,
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return "comboSelect({})"
	}
	return "comboSelect(" + string(b) + ")"
}
```

- [ ] **Step 2: Gate the dropdown "Create" item on `allowCreate`**

Same file, find the `<div v-if="query.trim() && !exactMatch"` block (around line 132) and change the `v-if` to require `allowCreate`:

Replace:
```html
<div
    v-if="query.trim() && !exactMatch"
    class="combo-item-create"
```
With:
```html
<div
    v-if="allowCreate && query.trim() && !exactMatch"
    class="combo-item-create"
```

(Do not touch the roaster sub-picker `v-if` at line ~219 — that's nested inside the bean modal flow and uses its own dropdown.)

- [ ] **Step 3: Read the flag in combo-select.js**

In `internal/web/assets/js/combo-select.js`, find the `comboSelect` factory (line 372) and add `allowCreate` to the returned object's config section. After the line `passthrough: config.passthrough || false,` (around line 383), insert:

```javascript
      // Default true so existing callsites (which don't pass the flag)
      // keep the current "+ Create new" behavior.
      allowCreate: config.allowCreate !== false,
```

- [ ] **Step 4: Short-circuit createNew() when disallowed**

In the same file, find the `createNew` method on the factory (search for `createNew(` — it's the method invoked by the dropdown create item). Add a guard as the first statement of the method body:

```javascript
      if (!this.allowCreate) return;
```

If `createNew` opens an inline create form (`showCreateForm = true`), also gate any sibling entry point (e.g. `startCreateForm`, if present) the same way. Search the file for `showCreateForm = true` and add the same guard on each path that sets it from a *user-driven* combo create (do not touch the roaster sub-picker, which has its own `startCreateRoaster`).

- [ ] **Step 5: Regenerate templ and verify build**

Run:
```bash
templ generate -f internal/web/components/combo_select.templ
go vet ./... && go build ./...
```
Expected: clean exit.

- [ ] **Step 6: Commit**

```bash
jj describe -m "feat(web): add allowCreate flag to combo-select"
```

---

## Task 3: Lock down brew form combo-selects

**Files:**
- Modify: `internal/arabica/web/pages/brew_form.templ`

- [ ] **Step 1: Switch bean / grinder / brewer combo configs**

In `internal/arabica/web/pages/brew_form.templ`, change three lines:

Line 193 — bean:
```go
// before
v-scope={ components.ComboSelectConfig("bean", "/api/beans", "/api/suggestions/beans", "bean_rkey", "Search or create a bean...", true) }
// after
v-scope={ components.ComboSelectConfigNoCreate("bean", "/api/beans", "/api/suggestions/beans", "bean_rkey", "Search beans...", true) }
```

Line 277 — grinder:
```go
// before
v-scope={ components.ComboSelectConfig("grinder", "/api/grinders", "/api/suggestions/grinders", "grinder_rkey", "Search or create a grinder...", false) }
// after
v-scope={ components.ComboSelectConfigNoCreate("grinder", "/api/grinders", "/api/suggestions/grinders", "grinder_rkey", "Search grinders...", false) }
```

Line 318 — brewer:
```go
// before
v-scope={ components.ComboSelectConfig("brewer", "/api/brewers", "/api/suggestions/brewers", "brewer_rkey", "Search or create a brew method...", false) }
// after
v-scope={ components.ComboSelectConfigNoCreate("brewer", "/api/brewers", "/api/suggestions/brewers", "brewer_rkey", "Search brew methods...", false) }
```

Leave the recipe combo at line 158 alone — recipes use `passthrough` and inline creation isn't relevant. (Roaster has no combo on the brew form directly; it lives in the bean modal, which we are intentionally not changing.)

- [ ] **Step 2: Regenerate and verify**

Run:
```bash
templ generate -f internal/arabica/web/pages/brew_form.templ
go vet ./... && go build ./...
```
Expected: clean exit.

- [ ] **Step 3: Commit**

```bash
jj describe -m "feat(arabica): disable inline entity creation in brew form"
```

---

## Task 4: Readiness redirect in `HandleBrewNew`

**Files:**
- Modify: `internal/arabica/handlers/brew.go`
- Test: `internal/arabica/handlers/onboarding_test.go` (created in this task)

The existing handler test setup hits 401 because OAuth context is nil; we lean on that pattern for the unauth case and add a tightly-scoped test for the redirect via a helper that lets us bypass the auth wall.

- [ ] **Step 1: Refactor the readiness gate as a testable helper**

In `internal/arabica/handlers/brew.go`, find `HandleBrewNew` (line 180). Add the readiness check after the auth check. Modify the function:

```go
import (
    // existing imports...
    "tangled.org/arabica.social/arabica/internal/database"
    "tangled.org/arabica.social/arabica/internal/onboarding"
)

func (h *Handlers) HandleBrewNew(w http.ResponseWriter, r *http.Request) {
    store, authenticated := h.GetAtprotoStore(r)
    if !authenticated {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }

    if !brewNewReady(r.Context(), store) {
        http.Redirect(w, r, "/#get-started", http.StatusFound)
        return
    }

    layoutData, _, _ := h.LayoutDataFromRequest(r, "New Brew")
    brewFormProps := coffeepages.BrewFormProps{
        Brew:           nil,
        RecipeRKey:     r.URL.Query().Get("recipe"),
        RecipeOwnerDID: r.URL.Query().Get("recipe_owner"),
    }
    if err := coffeepages.BrewFormPage(layoutData, brewFormProps).Render(r.Context(), w); err != nil {
        http.Error(w, "Failed to render page", http.StatusInternalServerError)
        log.Error().Err(err).Msg("Failed to render brew form")
    }
}

// brewNewReady wraps onboarding.CheckBrewReadiness with logging. A readiness
// check failure (rare; PDS down) is treated as "not ready" so the user gets
// the onboarding card instead of a blank brew form they can't submit.
func brewNewReady(ctx context.Context, store database.Store) bool {
    status, err := onboarding.CheckBrewReadiness(ctx, store)
    if err != nil {
        log.Warn().Err(err).Msg("brew-readiness check failed; treating as not ready")
        return false
    }
    return status.Ready()
}
```

- [ ] **Step 2: Write the readiness-helper test**

Create `internal/arabica/handlers/onboarding_test.go`:

```go
package coffeehandlers

import (
	"context"
	"errors"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/database"

	"github.com/stretchr/testify/assert"
)

func TestBrewNewReady_True(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	assert.True(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseWhenMissing(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseOnError(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, errors.New("pds down") },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/arabica/handlers/... -run TestBrewNewReady -v`
Expected: 3 tests PASS.

- [ ] **Step 4: Verify build**

Run: `templ generate && go vet ./... && go build ./...`
Expected: clean exit.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat(arabica): redirect /brews/new to onboarding when not ready"
```

---

## Task 5: `GetStartedCard` templ component

**Files:**
- Create: `internal/arabica/web/components/get_started_card.templ`

The card has four sub-sections (Brewers/Beans/Roasters/Grinders) and a footer status line. Each section renders a list of the user's existing items and a button that opens the existing entity create modal. The card is loaded/refreshed via HTMX.

- [ ] **Step 1: Write the component**

Create `internal/arabica/web/components/get_started_card.templ`. **Tabs only — no spaces.** A `templ fmt` post-tool hook runs automatically, but write tabs to start with.

```go
package coffeecomponents

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/onboarding"
)

// GetStartedCardProps is what the /api/get-started-card handler renders.
// Beans / Brewers / Grinders / Roasters are the user's current entities,
// shown as a short list per section.
type GetStartedCardProps struct {
	Readiness onboarding.ReadinessStatus
	Beans     []*arabica.Bean
	Brewers   []*arabica.Brewer
	Grinders  []*arabica.Grinder
	Roasters  []*arabica.Roaster
}

// GetStartedCard is the homepage onboarding surface. It always re-issues
// itself with the same id + hx-trigger so a `refreshManage` event
// (fired by entity-create modals on success) reloads it in place.
templ GetStartedCard(props GetStartedCardProps) {
	<section
		id="get-started"
		hx-get="/api/get-started-card"
		hx-trigger="refreshManage from:body"
		hx-swap="outerHTML"
		class="card p-4 sm:p-6 mb-6"
	>
		<header class="mb-4">
			<h2 class="text-xl font-bold text-primary">Get Started</h2>
			<p class="text-sm text-emphasis mt-1">
				Add at least one brewer and one bean before logging your first brew.
			</p>
		</header>

		@getStartedSection("Brewers", true, "brewer", brewerLabels(props.Brewers))
		@getStartedSection("Beans", true, "bean", beanLabels(props.Beans))
		@getStartedSection("Roasters", false, "roaster", roasterLabels(props.Roasters))
		@getStartedSection("Grinders", false, "grinder", grinderLabels(props.Grinders))

		@getStartedFooter(props.Readiness)
	</section>
}

// getStartedSection renders one entity bucket: heading + (required)/(optional)
// badge, existing items, and the "+ Add" button that opens the modal.
templ getStartedSection(title string, required bool, entityNoun string, items []string) {
	<div class="mb-4">
		<div class="flex items-center gap-2 mb-2">
			<h3 class="font-semibold text-primary">{ title }</h3>
			if required {
				<span class="badge-required text-xs uppercase tracking-wide">Required</span>
			} else {
				<span class="badge-optional text-xs uppercase tracking-wide">Optional</span>
			}
		</div>
		if len(items) > 0 {
			<ul class="text-sm text-emphasis mb-2 space-y-1">
				for _, label := range items {
					<li>· { label }</li>
				}
			</ul>
		} else {
			<p class="text-sm text-faint italic mb-2">None yet.</p>
		}
		<button
			type="button"
			class="btn-secondary text-sm"
			hx-get={ "/api/modals/" + entityNoun + "/new" }
			hx-target="#modal-container"
			hx-swap="innerHTML"
			onclick="setTimeout(function(){var d=document.querySelector('#modal-container dialog');if(d&&d.showModal)d.showModal();},0)"
		>
			+ Add a { entityNoun }
		</button>
	</div>
}

// getStartedFooter shows current status and unlocks the "log your first brew"
// CTA once both required prereqs are satisfied.
templ getStartedFooter(r onboarding.ReadinessStatus) {
	<div class="border-t border-divider pt-3 mt-2 flex items-center justify-between flex-wrap gap-3">
		<p class="text-sm text-emphasis">
			if r.HasBrewer {
				<span class="text-success">✓ brewer</span>
			} else {
				<span class="text-faint">✗ brewer</span>
			}
			<span class="mx-2 text-faint">·</span>
			if r.HasBean {
				<span class="text-success">✓ bean</span>
			} else {
				<span class="text-faint">✗ bean</span>
			}
			if r.Ready() {
				<span class="ml-2 font-medium">Ready to brew!</span>
			} else {
				<span class="ml-2 text-faint">— add 1 of each to start logging brews</span>
			}
		</p>
		if r.Ready() {
			<a href="/brews/new" class="btn-primary text-sm">
				Log your first brew →
			</a>
		}
	</div>
}

// --- label helpers ---------------------------------------------------

func brewerLabels(brewers []*arabica.Brewer) []string {
	out := make([]string, 0, len(brewers))
	for _, b := range brewers {
		if b == nil {
			continue
		}
		out = append(out, b.Name)
	}
	return out
}

func beanLabels(beans []*arabica.Bean) []string {
	out := make([]string, 0, len(beans))
	for _, b := range beans {
		if b == nil {
			continue
		}
		out = append(out, b.Name)
	}
	return out
}

func grinderLabels(grinders []*arabica.Grinder) []string {
	out := make([]string, 0, len(grinders))
	for _, g := range grinders {
		if g == nil {
			continue
		}
		out = append(out, g.Name)
	}
	return out
}

func roasterLabels(roasters []*arabica.Roaster) []string {
	out := make([]string, 0, len(roasters))
	for _, r := range roasters {
		if r == nil {
			continue
		}
		out = append(out, r.Name)
	}
	return out
}
```

> If `coffeecomponents` is not the actual package name in `internal/arabica/web/components/`, use whatever the existing files there declare (run `head -1 internal/arabica/web/components/*.templ | head -2` to confirm before editing). Likewise check existing CSS class names — `badge-required` / `badge-optional` / `text-success` / `border-divider` are placeholders; if the project uses different utility classes (e.g. raw Tailwind `bg-amber-100`), substitute matching ones. Grep existing components for `badge-` and `text-success` to confirm.

- [ ] **Step 2: Regenerate templ and verify build**

Run:
```bash
templ generate -f internal/arabica/web/components/get_started_card.templ
go vet ./... && go build ./...
```
Expected: clean exit.

- [ ] **Step 3: Commit**

```bash
jj describe -m "feat(arabica): add GetStartedCard onboarding component"
```

---

## Task 6: `/api/get-started-card` handler + route

**Files:**
- Create: `internal/arabica/handlers/onboarding.go`
- Modify: `internal/routing/routing.go`
- Test: append to `internal/arabica/handlers/onboarding_test.go`

- [ ] **Step 1: Implement the handler**

Create `internal/arabica/handlers/onboarding.go`:

```go
package coffeehandlers

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	coffeecomponents "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/database"
	"tangled.org/arabica.social/arabica/internal/onboarding"
)

// HandleGetStartedCard returns the rendered onboarding card. Reloaded via
// HTMX on every refreshManage event so newly-created entities show up.
//
// When the user is fully ready, the card still renders (with the green
// "Log your first brew" CTA). The home template decides whether to mount
// the card slot at all on the initial server render.
func (h *Handlers) HandleGetStartedCard(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	props, err := buildGetStartedCardProps(r.Context(), store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build get-started card props")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}

	if err := coffeecomponents.GetStartedCard(props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render get-started card")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

func buildGetStartedCardProps(ctx context.Context, store database.Store) (coffeecomponents.GetStartedCardProps, error) {
	status, err := onboarding.CheckBrewReadiness(ctx, store)
	if err != nil {
		return coffeecomponents.GetStartedCardProps{}, err
	}
	beans, err := store.ListBeans(ctx)
	if err != nil {
		return coffeecomponents.GetStartedCardProps{}, err
	}
	brewers, err := store.ListBrewers(ctx)
	if err != nil {
		return coffeecomponents.GetStartedCardProps{}, err
	}
	grinders, err := store.ListGrinders(ctx)
	if err != nil {
		return coffeecomponents.GetStartedCardProps{}, err
	}
	roasters, err := store.ListRoasters(ctx)
	if err != nil {
		return coffeecomponents.GetStartedCardProps{}, err
	}
	return coffeecomponents.GetStartedCardProps{
		Readiness: status,
		Beans:     beans,
		Brewers:   brewers,
		Grinders:  grinders,
		Roasters:  roasters,
	}, nil
}

// silence unused if MockStore doesn't have all funcs; keep this var off.
var _ = arabica.NSIDBean
```

(The trailing `var _ = arabica.NSIDBean` only stays if needed to keep the `arabica` import; remove it if `arabica.*` is referenced elsewhere in the file. If not needed, drop the import entirely.)

- [ ] **Step 2: Write the prop-builder test**

Append to `internal/arabica/handlers/onboarding_test.go`:

```go
func TestBuildGetStartedCardProps_Empty(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
		ListGrindersFunc: func(ctx context.Context) ([]*arabica.Grinder, error) { return nil, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, nil },
	}

	props, err := buildGetStartedCardProps(context.Background(), store)

	assert.NoError(t, err)
	assert.False(t, props.Readiness.Ready())
	assert.Empty(t, props.Beans)
	assert.Empty(t, props.Brewers)
}

func TestBuildGetStartedCardProps_Populated(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "b1", Name: "Ethiopia"}}, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "br1", Name: "V60"}}, nil },
		ListGrindersFunc: func(ctx context.Context) ([]*arabica.Grinder, error) { return nil, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return []*arabica.Roaster{{RKey: "r1", Name: "Onyx"}}, nil },
	}

	props, err := buildGetStartedCardProps(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, props.Readiness.Ready())
	assert.Len(t, props.Beans, 1)
	assert.Len(t, props.Brewers, 1)
	assert.Len(t, props.Roasters, 1)
	assert.Equal(t, "Ethiopia", props.Beans[0].Name)
}
```

> If `database.MockStore` lacks `ListGrindersFunc` / `ListRoastersFunc`, check `internal/database/store_mock.go` and add them following the existing pattern (`if m.ListXFunc != nil { return m.ListXFunc(ctx) }` plus a default empty return). This is a small additive change — keep it scoped.

- [ ] **Step 3: Register the route**

In `internal/routing/routing.go`, find the cluster of `mux.HandleFunc("GET /api/...")` for arabica (near the recipe modal routes around line 151). Add:

```go
mux.HandleFunc("GET /api/get-started-card", coffee.HandleGetStartedCard)
```

The exact insertion point: anywhere in the arabica-conditional block of the routing function (search for `coffee.HandleRecipeModalNew` and add nearby). If routing.go gates registration on `app.Name == "arabica"`, place it inside that block so oolong doesn't try to call it. (For oolong commit 2, we register a parallel oolong endpoint.)

- [ ] **Step 4: Run tests + build**

Run:
```bash
go test ./internal/arabica/handlers/... -v -run TestBuildGetStartedCardProps
templ generate
go vet ./... && go build ./...
```
Expected: 2 new tests PASS; build clean.

- [ ] **Step 5: Commit**

```bash
jj describe -m "feat(arabica): add /api/get-started-card endpoint"
```

---

## Task 7: Wire the home page (server-side readiness + slot + disabled CTA)

**Files:**
- Modify: `internal/web/pages/home.templ`
- Modify: `internal/web/components/shared.templ`
- Modify: `internal/handlers/feed.go`

- [ ] **Step 1: Extend `HomeProps` and render the card slot**

In `internal/web/pages/home.templ`:

1. Add the import for the onboarding package at the top:

```go
import (
    "tangled.org/arabica.social/arabica/internal/entities"
    "tangled.org/arabica.social/arabica/internal/onboarding"
    "tangled.org/arabica.social/arabica/internal/web/components"
)
```

2. Add a field to `HomeProps`:

```go
type HomeProps struct {
    IsAuthenticated bool
    UserDID         string
    AppName         string
    Descriptors     []*entities.Descriptor
    Readiness       *onboarding.ReadinessStatus // nil for unauth or non-arabica
}
```

3. In `HomeContent`, insert the card slot before the `WelcomeCard` call (but only for arabica + authenticated + not-ready). Replace the existing block:

```go
if props.IsAuthenticated {
    @components.WelcomeCard(components.WelcomeCardProps{
        IsAuthenticated: props.IsAuthenticated,
        UserDID:         props.UserDID,
        AppName:         props.AppName,
    })
    // ...
}
```

With:

```go
if props.IsAuthenticated {
    if props.AppName != "oolong" && props.Readiness != nil && !props.Readiness.Ready() {
        // GetStartedCard is rendered async via /api/get-started-card so the
        // hx-trigger="load, refreshManage from:body" keeps it fresh as the
        // user adds entities via the modal flow.
        <div id="get-started" hx-get="/api/get-started-card" hx-trigger="load, refreshManage from:body" hx-swap="outerHTML"></div>
    }
    @components.WelcomeCard(components.WelcomeCardProps{
        IsAuthenticated: props.IsAuthenticated,
        UserDID:         props.UserDID,
        AppName:         props.AppName,
        Ready:           props.Readiness == nil || props.Readiness.Ready(),
    })
    <!-- Modal container for entity create/edit dialogs opened from dashboard -->
    <div id="modal-container"></div>
    if props.AppName != "oolong" {
        <div id="incomplete-records-section" hx-get="/api/incomplete-records" hx-trigger="load, refreshManage from:body" hx-swap="innerHTML"></div>
        <div hx-get="/api/popular-recipes" hx-trigger="load" hx-swap="innerHTML"></div>
    }
}
```

(`Ready: props.Readiness == nil || props.Readiness.Ready()` means: oolong/unauth treat the user as ready; arabica auth users get the real flag.)

- [ ] **Step 2: Update `WelcomeCardProps` and disabled state**

In `internal/web/components/shared.templ`:

1. Find `WelcomeCardProps` (search for `type WelcomeCardProps`) and add the field:

```go
type WelcomeCardProps struct {
    IsAuthenticated bool
    UserDID         string
    AppName         string
    Ready           bool
}
```

2. In `welcomeAuthenticatedArabica` (line 409), change the "Log Brew" anchor block. Replace:

```go
<a href="/brews/new" class="home-action-primary block text-center py-4 px-4 rounded-xl">
    <span class="text-base font-semibold">Log Brew</span>
</a>
```

With:

```go
if props.Ready {
    <a href="/brews/new" class="home-action-primary block text-center py-4 px-4 rounded-xl">
        <span class="text-base font-semibold">Log Brew</span>
    </a>
} else {
    <a href="#get-started" class="home-action-primary block text-center py-4 px-4 rounded-xl opacity-60" aria-disabled="true">
        <span class="text-base font-semibold">Log Brew — finish setup ↑</span>
    </a>
}
```

3. Because `welcomeAuthenticatedArabica` now reads `props.Ready`, change its signature from `welcomeAuthenticatedArabica(userDID string)` to `welcomeAuthenticatedArabica(props WelcomeCardProps)`. Update the body to use `props.UserDID` wherever `userDID` appears, and update the call site in `WelcomeCard` (line 397) to pass `props` through:

```go
templ WelcomeCard(props WelcomeCardProps) {
    if props.IsAuthenticated {
        if props.AppName == "oolong" {
            @welcomeAuthenticatedOolong(props.UserDID)
        } else {
            @welcomeAuthenticatedArabica(props)
        }
    }
}
```

(The legacy two-arg `WelcomeAuthenticated(userDID string)` shim at line 405 should be updated to construct a `WelcomeCardProps{UserDID: userDID, Ready: true}` so it stays backward-compatible.)

- [ ] **Step 3: Populate readiness in `HomeHandler`**

In `internal/handlers/feed.go`, modify `HandleHome` (line 60). Add an import:

```go
import (
    // existing imports...
    "tangled.org/arabica.social/arabica/internal/onboarding"
)
```

Before constructing `homeProps`, derive readiness for authed arabica users:

```go
var readiness *onboarding.ReadinessStatus
if isAuthenticated && appName != "oolong" {
    if store, ok := h.GetAtprotoStore(r); ok {
        if status, err := onboarding.CheckBrewReadiness(r.Context(), store); err != nil {
            log.Warn().Err(err).Msg("readiness check failed; treating user as ready to avoid false block")
        } else {
            readiness = &status
        }
    }
}

homeProps := pages.HomeProps{
    IsAuthenticated: isAuthenticated,
    UserDID:         didStr,
    AppName:         appName,
    Descriptors:     descriptors,
    Readiness:       readiness,
}
```

> If `h.GetAtprotoStore` does not exist on `*Handler` (it's defined on the arabica `*Handlers`), find the equivalent on the shared handler — likely a method or helper that yields a `database.Store` for the request. Grep `grep -nE "func.*Handler.*Store" internal/handlers/*.go` to find it. If no such method exists, factor a small `storeFromRequest(r) (database.Store, bool)` helper using the same OAuth-session lookup that `coffeehandlers.GetAtprotoStore` uses, and call that. Keep the helper in `internal/handlers/` so both apps can use it.

If `nil` readiness means "treat as ready" (per the home template change in step 1), an error during readiness check falls through to the normal brew form — which then runs its own readiness check in `HandleBrewNew` and redirects properly. Two layers of safety.

- [ ] **Step 4: Regenerate and build**

Run:
```bash
templ generate
go vet ./... && go build ./...
```
Expected: clean exit.

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 6: Manual smoke test**

Start the dev server:
```bash
just run
```

In a private/incognito browser:
1. Log in with a fresh account (or one with zero beans + zero brewers).
2. Confirm the home page shows the "Get Started" card above the welcome grid, and the "Log Brew" button reads "Log Brew — finish setup ↑" and is dimmed.
3. Click "+ Add a brewer", complete the modal. Confirm card refreshes and brewer appears in the list. Status line shows `✓ brewer  ✗ bean`.
4. Click "+ Add a bean", complete the modal. Confirm card refreshes, status line shows `✓ brewer  ✓ bean — Ready to brew!`, and the green "Log your first brew →" CTA appears.
5. Click the CTA; `/brews/new` renders the brew form.
6. Confirm the bean / grinder / brewer combo-selects on the brew form **do not** show "+ Create new" in their dropdowns when you type a non-matching query.
7. Navigate directly to `/brews/new` in a separate session that has zero entities; confirm 302 redirect to `/#get-started`.
8. Refresh home — "Log Brew" is now the normal active button (since the prereqs are satisfied).

- [ ] **Step 7: Commit**

```bash
jj describe -m "feat(arabica): show onboarding card on home and gate Log Brew"
```

---

## Follow-up (separate commit, not in this plan)

Per the spec, the oolong variant lands as a second commit:
- Mirror `GetStartedCard` with steeper / tea / vendor / etc. labels.
- Add a corresponding `/api/oolong/get-started-card` route in the oolong-conditional block of routing.
- Add an oolong readiness check (probably `internal/onboarding/readiness.go` grows an `OolongReadiness` helper or moves into per-app readiness).
- Update `welcomeAuthenticatedOolong` for the disabled-state CTA.
- Render the slot in home.templ for the oolong branch.

---

## Self-Review Notes

- **Spec coverage:**
  - Prereq rule (bean + brewer) — Task 1.
  - Home `GetStartedCard` above WelcomeCard — Task 7 step 1.
  - "Log Brew" disabled-and-anchored — Task 7 step 2.
  - `/brews/new` 302 redirect — Task 4.
  - Brew form combo-selects lose inline create — Tasks 2 + 3.
  - Card layout (Brewer → Bean → Roaster → Grinder, badges, lists, "+ Add" buttons, footer status, "Log your first brew" CTA) — Task 5.
  - Readiness via existing collections, no persistence — Task 1.
  - HTMX `refreshManage` refresh — Task 5 (component) + Task 6 (handler).
  - Arabica-only this commit, oolong follow-up — flagged in Header + Follow-up.
  - Unit tests on readiness + props builder — Tasks 1 + 6.
- **No placeholders** other than the two explicit "if the codebase shape differs from what I observed, do X" notes in Tasks 5 and 7 — these are real "verify before editing" cues for the implementer, not deferred work.
- **Type consistency:** `ReadinessStatus` / `Ready()` / `HasBean` / `HasBrewer` used identically throughout. `GetStartedCardProps` fields match what the handler populates and what the templ component consumes.
