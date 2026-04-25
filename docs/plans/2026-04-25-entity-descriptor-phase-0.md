# Entity Descriptor — Phase 0 Foundation

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Introduce the `internal/entities` package with a `Descriptor` type and
registry, register all current record types, and migrate two representative call
sites (one templ, one Go) as proof. No behavior change; pure refactor.

**Parent spec:** `docs/entity-descriptor-refactor.md`

**Tech Stack:** Go 1.26, Templ, no new dependencies.

---

## Major Design Changes

### The new abstraction

A single struct, `entities.Descriptor`, captures the per-entity data that
callers across the codebase dispatch on. One descriptor is registered per record
type at package `init()`. Callers do `entities.Get(item.RecordType)` and read
fields off the descriptor instead of writing a `switch`.

The descriptor stores **data and small accessors** — not templ components, not
record conversion, not validation. Anything that genuinely varies in _structure_
per entity (form layouts, conversion logic, ref resolution) stays as code.

### Why a registry instead of generics or interfaces

- **Generics** force every caller to be parameterized on `T`, which doesn't work
  for templ files (templ has no type parameters) and would force the feed item
  to be generic too.
- **An interface on the model** (e.g., `Bean implements EntityDescriptor`) works
  for Go callers but not for templ — templ needs to dispatch on `RecordType` (a
  string) coming off a `FeedItem`, where the typed model pointer may be nil.
- **A registry keyed by `RecordType`** works in both Go and templ, requires zero
  changes to existing types, and stays opt-in: callers who don't need it ignore
  it.

### What is NOT changing in phase 0

- The `Store` interface and `AtprotoStore` implementation
- `UserCache` typed fields
- Per-entity record conversion (`records.go`)
- Per-entity templ form layouts
- The 5-switch structure of `feed.templ` (only `ActionText` is migrated this
  phase; the others are phase 1)
- Routing (entity routes still registered explicitly; loop comes in phase 6)

### What this phase enables

After phase 0, every subsequent phase migrates additional call sites onto the
same registry. The generic-store work in phase 3 reuses `Descriptor.NSID` for
collection lookup. The OG card work in phase 1 reuses `Descriptor.DisplayName`.
Phase 0's value is foundational, not direct LOC savings.

### Boundaries and invariants

- **The descriptor never holds entity data, only metadata.** A `Descriptor` for
  "bean" is a singleton; there is one of it for the whole program.
- **`any` is contained.** It appears on `Record` and `GetField` because templ
  can't be parameterized on `T`. Callers never store `any` — they extract a
  typed pointer or a string and move on.
- **Registration panics on duplicates.** Catches double-registration bugs at
  startup, before serving traffic.
- **Registration is package-private to `entities`.** No one outside the package
  can mutate the registry.

---

## Phase 0: Foundation

### Task 1: Create the `entities` package

**Files:**

- Create: `internal/entities/entities.go`
- Create: `internal/entities/entities_test.go`

**Step 1: Write the descriptor type and registry**

Create `internal/entities/entities.go`:

```go
// Package entities provides a registry of descriptors for each Arabica record
// type. A descriptor captures the per-entity data that callers in feed, templ,
// handlers, and ogcard dispatch on, replacing scattered switch statements with
// a single lookup.
package entities

import (
	"fmt"
	"sort"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// Descriptor describes one Arabica record type.
type Descriptor struct {
	Type        lexicons.RecordType
	NSID        string
	DisplayName string // "Bean"
	Noun        string // "bean" — appears in copy: "added a new bean"
	URLPath     string // "beans" — share URLs and routes

	// Record returns the typed record pointer from a FeedItem (e.g. item.Bean),
	// or nil if the FeedItem holds no record of this type.
	Record func(*feed.FeedItem) any

	// GetField extracts one named string field from a typed model pointer for
	// form prefill. Returns ("", false) if entity is nil or field is unknown.
	GetField func(entity any, field string) (string, bool)
}

var registry = map[lexicons.RecordType]*Descriptor{}

// Register adds a descriptor. Called once per entity at package init.
// Panics on duplicate registration to catch wiring bugs at startup.
func Register(d *Descriptor) {
	if _, ok := registry[d.Type]; ok {
		panic(fmt.Sprintf("entities: duplicate descriptor for %s", d.Type))
	}
	registry[d.Type] = d
}

// Get returns the descriptor for a record type, or nil if unregistered.
func Get(rt lexicons.RecordType) *Descriptor { return registry[rt] }

// All returns descriptors in stable order (by RecordType). Use for route loops.
func All() []*Descriptor {
	out := make([]*Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}
```

**Step 2: Write the registry test**

Create `internal/entities/entities_test.go` covering:

- `Get` returns the right descriptor for each registered type
- `Get` returns `nil` for an unknown type
- `All()` returns descriptors in sorted order
- Duplicate registration panics

Use `testify/assert` (project convention).

### Task 2: Register all current record types

**Files:**

- Create: `internal/entities/register.go`
- Create: `internal/entities/fields.go`

**Step 1: Write registration**

Create `internal/entities/register.go`:

