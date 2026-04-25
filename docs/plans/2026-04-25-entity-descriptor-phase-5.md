# Entity Descriptor — Phase 5': Modal Shell Extraction

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Extract the dialog/header/error/footer scaffolding shared across
the five entity modals into a single `ModalShell` component. Each modal
keeps its own templ body for the field markup. No DSL, no FieldSpec list —
just a shared shell.

**Parent spec:** `docs/entity-descriptor-refactor.md`
**Previous phase:** `docs/plans/2026-04-25-entity-descriptor-phase-1.md`

**Tech Stack:** Go 1.26, Templ.

---

## Major Design Changes

### What's actually duplicated across modals

After auditing all five modals, the **shell** is identical (modulo entity
name and URL):

1. `<dialog id="entity-modal" class="modal-dialog">` open
2. `<div class="modal-content">` wrapper
3. `<h3 class="modal-title">` with conditional "Edit X" / "Add X"
4. `<form>` with:
   - Conditional `hx-put` (edit) / `hx-post` (create) URL
   - Standard `hx-trigger`, `hx-swap`, `class`
   - `x-data="{ serverError: '' }"` (recipe extends this with pour state)
   - The big `hx-on::after-request` handler (identical, except recipe is
     **missing the 401 session-expired branch** — a latent bug we'll fix
     while we're here)
5. `<div x-show="serverError" ...>` server error display
6. **(per-entity field body — this is what stays in each modal's templ)**
7. Footer `<div class="flex gap-2 pt-2">` with Save/Cancel buttons
8. Close `</form></div></dialog>`

### What stays per-entity

- The field body — fieldsets, inputs, dropdowns, the bean modal's roaster
  picker, the bean rating slider, the brewer modal's conditional sections,
  the recipe modal's pour repeater
- Per-entity Alpine state — recipe needs `pours`, `addPour()`,
  `removePour()` plumbed through `ExtraXData`
- The bean modal's "bag is closed" checkbox (only shown on edit)

### The component

```go
type ModalShellProps struct {
    Type       lexicons.RecordType  // looked up via entities.Get()
    RKey       string               // "" → create (POST), non-empty → edit (PUT)
    ExtraXData string               // optional Alpine x-data; defaults to `{ serverError: '' }`
}
```

The shell looks up the descriptor, derives `DisplayName` and `URLPath`,
builds the action URL (`/api/{URLPath}` for create, `/api/{URLPath}/{rkey}`
for edit), and renders the wrapper. The body comes in as templ children.

### Why this beats the rejected FieldSpec design

- **No DSL**: each modal's body is plain templ markup you can read
  top-to-bottom
- **Bespoke widgets stay free**: the roaster picker, rating slider,
  conditional brewer fields, recipe pour repeater all stay where they are
- **Cross-cutting changes are still cheap**: changing the error display,
  rewriting the after-request handler, retitling buttons — all single-edit
- **LOC saved**: ~150 (5 modals × ~30 lines of shell each)
- **Risk**: low. If a future modal needs a different shell, it can opt out
  by not using `ModalShell`.

### What is NOT changing

- Field bodies (the entire point — they stay readable)
- `getStringValue` / `recordTypeOf` (already migrated in phase 0)
- Modal route registration (phase 6 work)
- Modal handler functions
- The `ConfirmDeleteModal` (different shape — confirmation, not edit form)

---

## Phase 5': Tasks

### Task 1: Create `ModalShell` component

**Files:**

- Create: `internal/web/components/modal_shell.templ`

**Step 1: Define the shell**

```templ
package components

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// ModalShellProps configures the shared dialog wrapper used by entity
// create/edit modals. Body content is passed as templ children.
type ModalShellProps struct {
	Type       lexicons.RecordType
	RKey       string // "" = create, non-empty = edit
	ExtraXData string // optional Alpine x-data (defaults to `{ serverError: '' }`)
}

// modalAfterRequest is the htmx after-request handler shared across
// entity modals. Closes the dialog on success, surfaces session-expired
// on 401, and otherwise displays a generic error in the form.
const modalAfterRequest = `if(event.detail.successful) { this.closest('dialog').close(); htmx.trigger('body', 'refreshManage'); } else if(event.detail.xhr && event.detail.xhr.status === 401) { this.closest('dialog').close(); window.__showSessionExpiredModal(); } else { this._x_dataStack[0].serverError = event.detail.xhr ? 'Something went wrong. Please try again.' : 'Connection error. Check your network.'; }`

func modalShellXData(extra string) string {
	if extra != "" {
		return extra
	}
	return "{ serverError: '' }"
}

func modalActionURL(d *entities.Descriptor, rkey string) string {
	if rkey != "" {
		return "/api/" + d.URLPath + "/" + rkey
	}
	return "/api/" + d.URLPath
}

templ ModalShell(props ModalShellProps) {
	if d := entities.Get(props.Type); d != nil {
		<dialog id="entity-modal" class="modal-dialog">
			<div class="modal-content">
				<h3 class="modal-title">
					if props.RKey != "" {
						Edit { d.DisplayName }
					} else {
						Add { d.DisplayName }
					}
				</h3>
				<form
					if props.RKey != "" {
						hx-put={ modalActionURL(d, props.RKey) }
					} else {
						hx-post={ modalActionURL(d, "") }
					}
					hx-trigger="submit"
					hx-swap="none"
					x-data={ modalShellXData(props.ExtraXData) }
					hx-on::after-request={ modalAfterRequest }
					class="space-y-5"
				>
					<div x-show="serverError" x-cloak class="bg-red-50 border border-red-200 rounded-lg p-3 text-sm text-red-800" x-text="serverError"></div>
					{ children... }
					<div class="flex gap-2 pt-2">
						<button type="submit" class="flex-1 btn-primary">Save</button>
						<button
							type="button"
							@click="$el.closest('dialog').close()"
							class="flex-1 btn-secondary"
						>
							Cancel
						</button>
					</div>
				</form>
			</div>
		</dialog>
	}
}
```

**Step 2: Generate**

```bash
templ generate -f internal/web/components/modal_shell.templ
```

### Task 2: Migrate `GrinderDialogModal` (proof of concept — simplest)

**Files:**

- Modify: `internal/web/components/dialog_modals.templ`

**Step 1: Replace shell, keep body**

Replace the `templ GrinderDialogModal(...)` definition. The body
(everything between the form open tag and the Save button) stays
verbatim, wrapped in a `@ModalShell(...) { ... }` call.

**Step 2: Regenerate and verify**

```bash
templ generate -f internal/web/components/dialog_modals.templ
go build ./...
```

Open the grinder modal in the browser. Confirm: Edit/Add label, form
submission, error display, footer buttons all work identically.

### Task 3: Migrate `RoasterDialogModal`

Same shape as grinder. Single edit + regenerate.

### Task 4: Migrate `BrewerDialogModal`

Same shape. The conditional espresso/pourover sections inside the body
stay as-is — they're field-body concerns, not shell concerns.

### Task 5: Migrate `BeanDialogModal`

The roaster picker (~80 lines of Alpine) and rating slider (Alpine state)
stay inside the body verbatim. The bean's "bag is closed" checkbox also
stays in the body. Only the shell scaffolding is extracted.

### Task 6: Migrate `RecipeDialogModal`

Recipe is the special case:

- **Fix the latent bug**: recipe's current `hx-on::after-request` is
  missing the 401 session-expired branch. Migrating to `ModalShell`
  fixes it automatically (the shell uses the canonical handler).
- **Plumb pour state through `ExtraXData`**: pass
  `fmt.Sprintf("{ serverError: '', pours: %s, addPour() { this.pours.push({water: '', time: ''}); }, removePour(i) { this.pours.splice(i, 1); } }", recipePourJSON(recipe))`
  as `ExtraXData`. The shell uses it instead of the default.

### Task 7: Verify

**Commands:**

```bash
go vet ./...
go build ./...
go test ./...
just run
```

**Manual smoke test (per modal):**

For each of bean, grinder, brewer, roaster, recipe:

1. Open the create modal — title reads "Add X", submitting POSTs to `/api/{path}`
2. Open the edit modal for an existing record — title reads "Edit X",
   submitting PUTs to `/api/{path}/{rkey}`
3. Trigger a server error (e.g. invalid input) — error display appears
4. Cancel button closes the dialog
5. Successful submit closes the dialog and triggers `refreshManage`

Bean-specific: roaster picker still works, rating slider still works.
Brewer-specific: espresso/pourover conditional fields still toggle.
Recipe-specific: pour add/remove still works; 401 now closes the dialog
and surfaces the session-expired modal (previously didn't).

**Expected delta:**

- ~150 LOC removed from `dialog_modals.templ` (5 × ~30 lines of shell)
- ~80 LOC added in `modal_shell.templ`
- Net: ~−70 LOC, plus a latent recipe bug fixed and zero DSL risk.

---

## Out of scope for phase 5'

- Field-body abstraction or FieldSpec DSL (rejected — see parent spec)
- Modal route loop (phase 6)
- Suggestions config from descriptor (phase 6)
- Migrating `ConfirmDeleteModal` (different shape; not part of the
  create/edit family)
