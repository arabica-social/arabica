# Entity Descriptor — Phase 1: Templ Data Switches

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Migrate the remaining `RecordType` switches in `feed.templ` (share
URL, share title, delete URL) onto the descriptor + new `FeedItem` accessor
methods. Replace hardcoded entity labels in OG card constructors with
`Descriptor.Noun`. No behavior change; pure refactor.

**Parent spec:** `docs/entity-descriptor-refactor.md`
**Previous phase:** `docs/plans/2026-04-25-entity-descriptor-phase-0.md`

**Tech Stack:** Go 1.26, Templ, no new dependencies.

---

## Major Design Changes

### Container-side accessors on `FeedItem`

Phase 0 added `(*FeedItem).Record()` to give callers a typed-pointer
accessor without a switch. Phase 1 extends the same pattern with two
more accessors:

- `(*FeedItem).RKey() string` — the rkey of whichever record is set
- `(*FeedItem).DisplayTitle() string` — a human-readable title for
  share UI; brew is special-cased (uses bean name)

Both methods follow the same convention: a single switch in
`feed/service.go`, co-located with the typed fields. Adding a new
entity to FeedItem requires updating both methods, which is obvious
because they live next to the field declarations.

### Rule of thumb confirmed

We are flattening **data switches** (URLs, labels, titles) but keeping
**rendering switches** (which templ component to invoke). The
`feed.templ` content-rendering switch at line 212 (dispatching to
`BeanContent`, `RoasterContent`, etc.) is **not** migrated — it
dispatches to genuinely different templ components, which the
descriptor pattern can't carry.

### What is NOT changing in phase 1

- The `Store` / `AtprotoStore` interfaces
- `UserCache` typed fields
- Per-entity record conversion (`records.go`)
- The content-rendering switch in `feed.templ` (line 212)
- Per-entity `Draw*Card` functions in `ogcard/entities.go` (the field
  rendering is genuinely different per entity)
- `getEditURL` (only handles brew today; not worth touching)

### What this phase enables

After phase 1, the feed templ has no remaining data switches over
`RecordType`. The next migration target (phase 5' modal shell
extraction) is independent and can land at any time.

---

## Phase 1: Tasks

### Task 1: Add `RKey()` and `DisplayTitle()` accessors to `FeedItem`

**Files:**

- Modify: `internal/feed/service.go`

**Step 1: Add `RKey()` method**

After the existing `Record()` method, add:

```go
// RKey returns the record key of whichever typed record is set on this
// FeedItem, or "" if none. Lets callers build URLs without a type switch.
func (f *FeedItem) RKey() string {
	switch f.RecordType {
	case lexicons.RecordTypeBean:
		if f.Bean != nil {
			return f.Bean.RKey
		}
	case lexicons.RecordTypeRoaster:
		if f.Roaster != nil {
			return f.Roaster.RKey
		}
	case lexicons.RecordTypeGrinder:
		if f.Grinder != nil {
			return f.Grinder.RKey
		}
	case lexicons.RecordTypeBrewer:
		if f.Brewer != nil {
			return f.Brewer.RKey
		}
	case lexicons.RecordTypeRecipe:
		if f.Recipe != nil {
			return f.Recipe.RKey
		}
	case lexicons.RecordTypeBrew:
		if f.Brew != nil {
			return f.Brew.RKey
		}
	}
	return ""
}
```

**Step 2: Add `DisplayTitle()` method**

```go
// DisplayTitle returns a human-readable title for share UI. Brew is
// special-cased: brews don't have a name field, so we fall back to the
// associated bean's name (or origin).
func (f *FeedItem) DisplayTitle() string {
	switch f.RecordType {
	case lexicons.RecordTypeBrew:
		if f.Brew != nil && f.Brew.Bean != nil {
			if f.Brew.Bean.Name != "" {
				return f.Brew.Bean.Name
			}
			return f.Brew.Bean.Origin
		}
		return "Coffee Brew"
	case lexicons.RecordTypeBean:
		if f.Bean != nil {
			if f.Bean.Name != "" {
				return f.Bean.Name
			}
			return f.Bean.Origin
		}
		return "Coffee Bean"
	case lexicons.RecordTypeRoaster:
		if f.Roaster != nil {
			return f.Roaster.Name
		}
		return "Roaster"
	case lexicons.RecordTypeGrinder:
		if f.Grinder != nil {
			return f.Grinder.Name
		}
		return "Grinder"
	case lexicons.RecordTypeBrewer:
		if f.Brewer != nil {
			return f.Brewer.Name
		}
		return "Brewer"
	case lexicons.RecordTypeRecipe:
		if f.Recipe != nil {
			return f.Recipe.Name
		}
		return "Recipe"
	}
	return "Arabica"
}
```

### Task 2: Migrate `getFeedItemShareURL` in `feed.templ`

**Files:**

- Modify: `internal/web/pages/feed.templ`