```go
package entities

import (
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	Register(&Descriptor{
		Type: lexicons.RecordTypeBean, NSID: atproto.NSIDBean,
		DisplayName: "Bean", Noun: "bean", URLPath: "beans",
		Record:   func(i *feed.FeedItem) any { return i.Bean },
		GetField: beanField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRoaster, NSID: atproto.NSIDRoaster,
		DisplayName: "Roaster", Noun: "roaster", URLPath: "roasters",
		Record:   func(i *feed.FeedItem) any { return i.Roaster },
		GetField: roasterField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeGrinder, NSID: atproto.NSIDGrinder,
		DisplayName: "Grinder", Noun: "grinder", URLPath: "grinders",
		Record:   func(i *feed.FeedItem) any { return i.Grinder },
		GetField: grinderField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrewer, NSID: atproto.NSIDBrewer,
		DisplayName: "Brewer", Noun: "brewer", URLPath: "brewers",
		Record:   func(i *feed.FeedItem) any { return i.Brewer },
		GetField: brewerField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRecipe, NSID: atproto.NSIDRecipe,
		DisplayName: "Recipe", Noun: "recipe", URLPath: "recipes",
		Record:   func(i *feed.FeedItem) any { return i.Recipe },
		GetField: recipeField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrew, NSID: atproto.NSIDBrew,
		DisplayName: "Brew", Noun: "brew", URLPath: "brews",
		Record:   func(i *feed.FeedItem) any { return i.Brew },
		GetField: nil, // brew has no edit modal that needs prefill
	})
	// Like is intentionally omitted — has no entity page or modal.
}
```

**Step 2: Write the per-entity field accessors**

Create `internal/entities/fields.go` with `beanField`, `roasterField`,
`grinderField`, `brewerField`, `recipeField`. Each is a small switch over the
field-name strings used by `dialog_modals.templ`. Example:

```go
func beanField(e any, field string) (string, bool) {
	b, ok := e.(*models.Bean)
	if !ok || b == nil {
		return "", false
	}
	switch field {
	case "name":        return b.Name, true
	case "origin":      return b.Origin, true
	case "variety":     return b.Variety, true
	case "process":     return b.Process, true
	case "description": return b.Description, true
	}
	return "", false
}
```

The Recipe case in the original `getStringValue` formats `coffee_amount` and
`water_amount` as `%.1f`. Move that formatting into `recipeField` — the
descriptor is the right home for it.

### Task 3: Migrate `feed.templ` `ActionText`

**Files:**

- Modify: `internal/web/pages/feed.templ`

**Step 1: Replace the switch**

Lines `289–348` of `internal/web/pages/feed.templ` contain six near-identical
cases that differ only in noun. Replace the whole `templ ActionText(...)` body
with:

```templ
templ ActionText(item *feed.FeedItem) {
	if d := entities.Get(item.RecordType); d != nil && d.Record(item) != nil {
		added a
		<a href={ templ.SafeURL(getFeedItemShareURL(item)) }
		   class="underline hover:text-brown-900">new { d.Noun }</a>
		{ " " }
		@components.TypeBadge(d.Noun)
	} else {
		{ item.Action }
	}
}
```

Add the `entities` import to the templ file.

**Step 2: Regenerate**

Run `templ generate -f internal/web/pages/feed.templ`.

### Task 4: Migrate `getStringValue` in `dialog_modals.templ`

**Files:**

- Modify: `internal/web/components/dialog_modals.templ`

**Step 1: Replace the function**

Lines `935–1016` of `internal/web/components/dialog_modals.templ` contain the
nested type+field switch. Replace with:

```go
func getStringValue(entity interface{}, field string) string {
	if entity == nil {
		return ""
	}
	rt := recordTypeOf(entity)
	d := entities.Get(rt)
	if d == nil || d.GetField == nil {
		return ""
	}
	v, _ := d.GetField(entity, field)
	return v
}

// recordTypeOf maps a typed model pointer to its RecordType. Local helper
// because callers here already hold a typed pointer; pulling this onto the
// model would require a phase-4 model unification.
func recordTypeOf(entity any) lexicons.RecordType {
	switch entity.(type) {
	case *models.Bean:    return lexicons.RecordTypeBean
	case *models.Roaster: return lexicons.RecordTypeRoaster
	case *models.Grinder: return lexicons.RecordTypeGrinder
	case *models.Brewer:  return lexicons.RecordTypeBrewer
	case *models.Recipe:  return lexicons.RecordTypeRecipe
	}
	return ""
}
```

Add the `entities` import.

**Step 2: Regenerate**

Run `templ generate -f internal/web/components/dialog_modals.templ`.

### Task 5: Verify

**Commands:**

```bash
go vet ./...
go build ./...
go test ./...
just run
```

**Manual smoke test:**

- Load `/feed` — `ActionText` for each entity type should render the same as
  before
- Open the bean edit modal from `/my-coffee` — fields should prefill identically
- Repeat for roaster, grinder, brewer, recipe modals

**Expected delta:**

- ~110 LOC removed (`ActionText` switch: −48, `getStringValue` switch: −62)
- ~90 LOC added (new `entities` package)
- Net flat, but the foundation now exists for phases 1–6 to compound on.

---

## Out of scope for phase 0

These belong to later phases (see parent spec for full rollout):

- Remaining four switches in `feed.templ` (phase 1)
- OG card consolidation (phase 1)
- Cache typed fields → map (phase 2)
- Generic store CRUD (phase 3)
- View handler unification (phase 4)
- Dialog modal consolidation (phase 5)
- Routing loop, suggestions config, dirty-tracking TTL (phase 6)

## Decisions to confirm before starting

1. **Package location:** `internal/entities/` — confirmed unless you'd prefer
   `internal/lexicons/`.
2. **`Get` return shape:** `*Descriptor` (nil on miss) — idiomatic Go. Switch to
   `(*Descriptor, bool)` if you'd rather be explicit.
3. **Like registration:** skipped in this phase (no entity page); revisit if a
   feed migration needs it.
