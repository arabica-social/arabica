# Plan: Add Cafe and Drink Record Types

## Context

Users have requested the ability to track cafe visits alongside home brews.
This adds two new AT Protocol record types:
- **Cafe** — an entity (like roaster) representing a coffee shop
- **Drink** — an experience record (like brew but simpler) for cafe visits

These are alpha lexicons (`social.arabica.alpha.cafe/drink`) designed so that
field names match what a future v1 unified brew+drink record would use. The
shared field names (`beanRef`, `tastingNotes`, `rating`, `createdAt`) are
identical between brew and drink, making a v1 union-typed merge straightforward.

## Lexicon Design

### Cafe (`social.arabica.alpha.cafe`)
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string (max 200) | yes | |
| location | string (max 200) | no | |
| website | string (uri, max 500) | no | |
| roasterRef | string (at-uri) | no | Link to roaster this cafe serves |
| sourceRef | string (at-uri) | no | Sourced-from reference |
| createdAt | datetime | yes | |

### Drink (`social.arabica.alpha.drink`)
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| cafeRef | string (at-uri) | yes | Link to cafe |
| beanRef | string (at-uri) | no | Same field name as brew (optional here) |
| drinkStyle | string (max 100) | no | e.g. "cortado", "pour over", "latte" |
| tastingNotes | string (max 2000) | no | Same field name as brew |
| rating | integer (1-10) | no | Same field name as brew |
| price | integer (min 0) | no | Price in cents |
| createdAt | datetime | yes | |

## v1 Migration Strategy

The field naming is intentional for a clean v1 migration path:

1. **Shared fields** (`beanRef`, `tastingNotes`, `rating`, `createdAt`) use
   identical names in both brew and drink. In v1, these become top-level fields
   on a unified `social.arabica.brew` record.

2. **Drink-specific fields** (`cafeRef`, `drinkStyle`, `price`) become a
   `cafeContext` union member in v1.

3. **Brew-specific fields** (`grinderRef`, `brewerRef`, `method`,
   `espressoParams`, `pouroverParams`, `pours`, etc.) become a `homeContext`
   union member in v1.

4. **Cafe entity** simply drops `alpha` from the NSID — no structural changes.

5. **Record conversion layer** (`internal/atproto/records.go`) is the only code
   that needs v1 schema awareness. Models and handlers stay the same.

The v1 unified lexicon would look like:
```json
{
  "id": "social.arabica.brew",
  "defs": {
    "main": {
      "record": {
        "properties": {
          "beanRef": { "type": "string", "format": "at-uri" },
          "tastingNotes": { "type": "string" },
          "rating": { "type": "integer" },
          "createdAt": { "type": "string", "format": "datetime" },
          "context": {
            "type": "union",
            "refs": ["#homeContext", "#cafeContext"]
          }
        }
      }
    },
    "homeContext": {
      "type": "object",
      "properties": {
        "grinderRef": {},
        "brewerRef": {},
        "method": {},
        "espressoParams": {},
        "pouroverParams": {}
      }
    },
    "cafeContext": {
      "type": "object",
      "properties": {
        "cafeRef": {},
        "drinkStyle": {},
        "price": {}
      }
    }
  }
}
```

## Implementation Phases

Each phase is independently verifiable with `nix develop -c go vet ./...` and
`nix develop -c go build ./...`.

---

### Phase 1: Foundation
Lexicons, constants, models — everything else depends on these.

#### Files to create
- `lexicons/social.arabica.alpha.cafe.json` — modeled on roaster.json
- `lexicons/social.arabica.alpha.drink.json` — new schema per table above

#### Files to modify

**`internal/atproto/nsid.go`** — add constants:
- `NSIDCafe = NSIDBase + ".cafe"` (between NSIDBean and NSIDComment)
- `NSIDDrink = NSIDBase + ".drink"` (between NSIDComment and NSIDGrinder)

**`internal/lexicons/record_type.go`** — add:
- `RecordTypeCafe RecordType = "cafe"` and `RecordTypeDrink RecordType = "drink"`
- Add both to `ParseRecordType` switch
- Add `DisplayName()` cases: "Cafe", "Drink"

