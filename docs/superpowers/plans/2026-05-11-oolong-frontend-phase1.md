# Phase 1: Descriptor Render Hooks — Implementation Plan

> **For agentic workers:** This is the Phase 1 plan from `docs/superpowers/specs/2026-05-11-oolong-frontend-split-design.md`. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Remove the three entity-dispatch `switch` sites from `internal/web/pages/feed.templ` by adding render-hook fields to `entities.Descriptor`, wired by an arabica bridge file. Zero behavior change.

**Architecture:** Each arabica descriptor gains three new fields — `RenderFeedContent func(*feed.FeedItem) templ.Component`, `FeedCardCompact bool`, `EditURL func(*feed.FeedItem) string`. A new bridge file in `internal/web/components/` populates these at init by referencing existing templ components. `feed.templ` calls into the descriptor instead of switching on `RecordType`. Bridge file will move to `internal/arabica/web/components/` in Phase 2.

**Tech Stack:** Go, templ, existing `entities` registry pattern.

---

## File Structure

**Modify:**
- `internal/entities/entities.go` — add three fields to `Descriptor`, import `templ` and `feed`
- `internal/web/pages/feed.templ` — replace three switches with descriptor dispatch

**Create:**
- `internal/web/components/feed_content.templ` — wrapper templ funcs returning the entity-specific clickable block for each non-brew entity (Brew already has `FeedBrewContentClickable` in `feed.templ`; it will move into this new file for symmetry)
- `internal/web/components/descriptor_bridge.go` — `init()` that assigns render hooks to each arabica descriptor
- `internal/web/components/descriptor_bridge_test.go` — verifies every arabica descriptor has hooks populated

---

## Tasks

### Task 1: Add render-hook fields to `entities.Descriptor`

**Files:**
- Modify: `internal/entities/entities.go`

- [ ] **Step 1:** Add imports for `github.com/a-h/templ` and `tangled.org/arabica.social/arabica/internal/feed`.

- [ ] **Step 2:** Add three fields to `Descriptor` after `RecordToModel`:

```go
// RenderFeedContent returns the entity-specific clickable block for
// feed.templ (anchor + content). nil means the entity does not appear
// in the feed and the feed shell should skip the content slot.
RenderFeedContent func(item *feed.FeedItem) templ.Component

// FeedCardCompact indicates the feed card uses the compact style
// (less padding for sparse content like grinder/brewer/roaster).
FeedCardCompact bool

// EditURL returns the dedicated edit-page URL for an item, or "" if
// the entity is edited via modal on the manage page.
EditURL func(item *feed.FeedItem) string
```

