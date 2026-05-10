# Oolong Phase 1: Lexicons + Entity Package — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define the 9 oolong lexicon JSON files under `lexicons/` and build the `internal/entities/oolong/` Go package (NSIDs, models, record conversions, field accessors, descriptor registration). After this phase, `go build ./...` and `go test ./...` succeed; oolong records can be marshalled to/from atproto record maps; the entity registry knows about them. No store, firehose, OAuth, routes, or UI yet — those are Phases 2-4.

**Architecture:** Mirror `internal/entities/arabica/` exactly. The oolong package is a sibling, not a refactor. The shared `internal/lexicons` package gains oolong RecordType constants. The shared `internal/entities` registry gains nine oolong descriptors via `init()`. Tests use `shutter` snapshots for record conversions (matches arabica's pattern) and `testify/assert` for everything else.

**Tech Stack:** Go 1.21+, `github.com/bluesky-social/indigo/atproto/syntax`, `github.com/stretchr/testify`, `github.com/ptdewey/shutter` (already in go.mod).

**Spec reference:** `docs/superpowers/specs/2026-05-10-oolong-tea-lexicons-design.md`

## Pre-existing infrastructure (read this before starting)

Tea multi-tenancy is much further along than the spec implied. `docs/tea-multitenant-refactor.md` Phases A–H are mostly complete:

- **`internal/atplatform/domain/app.go`** defines the `App` struct with `Name`, `NSIDBase`, `Descriptors`, `Brand`. Methods: `NSIDs()`, `OAuthScopes()`, `DescriptorByNSID()`, `DescriptorByType()`. App is the only app-specific value at runtime — handlers, firehose, OAuth all read from it.
- **`cmd/server/main.go`** runs both apps as concurrent subservers (arabica on 18910, oolong on 18920) by constructing `newArabicaApp()` and `newTeaApp()` from `cmd/server/apps.go`.
- **`newArabicaApp()`** sets `Descriptors: entities.All()` — relies on the global `entities` registry, which today holds only arabica's descriptors (registered via `internal/entities/arabica/register.go`'s `init()`).
- **`newTeaApp()`** currently sets `Descriptors: nil` with a comment explaining tea lexicons haven't been authored. The doc says it "boots, serves with empty descriptors and refuses entity routes."

This Phase 1 plan fills in oolong's descriptors. The wiring step (Task 22) does **two** things:

1. Update `newTeaApp()` to populate `Descriptors` with oolong's entity descriptors.
2. Switch both `newArabicaApp()` and `newTeaApp()` from `entities.All()` to a new `entities.AllForApp(nsidBase)` helper that filters by NSID prefix. Without this, the global registry holding both arabica and oolong descriptors would cause each app to see the other's entities.

The `apps_test.go` test that asserts arabica's NSIDs already exists and pins the 8 arabica NSIDs. After this plan it must still pass; we'll add a parallel `TestTeaApp_NSIDs` test.

---

## File Map

**Create (lexicons):**
- `lexicons/social.oolong.alpha.tea.json`
- `lexicons/social.oolong.alpha.brew.json`
- `lexicons/social.oolong.alpha.brewer.json`
- `lexicons/social.oolong.alpha.recipe.json`
- `lexicons/social.oolong.alpha.vendor.json`
- `lexicons/social.oolong.alpha.cafe.json`
- `lexicons/social.oolong.alpha.drink.json`
- `lexicons/social.oolong.alpha.comment.json`
- `lexicons/social.oolong.alpha.like.json`

**Create (Go entity package):**
- `internal/entities/oolong/nsid.go` — NSID constants
- `internal/entities/oolong/models.go` — shared constants, errors, helpers
- `internal/entities/oolong/models_tea.go`
- `internal/entities/oolong/models_brew.go` — Brew + MethodParams interface + concrete params
- `internal/entities/oolong/models_brewer.go`
- `internal/entities/oolong/models_recipe.go`
- `internal/entities/oolong/models_vendor.go`
- `internal/entities/oolong/models_cafe.go`
- `internal/entities/oolong/models_drink.go`
- `internal/entities/oolong/models_social.go` — Like, Comment
- `internal/entities/oolong/records.go` — small entities (vendor, cafe, like, comment)
- `internal/entities/oolong/records_tea.go`
- `internal/entities/oolong/records_brew.go`
- `internal/entities/oolong/records_brewer.go`
- `internal/entities/oolong/records_recipe.go`
- `internal/entities/oolong/records_drink.go`
- `internal/entities/oolong/fields.go` — field accessors for form prefill
- `internal/entities/oolong/options.go` — dropdown lists
- `internal/entities/oolong/register.go` — descriptor registration
- Test files: one per `models_*.go` and `records_*.go` group
- `internal/entities/oolong/__snapshots__/` — auto-created by shutter on first test run

**Modify:**
- `internal/lexicons/record_type.go` — add oolong RecordType constants and update `ParseRecordType`/`DisplayName`
- `cmd/arabica/main.go` (or wherever arabica registers its entities) — add `_ "tangled.org/arabica.social/arabica/internal/entities/oolong"` blank import to trigger `init()` registration

> **Why split models/records by entity:** Arabica's single `models.go` is 750 lines and growing. Oolong starts fresh — splitting by entity keeps each file focused and matches the CLAUDE.md guidance about small, focused files. Aggregate is one package, so callers still write `oolong.Tea`, `oolong.RecordToBrew`, etc.

---

## Conventions (apply to every record conversion)

These mirror arabica's patterns exactly. Reference them when writing each entity's conversion functions:

1. **Record map shape:** `RecordToX(record map[string]any, atURI string) (*X, error)` and `XToRecord(x *X, ...refs) (map[string]any, error)`. The record map's `$type` field equals the NSID constant.
2. **AT-URI parsing:** use `syntax.ParseATURI(atURI)` from indigo to extract `RKey`. Mirror `RecordToRecipe` lines 70-76.
3. **Numeric encoding:** integers in atproto for fixed-point — `temperature` is tenths of °C (`93.5°C → 935`), `leafGrams` is tenths of g, `coffeeAmount` style. Convert on encode (`int(value * 10)`) and decode (`value / 10.0`).
4. **JSON number type:** atproto records decoded from JSON give `float64` for all numbers; in-memory may give `int`. Use the existing `toFloat64(v any) (float64, bool)` helper from `arabica/records.go`. Copy it into oolong's `records.go` (no shared package yet — duplication is intentional for v1).
5. **Optional fields:** check the record map with `if v, ok := record["x"].(T); ok { ... }`. Skip silently if absent.
6. **Required fields:** if absent, return `fmt.Errorf("X is required")`.
7. **Time format:** `time.RFC3339`.
8. **`sourceRef`:** every record carries an optional `sourceRef` (string AT-URI). Always check for and round-trip it.

---

## Task 1: Lexicon JSON — `vendor` (smallest, no refs)