**`internal/models/models.go`** — add:
- `MaxDrinkStyleLength = 100` constant
- `Cafe` struct (follows Roaster pattern + `RoasterRKey` + joined `*Roaster`)
- `Drink` struct (CafeRKey, BeanRKey, DrinkStyle, TastingNotes, Rating, Price +
  joined `*Cafe`, `*Bean`)
- `CreateCafeRequest`, `UpdateCafeRequest` + `Validate()` methods
- `CreateDrinkRequest`, `UpdateDrinkRequest` + `Validate()` methods

---

### Phase 2: Data Layer
Record conversion, store interface, store implementation, cache.

**`internal/atproto/records.go`** — add:
- `CafeToRecord(cafe, roasterURI)` / `RecordToCafe(record, atURI)` — follows
  roaster pattern + roasterRef
- `DrinkToRecord(drink, cafeURI, beanURI)` / `RecordToDrink(record, atURI)` —
  cafeRef required, beanRef optional

**`internal/database/store.go`** — add to interface:
- Cafe: `CreateCafe`, `GetCafeByRKey`, `ListCafes`, `UpdateCafeByRKey`,
  `DeleteCafeByRKey`
- Drink: `CreateDrink`, `GetDrinkByRKey`, `ListDrinks`, `UpdateDrinkByRKey`,
  `DeleteDrinkByRKey`

**`internal/atproto/store.go`** — implement all 10 methods:
- Cafe methods follow roaster implementation pattern exactly
- Drink methods follow brew implementation pattern (builds cafeRef AT-URI,
  optional beanRef AT-URI)
- Add `LinkCafesToRoasters(cafes, roasters)` helper
- Add `LinkDrinksToCafes(drinks, cafes)` and `LinkDrinksToBeans(drinks, beans)`
  helpers

**`internal/atproto/cache.go`** — add:
- `Cafes []*models.Cafe` and `Drinks []*models.Drink` to `UserCache`
- Include in `clone()` method
- `SetCafes`, `SetDrinks`, `InvalidateCafes`, `InvalidateDrinks` on
  `SessionCache`

---

### Phase 3: Protocol Layer
OAuth scopes, firehose subscription and indexing.

**`internal/atproto/oauth.go`** — add to `scopes` slice:
- `"repo:" + NSIDCafe` (between NSIDBrewer and NSIDComment)
- `"repo:" + NSIDDrink` (between NSIDComment and NSIDGrinder)

**`internal/firehose/config.go`** — add to `ArabicaCollections`:
- `atproto.NSIDCafe`, `atproto.NSIDDrink`

**`internal/firehose/index.go`** — add:
- `Cafe *models.Cafe` and `Drink *models.Drink` to `FeedItem` struct
- `lexicons.RecordTypeCafe: true` and `lexicons.RecordTypeDrink: true` to
  `FeedableRecordTypes`
- `lexicons.RecordTypeCafe: atproto.NSIDCafe` and
  `lexicons.RecordTypeDrink: atproto.NSIDDrink` to `recordTypeToNSID`
- Switch cases in `recordToFeedItem`: cafe (simple, like roaster + resolve
  roasterRef), drink (resolve cafeRef + optional beanRef)

**`internal/feed/service.go`** — add:
- `Cafe *models.Cafe` and `Drink *models.Drink` to `FeedItem` struct
- Map these fields in the firehose→feed adapter conversion

---

### Phase 4: API Layer
Handlers and routing.

**`internal/handlers/entities.go`** — add:
- `HandleCafeCreate`, `HandleCafeUpdate`, `HandleCafeDelete` (follow roaster
  pattern)
- `HandleDrinkCreate`, `HandleDrinkUpdate`, `HandleDrinkDelete` (follow brew
  pattern, simpler)
- Update `HandleManagePartial` to fetch cafes + drinks in errgroup, link
  references, pass to props
- Update `HandleAPIListAll` to include cafes and drinks

**`internal/handlers/entity_views.go`** — add:
- `HandleCafeView` (follows roaster view pattern)
- `HandleDrinkView` (follows brew view pattern, simpler)
- `HandleCafeOGImage`, `HandleDrinkOGImage`