- [ ] **Step 3:** Run `go build ./...`. Expected: success (no callers populate the new fields yet, but they're optional).

- [ ] **Step 4:** Run `go test ./internal/entities/...`. Expected: existing tests pass.

---

### Task 2: Create per-entity feed-content wrappers

**Files:**
- Create: `internal/web/components/feed_content.templ`

The existing `feed.templ` switch wraps non-brew entities in an anchor:

```
<a href={ shareURL } class="block hover:opacity-90 transition-opacity">
    @components.BeanContent(item.Bean())
</a>
```

For Bean/Roaster/Grinder/Brewer it also has a nil-guard on `item.Bean()` etc. We replicate that wrapping in per-entity templ funcs so the descriptor's `RenderFeedContent` becomes a clean one-liner.

- [ ] **Step 1:** Write `internal/web/components/feed_content.templ`:

```templ
package components

import (
	"fmt"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/models/arabica"
)

// feedItemShareURL mirrors getFeedItemShareURL in pages/feed.templ.
// Duplicated here so this file is self-contained for the move to
// internal/arabica/web/components/ in Phase 2.
func feedItemShareURL(item *feed.FeedItem) string {
	if d := entities.Get(item.RecordType); d != nil {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/%s/%s?owner=%s", d.URLPath, rkey, item.Author.DID)
		}
	}
	return fmt.Sprintf("/profile/%s", item.Author.DID)
}

templ BeanFeedContent(item *feed.FeedItem) {
	if bean, ok := item.Record.(*arabica.Bean); ok && bean != nil {
		<a href={ templ.SafeURL(feedItemShareURL(item)) } class="block hover:opacity-90 transition-opacity">
			@BeanContent(bean)
		</a>
	}
}

templ RoasterFeedContent(item *feed.FeedItem) {
	if roaster, ok := item.Record.(*arabica.Roaster); ok && roaster != nil {
		<a href={ templ.SafeURL(feedItemShareURL(item)) } class="block hover:opacity-90 transition-opacity">
			@RoasterContent(roaster)
		</a>
	}
}

templ GrinderFeedContent(item *feed.FeedItem) {
	if grinder, ok := item.Record.(*arabica.Grinder); ok && grinder != nil {
		<a href={ templ.SafeURL(feedItemShareURL(item)) } class="block hover:opacity-90 transition-opacity">
			@GrinderContent(grinder)
		</a>
	}
}

templ BrewerFeedContent(item *feed.FeedItem) {
	if brewer, ok := item.Record.(*arabica.Brewer); ok && brewer != nil {
		<a href={ templ.SafeURL(feedItemShareURL(item)) } class="block hover:opacity-90 transition-opacity">
			@BrewerContent(brewer)
		</a>
	}
}
```

Note: `BeanContent`, `RoasterContent`, `GrinderContent`, `BrewerContent` already exist in `record_*.templ` files. The accessor methods `item.Bean()`, `item.Roaster()`, etc. used in the current switch were the typed-field accessors before Phase E; today `item.Record` is `any` and entities live there. Will verify correct accessor pattern by reading existing usages before running this.

- [ ] **Step 2:** Before writing the file, verify the current accessor pattern. Run:

```bash
grep -n "item\.Bean()\|item\.Record" internal/web/pages/feed.templ internal/feed/service.go | head -20
```

Adjust the templ to match. (`item.Bean()` returns `*arabica.Bean` from nil-safe accessor — keep that pattern if it still exists.)

- [ ] **Step 3:** Run `templ generate -f internal/web/components/feed_content.templ`.

- [ ] **Step 4:** Run `go build ./...`. Expected: success.

---

### Task 3: Create descriptor bridge file

**Files:**
- Create: `internal/web/components/descriptor_bridge.go`

- [ ] **Step 1:** Write the bridge:

```go
// Package components: descriptor_bridge.go wires arabica entities'
// templ render hooks into the entities.Descriptor registry. This
// file lives in internal/web/components/ during Phase 1 of the
// oolong-frontend-split refactor; it moves to
// internal/arabica/web/components/ in Phase 2 alongside the
// arabica-specific record_*.templ files.
package components

import (
	"fmt"

	"github.com/a-h/templ"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	// Bean
	if d := entities.Get(lexicons.RecordTypeBean); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return BeanFeedContent(item)
		}
	}
	// Roaster
	if d := entities.Get(lexicons.RecordTypeRoaster); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return RoasterFeedContent(item)
		}
		d.FeedCardCompact = true
	}
	// Grinder
	if d := entities.Get(lexicons.RecordTypeGrinder); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return GrinderFeedContent(item)
		}
		d.FeedCardCompact = true
	}
	// Brewer
	if d := entities.Get(lexicons.RecordTypeBrewer); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return BrewerFeedContent(item)
		}
		d.FeedCardCompact = true
	}
	// Recipe — uses FeedRecipeContent which still lives in feed.templ
	// for now; will be moved into feed_content.templ in Phase 2 along
	// with the rest of arabica's templ tree.
	if d := entities.Get(lexicons.RecordTypeRecipe); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return recipeFeedContentBridge(item)
		}
	}
	// Brew — uses FeedBrewContentClickable, currently in feed.templ.
	if d := entities.Get(lexicons.RecordTypeBrew); d != nil {
		d.RenderFeedContent = func(item *feed.FeedItem) templ.Component {
			return brewFeedContentBridge(item)
		}
		d.EditURL = func(item *feed.FeedItem) string {
			// Brew is the only entity with a dedicated edit page.
			// We can't import pages, so reconstruct the URL using
			// the descriptor's URLPath and the item's RKey.
			rkey := item.RKey()
			if rkey == "" {
				return ""
			}
			return fmt.Sprintf("/brews/%s/edit", rkey)
		}
	}
}
```

- [ ] **Step 2:** Add the two bridge funcs that defer to existing pages/feed.templ templ functions. Since `pages` imports `components` and not vice versa, we cannot call `pages.FeedBrewContentClickable` from here without a cycle. **Resolution:** move `FeedBrewContentClickable` and `FeedRecipeContent` from `internal/web/pages/feed.templ` into `internal/web/components/feed_content.templ` (they were always candidates for the components package; they live in pages today purely for proximity). Update `feed.templ` callers accordingly.

After the move:

```go
func brewFeedContentBridge(item *feed.FeedItem) templ.Component {
	return FeedBrewContentClickable(item)
}

func recipeFeedContentBridge(item *feed.FeedItem) templ.Component {
	return FeedRecipeContent(item)
}
```

Alternatively, drop the bridge funcs and assign directly:

```go
d.RenderFeedContent = FeedBrewContentClickable
```

(Templ-generated functions have signature `func(...) templ.Component`, which is exactly what the field expects. Verify generated signature before assigning directly.)

- [ ] **Step 3:** Run `go build ./...`. Expected: success.

---

### Task 4: Move `FeedBrewContentClickable` and `FeedRecipeContent` from pages/feed.templ to components/feed_content.templ

**Files:**
- Modify: `internal/web/pages/feed.templ` — remove the two templ definitions (lines around 306 and 436)
- Modify: `internal/web/components/feed_content.templ` — add the two definitions, prefixed with `components.` qualifier removed since they're now in this package
- Modify: `internal/web/pages/feed.templ` — change call sites from `@FeedBrewContentClickable(item)` to `@components.FeedBrewContentClickable(item)` and similarly for `@FeedRecipeContent(item)`

- [ ] **Step 1:** Read `internal/web/pages/feed.templ` lines 300-475 to capture both templ funcs verbatim.

- [ ] **Step 2:** Append them to `internal/web/components/feed_content.templ`, adjusting any package-qualified references (e.g. `components.X` becomes just `X`).

- [ ] **Step 3:** Delete the two templ funcs from `internal/web/pages/feed.templ`.

- [ ] **Step 4:** Update the two call sites in `feed.templ`:
  - Line ~219 `@FeedBrewContentClickable(item)` → `@components.FeedBrewContentClickable(item)`
  - Line ~247 `@FeedRecipeContent(item)` → `@components.FeedRecipeContent(item)`

- [ ] **Step 5:** Run `templ generate`.

- [ ] **Step 6:** Run `go build ./...` and `go vet ./...`. Expected: success.

---

### Task 5: Replace feed.templ line 217 switch with descriptor dispatch

**Files:**
- Modify: `internal/web/pages/feed.templ`

- [ ] **Step 1:** Replace the entire switch block (current lines 217-250) with:

```templ
if d := entities.Get(item.RecordType); d != nil && d.RenderFeedContent != nil {
	@d.RenderFeedContent(item)
}
```

- [ ] **Step 2:** Run `templ generate -f internal/web/pages/feed.templ`.

- [ ] **Step 3:** Run `go build ./...`. Expected: success.

---

### Task 6: Replace feed.templ line 482 (`feedCardClass`) switch with descriptor lookup

**Files:**
- Modify: `internal/web/pages/feed.templ`

- [ ] **Step 1:** Replace the switch at line ~482:

```go
func feedCardClass(item *feed.FeedItem) string {
	classes := "feed-card"
	if d := entities.Get(item.RecordType); d != nil {
		classes += " feed-card-" + d.Noun
		if d.FeedCardCompact {
			classes += " feed-card-compact"
		}
	}
	return classes
}
```

- [ ] **Step 2:** Run `go build ./...`. Expected: success.

---

### Task 7: Replace feed.templ line 516 (`getEditURL`) switch with descriptor lookup

**Files:**
- Modify: `internal/web/pages/feed.templ`

- [ ] **Step 1:** Replace `getEditURL`:

```go
func getEditURL(item *feed.FeedItem) string {
	if d := entities.Get(item.RecordType); d != nil && d.EditURL != nil {
		return d.EditURL(item)
	}
	return ""
}
```

- [ ] **Step 2:** Remove the now-unused `lexicons` import from `feed.templ` if no other references remain. Run `goimports -w internal/web/pages/feed_templ.go` after regenerating.

- [ ] **Step 3:** Run `templ generate` then `go build ./...` and `go vet ./...`. Expected: success.

---

### Task 8: Add bridge test

**Files:**
- Create: `internal/web/components/descriptor_bridge_test.go`

- [ ] **Step 1:** Write the test:

```go
package components

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestDescriptorBridge_AllArabicaEntitiesHaveFeedRenderer(t *testing.T) {
	want := []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeRoaster,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeBrewer,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeBrew,
	}
	for _, rt := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		assert.NotNil(t, d.RenderFeedContent, "RenderFeedContent not wired for %s", rt)
	}
}

func TestDescriptorBridge_BrewHasEditURL(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeBrew)
	assert.NotNil(t, d.EditURL)
}

func TestDescriptorBridge_CompactEntities(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeRoaster,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeBrewer,
	} {
		d := entities.Get(rt)
		assert.True(t, d.FeedCardCompact, "%s should be FeedCardCompact", rt)
	}
}
```

- [ ] **Step 2:** Run `go test ./internal/web/components/... -run TestDescriptorBridge -v`. Expected: PASS.

---

### Task 9: Run full test suite and smoke check

- [ ] **Step 1:** Run `go vet ./...`. Expected: no output.

- [ ] **Step 2:** Run `go build ./...`. Expected: success.

- [ ] **Step 3:** Run `go test ./...`. Expected: all green.

- [ ] **Step 4:** Manual smoke (only if user wants it): `just run`, load `/feed`, verify each entity card type renders.

---

### Task 10: Commit

- [ ] **Step 1:**

```bash
git add internal/entities/entities.go \
        internal/web/components/feed_content.templ \
        internal/web/components/feed_content_templ.go \
        internal/web/components/descriptor_bridge.go \
        internal/web/components/descriptor_bridge_test.go \
        internal/web/pages/feed.templ \
        internal/web/pages/feed_templ.go \
        docs/superpowers/plans/2026-05-11-oolong-frontend-phase1.md
```

- [ ] **Step 2:**

```bash
git commit -m "refactor(web): descriptor-driven feed entity dispatch

Remove the three entity-type switches in pages/feed.templ by adding
RenderFeedContent, FeedCardCompact, and EditURL fields to
entities.Descriptor. An arabica bridge file in web/components/ wires
the hooks at init by referencing existing record_*.templ components.

Phase 1 of docs/superpowers/specs/2026-05-11-oolong-frontend-split-design.md.

Zero behavior change for arabica; this unblocks oolong from
contributing its own entity render hooks without touching shared
feed.templ.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```