**Files:**
- Create: `lexicons/social.oolong.alpha.vendor.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.vendor",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A tea vendor — seller, blender, or importer of tea",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": {
            "type": "string",
            "maxLength": 200,
            "description": "Name of the vendor (e.g., 'Spirit Tea', 'Tezumi', 'Yunnan Sourcing')"
          },
          "location": {
            "type": "string",
            "maxLength": 200,
            "description": "Location of the vendor (e.g., 'Chicago, IL', 'Tokyo, Japan')"
          },
          "website": {
            "type": "string",
            "format": "uri",
            "maxLength": 500,
            "description": "Vendor's website URL"
          },
          "description": {
            "type": "string",
            "maxLength": 1000,
            "description": "Description or notes about the vendor"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime",
            "description": "Timestamp when the vendor record was created"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI of the record this entity was sourced from"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses**

Run: `python3 -m json.tool lexicons/social.oolong.alpha.vendor.json > /dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add lexicons/social.oolong.alpha.vendor.json
git commit -m "feat(oolong): add vendor lexicon"
```

---

## Task 2: Lexicon JSON — `cafe`

**Files:**
- Create: `lexicons/social.oolong.alpha.cafe.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.cafe",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A cafe or teahouse where tea is served",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": {
            "type": "string",
            "maxLength": 200,
            "description": "Name of the cafe or teahouse"
          },
          "location": {
            "type": "string",
            "maxLength": 200,
            "description": "Human-readable location (e.g., 'Brooklyn, NY', 'Tokyo')"
          },
          "address": {
            "type": "string",
            "maxLength": 500,
            "description": "Street address"
          },
          "website": {
            "type": "string",
            "format": "uri",
            "maxLength": 500
          },
          "description": {
            "type": "string",
            "maxLength": 1000
          },
          "vendorRef": {
            "type": "string",
            "format": "at-uri",
            "description": "Optional AT-URI to a vendor record if the cafe is operated by a vendor"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses**

Run: `python3 -m json.tool lexicons/social.oolong.alpha.cafe.json > /dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add lexicons/social.oolong.alpha.cafe.json
git commit -m "feat(oolong): add cafe lexicon"
```

---

## Task 3: Lexicon JSON — `like` and `comment`

**Files:**
- Create: `lexicons/social.oolong.alpha.like.json`
- Create: `lexicons/social.oolong.alpha.comment.json`

- [ ] **Step 1: Write `like.json`**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.like",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A like on an Oolong record (or any AT-URI-addressable record)",
      "record": {
        "type": "object",
        "required": ["subject", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The AT-URI and CID of the record being liked"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Write `comment.json`**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.comment",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A comment on an Oolong record (or any AT-URI-addressable record)",
      "record": {
        "type": "object",
        "required": ["subject", "text", "createdAt"],
        "properties": {
          "subject": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "The AT-URI and CID of the record being commented on"
          },
          "text": {
            "type": "string",
            "maxLength": 1000,
            "maxGraphemes": 300
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "parent": {
            "type": "ref",
            "ref": "com.atproto.repo.strongRef",
            "description": "Optional parent comment reference for replies"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 3: Validate both parse**

Run: `for f in lexicons/social.oolong.alpha.{like,comment}.json; do python3 -m json.tool "$f" > /dev/null && echo "$f OK"; done`
Expected: two `OK` lines.

- [ ] **Step 4: Commit**

```bash
git add lexicons/social.oolong.alpha.like.json lexicons/social.oolong.alpha.comment.json
git commit -m "feat(oolong): add like and comment lexicons"
```

---

## Task 4: Lexicon JSON — `tea` (with processing array)

**Files:**
- Create: `lexicons/social.oolong.alpha.tea.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.tea",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A tea variety tracked by the user",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": {
            "type": "string",
            "maxLength": 200,
            "description": "Name of the tea (e.g., 'Long Jing 2024 Spring', 'Da Hong Pao')"
          },
          "category": {
            "type": "string",
            "maxLength": 50,
            "knownValues": [
              "green", "yellow", "white", "oolong", "red", "dark",
              "flavored", "blend", "other"
            ],
            "description": "Broad tea category. Known values: green, yellow, white, oolong, red, dark, flavored, blend, other"
          },
          "subStyle": {
            "type": "string",
            "maxLength": 200,
            "description": "Specific style (e.g., 'Yue Guang Bai', 'Baozhong', 'Wuyi', 'Raw Puerh', 'Hei Cha')"
          },
          "origin": {
            "type": "string",
            "maxLength": 200,
            "description": "Region or country of origin"
          },
          "cultivar": {
            "type": "string",
            "maxLength": 200,
            "description": "Plant cultivar (e.g., 'Yabukita', 'Tieguanyin', 'Da Ye')"
          },
          "cultivarRef": {
            "type": "string",
            "format": "at-uri",
            "description": "Reserved for future cultivar records"
          },
          "farm": {
            "type": "string",
            "maxLength": 200
          },
          "farmRef": {
            "type": "string",
            "format": "at-uri",
            "description": "Reserved for future farm records"
          },
          "harvestYear": {
            "type": "integer",
            "minimum": 1900,
            "maximum": 2100
          },
          "processing": {
            "type": "array",
            "description": "Ordered processing steps (e.g., [{step:'shaded',detail:'20 days'}, {step:'steamed'}, {step:'rolled'}])",
            "items": {
              "type": "ref",
              "ref": "#processingStep"
            }
          },
          "description": {
            "type": "string",
            "maxLength": 1000
          },
          "vendorRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI to a social.oolong.alpha.vendor record"
          },
          "rating": {
            "type": "integer",
            "minimum": 1,
            "maximum": 10
          },
          "closed": {
            "type": "boolean",
            "description": "Whether the bag/pouch is finished"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri"
          }
        }
      }
    },
    "processingStep": {
      "type": "object",
      "description": "A single step in the tea's processing chain",
      "required": ["step"],
      "properties": {
        "step": {
          "type": "string",
          "maxLength": 50,
          "knownValues": [
            "steamed", "pan-fired", "rolled", "oxidized", "shaded",
            "roasted", "aged", "fermented", "compressed", "scented",
            "smoked", "blended", "flavored", "other"
          ]
        },
        "detail": {
          "type": "string",
          "maxLength": 300,
          "description": "Freeform detail (e.g., '20 days under kanreisha', 'stone-pressed cake', '1995 vintage')"
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses**

Run: `python3 -m json.tool lexicons/social.oolong.alpha.tea.json > /dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add lexicons/social.oolong.alpha.tea.json
git commit -m "feat(oolong): add tea lexicon"
```

---

## Task 5: Lexicon JSON — `brewer`

**Files:**
- Create: `lexicons/social.oolong.alpha.brewer.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.brewer",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A tea brewing vessel",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": {
            "type": "string",
            "maxLength": 200,
            "description": "Name of the brewer (e.g., '120ml gaiwan', 'Tokoname kyusu')"
          },
          "style": {
            "type": "string",
            "maxLength": 50,
            "knownValues": [
              "gaiwan", "kyusu", "teapot", "matcha-bowl",
              "infuser", "thermos", "other"
            ],
            "description": "Vessel style. Known values: gaiwan, kyusu, teapot, matcha-bowl, infuser, thermos, other"
          },
          "capacityMl": {
            "type": "integer",
            "minimum": 0,
            "description": "Vessel capacity in millilitres"
          },
          "material": {
            "type": "string",
            "maxLength": 200,
            "description": "Material (e.g., 'porcelain', 'yixing clay', 'glass', 'cast iron')"
          },
          "description": {
            "type": "string",
            "maxLength": 1000
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses**

Run: `python3 -m json.tool lexicons/social.oolong.alpha.brewer.json > /dev/null && echo OK`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add lexicons/social.oolong.alpha.brewer.json
git commit -m "feat(oolong): add brewer lexicon"
```

---

## Task 6: Lexicon JSON — `drink`

**Files:**
- Create: `lexicons/social.oolong.alpha.drink.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.drink",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A tea ordered at a cafe",
      "record": {
        "type": "object",
        "required": ["cafeRef", "createdAt"],
        "properties": {
          "cafeRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI to a social.oolong.alpha.cafe record"
          },
          "teaRef": {
            "type": "string",
            "format": "at-uri",
            "description": "Optional AT-URI to a social.oolong.alpha.tea record"
          },
          "name": {
            "type": "string",
            "maxLength": 200,
            "description": "Menu item name (e.g., 'Iced hojicha latte', 'Jasmine pearls')"
          },
          "style": {
            "type": "string",
            "maxLength": 50,
            "knownValues": ["gongfu", "matcha", "longSteep", "milkTea", "other"]
          },
          "description": {
            "type": "string",
            "maxLength": 500
          },
          "rating": {
            "type": "integer",
            "minimum": 1,
            "maximum": 10
          },
          "tastingNotes": {
            "type": "string",
            "maxLength": 2000
          },
          "priceUsdCents": {
            "type": "integer",
            "minimum": 0,
            "description": "Optional price in US cents"
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses + commit**

```bash
python3 -m json.tool lexicons/social.oolong.alpha.drink.json > /dev/null && echo OK
git add lexicons/social.oolong.alpha.drink.json
git commit -m "feat(oolong): add drink lexicon"
```

Expected: `OK`, then a successful commit.

---

## Task 7: Lexicon JSON — `brew` (the polymorphic union)

**Files:**
- Create: `lexicons/social.oolong.alpha.brew.json`

This lexicon declares the `methodParams` union and the three sub-block types.

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.brew",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A tea brewing session",
      "record": {
        "type": "object",
        "required": ["teaRef", "style", "createdAt"],
        "properties": {
          "teaRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI to a social.oolong.alpha.tea record"
          },
          "style": {
            "type": "string",
            "maxLength": 50,
            "knownValues": ["gongfu", "matcha", "longSteep", "milkTea"],
            "description": "Brewing style. Required. Canonical filterable axis."
          },
          "brewerRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI to a social.oolong.alpha.brewer record"
          },
          "recipeRef": {
            "type": "string",
            "format": "at-uri",
            "description": "AT-URI to a social.oolong.alpha.recipe record"
          },
          "temperature": {
            "type": "integer",
            "minimum": 0,
            "maximum": 1000,
            "description": "Water temperature in tenths of °C (e.g. 850 = 85.0°C)"
          },
          "leafGrams": {
            "type": "integer",
            "minimum": 0,
            "description": "Leaf weight in tenths of a gram (e.g. 50 = 5.0g)"
          },
          "vesselMl": {
            "type": "integer",
            "minimum": 0,
            "description": "Total water / vessel volume in millilitres"
          },
          "timeSeconds": {
            "type": "integer",
            "minimum": 0,
            "description": "Total brew time in seconds. Primarily for longSteep and milkTea; gongfu records per-steep times inside #gongfuParams"
          },
          "tastingNotes": {
            "type": "string",
            "maxLength": 2000
          },
          "rating": {
            "type": "integer",
            "minimum": 1,
            "maximum": 10
          },
          "methodParams": {
            "type": "union",
            "refs": [
              "#gongfuParams",
              "#matchaParams",
              "#milkTeaParams"
            ],
            "description": "Style-specific parameters. Absent for longSteep style."
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    },
    "gongfuParams": {
      "type": "object",
      "description": "Parameters specific to gong fu (flash steep) brewing",
      "properties": {
        "rinse": {
          "type": "boolean",
          "description": "Whether the leaf was rinsed before brewing"
        },
        "rinseSeconds": {
          "type": "integer",
          "minimum": 0
        },
        "steeps": {
          "type": "array",
          "items": {
            "type": "ref",
            "ref": "#steep"
          }
        },
        "totalSteeps": {
          "type": "integer",
          "minimum": 0,
          "description": "Total steeps performed, in case not every one was logged"
        }
      }
    },
    "steep": {
      "type": "object",
      "required": ["number", "timeSeconds"],
      "properties": {
        "number": {
          "type": "integer",
          "minimum": 1,
          "description": "1-indexed steep number"
        },
        "timeSeconds": {
          "type": "integer",
          "minimum": 0
        },
        "temperature": {
          "type": "integer",
          "minimum": 0,
          "maximum": 1000,
          "description": "Temperature override for this steep, tenths of °C"
        },
        "tastingNotes": {
          "type": "string",
          "maxLength": 500
        },
        "rating": {
          "type": "integer",
          "minimum": 1,
          "maximum": 10
        }
      }
    },
    "matchaParams": {
      "type": "object",
      "description": "Parameters specific to matcha preparation",
      "properties": {
        "preparation": {
          "type": "string",
          "maxLength": 50,
          "knownValues": ["usucha", "koicha", "iced", "other"]
        },
        "sieved": {
          "type": "boolean"
        },
        "whiskType": {
          "type": "string",
          "maxLength": 200
        },
        "waterMl": {
          "type": "integer",
          "minimum": 0
        }
      }
    },
    "milkTeaParams": {
      "type": "object",
      "description": "Parameters specific to milk tea / tea-based beverages",
      "properties": {
        "preparation": {
          "type": "string",
          "maxLength": 300
        },
        "ingredients": {
          "type": "array",
          "items": {
            "type": "ref",
            "ref": "#ingredient"
          }
        },
        "iced": {
          "type": "boolean"
        }
      }
    },
    "ingredient": {
      "type": "object",
      "required": ["name"],
      "properties": {
        "name": {
          "type": "string",
          "maxLength": 200
        },
        "amount": {
          "type": "integer",
          "minimum": 0,
          "description": "Quantity in tenths of `unit`"
        },
        "unit": {
          "type": "string",
          "maxLength": 20,
          "knownValues": ["g", "ml", "tsp", "tbsp", "cup", "pcs", "other"]
        },
        "notes": {
          "type": "string",
          "maxLength": 200
        }
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses + commit**

```bash
python3 -m json.tool lexicons/social.oolong.alpha.brew.json > /dev/null && echo OK
git add lexicons/social.oolong.alpha.brew.json
git commit -m "feat(oolong): add brew lexicon"
```

Expected: `OK`, successful commit.

---

## Task 8: Lexicon JSON — `recipe`

The recipe lexicon redeclares the same `gongfuParams`/`matchaParams`/`milkTeaParams`/`steep`/`ingredient` shapes because lexicon refs are scoped to a single document. The Go entity package will share types between brew and recipe (Task 14).

**Files:**
- Create: `lexicons/social.oolong.alpha.recipe.json`

- [ ] **Step 1: Write the lexicon JSON**

```json
{
  "lexicon": 1,
  "id": "social.oolong.alpha.recipe",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "description": "A reusable tea brewing template",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": {
            "type": "string",
            "maxLength": 200
          },
          "brewerRef": {
            "type": "string",
            "format": "at-uri"
          },
          "style": {
            "type": "string",
            "maxLength": 50,
            "knownValues": ["gongfu", "matcha", "longSteep", "milkTea"]
          },
          "teaRef": {
            "type": "string",
            "format": "at-uri",
            "description": "Optional — recipes may be tea-specific or generic"
          },
          "temperature": {
            "type": "integer",
            "minimum": 0,
            "maximum": 1000
          },
          "timeSeconds": {
            "type": "integer",
            "minimum": 0
          },
          "leafGrams": {
            "type": "integer",
            "minimum": 0
          },
          "vesselMl": {
            "type": "integer",
            "minimum": 0
          },
          "methodParams": {
            "type": "union",
            "refs": [
              "#gongfuParams",
              "#matchaParams",
              "#milkTeaParams"
            ]
          },
          "notes": {
            "type": "string",
            "maxLength": 2000
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          },
          "sourceRef": {
            "type": "string",
            "format": "at-uri"
          }
        }
      }
    },
    "gongfuParams": {
      "type": "object",
      "properties": {
        "rinse": {"type": "boolean"},
        "rinseSeconds": {"type": "integer", "minimum": 0},
        "steeps": {"type": "array", "items": {"type": "ref", "ref": "#steep"}},
        "totalSteeps": {"type": "integer", "minimum": 0}
      }
    },
    "steep": {
      "type": "object",
      "required": ["number", "timeSeconds"],
      "properties": {
        "number": {"type": "integer", "minimum": 1},
        "timeSeconds": {"type": "integer", "minimum": 0},
        "temperature": {"type": "integer", "minimum": 0, "maximum": 1000},
        "tastingNotes": {"type": "string", "maxLength": 500},
        "rating": {"type": "integer", "minimum": 1, "maximum": 10}
      }
    },
    "matchaParams": {
      "type": "object",
      "properties": {
        "preparation": {
          "type": "string",
          "maxLength": 50,
          "knownValues": ["usucha", "koicha", "iced", "other"]
        },
        "sieved": {"type": "boolean"},
        "whiskType": {"type": "string", "maxLength": 200},
        "waterMl": {"type": "integer", "minimum": 0}
      }
    },
    "milkTeaParams": {
      "type": "object",
      "properties": {
        "preparation": {"type": "string", "maxLength": 300},
        "ingredients": {"type": "array", "items": {"type": "ref", "ref": "#ingredient"}},
        "iced": {"type": "boolean"}
      }
    },
    "ingredient": {
      "type": "object",
      "required": ["name"],
      "properties": {
        "name": {"type": "string", "maxLength": 200},
        "amount": {"type": "integer", "minimum": 0},
        "unit": {
          "type": "string",
          "maxLength": 20,
          "knownValues": ["g", "ml", "tsp", "tbsp", "cup", "pcs", "other"]
        },
        "notes": {"type": "string", "maxLength": 200}
      }
    }
  }
}
```

- [ ] **Step 2: Validate JSON parses + commit**

```bash
python3 -m json.tool lexicons/social.oolong.alpha.recipe.json > /dev/null && echo OK
git add lexicons/social.oolong.alpha.recipe.json
git commit -m "feat(oolong): add recipe lexicon"
```

Expected: `OK`, successful commit.

---

## Task 9: Add oolong RecordType constants to shared lexicons package

The shared `internal/lexicons/record_type.go` package provides typed `RecordType` constants used by the entity registry. Add constants for the nine oolong record types and update `ParseRecordType` and `DisplayName` accordingly. Use an `oolong-` prefix on the underlying string values to avoid collisions with arabica's existing types (e.g., both have a "brew" — distinct types must have distinct values).

**Files:**
- Modify: `internal/lexicons/record_type.go`
- Test: `internal/lexicons/record_type_test.go` (create if absent)

- [ ] **Step 1: Read current `record_type.go`**

Run: `cat internal/lexicons/record_type.go`
Note the existing `RecordType*` constants and the `ParseRecordType` / `DisplayName` switches.

- [ ] **Step 2: Write a failing test**

Create `internal/lexicons/record_type_test.go`:

```go
package lexicons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOolongRecordTypes(t *testing.T) {
	cases := []struct {
		raw   string
		want  RecordType
		label string
	}{
		{"oolong-tea", RecordTypeOolongTea, "Tea"},
		{"oolong-brew", RecordTypeOolongBrew, "Tea Brew"},
		{"oolong-brewer", RecordTypeOolongBrewer, "Tea Brewer"},
		{"oolong-recipe", RecordTypeOolongRecipe, "Tea Recipe"},
		{"oolong-vendor", RecordTypeOolongVendor, "Tea Vendor"},
		{"oolong-cafe", RecordTypeOolongCafe, "Tea Cafe"},
		{"oolong-drink", RecordTypeOolongDrink, "Tea Drink"},
		{"oolong-comment", RecordTypeOolongComment, "Tea Comment"},
		{"oolong-like", RecordTypeOolongLike, "Tea Like"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseRecordType(tc.raw))
			assert.Equal(t, tc.label, tc.want.DisplayName())
		})
	}
}

func TestArabicaRecordTypesUnchanged(t *testing.T) {
	assert.Equal(t, RecordTypeBean, ParseRecordType("bean"))
	assert.Equal(t, "Bean", RecordTypeBean.DisplayName())
}
```

- [ ] **Step 3: Run the test — expect compile failure**

Run: `go test ./internal/lexicons/... -run TestOolong -v`
Expected: build error — `RecordTypeOolongTea` undefined.

- [ ] **Step 4: Add the constants and update switches**

Edit `internal/lexicons/record_type.go`. After the existing `const` block, add:

```go
const (
	RecordTypeOolongTea     RecordType = "oolong-tea"
	RecordTypeOolongBrew    RecordType = "oolong-brew"
	RecordTypeOolongBrewer  RecordType = "oolong-brewer"
	RecordTypeOolongRecipe  RecordType = "oolong-recipe"
	RecordTypeOolongVendor  RecordType = "oolong-vendor"
	RecordTypeOolongCafe    RecordType = "oolong-cafe"
	RecordTypeOolongDrink   RecordType = "oolong-drink"
	RecordTypeOolongComment RecordType = "oolong-comment"
	RecordTypeOolongLike    RecordType = "oolong-like"
)
```

Update `ParseRecordType`:

```go
func ParseRecordType(s string) RecordType {
	switch RecordType(s) {
	case RecordTypeBean, RecordTypeBrew, RecordTypeBrewer, RecordTypeGrinder, RecordTypeRecipe, RecordTypeRoaster, RecordTypeLike:
		return RecordType(s)
	case RecordTypeOolongTea, RecordTypeOolongBrew, RecordTypeOolongBrewer,
		RecordTypeOolongRecipe, RecordTypeOolongVendor, RecordTypeOolongCafe,
		RecordTypeOolongDrink, RecordTypeOolongComment, RecordTypeOolongLike:
		return RecordType(s)
	default:
		return ""
	}
}
```

Update `DisplayName`:

```go
func (r RecordType) DisplayName() string {
	switch r {
	case RecordTypeBean:
		return "Bean"
	case RecordTypeBrew:
		return "Brew"
	case RecordTypeBrewer:
		return "Brewer"
	case RecordTypeGrinder:
		return "Grinder"
	case RecordTypeLike:
		return "Like"
	case RecordTypeRecipe:
		return "Recipe"
	case RecordTypeRoaster:
		return "Roaster"
	case RecordTypeOolongTea:
		return "Tea"
	case RecordTypeOolongBrew:
		return "Tea Brew"
	case RecordTypeOolongBrewer:
		return "Tea Brewer"
	case RecordTypeOolongRecipe:
		return "Tea Recipe"
	case RecordTypeOolongVendor:
		return "Tea Vendor"
	case RecordTypeOolongCafe:
		return "Tea Cafe"
	case RecordTypeOolongDrink:
		return "Tea Drink"
	case RecordTypeOolongComment:
		return "Tea Comment"
	case RecordTypeOolongLike:
		return "Tea Like"
	default:
		return string(r)
	}
}
```

> **Why "Tea Brew" not just "Brew":** the registry keys descriptors by `RecordType`, and the display name appears in copy. Since arabica has its own "Brew", oolong's display name disambiguates. UI surfaces inside the oolong vertical can override copy via local templ helpers if needed.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/lexicons/... -v`
Expected: all pass.

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 7: Commit**

```bash
git add internal/lexicons/record_type.go internal/lexicons/record_type_test.go
git commit -m "feat(lexicons): add oolong record type constants"
```

---

## Task 10: Create oolong package skeleton — `nsid.go` + shared `models.go`

**Files:**
- Create: `internal/entities/oolong/nsid.go`
- Create: `internal/entities/oolong/models.go`
- Test: `internal/entities/oolong/nsid_test.go`

- [ ] **Step 1: Write a failing nsid test**

Create `internal/entities/oolong/nsid_test.go`:

```go
package oolong

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNSIDs(t *testing.T) {
	assert.Equal(t, "social.oolong.alpha", NSIDBase)
	assert.Equal(t, "social.oolong.alpha.tea", NSIDTea)
	assert.Equal(t, "social.oolong.alpha.brew", NSIDBrew)
	assert.Equal(t, "social.oolong.alpha.brewer", NSIDBrewer)
	assert.Equal(t, "social.oolong.alpha.recipe", NSIDRecipe)
	assert.Equal(t, "social.oolong.alpha.vendor", NSIDVendor)
	assert.Equal(t, "social.oolong.alpha.cafe", NSIDCafe)
	assert.Equal(t, "social.oolong.alpha.drink", NSIDDrink)
	assert.Equal(t, "social.oolong.alpha.comment", NSIDComment)
	assert.Equal(t, "social.oolong.alpha.like", NSIDLike)
}
```

- [ ] **Step 2: Run test — expect compile failure**

Run: `go test ./internal/entities/oolong/... -v`
Expected: package not found / undefined symbols.

- [ ] **Step 3: Create `nsid.go`**

```go
// Package oolong provides typed Go models, atproto record conversions, and
// entity descriptors for the oolong tea-tracking sister app. NSIDs and
// behavior live here (not in the shared internal/atproto or internal/lexicons
// packages) because they are oolong-specific. Arabica has a sibling package
// at internal/entities/arabica with the same shape.
package oolong

const (
	// NSIDBase is the base namespace for all Oolong lexicons.
	NSIDBase = "social.oolong.alpha"

	NSIDTea     = NSIDBase + ".tea"
	NSIDBrew    = NSIDBase + ".brew"
	NSIDBrewer  = NSIDBase + ".brewer"
	NSIDRecipe  = NSIDBase + ".recipe"
	NSIDVendor  = NSIDBase + ".vendor"
	NSIDCafe    = NSIDBase + ".cafe"
	NSIDDrink   = NSIDBase + ".drink"
	NSIDComment = NSIDBase + ".comment"
	NSIDLike    = NSIDBase + ".like"
)
```

- [ ] **Step 4: Create `models.go` — shared constants, errors, helpers**

```go
package oolong

import "errors"

// Field length limits, mirrored from lexicon JSONs.
const (
	MaxNameLength         = 200
	MaxLocationLength     = 200
	MaxAddressLength      = 500
	MaxWebsiteLength      = 500
	MaxDescriptionLength  = 1000
	MaxNotesLength        = 2000
	MaxOriginLength       = 200
	MaxCultivarLength     = 200
	MaxFarmLength         = 200
	MaxSubStyleLength     = 200
	MaxCategoryLength     = 50
	MaxStepLength         = 50
	MaxStepDetailLength   = 300
	MaxBrewerStyleLength  = 50
	MaxMaterialLength     = 200
	MaxMenuItemLength     = 200
	MaxStyleLength        = 50
	MaxTastingNotesLength = 2000
	MaxSteepNotesLength   = 500
	MaxPreparationLength  = 300
	MaxIngredientName     = 200
	MaxIngredientUnit     = 20
	MaxIngredientNotes    = 200
	MaxWhiskTypeLength    = 200
	MaxCommentText        = 1000
	MaxCommentGraphemes   = 300
)

// Validation errors. Mirrors arabica's error sentinel pattern.
var (
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name is too long")
	ErrFieldTooLong      = errors.New("field value is too long")
	ErrLocationTooLong   = errors.New("location is too long")
	ErrWebsiteTooLong    = errors.New("website is too long")
	ErrDescTooLong       = errors.New("description is too long")
	ErrRatingOutOfRange  = errors.New("rating must be between 1 and 10")
	ErrTeaRefRequired    = errors.New("teaRef is required")
	ErrCafeRefRequired   = errors.New("cafeRef is required")
	ErrSubjectRequired   = errors.New("subject is required")
	ErrTextRequired      = errors.New("text is required")
	ErrTextTooLong       = errors.New("text is too long")
	ErrParentInvalid     = errors.New("parent_uri and parent_cid must be provided together")
	ErrStyleRequired     = errors.New("style is required")
	ErrStyleInvalid      = errors.New("style is not a known value")
	ErrCategoryInvalid   = errors.New("category is not a known value")
	ErrIngredientNoName  = errors.New("ingredient name is required")
)

// toFloat64 extracts a numeric value from an interface{} that may be int or
// float64. JSON decoding produces float64; in-memory maps may contain int.
// Mirrors the helper of the same name in internal/entities/arabica/records.go.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -v`
Expected: `TestNSIDs` passes.

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 7: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): scaffold entity package — NSIDs, shared constants"
```

---

## Task 11: `Vendor` — model, conversions, tests

**Files:**
- Create: `internal/entities/oolong/models_vendor.go`
- Create: `internal/entities/oolong/records_vendor.go`
- Create: `internal/entities/oolong/records_vendor_test.go`

- [ ] **Step 1: Write `models_vendor.go`**

```go
package oolong

import "time"

type Vendor struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateVendorRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateVendorRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

func (r *CreateVendorRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateVendorRequest) Validate() error {
	c := CreateVendorRequest(*r)
	return c.Validate()
}
```

- [ ] **Step 2: Write a failing test for `VendorToRecord`**

Create `internal/entities/oolong/records_vendor_test.go`:

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVendorToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full vendor", func(t *testing.T) {
		v := &Vendor{
			Name:        "Spirit Tea",
			Location:    "Chicago, IL",
			Website:     "https://spirittea.co",
			Description: "Importer focused on Chinese and Taiwanese teas",
			CreatedAt:   createdAt,
		}
		record, err := VendorToRecord(v)
		require.NoError(t, err)
		shutter.Snap(t, "VendorToRecord/full vendor", record)
	})

	t.Run("minimal vendor", func(t *testing.T) {
		v := &Vendor{Name: "Tezumi", CreatedAt: createdAt}
		record, err := VendorToRecord(v)
		require.NoError(t, err)
		shutter.Snap(t, "VendorToRecord/minimal vendor", record)
	})

	t.Run("error without name", func(t *testing.T) {
		v := &Vendor{CreatedAt: createdAt}
		_, err := VendorToRecord(v)
		assert.ErrorIs(t, err, ErrNameRequired)
	})
}

func TestRecordToVendor(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":       NSIDVendor,
			"name":        "Spirit Tea",
			"location":    "Chicago, IL",
			"website":     "https://spirittea.co",
			"description": "Importer",
			"createdAt":   "2026-05-10T12:00:00Z",
		}
		v, err := RecordToVendor(record, "at://did:plc:test/social.oolong.alpha.vendor/abc123")
		require.NoError(t, err)
		assert.Equal(t, "abc123", v.RKey)
		assert.Equal(t, "Spirit Tea", v.Name)
		assert.Equal(t, "Chicago, IL", v.Location)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDVendor,
			"createdAt": "2026-05-10T12:00:00Z",
		}
		_, err := RecordToVendor(record, "")
		assert.Error(t, err)
	})
}

func TestVendorRoundTrip(t *testing.T) {
	original := &Vendor{
		Name:        "Yunnan Sourcing",
		Location:    "Kunming, Yunnan",
		Website:     "https://yunnansourcing.com",
		Description: "Direct from Yunnan",
		SourceRef:   "at://did:plc:other/social.oolong.alpha.vendor/source",
		CreatedAt:   time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	record, err := VendorToRecord(original)
	require.NoError(t, err)
	round, err := RecordToVendor(record, "at://did:plc:test/social.oolong.alpha.vendor/abc")
	require.NoError(t, err)
	round.RKey = ""
	assert.Equal(t, original, round)
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run Vendor -v`
Expected: undefined `VendorToRecord`, `RecordToVendor`.

- [ ] **Step 4: Implement `records_vendor.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func VendorToRecord(v *Vendor) (map[string]any, error) {
	if v.Name == "" {
		return nil, ErrNameRequired
	}
	record := map[string]any{
		"$type":     NSIDVendor,
		"name":      v.Name,
		"createdAt": v.CreatedAt.Format(time.RFC3339),
	}
	if v.Location != "" {
		record["location"] = v.Location
	}
	if v.Website != "" {
		record["website"] = v.Website
	}
	if v.Description != "" {
		record["description"] = v.Description
	}
	if v.SourceRef != "" {
		record["sourceRef"] = v.SourceRef
	}
	return record, nil
}

func RecordToVendor(record map[string]any, atURI string) (*Vendor, error) {
	v := &Vendor{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		v.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	v.Name = name

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	v.CreatedAt = t

	if s, ok := record["location"].(string); ok {
		v.Location = s
	}
	if s, ok := record["website"].(string); ok {
		v.Website = s
	}
	if s, ok := record["description"].(string); ok {
		v.Description = s
	}
	if s, ok := record["sourceRef"].(string); ok {
		v.SourceRef = s
	}
	return v, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run Vendor -v`
Expected: all pass. First run creates `__snapshots__/vendortorecord/` files; review them by eye.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): vendor model and record conversions"
```

---

## Task 12: `Cafe` — model, conversions, tests

Mirror Task 11. Cafe has one optional ref (`vendorRef`), so `CafeToRecord` takes a `vendorURI` parameter that matches arabica's pattern from `BeanToRecord(bean, roasterURI)`.

**Files:**
- Create: `internal/entities/oolong/models_cafe.go`
- Create: `internal/entities/oolong/records_cafe.go`
- Create: `internal/entities/oolong/records_cafe_test.go`

- [ ] **Step 1: Write `models_cafe.go`**

```go
package oolong

import "time"

type Cafe struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	Address     string    `json:"address"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	VendorRKey  string    `json:"vendor_rkey,omitempty"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Joined data for display
	Vendor *Vendor `json:"vendor,omitempty"`
}

type CreateCafeRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Address     string `json:"address"`
	Website     string `json:"website"`
	Description string `json:"description"`
	VendorRKey  string `json:"vendor_rkey,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateCafeRequest CreateCafeRequest

func (r *CreateCafeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Address) > MaxAddressLength {
		return ErrFieldTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateCafeRequest) Validate() error {
	c := CreateCafeRequest(*r)
	return c.Validate()
}
```

- [ ] **Step 2: Write `records_cafe_test.go` — failing tests**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	t.Run("full cafe", func(t *testing.T) {
		c := &Cafe{
			Name:        "Floating Mountain",
			Location:    "New York, NY",
			Address:     "243 W 72nd St",
			Website:     "https://floatingmountain.tea",
			Description: "Specialty teahouse",
			CreatedAt:   createdAt,
		}
		vendorURI := "at://did:plc:test/social.oolong.alpha.vendor/v1"
		rec, err := CafeToRecord(c, vendorURI)
		require.NoError(t, err)
		shutter.Snap(t, "CafeToRecord/full cafe", rec)
	})
	t.Run("minimal cafe", func(t *testing.T) {
		c := &Cafe{Name: "Tea Spot", CreatedAt: createdAt}
		rec, err := CafeToRecord(c, "")
		require.NoError(t, err)
		shutter.Snap(t, "CafeToRecord/minimal cafe", rec)
	})
}

func TestRecordToCafe(t *testing.T) {
	rec := map[string]any{
		"$type":     NSIDCafe,
		"name":      "Floating Mountain",
		"location":  "New York, NY",
		"vendorRef": "at://did:plc:test/social.oolong.alpha.vendor/v1",
		"createdAt": "2026-05-10T12:00:00Z",
	}
	c, err := RecordToCafe(rec, "at://did:plc:test/social.oolong.alpha.cafe/cafe1")
	require.NoError(t, err)
	assert.Equal(t, "cafe1", c.RKey)
	assert.Equal(t, "Floating Mountain", c.Name)
	assert.NotEmpty(t, c.VendorRKey)
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run Cafe -v`
Expected: undefined `CafeToRecord`, `RecordToCafe`.

- [ ] **Step 4: Implement `records_cafe.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func CafeToRecord(c *Cafe, vendorURI string) (map[string]any, error) {
	if c.Name == "" {
		return nil, ErrNameRequired
	}
	record := map[string]any{
		"$type":     NSIDCafe,
		"name":      c.Name,
		"createdAt": c.CreatedAt.Format(time.RFC3339),
	}
	if c.Location != "" {
		record["location"] = c.Location
	}
	if c.Address != "" {
		record["address"] = c.Address
	}
	if c.Website != "" {
		record["website"] = c.Website
	}
	if c.Description != "" {
		record["description"] = c.Description
	}
	if vendorURI != "" {
		record["vendorRef"] = vendorURI
	}
	if c.SourceRef != "" {
		record["sourceRef"] = c.SourceRef
	}
	return record, nil
}

func RecordToCafe(record map[string]any, atURI string) (*Cafe, error) {
	c := &Cafe{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		c.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	c.Name = name

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	c.CreatedAt = t

	if s, ok := record["location"].(string); ok {
		c.Location = s
	}
	if s, ok := record["address"].(string); ok {
		c.Address = s
	}
	if s, ok := record["website"].(string); ok {
		c.Website = s
	}
	if s, ok := record["description"].(string); ok {
		c.Description = s
	}
	if s, ok := record["vendorRef"].(string); ok && s != "" {
		// Store the AT-URI's RKey for now; full URI is preserved via the joined Vendor when resolved later.
		parsed, err := syntax.ParseATURI(s)
		if err == nil {
			c.VendorRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["sourceRef"].(string); ok {
		c.SourceRef = s
	}
	return c, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run Cafe -v`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): cafe model and record conversions"
```

---

## Task 13: `Like` and `Comment` — models, conversions, tests

These are nearly identical to arabica's like/comment. Reference `internal/entities/arabica/records.go` lines 676-820 for the canonical patterns. The only difference is NSID strings.

**Files:**
- Create: `internal/entities/oolong/models_social.go`
- Create: `internal/entities/oolong/records_social.go`
- Create: `internal/entities/oolong/records_social_test.go`

- [ ] **Step 1: Write `models_social.go`**

```go
package oolong

import (
	"errors"
	"time"
	"unicode/utf8"
)

type Like struct {
	RKey       string    `json:"rkey"`
	SubjectURI string    `json:"subject_uri"`
	SubjectCID string    `json:"subject_cid"`
	CreatedAt  time.Time `json:"created_at"`
	ActorDID   string    `json:"actor_did,omitempty"`
}

type CreateLikeRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
}

type Comment struct {
	RKey       string    `json:"rkey"`
	CID        string    `json:"cid,omitempty"`
	SubjectURI string    `json:"subject_uri"`
	SubjectCID string    `json:"subject_cid"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	ActorDID   string    `json:"actor_did,omitempty"`
	ParentURI  string    `json:"parent_uri,omitempty"`
	ParentCID  string    `json:"parent_cid,omitempty"`
}

type CreateCommentRequest struct {
	SubjectURI string `json:"subject_uri"`
	SubjectCID string `json:"subject_cid"`
	Text       string `json:"text"`
	ParentURI  string `json:"parent_uri,omitempty"`
	ParentCID  string `json:"parent_cid,omitempty"`
}

func (r *CreateLikeRequest) Validate() error {
	if r.SubjectURI == "" {
		return ErrSubjectRequired
	}
	if r.SubjectCID == "" {
		return errors.New("subject_cid is required")
	}
	return nil
}

func (r *CreateCommentRequest) Validate() error {
	if r.Text == "" {
		return ErrTextRequired
	}
	if len(r.Text) > MaxCommentText {
		return ErrTextTooLong
	}
	if utf8.RuneCountInString(r.Text) > MaxCommentGraphemes {
		return ErrTextTooLong
	}
	if r.SubjectURI == "" {
		return ErrSubjectRequired
	}
	if r.SubjectCID == "" {
		return errors.New("subject_cid is required")
	}
	if (r.ParentURI != "" && r.ParentCID == "") || (r.ParentURI == "" && r.ParentCID != "") {
		return ErrParentInvalid
	}
	return nil
}
```

- [ ] **Step 2: Write `records_social_test.go`**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLikeRoundTrip(t *testing.T) {
	original := &Like{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := LikeToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "LikeToRecord/full like", rec)

	round, err := RecordToLike(rec, "at://did:plc:test/social.oolong.alpha.like/like1")
	require.NoError(t, err)
	assert.Equal(t, "like1", round.RKey)
	assert.Equal(t, original.SubjectURI, round.SubjectURI)
	assert.Equal(t, original.SubjectCID, round.SubjectCID)
}

func TestCommentRoundTrip(t *testing.T) {
	original := &Comment{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		Text:       "Beautiful first steep",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := CommentToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "CommentToRecord/full comment", rec)

	round, err := RecordToComment(rec, "at://did:plc:test/social.oolong.alpha.comment/c1")
	require.NoError(t, err)
	assert.Equal(t, "c1", round.RKey)
	assert.Equal(t, original.Text, round.Text)
}

func TestCommentWithParent(t *testing.T) {
	original := &Comment{
		SubjectURI: "at://did:plc:author/social.oolong.alpha.tea/tea1",
		SubjectCID: "bafyreig...",
		Text:       "Reply",
		ParentURI:  "at://did:plc:author/social.oolong.alpha.comment/parent",
		ParentCID:  "bafyreig...",
		CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := CommentToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "CommentToRecord/with parent", rec)
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run "Like|Comment" -v`
Expected: undefined `LikeToRecord`, `RecordToLike`, `CommentToRecord`, `RecordToComment`.

- [ ] **Step 4: Implement `records_social.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ========== Like ==========

func LikeToRecord(l *Like) (map[string]any, error) {
	if l.SubjectURI == "" {
		return nil, ErrSubjectRequired
	}
	return map[string]any{
		"$type": NSIDLike,
		"subject": map[string]any{
			"uri": l.SubjectURI,
			"cid": l.SubjectCID,
		},
		"createdAt": l.CreatedAt.Format(time.RFC3339),
	}, nil
}

func RecordToLike(record map[string]any, atURI string) (*Like, error) {
	l := &Like{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		l.RKey = parsed.RecordKey().String()
	}
	subj, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, ErrSubjectRequired
	}
	if uri, ok := subj["uri"].(string); ok {
		l.SubjectURI = uri
	}
	if cid, ok := subj["cid"].(string); ok {
		l.SubjectCID = cid
	}
	if createdStr, ok := record["createdAt"].(string); ok {
		t, err := time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid createdAt: %w", err)
		}
		l.CreatedAt = t
	}
	return l, nil
}

// ========== Comment ==========

func CommentToRecord(c *Comment) (map[string]any, error) {
	if c.SubjectURI == "" {
		return nil, ErrSubjectRequired
	}
	if c.Text == "" {
		return nil, ErrTextRequired
	}
	rec := map[string]any{
		"$type": NSIDComment,
		"subject": map[string]any{
			"uri": c.SubjectURI,
			"cid": c.SubjectCID,
		},
		"text":      c.Text,
		"createdAt": c.CreatedAt.Format(time.RFC3339),
	}
	if c.ParentURI != "" && c.ParentCID != "" {
		rec["parent"] = map[string]any{
			"uri": c.ParentURI,
			"cid": c.ParentCID,
		}
	}
	return rec, nil
}

func RecordToComment(record map[string]any, atURI string) (*Comment, error) {
	c := &Comment{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		c.RKey = parsed.RecordKey().String()
	}
	subj, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, ErrSubjectRequired
	}
	if uri, ok := subj["uri"].(string); ok {
		c.SubjectURI = uri
	}
	if cid, ok := subj["cid"].(string); ok {
		c.SubjectCID = cid
	}
	text, ok := record["text"].(string)
	if !ok {
		return nil, ErrTextRequired
	}
	c.Text = text

	if createdStr, ok := record["createdAt"].(string); ok {
		t, err := time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid createdAt: %w", err)
		}
		c.CreatedAt = t
	}
	if parent, ok := record["parent"].(map[string]any); ok {
		if uri, ok := parent["uri"].(string); ok {
			c.ParentURI = uri
		}
		if cid, ok := parent["cid"].(string); ok {
			c.ParentCID = cid
		}
	}
	return c, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run "Like|Comment" -v`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): like and comment models and record conversions"
```

---

## Task 14: `Brewer` — model, conversions, tests

**Files:**
- Create: `internal/entities/oolong/models_brewer.go`
- Create: `internal/entities/oolong/records_brewer.go`
- Create: `internal/entities/oolong/records_brewer_test.go`

- [ ] **Step 1: Write `models_brewer.go`**

```go
package oolong

import (
	"strings"
	"time"
)

const (
	BrewerStyleGaiwan     = "gaiwan"
	BrewerStyleKyusu      = "kyusu"
	BrewerStyleTeapot     = "teapot"
	BrewerStyleMatchaBowl = "matcha-bowl"
	BrewerStyleInfuser    = "infuser"
	BrewerStyleThermos    = "thermos"
	BrewerStyleOther      = "other"
)

var BrewerStyleKnownValues = []string{
	BrewerStyleGaiwan,
	BrewerStyleKyusu,
	BrewerStyleTeapot,
	BrewerStyleMatchaBowl,
	BrewerStyleInfuser,
	BrewerStyleThermos,
	BrewerStyleOther,
}

var BrewerStyleLabels = map[string]string{
	BrewerStyleGaiwan:     "Gaiwan",
	BrewerStyleKyusu:      "Kyusu",
	BrewerStyleTeapot:     "Teapot",
	BrewerStyleMatchaBowl: "Matcha Bowl",
	BrewerStyleInfuser:    "Infuser",
	BrewerStyleThermos:    "Thermos",
	BrewerStyleOther:      "Other",
}

// NormalizeBrewerStyle maps freeform strings to canonical values.
// Returns the input unchanged for unknown values.
func NormalizeBrewerStyle(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch lower {
	case "gaiwan":
		return BrewerStyleGaiwan
	case "kyusu", "kyuusu":
		return BrewerStyleKyusu
	case "teapot", "yixing", "zisha":
		return BrewerStyleTeapot
	case "matcha-bowl", "matcha bowl", "chawan":
		return BrewerStyleMatchaBowl
	case "infuser", "tea bag", "strainer":
		return BrewerStyleInfuser
	case "thermos", "grandpa", "bottle":
		return BrewerStyleThermos
	case "other":
		return BrewerStyleOther
	default:
		return raw
	}
}

type Brewer struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Style       string    `json:"style"`
	CapacityMl  int       `json:"capacity_ml"`
	Material    string    `json:"material"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateBrewerRequest struct {
	Name        string `json:"name"`
	Style       string `json:"style"`
	CapacityMl  int    `json:"capacity_ml"`
	Material    string `json:"material"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateBrewerRequest CreateBrewerRequest

func (r *CreateBrewerRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Style) > MaxBrewerStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Material) > MaxMaterialLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateBrewerRequest) Validate() error {
	c := CreateBrewerRequest(*r)
	return c.Validate()
}
```

- [ ] **Step 2: Write `records_brewer_test.go`**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewerRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brewer{
		Name:        "Tokoname kyusu",
		Style:       BrewerStyleKyusu,
		CapacityMl:  180,
		Material:    "stoneware",
		Description: "Side-handle, fine mesh",
		CreatedAt:   createdAt,
	}
	rec, err := BrewerToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "BrewerToRecord/full brewer", rec)

	round, err := RecordToBrewer(rec, "at://did:plc:test/social.oolong.alpha.brewer/b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	assert.Equal(t, original.CapacityMl, round.CapacityMl)
	assert.Equal(t, original.Material, round.Material)
}

func TestNormalizeBrewerStyle(t *testing.T) {
	assert.Equal(t, BrewerStyleKyusu, NormalizeBrewerStyle("Kyuusu"))
	assert.Equal(t, BrewerStyleTeapot, NormalizeBrewerStyle("Yixing"))
	assert.Equal(t, "unknown-thing", NormalizeBrewerStyle("unknown-thing"))
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run Brewer -v`

- [ ] **Step 4: Implement `records_brewer.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func BrewerToRecord(b *Brewer) (map[string]any, error) {
	if b.Name == "" {
		return nil, ErrNameRequired
	}
	rec := map[string]any{
		"$type":     NSIDBrewer,
		"name":      b.Name,
		"createdAt": b.CreatedAt.Format(time.RFC3339),
	}
	if b.Style != "" {
		rec["style"] = b.Style
	}
	if b.CapacityMl > 0 {
		rec["capacityMl"] = b.CapacityMl
	}
	if b.Material != "" {
		rec["material"] = b.Material
	}
	if b.Description != "" {
		rec["description"] = b.Description
	}
	if b.SourceRef != "" {
		rec["sourceRef"] = b.SourceRef
	}
	return rec, nil
}

func RecordToBrewer(record map[string]any, atURI string) (*Brewer, error) {
	b := &Brewer{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		b.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	b.Name = name

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	b.CreatedAt = t

	if s, ok := record["style"].(string); ok {
		b.Style = s
	}
	if v, ok := toFloat64(record["capacityMl"]); ok {
		b.CapacityMl = int(v)
	}
	if s, ok := record["material"].(string); ok {
		b.Material = s
	}
	if s, ok := record["description"].(string); ok {
		b.Description = s
	}
	if s, ok := record["sourceRef"].(string); ok {
		b.SourceRef = s
	}
	return b, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run Brewer -v`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): brewer model and record conversions"
```

---

## Task 15: `Tea` — model with processing array, conversions, tests

Tea is the largest non-brew record. It carries a `processing` array of `ProcessingStep` and a `vendorRef` AT-URI. Reserved-for-future `cultivarRef` and `farmRef` slots are stored as plain strings on the Go model — they'll be populated/used later when those records exist.

**Files:**
- Create: `internal/entities/oolong/models_tea.go`
- Create: `internal/entities/oolong/records_tea.go`
- Create: `internal/entities/oolong/records_tea_test.go`

- [ ] **Step 1: Write `models_tea.go`**

```go
package oolong

import "time"

const (
	CategoryGreen    = "green"
	CategoryYellow   = "yellow"
	CategoryWhite    = "white"
	CategoryOolong   = "oolong"
	CategoryRed      = "red"
	CategoryDark     = "dark"
	CategoryFlavored = "flavored"
	CategoryBlend    = "blend"
	CategoryOther    = "other"
)

var CategoryKnownValues = []string{
	CategoryGreen, CategoryYellow, CategoryWhite, CategoryOolong,
	CategoryRed, CategoryDark, CategoryFlavored, CategoryBlend, CategoryOther,
}

var CategoryLabels = map[string]string{
	CategoryGreen:    "Green",
	CategoryYellow:   "Yellow",
	CategoryWhite:    "White",
	CategoryOolong:   "Oolong",
	CategoryRed:      "Red / Black",
	CategoryDark:     "Dark / Fermented",
	CategoryFlavored: "Flavored",
	CategoryBlend:    "Blend",
	CategoryOther:    "Other",
}

const (
	ProcessingSteamed    = "steamed"
	ProcessingPanFired   = "pan-fired"
	ProcessingRolled     = "rolled"
	ProcessingOxidized   = "oxidized"
	ProcessingShaded     = "shaded"
	ProcessingRoasted    = "roasted"
	ProcessingAged       = "aged"
	ProcessingFermented  = "fermented"
	ProcessingCompressed = "compressed"
	ProcessingScented    = "scented"
	ProcessingSmoked     = "smoked"
	ProcessingBlended    = "blended"
	ProcessingFlavored   = "flavored"
	ProcessingOther      = "other"
)

var ProcessingKnownValues = []string{
	ProcessingSteamed, ProcessingPanFired, ProcessingRolled, ProcessingOxidized,
	ProcessingShaded, ProcessingRoasted, ProcessingAged, ProcessingFermented,
	ProcessingCompressed, ProcessingScented, ProcessingSmoked, ProcessingBlended,
	ProcessingFlavored, ProcessingOther,
}

type ProcessingStep struct {
	Step   string `json:"step"`
	Detail string `json:"detail,omitempty"`
}

type Tea struct {
	RKey         string           `json:"rkey"`
	Name         string           `json:"name"`
	Category     string           `json:"category"`
	SubStyle     string           `json:"sub_style"`
	Origin       string           `json:"origin"`
	Cultivar     string           `json:"cultivar"`
	CultivarRef  string           `json:"cultivar_ref,omitempty"`
	Farm         string           `json:"farm"`
	FarmRef      string           `json:"farm_ref,omitempty"`
	HarvestYear  int              `json:"harvest_year,omitempty"`
	Processing   []ProcessingStep `json:"processing,omitempty"`
	Description  string           `json:"description"`
	VendorRKey   string           `json:"vendor_rkey,omitempty"`
	Rating       *int             `json:"rating,omitempty"`
	Closed       bool             `json:"closed"`
	SourceRef    string           `json:"source_ref,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`

	// Joined data for display
	Vendor *Vendor `json:"vendor,omitempty"`
}

type CreateTeaRequest struct {
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	SubStyle    string           `json:"sub_style"`
	Origin      string           `json:"origin"`
	Cultivar    string           `json:"cultivar"`
	Farm        string           `json:"farm"`
	HarvestYear int              `json:"harvest_year,omitempty"`
	Processing  []ProcessingStep `json:"processing,omitempty"`
	Description string           `json:"description"`
	VendorRKey  string           `json:"vendor_rkey,omitempty"`
	Rating      *int             `json:"rating,omitempty"`
	Closed      bool             `json:"closed"`
	SourceRef   string           `json:"source_ref,omitempty"`
}

type UpdateTeaRequest CreateTeaRequest

func (r *CreateTeaRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Category) > MaxCategoryLength {
		return ErrFieldTooLong
	}
	if r.Category != "" && !isKnownValue(r.Category, CategoryKnownValues) {
		// Open enum: unknown values are allowed but validated separately if needed.
		// Mirror arabica's pattern of preserving unknowns.
	}
	if len(r.SubStyle) > MaxSubStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Origin) > MaxOriginLength {
		return ErrFieldTooLong
	}
	if len(r.Cultivar) > MaxCultivarLength {
		return ErrFieldTooLong
	}
	if len(r.Farm) > MaxFarmLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	for _, p := range r.Processing {
		if len(p.Step) > MaxStepLength {
			return ErrFieldTooLong
		}
		if len(p.Detail) > MaxStepDetailLength {
			return ErrFieldTooLong
		}
	}
	return nil
}

func (r *UpdateTeaRequest) Validate() error {
	c := CreateTeaRequest(*r)
	return c.Validate()
}

// IsIncomplete returns true if the tea is missing key classification fields.
func (t *Tea) IsIncomplete() bool {
	return t.Category == "" || t.Origin == ""
}

func (t *Tea) MissingFields() []string {
	var missing []string
	if t.Category == "" {
		missing = append(missing, "category")
	}
	if t.Origin == "" {
		missing = append(missing, "origin")
	}
	return missing
}

func isKnownValue(s string, known []string) bool {
	for _, k := range known {
		if s == k {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Write `records_tea_test.go`**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeaToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full tea with processing", func(t *testing.T) {
		rating := 9
		tea := &Tea{
			Name:        "Long Jing 2024 Spring",
			Category:    CategoryGreen,
			SubStyle:    "Xi Hu Long Jing",
			Origin:      "Hangzhou, Zhejiang",
			Cultivar:    "Qun Ti",
			Farm:        "Mei Jia Wu village",
			HarvestYear: 2024,
			Processing: []ProcessingStep{
				{Step: ProcessingPanFired, Detail: "Hand-fired in iron wok"},
			},
			Description: "Pre-Qingming pluck",
			Rating:      &rating,
			CreatedAt:   createdAt,
		}
		vendorURI := "at://did:plc:test/social.oolong.alpha.vendor/v1"
		rec, err := TeaToRecord(tea, vendorURI)
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/full tea with processing", rec)
	})

	t.Run("minimal tea", func(t *testing.T) {
		tea := &Tea{Name: "Generic green", CreatedAt: createdAt}
		rec, err := TeaToRecord(tea, "")
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/minimal tea", rec)
	})

	t.Run("tea with multiple processing steps", func(t *testing.T) {
		tea := &Tea{
			Name:     "Gyokuro",
			Category: CategoryGreen,
			SubStyle: "Gyokuro",
			Origin:   "Yame",
			Processing: []ProcessingStep{
				{Step: ProcessingShaded, Detail: "20 days under kanreisha"},
				{Step: ProcessingSteamed},
				{Step: ProcessingRolled},
			},
			CreatedAt: createdAt,
		}
		rec, err := TeaToRecord(tea, "")
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/multiple processing steps", rec)
	})
}

func TestRecordToTea(t *testing.T) {
	t.Run("full record with processing", func(t *testing.T) {
		rec := map[string]any{
			"$type":       NSIDTea,
			"name":        "Long Jing 2024 Spring",
			"category":    "green",
			"subStyle":    "Xi Hu Long Jing",
			"origin":      "Hangzhou, Zhejiang",
			"cultivar":    "Qun Ti",
			"farm":        "Mei Jia Wu",
			"harvestYear": float64(2024),
			"processing": []any{
				map[string]any{"step": "pan-fired", "detail": "Hand-fired"},
				map[string]any{"step": "rolled"},
			},
			"description": "Pre-Qingming",
			"vendorRef":   "at://did:plc:test/social.oolong.alpha.vendor/v1",
			"rating":      float64(9),
			"closed":      false,
			"createdAt":   "2026-05-10T12:00:00Z",
		}
		tea, err := RecordToTea(rec, "at://did:plc:test/social.oolong.alpha.tea/tea1")
		require.NoError(t, err)
		assert.Equal(t, "tea1", tea.RKey)
		assert.Equal(t, CategoryGreen, tea.Category)
		assert.Equal(t, 2024, tea.HarvestYear)
		assert.Len(t, tea.Processing, 2)
		assert.Equal(t, ProcessingPanFired, tea.Processing[0].Step)
		assert.Equal(t, "Hand-fired", tea.Processing[0].Detail)
		assert.Equal(t, ProcessingRolled, tea.Processing[1].Step)
		assert.Empty(t, tea.Processing[1].Detail)
		require.NotNil(t, tea.Rating)
		assert.Equal(t, 9, *tea.Rating)
		assert.NotEmpty(t, tea.VendorRKey)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		_, err := RecordToTea(map[string]any{"createdAt": "2026-05-10T12:00:00Z"}, "")
		assert.ErrorIs(t, err, ErrNameRequired)
	})
}

func TestTeaRoundTrip(t *testing.T) {
	rating := 8
	original := &Tea{
		Name:        "Da Hong Pao",
		Category:    CategoryOolong,
		SubStyle:    "Wuyi rock tea",
		Origin:      "Wuyi Mountains",
		Cultivar:    "Da Hong Pao",
		HarvestYear: 2024,
		Processing: []ProcessingStep{
			{Step: ProcessingOxidized, Detail: "60-70%"},
			{Step: ProcessingRoasted, Detail: "Charcoal, medium"},
		},
		Rating:    &rating,
		Closed:    false,
		CreatedAt: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := TeaToRecord(original, "")
	require.NoError(t, err)
	round, err := RecordToTea(rec, "at://did:plc:test/social.oolong.alpha.tea/abc")
	require.NoError(t, err)
	round.RKey = ""
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Processing, round.Processing)
	assert.Equal(t, *original.Rating, *round.Rating)
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run Tea -v`

- [ ] **Step 4: Implement `records_tea.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TeaToRecord(t *Tea, vendorURI string) (map[string]any, error) {
	if t.Name == "" {
		return nil, ErrNameRequired
	}
	rec := map[string]any{
		"$type":     NSIDTea,
		"name":      t.Name,
		"createdAt": t.CreatedAt.Format(time.RFC3339),
	}
	if t.Category != "" {
		rec["category"] = t.Category
	}
	if t.SubStyle != "" {
		rec["subStyle"] = t.SubStyle
	}
	if t.Origin != "" {
		rec["origin"] = t.Origin
	}
	if t.Cultivar != "" {
		rec["cultivar"] = t.Cultivar
	}
	if t.CultivarRef != "" {
		rec["cultivarRef"] = t.CultivarRef
	}
	if t.Farm != "" {
		rec["farm"] = t.Farm
	}
	if t.FarmRef != "" {
		rec["farmRef"] = t.FarmRef
	}
	if t.HarvestYear > 0 {
		rec["harvestYear"] = t.HarvestYear
	}
	if len(t.Processing) > 0 {
		steps := make([]map[string]any, len(t.Processing))
		for i, p := range t.Processing {
			m := map[string]any{"step": p.Step}
			if p.Detail != "" {
				m["detail"] = p.Detail
			}
			steps[i] = m
		}
		rec["processing"] = steps
	}
	if t.Description != "" {
		rec["description"] = t.Description
	}
	if vendorURI != "" {
		rec["vendorRef"] = vendorURI
	}
	if t.Rating != nil {
		rec["rating"] = *t.Rating
	}
	if t.Closed {
		rec["closed"] = true
	}
	if t.SourceRef != "" {
		rec["sourceRef"] = t.SourceRef
	}
	return rec, nil
}

func RecordToTea(record map[string]any, atURI string) (*Tea, error) {
	t := &Tea{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		t.RKey = parsed.RecordKey().String()
	}

	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	t.Name = name

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	created, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	t.CreatedAt = created

	if s, ok := record["category"].(string); ok {
		t.Category = s
	}
	if s, ok := record["subStyle"].(string); ok {
		t.SubStyle = s
	}
	if s, ok := record["origin"].(string); ok {
		t.Origin = s
	}
	if s, ok := record["cultivar"].(string); ok {
		t.Cultivar = s
	}
	if s, ok := record["cultivarRef"].(string); ok {
		t.CultivarRef = s
	}
	if s, ok := record["farm"].(string); ok {
		t.Farm = s
	}
	if s, ok := record["farmRef"].(string); ok {
		t.FarmRef = s
	}
	if v, ok := toFloat64(record["harvestYear"]); ok {
		t.HarvestYear = int(v)
	}
	if raw, ok := record["processing"].([]any); ok {
		t.Processing = make([]ProcessingStep, 0, len(raw))
		for _, item := range raw {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			step, _ := m["step"].(string)
			if step == "" {
				continue
			}
			ps := ProcessingStep{Step: step}
			if d, ok := m["detail"].(string); ok {
				ps.Detail = d
			}
			t.Processing = append(t.Processing, ps)
		}
	}
	if s, ok := record["description"].(string); ok {
		t.Description = s
	}
	if s, ok := record["vendorRef"].(string); ok && s != "" {
		parsed, err := syntax.ParseATURI(s)
		if err == nil {
			t.VendorRKey = parsed.RecordKey().String()
		}
	}
	if v, ok := toFloat64(record["rating"]); ok {
		r := int(v)
		t.Rating = &r
	}
	if b, ok := record["closed"].(bool); ok {
		t.Closed = b
	}
	if s, ok := record["sourceRef"].(string); ok {
		t.SourceRef = s
	}
	return t, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run Tea -v`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): tea model with processing array and conversions"
```

---

## Task 16: `Drink` — model, conversions, tests

**Files:**
- Create: `internal/entities/oolong/models_drink.go`
- Create: `internal/entities/oolong/records_drink.go`
- Create: `internal/entities/oolong/records_drink_test.go`

- [ ] **Step 1: Write `models_drink.go`**

```go
package oolong

import "time"

type Drink struct {
	RKey          string    `json:"rkey"`
	CafeRKey      string    `json:"cafe_rkey"`
	TeaRKey       string    `json:"tea_rkey,omitempty"`
	Name          string    `json:"name"`
	Style         string    `json:"style"`
	Description   string    `json:"description"`
	Rating        int       `json:"rating,omitempty"`
	TastingNotes  string    `json:"tasting_notes"`
	PriceUsdCents int       `json:"price_usd_cents,omitempty"`
	SourceRef     string    `json:"source_ref,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	// Joined data for display
	Cafe *Cafe `json:"cafe,omitempty"`
	Tea  *Tea  `json:"tea,omitempty"`
}

type CreateDrinkRequest struct {
	CafeRKey      string `json:"cafe_rkey"`
	TeaRKey       string `json:"tea_rkey,omitempty"`
	Name          string `json:"name"`
	Style         string `json:"style"`
	Description   string `json:"description"`
	Rating        int    `json:"rating,omitempty"`
	TastingNotes  string `json:"tasting_notes"`
	PriceUsdCents int    `json:"price_usd_cents,omitempty"`
	SourceRef     string `json:"source_ref,omitempty"`
}

type UpdateDrinkRequest CreateDrinkRequest

func (r *CreateDrinkRequest) Validate() error {
	if r.CafeRKey == "" {
		return ErrCafeRefRequired
	}
	if len(r.Name) > MaxMenuItemLength {
		return ErrFieldTooLong
	}
	if len(r.Style) > MaxStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > 500 {
		return ErrDescTooLong
	}
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	if r.Rating != 0 && (r.Rating < 1 || r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}

func (r *UpdateDrinkRequest) Validate() error {
	c := CreateDrinkRequest(*r)
	return c.Validate()
}
```

- [ ] **Step 2: Write `records_drink_test.go`**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrinkRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Drink{
		Name:          "Iced hojicha latte",
		Style:         "milkTea",
		Rating:        8,
		TastingNotes:  "Toasty, smooth",
		PriceUsdCents: 750,
		CreatedAt:     createdAt,
	}
	cafeURI := "at://did:plc:test/social.oolong.alpha.cafe/c1"
	teaURI := "at://did:plc:test/social.oolong.alpha.tea/t1"

	rec, err := DrinkToRecord(original, cafeURI, teaURI)
	require.NoError(t, err)
	shutter.Snap(t, "DrinkToRecord/full drink", rec)

	round, err := RecordToDrink(rec, "at://did:plc:test/social.oolong.alpha.drink/d1")
	require.NoError(t, err)
	assert.Equal(t, "d1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	assert.Equal(t, original.PriceUsdCents, round.PriceUsdCents)
	assert.NotEmpty(t, round.CafeRKey)
	assert.NotEmpty(t, round.TeaRKey)
}

func TestDrinkRequiresCafeRef(t *testing.T) {
	d := &Drink{Name: "Tea", CreatedAt: time.Now()}
	_, err := DrinkToRecord(d, "", "")
	assert.ErrorIs(t, err, ErrCafeRefRequired)
}
```

- [ ] **Step 3: Implement `records_drink.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func DrinkToRecord(d *Drink, cafeURI, teaURI string) (map[string]any, error) {
	if cafeURI == "" {
		return nil, ErrCafeRefRequired
	}
	rec := map[string]any{
		"$type":     NSIDDrink,
		"cafeRef":   cafeURI,
		"createdAt": d.CreatedAt.Format(time.RFC3339),
	}
	if teaURI != "" {
		rec["teaRef"] = teaURI
	}
	if d.Name != "" {
		rec["name"] = d.Name
	}
	if d.Style != "" {
		rec["style"] = d.Style
	}
	if d.Description != "" {
		rec["description"] = d.Description
	}
	if d.Rating > 0 {
		rec["rating"] = d.Rating
	}
	if d.TastingNotes != "" {
		rec["tastingNotes"] = d.TastingNotes
	}
	if d.PriceUsdCents > 0 {
		rec["priceUsdCents"] = d.PriceUsdCents
	}
	if d.SourceRef != "" {
		rec["sourceRef"] = d.SourceRef
	}
	return rec, nil
}

func RecordToDrink(record map[string]any, atURI string) (*Drink, error) {
	d := &Drink{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		d.RKey = parsed.RecordKey().String()
	}
	cafeRef, ok := record["cafeRef"].(string)
	if !ok || cafeRef == "" {
		return nil, ErrCafeRefRequired
	}
	cafeParsed, err := syntax.ParseATURI(cafeRef)
	if err != nil {
		return nil, fmt.Errorf("invalid cafeRef: %w", err)
	}
	d.CafeRKey = cafeParsed.RecordKey().String()

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	d.CreatedAt = t

	if s, ok := record["teaRef"].(string); ok && s != "" {
		teaParsed, err := syntax.ParseATURI(s)
		if err == nil {
			d.TeaRKey = teaParsed.RecordKey().String()
		}
	}
	if s, ok := record["name"].(string); ok {
		d.Name = s
	}
	if s, ok := record["style"].(string); ok {
		d.Style = s
	}
	if s, ok := record["description"].(string); ok {
		d.Description = s
	}
	if v, ok := toFloat64(record["rating"]); ok {
		d.Rating = int(v)
	}
	if s, ok := record["tastingNotes"].(string); ok {
		d.TastingNotes = s
	}
	if v, ok := toFloat64(record["priceUsdCents"]); ok {
		d.PriceUsdCents = int(v)
	}
	if s, ok := record["sourceRef"].(string); ok {
		d.SourceRef = s
	}
	return d, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/entities/oolong/... -run Drink -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): drink model and record conversions"
```

---

## Task 17: `Brew` model + `MethodParams` union (Go-side)

This is the largest task. The Go model uses a sealed-interface pattern to mirror the lexicon's union: `MethodParams` is an interface implemented only by `*GongfuParams`, `*MatchaParams`, and `*MilkTeaParams`. The `Brew` struct holds one `MethodParams` field; callers set whichever concrete value applies. For `style: "longSteep"` the field is nil.

The `$type` discriminator on the methodParams sub-map uses the canonical atproto fragment form: `social.oolong.alpha.brew#gongfuParams`, etc.

`Steep`, `Ingredient`, and the three params types live in `models_brew.go` and are reused by Recipe (Task 18).

**Files:**
- Create: `internal/entities/oolong/models_brew.go`
- Create: `internal/entities/oolong/records_brew.go`
- Create: `internal/entities/oolong/records_brew_test.go`

- [ ] **Step 1: Write `models_brew.go`**

```go
package oolong

import "time"

const (
	StyleGongfu    = "gongfu"
	StyleMatcha    = "matcha"
	StyleLongSteep = "longSteep"
	StyleMilkTea   = "milkTea"
)

var StyleKnownValues = []string{StyleGongfu, StyleMatcha, StyleLongSteep, StyleMilkTea}

var StyleLabels = map[string]string{
	StyleGongfu:    "Gong Fu",
	StyleMatcha:    "Matcha",
	StyleLongSteep: "Long Steep",
	StyleMilkTea:   "Milk Tea",
}

const (
	MatchaPrepUsucha = "usucha"
	MatchaPrepKoicha = "koicha"
	MatchaPrepIced   = "iced"
	MatchaPrepOther  = "other"
)

var MatchaPreparationKnownValues = []string{
	MatchaPrepUsucha, MatchaPrepKoicha, MatchaPrepIced, MatchaPrepOther,
}

const (
	IngredientUnitG     = "g"
	IngredientUnitMl    = "ml"
	IngredientUnitTsp   = "tsp"
	IngredientUnitTbsp  = "tbsp"
	IngredientUnitCup   = "cup"
	IngredientUnitPcs   = "pcs"
	IngredientUnitOther = "other"
)

var IngredientUnitKnownValues = []string{
	IngredientUnitG, IngredientUnitMl, IngredientUnitTsp, IngredientUnitTbsp,
	IngredientUnitCup, IngredientUnitPcs, IngredientUnitOther,
}

// MethodParams is a sealed union of style-specific brewing parameters. One of
// *GongfuParams, *MatchaParams, *MilkTeaParams. Nil for longSteep style.
type MethodParams interface {
	isMethodParams()
	atprotoTypeName() string
}

type GongfuParams struct {
	Rinse        bool    `json:"rinse"`
	RinseSeconds int     `json:"rinse_seconds"`
	Steeps       []Steep `json:"steeps,omitempty"`
	TotalSteeps  int     `json:"total_steeps,omitempty"`
}

func (*GongfuParams) isMethodParams()       {}
func (*GongfuParams) atprotoTypeName() string { return NSIDBrew + "#gongfuParams" }

// Steep represents one steep in a gong fu session. Temperature in tenths of °C
// (0 means "use brew.Temperature default"). Rating 0 means "no rating given".
type Steep struct {
	Number       int    `json:"number"`
	TimeSeconds  int    `json:"time_seconds"`
	Temperature  int    `json:"temperature,omitempty"`
	TastingNotes string `json:"tasting_notes,omitempty"`
	Rating       int    `json:"rating,omitempty"`
}

type MatchaParams struct {
	Preparation string `json:"preparation"`
	Sieved      bool   `json:"sieved"`
	WhiskType   string `json:"whisk_type"`
	WaterMl     int    `json:"water_ml"`
}

func (*MatchaParams) isMethodParams()       {}
func (*MatchaParams) atprotoTypeName() string { return NSIDBrew + "#matchaParams" }

type MilkTeaParams struct {
	Preparation string       `json:"preparation"`
	Ingredients []Ingredient `json:"ingredients,omitempty"`
	Iced        bool         `json:"iced"`
}

func (*MilkTeaParams) isMethodParams()       {}
func (*MilkTeaParams) atprotoTypeName() string { return NSIDBrew + "#milkTeaParams" }

// Ingredient on a milk tea / tea-based beverage. Amount is in float units
// (lexicon stores tenths of `unit` as integer; conversion happens at the
// record boundary).
type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	Notes  string  `json:"notes,omitempty"`
}

type Brew struct {
	RKey         string       `json:"rkey"`
	TeaRKey      string       `json:"tea_rkey"`
	Style        string       `json:"style"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	RecipeRKey   string       `json:"recipe_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	TastingNotes string       `json:"tasting_notes,omitempty"`
	Rating       int          `json:"rating,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`

	// Joined data for display
	Tea       *Tea     `json:"tea,omitempty"`
	BrewerObj *Brewer  `json:"brewer_obj,omitempty"`
	RecipeObj *Recipe  `json:"recipe_obj,omitempty"`
}

type CreateBrewRequest struct {
	TeaRKey      string       `json:"tea_rkey"`
	Style        string       `json:"style"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	RecipeRKey   string       `json:"recipe_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	TastingNotes string       `json:"tasting_notes,omitempty"`
	Rating       int          `json:"rating,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
}

func (r *CreateBrewRequest) Validate() error {
	if r.TeaRKey == "" {
		return ErrTeaRefRequired
	}
	if r.Style == "" {
		return ErrStyleRequired
	}
	if !isKnownValue(r.Style, StyleKnownValues) {
		return ErrStyleInvalid
	}
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	if r.Rating != 0 && (r.Rating < 1 || r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	// methodParams alignment with style
	switch r.Style {
	case StyleGongfu:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*GongfuParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleMatcha:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*MatchaParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleMilkTea:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*MilkTeaParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleLongSteep:
		if r.MethodParams != nil {
			return ErrStyleInvalid // longSteep takes no params
		}
	}
	return nil
}
```

- [ ] **Step 2: Write `records_brew_test.go` with cases for each style**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewToRecord_Gongfu(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleGongfu,
		Temperature:  95.0,
		LeafGrams:    5.5,
		VesselMl:     120,
		TastingNotes: "Sweet, nutty",
		Rating:       8,
		MethodParams: &GongfuParams{
			Rinse:        true,
			RinseSeconds: 5,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10, TastingNotes: "Bright, floral"},
				{Number: 2, TimeSeconds: 12, TastingNotes: "Deeper, nutty"},
				{Number: 3, TimeSeconds: 15, Temperature: 980},
			},
			TotalSteeps: 6,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/gongfu full", rec)
}

func TestBrewToRecord_Matcha(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleMatcha,
		Temperature:  75.0,
		LeafGrams:    2.0,
		VesselMl:     70,
		TastingNotes: "Umami forward",
		Rating:       9,
		MethodParams: &MatchaParams{
			Preparation: MatchaPrepUsucha,
			Sieved:      true,
			WhiskType:   "chasen 80-prong",
			WaterMl:     70,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/matcha full", rec)
}

func TestBrewToRecord_MilkTea(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:       StyleMilkTea,
		Temperature: 95.0,
		TimeSeconds: 600,
		Rating:      7,
		MethodParams: &MilkTeaParams{
			Preparation: "stovetop simmer",
			Ingredients: []Ingredient{
				{Name: "Whole milk", Amount: 200, Unit: IngredientUnitMl},
				{Name: "Cardamom pods", Amount: 4, Unit: IngredientUnitPcs},
				{Name: "Sugar", Amount: 1.5, Unit: IngredientUnitTsp, Notes: "Demerara"},
			},
			Iced: false,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/milkTea full", rec)
}

func TestBrewToRecord_LongSteep(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleLongSteep,
		Temperature:  85.0,
		LeafGrams:    3.0,
		VesselMl:     500,
		TimeSeconds:  240,
		TastingNotes: "Western brewed, robust",
		Rating:       7,
		// MethodParams intentionally nil
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/longSteep no params", rec)
}

func TestBrewRoundTrip_Gongfu(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:       StyleGongfu,
		Temperature: 95.0,
		LeafGrams:   5.5,
		VesselMl:    120,
		Rating:      8,
		MethodParams: &GongfuParams{
			Rinse:        true,
			RinseSeconds: 5,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10, TastingNotes: "Bright"},
				{Number: 2, TimeSeconds: 12},
			},
		},
		CreatedAt: createdAt,
	}
	teaURI := "at://did:plc:test/social.oolong.alpha.tea/t1"
	rec, err := BrewToRecord(original, teaURI, "", "")
	require.NoError(t, err)

	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", round.RKey)
	assert.Equal(t, StyleGongfu, round.Style)
	assert.Equal(t, 95.0, round.Temperature)

	gp, ok := round.MethodParams.(*GongfuParams)
	require.True(t, ok, "expected gongfuParams")
	assert.True(t, gp.Rinse)
	assert.Len(t, gp.Steeps, 2)
	assert.Equal(t, "Bright", gp.Steeps[0].TastingNotes)
}

func TestBrewRoundTrip_Matcha(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style: StyleMatcha,
		MethodParams: &MatchaParams{
			Preparation: MatchaPrepKoicha,
			Sieved:      true,
			WhiskType:   "chasen 120-prong",
			WaterMl:     30,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b2")
	require.NoError(t, err)
	mp, ok := round.MethodParams.(*MatchaParams)
	require.True(t, ok)
	assert.Equal(t, MatchaPrepKoicha, mp.Preparation)
	assert.Equal(t, 30, mp.WaterMl)
}

func TestBrewRoundTrip_MilkTea(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style: StyleMilkTea,
		MethodParams: &MilkTeaParams{
			Preparation: "shaken iced",
			Ingredients: []Ingredient{
				{Name: "Whole milk", Amount: 200, Unit: "ml"},
				{Name: "Sugar", Amount: 1.5, Unit: "tsp", Notes: "Demerara"},
			},
			Iced: true,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b3")
	require.NoError(t, err)
	mtp, ok := round.MethodParams.(*MilkTeaParams)
	require.True(t, ok)
	assert.True(t, mtp.Iced)
	require.Len(t, mtp.Ingredients, 2)
	assert.Equal(t, "Whole milk", mtp.Ingredients[0].Name)
	assert.Equal(t, 200.0, mtp.Ingredients[0].Amount)
	assert.Equal(t, 1.5, mtp.Ingredients[1].Amount)
	assert.Equal(t, "Demerara", mtp.Ingredients[1].Notes)
}

func TestBrewRoundTrip_LongSteepNoParams(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:       StyleLongSteep,
		Temperature: 85.0,
		LeafGrams:   3.0,
		VesselMl:    500,
		TimeSeconds: 240,
		CreatedAt:   createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b4")
	require.NoError(t, err)
	assert.Equal(t, StyleLongSteep, round.Style)
	assert.Nil(t, round.MethodParams)
	assert.Equal(t, 85.0, round.Temperature)
	assert.Equal(t, 240, round.TimeSeconds)
}

func TestBrewRequiresTeaRef(t *testing.T) {
	_, err := BrewToRecord(&Brew{Style: StyleGongfu}, "", "", "")
	assert.ErrorIs(t, err, ErrTeaRefRequired)
}

func TestBrewRequiresStyle(t *testing.T) {
	_, err := BrewToRecord(&Brew{}, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	assert.ErrorIs(t, err, ErrStyleRequired)
}
```

- [ ] **Step 3: Run — expect compile failure**

Run: `go test ./internal/entities/oolong/... -run Brew -v`

- [ ] **Step 4: Implement `records_brew.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// BrewToRecord converts a Brew to an atproto record map.
// The teaURI parameter is required (non-empty); brewerURI and recipeURI are optional.
func BrewToRecord(b *Brew, teaURI, brewerURI, recipeURI string) (map[string]any, error) {
	if teaURI == "" {
		return nil, ErrTeaRefRequired
	}
	if b.Style == "" {
		return nil, ErrStyleRequired
	}
	rec := map[string]any{
		"$type":     NSIDBrew,
		"teaRef":    teaURI,
		"style":     b.Style,
		"createdAt": b.CreatedAt.Format(time.RFC3339),
	}
	if brewerURI != "" {
		rec["brewerRef"] = brewerURI
	}
	if recipeURI != "" {
		rec["recipeRef"] = recipeURI
	}
	if b.Temperature > 0 {
		rec["temperature"] = int(b.Temperature * 10)
	}
	if b.LeafGrams > 0 {
		rec["leafGrams"] = int(b.LeafGrams * 10)
	}
	if b.VesselMl > 0 {
		rec["vesselMl"] = b.VesselMl
	}
	if b.TimeSeconds > 0 {
		rec["timeSeconds"] = b.TimeSeconds
	}
	if b.TastingNotes != "" {
		rec["tastingNotes"] = b.TastingNotes
	}
	if b.Rating > 0 {
		rec["rating"] = b.Rating
	}
	if b.MethodParams != nil {
		rec["methodParams"] = methodParamsToMap(b.MethodParams)
	}
	return rec, nil
}

// methodParamsToMap encodes a MethodParams value as the lexicon union shape:
// a map carrying $type plus the params fields.
func methodParamsToMap(mp MethodParams) map[string]any {
	m := map[string]any{"$type": mp.atprotoTypeName()}
	switch p := mp.(type) {
	case *GongfuParams:
		if p.Rinse {
			m["rinse"] = true
		}
		if p.RinseSeconds > 0 {
			m["rinseSeconds"] = p.RinseSeconds
		}
		if len(p.Steeps) > 0 {
			steeps := make([]map[string]any, len(p.Steeps))
			for i, s := range p.Steeps {
				sm := map[string]any{
					"number":      s.Number,
					"timeSeconds": s.TimeSeconds,
				}
				if s.Temperature > 0 {
					sm["temperature"] = s.Temperature
				}
				if s.TastingNotes != "" {
					sm["tastingNotes"] = s.TastingNotes
				}
				if s.Rating > 0 {
					sm["rating"] = s.Rating
				}
				steeps[i] = sm
			}
			m["steeps"] = steeps
		}
		if p.TotalSteeps > 0 {
			m["totalSteeps"] = p.TotalSteeps
		}
	case *MatchaParams:
		if p.Preparation != "" {
			m["preparation"] = p.Preparation
		}
		if p.Sieved {
			m["sieved"] = true
		}
		if p.WhiskType != "" {
			m["whiskType"] = p.WhiskType
		}
		if p.WaterMl > 0 {
			m["waterMl"] = p.WaterMl
		}
	case *MilkTeaParams:
		if p.Preparation != "" {
			m["preparation"] = p.Preparation
		}
		if len(p.Ingredients) > 0 {
			ings := make([]map[string]any, len(p.Ingredients))
			for i, ing := range p.Ingredients {
				im := map[string]any{"name": ing.Name}
				if ing.Amount > 0 {
					im["amount"] = int(ing.Amount * 10)
				}
				if ing.Unit != "" {
					im["unit"] = ing.Unit
				}
				if ing.Notes != "" {
					im["notes"] = ing.Notes
				}
				ings[i] = im
			}
			m["ingredients"] = ings
		}
		if p.Iced {
			m["iced"] = true
		}
	}
	return m
}

// RecordToBrew converts an atproto record map to a Brew.
func RecordToBrew(record map[string]any, atURI string) (*Brew, error) {
	b := &Brew{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		b.RKey = parsed.RecordKey().String()
	}
	teaRef, ok := record["teaRef"].(string)
	if !ok || teaRef == "" {
		return nil, ErrTeaRefRequired
	}
	teaParsed, err := syntax.ParseATURI(teaRef)
	if err != nil {
		return nil, fmt.Errorf("invalid teaRef: %w", err)
	}
	b.TeaRKey = teaParsed.RecordKey().String()

	style, ok := record["style"].(string)
	if !ok || style == "" {
		return nil, ErrStyleRequired
	}
	b.Style = style

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	b.CreatedAt = t

	if s, ok := record["brewerRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			b.BrewerRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["recipeRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			b.RecipeRKey = parsed.RecordKey().String()
		}
	}
	if v, ok := toFloat64(record["temperature"]); ok {
		b.Temperature = v / 10.0
	}
	if v, ok := toFloat64(record["leafGrams"]); ok {
		b.LeafGrams = v / 10.0
	}
	if v, ok := toFloat64(record["vesselMl"]); ok {
		b.VesselMl = int(v)
	}
	if v, ok := toFloat64(record["timeSeconds"]); ok {
		b.TimeSeconds = int(v)
	}
	if s, ok := record["tastingNotes"].(string); ok {
		b.TastingNotes = s
	}
	if v, ok := toFloat64(record["rating"]); ok {
		b.Rating = int(v)
	}
	if mp, ok := record["methodParams"].(map[string]any); ok {
		b.MethodParams = parseMethodParams(mp)
	}
	return b, nil
}

// parseMethodParams reads the union $type discriminator and dispatches to the
// matching concrete params decoder. Returns nil if $type is unknown or absent.
func parseMethodParams(m map[string]any) MethodParams {
	tname, _ := m["$type"].(string)
	switch tname {
	case NSIDBrew + "#gongfuParams", NSIDRecipe + "#gongfuParams":
		return decodeGongfuParams(m)
	case NSIDBrew + "#matchaParams", NSIDRecipe + "#matchaParams":
		return decodeMatchaParams(m)
	case NSIDBrew + "#milkTeaParams", NSIDRecipe + "#milkTeaParams":
		return decodeMilkTeaParams(m)
	}
	return nil
}

func decodeGongfuParams(m map[string]any) *GongfuParams {
	p := &GongfuParams{}
	if b, ok := m["rinse"].(bool); ok {
		p.Rinse = b
	}
	if v, ok := toFloat64(m["rinseSeconds"]); ok {
		p.RinseSeconds = int(v)
	}
	if raw, ok := m["steeps"].([]any); ok {
		p.Steeps = make([]Steep, 0, len(raw))
		for _, item := range raw {
			sm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			s := Steep{}
			if v, ok := toFloat64(sm["number"]); ok {
				s.Number = int(v)
			}
			if v, ok := toFloat64(sm["timeSeconds"]); ok {
				s.TimeSeconds = int(v)
			}
			if v, ok := toFloat64(sm["temperature"]); ok {
				s.Temperature = int(v)
			}
			if t, ok := sm["tastingNotes"].(string); ok {
				s.TastingNotes = t
			}
			if v, ok := toFloat64(sm["rating"]); ok {
				s.Rating = int(v)
			}
			p.Steeps = append(p.Steeps, s)
		}
	}
	if v, ok := toFloat64(m["totalSteeps"]); ok {
		p.TotalSteeps = int(v)
	}
	return p
}

func decodeMatchaParams(m map[string]any) *MatchaParams {
	p := &MatchaParams{}
	if s, ok := m["preparation"].(string); ok {
		p.Preparation = s
	}
	if b, ok := m["sieved"].(bool); ok {
		p.Sieved = b
	}
	if s, ok := m["whiskType"].(string); ok {
		p.WhiskType = s
	}
	if v, ok := toFloat64(m["waterMl"]); ok {
		p.WaterMl = int(v)
	}
	return p
}

func decodeMilkTeaParams(m map[string]any) *MilkTeaParams {
	p := &MilkTeaParams{}
	if s, ok := m["preparation"].(string); ok {
		p.Preparation = s
	}
	if raw, ok := m["ingredients"].([]any); ok {
		p.Ingredients = make([]Ingredient, 0, len(raw))
		for _, item := range raw {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := im["name"].(string)
			if name == "" {
				continue
			}
			ing := Ingredient{Name: name}
			if v, ok := toFloat64(im["amount"]); ok {
				ing.Amount = v / 10.0
			}
			if u, ok := im["unit"].(string); ok {
				ing.Unit = u
			}
			if n, ok := im["notes"].(string); ok {
				ing.Notes = n
			}
			p.Ingredients = append(p.Ingredients, ing)
		}
	}
	if b, ok := m["iced"].(bool); ok {
		p.Iced = b
	}
	return p
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/oolong/... -run Brew -v`
Expected: all pass.

- [ ] **Step 6: Inspect snapshots**

Run: `ls internal/entities/oolong/__snapshots__/` and read each file. Verify:
- `gongfu` snapshot has `methodParams.$type == "social.oolong.alpha.brew#gongfuParams"`
- `matcha` snapshot has `$type` matching `#matchaParams`
- `milkTea` snapshot has `$type` matching `#milkTeaParams` and ingredient amounts encoded as integer tenths
- `longSteep` snapshot has NO `methodParams` key

- [ ] **Step 7: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): brew model with methodParams union and conversions"
```

> **Note on the union `$type` discriminator across brew and recipe:**
> `methodParamsToMap` always emits the canonical `social.oolong.alpha.brew#gongfuParams` (etc.) form, even when the params are embedded inside a recipe record. `parseMethodParams` accepts both `brew#...` and `recipe#...` variants. This treats brew.json as the canonical home for the params shapes; if a future lexicon cleanup makes recipe.json reference brew's defs directly, no Go change is needed.

---

## Task 18: `Recipe` — model and conversions

Recipe reuses the `MethodParams`, `Steep`, and `Ingredient` types from `models_brew.go`. The Recipe-specific helpers (`Interpolate`, joined data fields) match arabica's pattern from `internal/entities/arabica/models.go` lines 204-258.

**Files:**
- Create: `internal/entities/oolong/models_recipe.go`
- Create: `internal/entities/oolong/records_recipe.go`
- Create: `internal/entities/oolong/records_recipe_test.go`

- [ ] **Step 1: Write `models_recipe.go`**

```go
package oolong

import "time"

type Recipe struct {
	RKey         string       `json:"rkey"`
	Name         string       `json:"name"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	Style        string       `json:"style,omitempty"`
	TeaRKey      string       `json:"tea_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	SourceRef    string       `json:"source_ref,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`

	// Joined data for display
	BrewerObj *Brewer `json:"brewer_obj,omitempty"`
	Tea       *Tea    `json:"tea,omitempty"`

	// Derived display fields, populated by handlers
	AuthorDID     string `json:"author_did,omitempty"`
	AuthorHandle  string `json:"author_handle,omitempty"`
	AuthorAvatar  string `json:"author_avatar,omitempty"`
	AuthorDisplay string `json:"author_display,omitempty"`

	SourceAuthorHandle  string `json:"source_author_handle,omitempty"`
	SourceAuthorAvatar  string `json:"source_author_avatar,omitempty"`
	SourceAuthorDisplay string `json:"source_author_display,omitempty"`

	ForkCount     int      `json:"fork_count,omitempty"`
	BrewCount     int      `json:"brew_count,omitempty"`
	ForkerAvatars []string `json:"forker_avatars,omitempty"`
}

type CreateRecipeRequest struct {
	Name         string       `json:"name"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	Style        string       `json:"style,omitempty"`
	TeaRKey      string       `json:"tea_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	SourceRef    string       `json:"source_ref,omitempty"`
}

type UpdateRecipeRequest CreateRecipeRequest

func (r *CreateRecipeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if r.Style != "" && !isKnownValue(r.Style, StyleKnownValues) {
		return ErrStyleInvalid
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrFieldTooLong
	}
	return nil
}

func (r *UpdateRecipeRequest) Validate() error {
	c := CreateRecipeRequest(*r)
	return c.Validate()
}
```

- [ ] **Step 2: Write `records_recipe_test.go`**

```go
package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipeRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Recipe{
		Name:        "Standard gaiwan gongfu",
		Style:       StyleGongfu,
		Temperature: 95.0,
		LeafGrams:   5.5,
		VesselMl:    120,
		MethodParams: &GongfuParams{
			Rinse: true,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10},
				{Number: 2, TimeSeconds: 12},
				{Number: 3, TimeSeconds: 15},
			},
			TotalSteeps: 6,
		},
		Notes:     "Adjust steep times to taste",
		CreatedAt: createdAt,
	}
	rec, err := RecipeToRecord(original, "", "")
	require.NoError(t, err)
	shutter.Snap(t, "RecipeToRecord/gongfu standard", rec)

	round, err := RecordToRecipe(rec, "at://did:plc:test/social.oolong.alpha.recipe/r1")
	require.NoError(t, err)
	assert.Equal(t, "r1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	gp, ok := round.MethodParams.(*GongfuParams)
	require.True(t, ok)
	assert.Len(t, gp.Steeps, 3)
}

func TestRecipeMissingName(t *testing.T) {
	_, err := RecipeToRecord(&Recipe{CreatedAt: time.Now()}, "", "")
	assert.ErrorIs(t, err, ErrNameRequired)
}
```

- [ ] **Step 3: Implement `records_recipe.go`**

```go
package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func RecipeToRecord(r *Recipe, brewerURI, teaURI string) (map[string]any, error) {
	if r.Name == "" {
		return nil, ErrNameRequired
	}
	rec := map[string]any{
		"$type":     NSIDRecipe,
		"name":      r.Name,
		"createdAt": r.CreatedAt.Format(time.RFC3339),
	}
	if brewerURI != "" {
		rec["brewerRef"] = brewerURI
	}
	if teaURI != "" {
		rec["teaRef"] = teaURI
	}
	if r.Style != "" {
		rec["style"] = r.Style
	}
	if r.Temperature > 0 {
		rec["temperature"] = int(r.Temperature * 10)
	}
	if r.TimeSeconds > 0 {
		rec["timeSeconds"] = r.TimeSeconds
	}
	if r.LeafGrams > 0 {
		rec["leafGrams"] = int(r.LeafGrams * 10)
	}
	if r.VesselMl > 0 {
		rec["vesselMl"] = r.VesselMl
	}
	if r.MethodParams != nil {
		rec["methodParams"] = methodParamsToMap(r.MethodParams)
	}
	if r.Notes != "" {
		rec["notes"] = r.Notes
	}
	if r.SourceRef != "" {
		rec["sourceRef"] = r.SourceRef
	}
	return rec, nil
}

func RecordToRecipe(record map[string]any, atURI string) (*Recipe, error) {
	r := &Recipe{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		r.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	r.Name = name

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	r.CreatedAt = t

	if s, ok := record["brewerRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			r.BrewerRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["teaRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			r.TeaRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["style"].(string); ok {
		r.Style = s
	}
	if v, ok := toFloat64(record["temperature"]); ok {
		r.Temperature = v / 10.0
	}
	if v, ok := toFloat64(record["timeSeconds"]); ok {
		r.TimeSeconds = int(v)
	}
	if v, ok := toFloat64(record["leafGrams"]); ok {
		r.LeafGrams = v / 10.0
	}
	if v, ok := toFloat64(record["vesselMl"]); ok {
		r.VesselMl = int(v)
	}
	if mp, ok := record["methodParams"].(map[string]any); ok {
		r.MethodParams = parseMethodParams(mp)
	}
	if s, ok := record["notes"].(string); ok {
		r.Notes = s
	}
	if s, ok := record["sourceRef"].(string); ok {
		r.SourceRef = s
	}
	return r, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/entities/oolong/... -run Recipe -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): recipe model and record conversions"
```

---

## Task 19: `fields.go` — form-prefill field accessors

Mirror `internal/entities/arabica/fields.go`. One function per entity that has an edit modal, returning string-typed field values for prefill.

**Files:**
- Create: `internal/entities/oolong/fields.go`

- [ ] **Step 1: Write `fields.go`**

```go
package oolong

import "fmt"

func teaField(e any, field string) (string, bool) {
	t, ok := e.(*Tea)
	if !ok || t == nil {
		return "", false
	}
	switch field {
	case "name":
		return t.Name, true
	case "category":
		return t.Category, true
	case "sub_style":
		return t.SubStyle, true
	case "origin":
		return t.Origin, true
	case "cultivar":
		return t.Cultivar, true
	case "farm":
		return t.Farm, true
	case "description":
		return t.Description, true
	}
	return "", false
}

func vendorField(e any, field string) (string, bool) {
	v, ok := e.(*Vendor)
	if !ok || v == nil {
		return "", false
	}
	switch field {
	case "name":
		return v.Name, true
	case "location":
		return v.Location, true
	case "website":
		return v.Website, true
	case "description":
		return v.Description, true
	}
	return "", false
}

func brewerField(e any, field string) (string, bool) {
	b, ok := e.(*Brewer)
	if !ok || b == nil {
		return "", false
	}
	switch field {
	case "name":
		return b.Name, true
	case "style":
		return b.Style, true
	case "material":
		return b.Material, true
	case "description":
		return b.Description, true
	case "capacity_ml":
		if b.CapacityMl > 0 {
			return fmt.Sprintf("%d", b.CapacityMl), true
		}
		return "", false
	}
	return "", false
}

func cafeField(e any, field string) (string, bool) {
	c, ok := e.(*Cafe)
	if !ok || c == nil {
		return "", false
	}
	switch field {
	case "name":
		return c.Name, true
	case "location":
		return c.Location, true
	case "address":
		return c.Address, true
	case "website":
		return c.Website, true
	case "description":
		return c.Description, true
	}
	return "", false
}

func recipeField(e any, field string) (string, bool) {
	r, ok := e.(*Recipe)
	if !ok || r == nil {
		return "", false
	}
	switch field {
	case "name":
		return r.Name, true
	case "style":
		return r.Style, true
	case "notes":
		return r.Notes, true
	case "leaf_grams":
		if r.LeafGrams > 0 {
			return fmt.Sprintf("%.1f", r.LeafGrams), true
		}
		return "", false
	}
	return "", false
}

func drinkField(e any, field string) (string, bool) {
	d, ok := e.(*Drink)
	if !ok || d == nil {
		return "", false
	}
	switch field {
	case "name":
		return d.Name, true
	case "style":
		return d.Style, true
	case "description":
		return d.Description, true
	case "tasting_notes":
		return d.TastingNotes, true
	}
	return "", false
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/entities/oolong/fields.go
git commit -m "feat(oolong): form-prefill field accessors"
```

---

## Task 20: `options.go` — dropdown lists for forms

**Files:**
- Create: `internal/entities/oolong/options.go`

- [ ] **Step 1: Write `options.go`**

```go
package oolong

// Dropdown options for oolong form fields. These are surface-level convenience
// re-exports of the knownValues from the lexicons; callers that want labels
// should prefer the *Labels maps in models_*.go.

var (
	// Categories are the broad tea classifications.
	Categories = CategoryKnownValues

	// BrewerStyles are vessel styles (gaiwan, kyusu, ...).
	BrewerStyles = BrewerStyleKnownValues

	// BrewStyles are the four brewing styles (gongfu, matcha, longSteep, milkTea).
	BrewStyles = StyleKnownValues

	// MatchaPreparations enumerate matcha prep variants.
	MatchaPreparations = MatchaPreparationKnownValues

	// IngredientUnits are units of measure for milk-tea ingredients.
	IngredientUnits = IngredientUnitKnownValues

	// ProcessingSteps are tea processing chain steps.
	ProcessingSteps = ProcessingKnownValues
)
```

- [ ] **Step 2: Verify build + commit**

```bash
go build ./...
git add internal/entities/oolong/options.go
git commit -m "feat(oolong): dropdown option lists"
```

Expected: clean build, successful commit.

---

## Task 21: `register.go` — entity descriptor registration

Mirror `internal/entities/arabica/register.go`. Register seven descriptors via `init()` — `tea`, `vendor`, `brewer`, `recipe`, `brew`, `cafe`, `drink`. The shared registry's `Register` panics on duplicate `Type`, so the disjoint oolong RecordType values (Task 9) are what makes this safe.

`like` and `comment` are intentionally **not** registered. `App.NSIDs()` (in `internal/atplatform/domain/app.go`) appends both of them unconditionally to the descriptor NSIDs, so registering them as descriptors would produce duplicates. Arabica follows the same pattern — its `register.go` also skips like and comment.

The `Comment` and `Like` Go types and their `*ToRecord`/`RecordTo*` functions still exist (Task 13) — they're consumed by handlers and the firehose, just not via the descriptor registry.

**Files:**
- Create: `internal/entities/oolong/register.go`
- Create: `internal/entities/oolong/registry_test.go`

- [ ] **Step 1: Write `registry_test.go` first**

```go
package oolong

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestOolongDescriptorsRegistered(t *testing.T) {
	cases := []struct {
		rt    lexicons.RecordType
		nsid  string
		label string
	}{
		{lexicons.RecordTypeOolongTea, NSIDTea, "Tea"},
		{lexicons.RecordTypeOolongBrew, NSIDBrew, "Tea Brew"},
		{lexicons.RecordTypeOolongBrewer, NSIDBrewer, "Tea Brewer"},
		{lexicons.RecordTypeOolongRecipe, NSIDRecipe, "Tea Recipe"},
		{lexicons.RecordTypeOolongVendor, NSIDVendor, "Tea Vendor"},
		{lexicons.RecordTypeOolongCafe, NSIDCafe, "Tea Cafe"},
		{lexicons.RecordTypeOolongDrink, NSIDDrink, "Tea Drink"},
	}
	for _, tc := range cases {
		t.Run(string(tc.rt), func(t *testing.T) {
			d := entities.Get(tc.rt)
			if assert.NotNil(t, d, "missing descriptor for %s", tc.rt) {
				assert.Equal(t, tc.nsid, d.NSID)
				assert.Equal(t, tc.label, d.DisplayName)
			}
		})
	}
}

func TestOolongLikeNotRegistered(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongLike)
	assert.Nil(t, d, "like is intentionally not registered (App.NSIDs() appends it)")
}

func TestOolongCommentNotRegistered(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongComment)
	assert.Nil(t, d, "comment is intentionally not registered (App.NSIDs() appends it)")
}

func TestOolongDescriptorsByNSID(t *testing.T) {
	d := entities.GetByNSID(NSIDTea)
	if assert.NotNil(t, d) {
		assert.Equal(t, lexicons.RecordTypeOolongTea, d.Type)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/entities/oolong/... -run Registered -v`
Expected: all subtests fail because no `init()` has run.

- [ ] **Step 3: Write `register.go`**

```go
package oolong

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongTea,
		NSID:            NSIDTea,
		DisplayName:     "Tea",
		Noun:            "tea",
		URLPath:         "teas",
		FeedFilterLabel: "Teas",
		GetField:        teaField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToTea(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongVendor,
		NSID:            NSIDVendor,
		DisplayName:     "Tea Vendor",
		Noun:            "vendor",
		URLPath:         "vendors",
		FeedFilterLabel: "", // reference entity — no dedicated feed tab
		GetField:        vendorField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToVendor(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongBrewer,
		NSID:            NSIDBrewer,
		DisplayName:     "Tea Brewer",
		Noun:            "brewer",
		URLPath:         "brewers",
		FeedFilterLabel: "Brewers",
		GetField:        brewerField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrewer(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongRecipe,
		NSID:            NSIDRecipe,
		DisplayName:     "Tea Recipe",
		Noun:            "recipe",
		URLPath:         "recipes",
		FeedFilterLabel: "Recipes",
		GetField:        recipeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRecipe(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongBrew,
		NSID:            NSIDBrew,
		DisplayName:     "Tea Brew",
		Noun:            "brew",
		URLPath:         "brews",
		FeedFilterLabel: "Brews",
		GetField:        nil, // brew has no edit modal that needs prefill
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrew(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongCafe,
		NSID:            NSIDCafe,
		DisplayName:     "Tea Cafe",
		Noun:            "cafe",
		URLPath:         "cafes",
		FeedFilterLabel: "Cafes",
		GetField:        cafeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToCafe(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongDrink,
		NSID:            NSIDDrink,
		DisplayName:     "Tea Drink",
		Noun:            "drink",
		URLPath:         "drinks",
		FeedFilterLabel: "Drinks",
		GetField:        drinkField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToDrink(rec, uri)
		},
	})
	// Comment and Like are intentionally NOT registered.
	// App.NSIDs() in internal/atplatform/domain/app.go appends them
	// unconditionally — registering them as descriptors would produce
	// duplicates. Same convention as internal/entities/arabica/register.go.
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/entities/oolong/... -run "Registered|ByNSID|NotRegistered" -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/entities/oolong/
git commit -m "feat(oolong): register entity descriptors"
```

---

## Task 22: Per-app descriptor filter helper

Right now `newArabicaApp()` calls `entities.All()`, relying on the fact that only arabica's descriptors are registered globally. After Task 21 the global registry holds both arabica and oolong descriptors, so both apps would see each other's entities. Add a per-NSID-prefix filter helper.

**Files:**
- Modify: `internal/entities/entities.go`
- Test: `internal/entities/entities_test.go` (add to existing file)

- [ ] **Step 1: Read the current `entities.go`**

Run: `cat internal/entities/entities.go`
Confirm the existing `All()` function shape.

- [ ] **Step 2: Write a failing test**

Append to `internal/entities/entities_test.go`:

```go
func TestAllForApp_filtersByNSIDPrefix(t *testing.T) {
	// This test runs after init() registrations from arabica AND oolong (when
	// both packages are blank-imported by tests). Confirm the filter splits
	// them correctly. Imports needed in this test file:
	//   _ "tangled.org/arabica.social/arabica/internal/entities/arabica"
	//   _ "tangled.org/arabica.social/arabica/internal/entities/oolong"
	arab := entities.AllForApp("social.arabica.alpha")
	for _, d := range arab {
		assert.True(t, strings.HasPrefix(d.NSID, "social.arabica.alpha."),
			"arabica filter leaked NSID %s", d.NSID)
	}
	assert.NotEmpty(t, arab, "expected arabica descriptors")

	tea := entities.AllForApp("social.oolong.alpha")
	for _, d := range tea {
		assert.True(t, strings.HasPrefix(d.NSID, "social.oolong.alpha."),
			"oolong filter leaked NSID %s", d.NSID)
	}
	assert.NotEmpty(t, tea, "expected oolong descriptors")

	// No overlap.
	for _, a := range arab {
		for _, o := range tea {
			assert.NotEqual(t, a.NSID, o.NSID)
		}
	}
}
```

If `entities_test.go` doesn't exist or doesn't already blank-import both packages, add the imports at the top:

```go
import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "tangled.org/arabica.social/arabica/internal/entities/arabica"
	_ "tangled.org/arabica.social/arabica/internal/entities/oolong"
)
```

- [ ] **Step 3: Run — expect failure**

Run: `go test ./internal/entities/... -run AllForApp -v`
Expected: undefined `entities.AllForApp`.

- [ ] **Step 4: Implement `AllForApp` in `entities.go`**

Add to `internal/entities/entities.go`:

```go
// AllForApp returns descriptors whose NSID begins with `nsidBase + "."`,
// in stable RecordType order. Use this from per-app App constructors so
// that the global registry (which may hold descriptors from sister apps)
// doesn't leak across app boundaries.
func AllForApp(nsidBase string) []*Descriptor {
	prefix := nsidBase + "."
	out := make([]*Descriptor, 0, len(registry))
	for _, d := range registry {
		if strings.HasPrefix(d.NSID, prefix) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}
```

Add `"strings"` to the package imports.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/entities/... -run AllForApp -v`
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/entities/entities.go internal/entities/entities_test.go
git commit -m "feat(entities): AllForApp filters descriptors by NSID base"
```

---

## Task 23: Wire oolong descriptors into `newTeaApp()`

**Files:**
- Modify: `cmd/server/apps.go`
- Modify: `cmd/server/apps_test.go`

- [ ] **Step 1: Read current `apps.go` and `apps_test.go`**

Run: `cat cmd/server/apps.go cmd/server/apps_test.go`
Note `newArabicaApp` uses `entities.All()` and the test asserts an exact list of arabica NSIDs.

- [ ] **Step 2: Update `newArabicaApp()` to filter**

Edit `cmd/server/apps.go`. Change:

```go
Descriptors: entities.All(),
```

to:

```go
Descriptors: entities.AllForApp(arabica.NSIDBase),
```

- [ ] **Step 3: Update `newTeaApp()` to wire oolong descriptors**

In the same file, update `newTeaApp()`. Add an oolong import:

```go
import (
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
)
```

Use the import (Go won't compile unused imports — use it via a blank reference if oolong's `NSIDBase` is the natural call):

```go
func newTeaApp() *domain.App {
	return &domain.App{
		Name:        teaAppName,
		NSIDBase:    oolong.NSIDBase,
		Descriptors: entities.AllForApp(oolong.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: strings.ToUpper(teaAppName[:1]) + teaAppName[1:],
			Tagline:     "Your tea, your data",
		},
	}
}
```

> **Note:** the existing `NSIDBase: "social." + teaAppName + ".alpha"` literal is replaced by the package constant `oolong.NSIDBase` for single-source-of-truth. Verify they are equal — `oolong.NSIDBase == "social.oolong.alpha"` and `teaAppName == "oolong"` ensures `"social." + "oolong" + ".alpha" == "social.oolong.alpha"`. ✓

- [ ] **Step 4: Add `TestTeaApp_NSIDs` to `apps_test.go`**

Append to `cmd/server/apps_test.go`:

```go
func TestTeaApp_NSIDs(t *testing.T) {
	app := newTeaApp()
	got := app.NSIDs()
	sort.Strings(got)

	want := []string{
		"social.oolong.alpha.brew",
		"social.oolong.alpha.brewer",
		"social.oolong.alpha.cafe",
		"social.oolong.alpha.comment",
		"social.oolong.alpha.drink",
		"social.oolong.alpha.like",
		"social.oolong.alpha.recipe",
		"social.oolong.alpha.tea",
		"social.oolong.alpha.vendor",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}

func TestTeaApp_OAuthScopes(t *testing.T) {
	app := newTeaApp()
	got := app.OAuthScopes()
	sort.Strings(got)

	want := []string{
		"atproto",
		"repo:social.oolong.alpha.brew",
		"repo:social.oolong.alpha.brewer",
		"repo:social.oolong.alpha.cafe",
		"repo:social.oolong.alpha.comment",
		"repo:social.oolong.alpha.drink",
		"repo:social.oolong.alpha.like",
		"repo:social.oolong.alpha.recipe",
		"repo:social.oolong.alpha.tea",
		"repo:social.oolong.alpha.vendor",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}
```

> **Note:** `App.NSIDs()` (in `internal/atplatform/domain/app.go`) appends `.like` and `.comment` to the descriptor NSIDs. Task 21 deliberately does **not** register `comment` or `like` as descriptors — `App.NSIDs()` adds them itself. This way the expected NSID list shown above includes both without duplicates.

- [ ] **Step 5: Verify build + full test suite**

Run: `go build ./...`
Expected: clean.

Run: `go test ./...`
Expected: all pass. Pay attention to `cmd/server/apps_test.go` — both arabica and tea NSID tests must pass.

Run: `go vet ./...`
Expected: clean.

- [ ] **Step 6: Verify all 9 lexicon files exist**

Run: `ls lexicons/social.oolong.alpha.* | wc -l`
Expected: `9`.

- [ ] **Step 7: Smoke-test the running binary**

Run: `go build -o /tmp/server ./cmd/server && /tmp/server --help 2>&1 | head -20`
(Adjust if the binary takes different flags; just confirm it links and starts argument parsing.)

Run: `rm /tmp/server`

- [ ] **Step 8: Commit**

```bash
git add cmd/server/apps.go cmd/server/apps_test.go internal/entities/oolong/
git commit -m "feat(oolong): wire descriptors into newTeaApp"
```

---

## Phase 1 Definition of Done

After Task 23:

- All 9 lexicon JSON files exist under `lexicons/` and parse as JSON.
- All 9 NSID constants are exposed from `internal/entities/oolong`.
- All 9 model types (`Tea`, `Brew`, `Brewer`, `Recipe`, `Vendor`, `Cafe`, `Drink`, `Comment`, `Like`) have `XToRecord` and `RecordToX` functions with shutter-snapshot tests covering at least one full and one minimal case each.
- The `methodParams` union round-trips correctly for `gongfu`, `matcha`, `milkTea`, and `longSteep` (no params) styles.
- `internal/entities` registry contains 7 oolong descriptors (`tea`, `vendor`, `brewer`, `recipe`, `brew`, `cafe`, `drink` — `comment` and `like` are intentionally excluded; `App.NSIDs()` appends them).
- `entities.AllForApp(nsidBase)` filters the global registry by app, and both `newArabicaApp()` and `newTeaApp()` use it to populate their `Descriptors`.
- `cmd/server/apps_test.go` has both `TestArabicaApp_NSIDs` (existing, unchanged expected list) and a new `TestTeaApp_NSIDs` asserting oolong's 9 NSIDs.
- `go build ./...`, `go vet ./...`, `go test ./...` all succeed.
- The CLAUDE.md "Adding a new entity type" checklist is **not** executed for oolong yet — that work belongs to Phases 2-4 (store, firehose, OAuth, routes, UI).

## Phases 2-4 (preview, not part of this plan)

- **Phase 2** — Store, witness cache, session cache, firehose pipeline. Most of this is registering oolong NSIDs with existing infrastructure (`internal/atproto/store_generic.go`, `internal/firehose/config.go`, etc.). The big new work is `feed.FeedItem` extension for oolong-specific fields.
- **Phase 3** — OAuth scope, routes, handlers. Per-entity CRUD handlers, page routes, modal endpoints, OG card generators. A lot of code, mostly mirroring arabica's handlers package.
- **Phase 4** — UI. Templ pages, components, combo-select integration, navigation. Likely splits into multiple plans (forms, views, feed, navigation).

Each phase gets its own spec and plan, written when the previous phase lands.