**Step 1: Replace the switch**

Replace lines `471-499` (`getFeedItemShareURL`) with:

```go
func getFeedItemShareURL(item *feed.FeedItem) string {
	if d := entities.Get(item.RecordType); d != nil {
		if rkey := item.RKey(); rkey != "" {
			return fmt.Sprintf("/%s/%s?owner=%s", d.URLPath, rkey, item.Author.DID)
		}
	}
	return fmt.Sprintf("/profile/%s", item.Author.DID)
}
```

**Step 2: Regenerate**

```bash
templ generate -f internal/web/pages/feed.templ
```

### Task 3: Migrate `getFeedItemShareTitle` in `feed.templ`

**Files:**

- Modify: `internal/web/pages/feed.templ`

**Step 1: Replace the switch**

Replace lines `501-541` (`getFeedItemShareTitle`) with:

```go
func getFeedItemShareTitle(item *feed.FeedItem) string {
	if title := item.DisplayTitle(); title != "" {
		return title
	}
	return "Arabica"
}
```

The per-type fallbacks (e.g., "Coffee Bean", "Roaster") now live in
`(*FeedItem).DisplayTitle()`.

### Task 4: Migrate `getDeleteURL` in `feed.templ`

**Files:**

- Modify: `internal/web/pages/feed.templ`

**Step 1: Replace the switch**

Brew has an asymmetric delete URL (`/brews/{rkey}` instead of
`/api/brews/{rkey}`) — leave it as an explicit branch, don't smuggle
the asymmetry into the descriptor. Replace lines `564-592`
(`getDeleteURL`) with:

```go
func getDeleteURL(item *feed.FeedItem) string {
	rkey := item.RKey()
	if rkey == "" {
		return ""
	}
	if item.RecordType == lexicons.RecordTypeBrew {
		return fmt.Sprintf("/brews/%s", rkey)
	}
	if d := entities.Get(item.RecordType); d != nil {
		return fmt.Sprintf("/api/%s/%s", d.URLPath, rkey)
	}
	return ""
}
```

**Step 2: Regenerate**

```bash
templ generate -f internal/web/pages/feed.templ
```

### Task 5: OG card label uses `Descriptor.Noun`

**Files:**

- Modify: `internal/ogcard/entities.go`

**Step 1: Replace hardcoded labels**

In the four simple Draw functions (`DrawBeanCard`, `DrawRoasterCard`,
`DrawGrinderCard`, `DrawBrewerCard`), replace the second argument to
`newTypedCard` with a descriptor lookup. Example:

Before:
```go
card, err := newTypedCard(AccentBean, "bean")
```

After:
```go
card, err := newTypedCard(AccentBean, entities.Get(lexicons.RecordTypeBean).Noun)
```

Apply to all four. Leave `DrawRecipeCard` as-is — its label is
`recipeType+" recipe"` (e.g., "espresso recipe"), which is bespoke and
can't be expressed by `Noun` alone.

`DrawBrewCard` in `ogcard/brew.go` is also bespoke (brew is a unique
case across the refactor); leave it alone.

**Note on accent colors:** the `AccentX` constants stay in
`ogcard/brew.go`. Putting `color.RGBA` on `Descriptor` would force
`entities` to import `image/color`, and the colors are an OG-card
implementation detail. If a future phase consolidates the Draw
functions further, an `accentByType` map can live in `ogcard` itself.

### Task 6: Verify

**Commands:**

```bash
go vet ./...
go build ./...
go test ./...
just run
```

**Manual smoke test:**

- Load `/feed`:
  - Click each entity type's card — share URL should match the
    `/{entity}/{rkey}?owner={did}` pattern as before
  - Click the share button — title should match (e.g., bean's name)
  - Click delete on each entity (without confirming) — URL should be
    `/api/{entity}/{rkey}` for non-brew, `/brews/{rkey}` for brew
- Hit each OG card endpoint (e.g., `/api/og/beans/{rkey}`) — labels
  in the corner should still read "bean", "roaster", etc.

**Expected delta:**

- `feed.templ`: ~120 LOC removed (three switches collapsed into helpers
  using descriptor + accessor methods)
- `feed/service.go`: ~70 LOC added (RKey + DisplayTitle methods)
- `ogcard/entities.go`: ~5 LOC changed (label arg)
- Net: ~−45 LOC, plus the wins from killing three more switch sites
  and proving the descriptor + container-accessor pattern composes.

---

## Out of scope for phase 1

These belong to later phases (see parent spec):

- Modal shell extraction (phase 5')
- Cache typed fields → map (phase 2, deferred)
- Generic store CRUD (phase 3, deferred)
- View handler unification (phase 4, deferred)
- Routing loop, suggestions config, dirty-tracking TTL (phase 6, deferred)
- Per-entity OG card Draw consolidation (intentionally not done — see
  phase 5 / FieldSpec discussion in parent spec)