**`internal/handlers/modals.go`** — add:
- `HandleCafeModalNew`, `HandleCafeModalEdit` (follow roaster modal pattern +
  roaster dropdown)

**`internal/atproto/resolver.go`** — add:
- `ResolveCafeRef` function (follows roaster ref resolution pattern)
- Drink references resolved inline in handlers (cafeRef + optional beanRef)

**`internal/routing/routing.go`** — add routes:
```
GET    /cafes/{id}/og-image    → HandleCafeOGImage
GET    /cafes/{id}             → HandleCafeView
GET    /drinks/{id}/og-image   → HandleDrinkOGImage
GET    /drinks/{id}            → HandleDrinkView
GET    /drinks/new             → HandleDrinkNew
GET    /drinks/{id}/edit       → HandleDrinkEdit
GET    /drinks                 → HandleDrinkList
POST   /api/cafes              → HandleCafeCreate (COP)
PUT    /api/cafes/{id}         → HandleCafeUpdate (COP)
DELETE /api/cafes/{id}         → HandleCafeDelete (COP)
POST   /api/drinks             → HandleDrinkCreate (COP)
PUT    /api/drinks/{id}        → HandleDrinkUpdate (COP)
DELETE /api/drinks/{id}        → HandleDrinkDelete (COP)
GET    /api/modals/cafe/new    → HandleCafeModalNew
GET    /api/modals/cafe/{id}   → HandleCafeModalEdit
```

---

### Phase 5: UI Layer
Templates and components.

#### Files to create
- `internal/web/components/record_cafe.templ` — `CafeContent(cafe)`: name,
  location, website, linked roaster
- `internal/web/components/record_drink.templ` — `DrinkContent(drink)`: cafe,
  drink style, rating, notes, price, bean
- `internal/web/pages/cafe_view.templ` — detail page (follows
  roaster_view.templ)
- `internal/web/pages/drink_view.templ` — detail page (follows brew_view.templ,
  simpler)
- `internal/web/pages/drink_form.templ` — creation/edit form with cafe selector
  (required), bean selector (optional), drink style, tasting notes, rating, price
- `internal/web/pages/drink_list.templ` — list page (follows brew_list.templ
  pattern)

#### Files to modify

**`internal/web/components/manage_partial.templ`** — add:
- `Cafes []*models.Cafe`, `Drinks []*models.Drink`,
  `CafeDrinkCounts map[string]int` to props
- New "Cafes" tab with cafe cards and "+ Add Cafe" button
- New "Drinks" tab or section showing drink entries

**`internal/web/components/entity_tables.templ`** — add:
- `CafesTableProps` and `CafesTable` / `CafeCard` templ functions

**`internal/web/components/dialog_modals.templ`** — add:
- `CafeDialogModal(cafe, roasters)` — follows roaster modal + optional roaster
  `<select>`

**`internal/web/pages/feed.templ`** — add:
- Filter tabs: `{Label: "Cafes", Value: "cafe"}`,
  `{Label: "Drinks", Value: "drink"}`
- Record content switch cases for `RecordTypeCafe` and `RecordTypeDrink`
- Update `ActionText`, share URL, edit URL, delete URL switches

**`internal/web/components/header.templ`** — add navigation link for drinks
(like brews has)

---

### Phase 6: Suggestions
Entity autocomplete for cafes.

**`internal/handlers/suggestions.go`** — add `"cafes": atproto.NSIDCafe` to
`entityTypeToNSID`

**`internal/suggestions/suggestions.go`** — add cafe config:
- allFields: name, location, website
- searchFields: name, location
- dedupKey: cafeDedupKey (fuzzy name + normalized location, same as roaster
  pattern)

Suggestions route already handles arbitrary entity types via the map — no route
change needed.

---

## Verification

After each phase:
```bash
nix develop -c go vet ./...
nix develop -c go build ./...
nix develop -c go test ./...
```

After Phase 5 (UI complete):
```bash
nix develop -c templ generate
nix develop -c go run cmd/server/main.go
```
- Visit `/manage` — verify Cafes and Drinks tabs appear
- Create a cafe via the modal
- Create a drink via `/drinks/new`
- Check the community feed shows both new record types
- Verify cafe/drink detail pages load at `/cafes/{id}` and `/drinks/{id}`
