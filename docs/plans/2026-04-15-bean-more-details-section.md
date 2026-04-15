# Plan: Bean "More Details" Expandable Section

## Context
The bean creation modal is already fairly large. Power users want fields like producer (farm/estate) and elevation, but adding them visibly would overwhelm casual users. Solution: move variety + new fields into a collapsible "More Details" section that auto-expands on edit when populated.

## Phase 1: Data Layer

**`lexicons/social.arabica.alpha.bean.json`** — Add `producer` (string, maxLength 200) and `elevation` (string, maxLength 100) as optional properties.

**`internal/models/models.go`** — Follow Variety/Process pattern exactly:
- Add `MaxProducerLength = 200`, `MaxElevationLength = 100` constants
- Add `Producer`, `Elevation` string fields to `Bean`, `CreateBeanRequest`, `UpdateBeanRequest`
- Add length validation in both `Validate()` methods

**`internal/atproto/records.go`** — Add producer/elevation to `BeanToRecord()` and `RecordToBean()` using same `if != ""` / type-assertion pattern as variety/process.

## Phase 2: Handlers

**`internal/handlers/entities.go`** — Add `Producer: r.FormValue("producer")` and `Elevation: r.FormValue("elevation")` to both `HandleBeanCreate` and `HandleBeanUpdate` form decoders.

## Phase 3: Modal UI (main change)

**`internal/web/components/dialog_modals.templ`**:
1. **Remove** the variety input from the "Origin Details" fieldset (keep roaster, roast level, process there)
2. **Add** a new expandable section between "Origin Details" and "Notes & Rating":
   - Go helper `beanMoreDetailsInit(bean)` returns `{ showMore: true }` or `{ showMore: false }` based on whether variety/producer/elevation have values
   - When collapsed: shows `+ More details (variety, producer, elevation)` as a quiet text link
   - When expanded: shows a fieldset with variety, producer, elevation inputs
   - Alpine.js `x-show` keeps inputs in DOM even when hidden (fields submit as empty strings = correct behavior)
3. **Add** `"producer"` and `"elevation"` cases to `getStringValue()` helper

## Phase 4: Display Views

**`internal/web/pages/bean_view.templ`**:
- Add `DetailField` entries for Producer and Elevation in the detail grid
- Add `producer` and `elevation` to `beanBaseJSON()` for edit round-tripping
- Pick appropriate icons (check existing icon set, add IconMountain if needed)

**`internal/ogcard/entities.go`**:
- Optionally append producer to the details line in `DrawBeanCard` (elevation too niche for OG cards)

**Skip adding to BeanSummary and BeanCard** — these are detail-level fields; the card/summary already shows origin/variety/roast/process and would get cluttered.

## Phase 5: Verify

```bash
templ generate
go vet ./...
go build ./...
```

Manual checks:
- Create bean without expanding More Details — variety/producer/elevation should save as empty
- Create bean with all three fields — should persist and display on view page
- Edit a bean with populated fields — More Details section should auto-expand

## Files Modified
- `lexicons/social.arabica.alpha.bean.json`
- `internal/models/models.go`
- `internal/atproto/records.go`
- `internal/handlers/entities.go`
- `internal/web/components/dialog_modals.templ` (main UI change)
- `internal/web/pages/bean_view.templ`
- `internal/ogcard/entities.go`
