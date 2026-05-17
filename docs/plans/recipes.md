# Recipe Evolution: From Structured to Freeform

## Problem

Recipes today are pourover templates. They assume a fixed set of parameters
(coffee amount, water amount, grind size, brewer, pours) that don't generalize
to espresso, AeroPress, cold brew, milk drinks, or novel methods. Users can't
express "steam milk to 65C" or "pull a 1:2 shot in 28s" within the current
structure.

The goal: let power users build custom recipes from composable parts, while
keeping simple recipes simple and preserving the ability to filter/compare
across recipes.

## Design Axes

Two tensions shape the design space:

1. **Structure vs. Freedom** — Fixed fields enable queries and comparison.
   Freeform fields enable creativity and method diversity.
2. **Simple vs. Power** — A basic user wants to log "18g in, 300g out, medium
   grind." A power user wants timed pour sequences, temperature profiles, and
   milk steaming steps.

The sweet spot is a design that serves both without forcing either into the
other's workflow.

## The Spectrum

### Level 0: What We Have Now

```
name, brewerRef, coffeeAmount, waterAmount, pours[], notes
```

Fixed schema. Works for pourover. Breaks for everything else. Pours are the
only "composable" element, and they're locked to water amount + time.

### Level 1: Core + Extensions

Keep the fields that are universal to nearly all coffee preparation, and add an
open union array for everything else.

**Core fields (always present, queryable):**
- `name` — recipe name
- `coffeeAmount` — dose in tenths of grams (universal to all methods)
- `notes` — freeform text
- `sourceRef` — fork provenance

**Optional structured fields (queryable when present):**
- `waterAmount` — total water (most methods, but not all — e.g., espresso
  yield is measured differently)
- `brewerRef` / `brewerType` — gear reference

**Extensions (open union array):**
```json
"extensions": {
  "type": "array",
  "items": {
    "type": "union",
    "refs": [
      "#pourStep",
      "#waitStep",
      "#tempStep",
      "#pressStep",
      "#steamStep",
      "#textParam",
      "#gearRef"
    ]
  }
}
```

This is the most conservative evolution. Simple recipes look identical to
today. Power users bolt on steps and parameters. Filtering works on core
fields. The split between "core" and "extension" can feel arbitrary though —
why is grind size a core field but brew temperature isn't?

**Good for:** incremental migration, backwards compat, keeping simple things
simple.

### Level 2: Minimal Core + Rich Parameters

Shrink the core to just what's truly universal, and push everything else into
typed parameter blocks.

**Core fields:**
- `name`
- `coffeeAmount` — the one thing every coffee recipe has
- `notes`
- `sourceRef`

**Parameters (open union array):**
```json
"parameters": {
  "type": "array",
  "items": {
    "type": "union",
    "refs": [
      "#weightParam",
      "#ratioParam",
      "#tempParam",
      "#timeParam",
      "#textParam",
      "#gearRef",
      "#ingredientRef"
    ]
  }
}
```

Each parameter carries a `label` (user-defined display name) and typed value:

| Type             | Fields                        | Example                        |
|------------------|-------------------------------|--------------------------------|
| `#weightParam`   | `label`, `grams` (int/10ths)  | "Water": 3000 (= 300.0g)      |
| `#ratioParam`    | `label`, `ratio` (float-ish)  | "Brew Ratio": 16.7             |
| `#tempParam`     | `label`, `celsius` (int/10th) | "Brew Temp": 930 (= 93.0C)    |
| `#timeParam`     | `label`, `seconds` (int)      | "Bloom Time": 45               |
| `#textParam`     | `label`, `value` (string)     | "Grind": "18 clicks on C40"   |
| `#gearRef`       | `label`, `ref` (at-uri)       | "Brewer": at://did/collection/rkey |
| `#ingredientRef` | `label`, `ref` (at-uri)       | "Bean": at://did/collection/rkey   |

**Steps (separate open union array for process):**
```json
"steps": {
  "type": "array",
  "items": {
    "type": "union",
    "refs": [
      "#pourStep",
      "#waitStep",
      "#stirStep",
      "#pressStep",
      "#steamStep",
      "#customStep"
    ]
  }
}
```

Steps are ordered and describe the process:

| Type          | Fields                              | Example                    |
|---------------|-------------------------------------|----------------------------|
| `#pourStep`   | `label?`, `grams`, `seconds`        | Bloom: 50g at 0:00         |
| `#waitStep`   | `label?`, `seconds`                 | Wait 45s                   |
| `#stirStep`   | `label?`, `technique?`              | Rao spin                   |
| `#pressStep`  | `label?`, `seconds?`               | Plunge over 30s            |
| `#steamStep`  | `label?`, `milkType?`, `tempC?`    | Steam oat milk to 65C      |
| `#customStep` | `label`, `description`              | "Swirl the V60 3 times"    |

This gives you two dimensions: **what goes in** (parameters) and **what you do**
(steps). A pourover recipe might have 3 parameters and 5 steps. An espresso
recipe might have 6 parameters and 1 step. A milk drink might chain espresso
extraction into steaming into latte art.

**Good for:** method diversity, power users, recipe builder UI.

### Level 3: Everything is a Facet

The most freeform option. No core fields beyond `name`. The recipe is a bag of
typed facets, each one self-describing.

```json
{
  "name": "Morning Espresso",
  "facets": [
    { "$type": "#weightParam", "label": "Dose", "grams": 180 },
    { "$type": "#weightParam", "label": "Yield", "grams": 360 },
    { "$type": "#timeParam", "label": "Shot Time", "seconds": 28 },
    { "$type": "#tempParam", "label": "Brew Temp", "celsius": 930 },
    { "$type": "#gearRef", "label": "Machine", "ref": "at://..." },
    { "$type": "#textParam", "label": "Grind", "value": "Setting 2.5" },
    { "$type": "#steamStep", "label": "Milk", "milkType": "Oat", "tempC": 650 },
    { "$type": "#customStep", "label": "Latte Art", "description": "Tulip" }
  ],
  "notes": "Pull shot first, steam while extracting",
  "sourceRef": "at://..."
}
```

Maximum flexibility. But you lose the ability to query "all recipes with >15g
dose" unless you define conventions about label names, which is fragile. Also
mixes parameters and process into one flat list — rendering order matters but
semantic grouping is lost.

**Good for:** maximum creative freedom. **Bad for:** filtering, comparison,
consistent UI.

## Recommended Approach: Level 2 Hybrid

Level 2 hits the sweet spot. Here's why:

### coffeeAmount stays in core

Every coffee recipe starts with a dose. Keeping it as a fixed field means:
- Explore page can filter by dose range
- Ratio computation works reliably (coffeeAmount + a waterAmount param or a
  ratioParam)
- Simple recipes need zero extensions — just fill in the core

### waterAmount moves to a parameter (with a convention)

Water/yield is method-dependent. For pourover it's total water. For espresso
it's liquid yield. For cold brew it's steep water. Making it a `weightParam`
with a conventional label handles all of these, but you lose trivial ratio
computation unless the UI knows to look for the first `weightParam` or a
`ratioParam`.

**Alternative:** keep `waterAmount` in core too. It's *almost* universal, and
having both dose and water in core makes ratio filtering work everywhere. The
only methods where it's awkward are Turkish coffee (no measured water) and
cupping — edge cases you can ignore for now.

### Parameters are the "what"

Typed building blocks for inputs: grind settings, temperatures, gear
references, ingredient refs. The type system means the UI can render
appropriate inputs (number spinner for weight, temperature picker for temp,
gear selector for refs).

### Steps are the "how"

Ordered process instructions. Pour schedules, wait times, stir techniques,
press actions. Having steps separate from parameters means you can show them
differently in the UI — parameters in a summary card, steps in a timeline.

### The customStep / textParam escape hatches

Power users can always add a `customStep` or `textParam` when the typed
options don't cover their needs. This prevents the system from being limiting
while still encouraging structured data when it fits.

## What This Looks Like in Practice

### Simple Pourover (basic user)

Core fields only, no extensions needed:

```
name: "Daily V60"
coffeeAmount: 180        (18.0g)
waterAmount: 3000        (300.0g)  [if kept in core]
brewerRef: at://did/...brewer/abc
notes: "Standard recipe, nothing fancy"
```

Identical to today. Zero learning curve.

### Detailed Pourover (power user)

Core fields plus steps:

```
name: "Hoffmann V60"
coffeeAmount: 150        (15.0g)
waterAmount: 2500        (250.0g)
brewerRef: at://did/...brewer/abc

parameters:
  tempParam("Water Temp", 950)       (95.0C)
  gearRef("Grinder", at://did/...grinder/xyz)
  textParam("Filter", "Cafec Abaca")

steps:
  pourStep("Bloom", 50g, 0s)
  waitStep("Bloom Wait", 45s)
  pourStep("Main Pour", 200g, 45s)
  stirStep("Swirl")
  waitStep("Drawdown", 60s)
```

### Espresso

```
name: "Morning Shot"
coffeeAmount: 180        (18.0g)

parameters:
  weightParam("Yield", 360)          (36.0g)
  timeParam("Shot Time", 28)
  tempParam("Brew Temp", 930)
  gearRef("Machine", at://did/...brewer/abc)
  textParam("Grind", "2.5 on Niche")

steps:
  customStep("Prep", "WDT, tamp level, pull shot")
```

### Oat Latte

```
name: "Oat Flat White"
coffeeAmount: 180        (18.0g)

parameters:
  weightParam("Yield", 360)
  timeParam("Shot Time", 28)
  gearRef("Machine", at://did/...brewer/abc)
  ingredientRef("Bean", at://did/...bean/xyz)

steps:
  extractStep("Pull Shot")
  steamStep("Steam Milk", milkType: "Oat", tempC: 650)
  customStep("Pour", "Flat white dot pattern")
```

### Cold Brew

```
name: "Weekend Cold Brew"
coffeeAmount: 700        (70.0g)

parameters:
  weightParam("Water", 7000)
  textParam("Grind", "Coarse")
  tempParam("Steep Temp", 40)        (4.0C / fridge)
  timeParam("Steep Time", 57600)     (16 hours)

steps:
  customStep("Combine", "Add grounds to jar, pour water, stir")
  waitStep("Steep", 57600)
  customStep("Filter", "Strain through Chemex filter, dilute 1:1")
```

## Recipe Builder UI

The UI becomes a two-panel recipe builder:

**Left panel: Parameters**
- Coffee amount always visible (core)
- Water amount visible by default (core or auto-added param)
- "Add parameter" dropdown: Weight, Temp, Time, Text, Gear, Ingredient
- Each parameter rendered with appropriate input for its type
- Drag to reorder

**Right panel: Steps (optional)**
- "Add step" dropdown: Pour, Wait, Stir, Press, Steam, Custom
- Each step rendered as a timeline card
- Drag to reorder
- Collapse/expand for complex recipes

**Simple mode:** Just the core fields — name, dose, water, grind, brewer,
notes. No parameters panel, no steps panel. Looks like today's form.

**Power mode:** Toggle or auto-expand when user adds first parameter or step.
Or just always show the "Add parameter" / "Add step" buttons below the core
fields.

## Migration

The current lexicon can evolve to Level 2 without breaking existing records:

1. Existing fields (`coffeeAmount`, `waterAmount`, `brewerRef`,
   `brewerType`, `pours`) remain and continue to work
2. Add `parameters` and `steps` as new optional array fields
3. Existing `pours` could be deprecated in favor of `steps` with `#pourStep`,
   but old records with `pours` still parse fine
4. New UI writes to both `pours` (backwards compat) and `steps` (new format)
   during transition, then drops `pours` in a future version

No data migration needed. Old records just lack the new fields.

## Lexicon Sketch

```json
{
  "lexicon": 1,
  "id": "social.arabica.alpha.recipe",
  "defs": {
    "main": {
      "type": "record",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["name", "createdAt"],
        "properties": {
          "name": { "type": "string", "maxLength": 200 },
          "coffeeAmount": { "type": "integer", "minimum": 0 },
          "waterAmount": { "type": "integer", "minimum": 0 },
          "brewerRef": { "type": "string", "format": "at-uri" },
          "brewerType": { "type": "string", "maxLength": 100 },
          "parameters": {
            "type": "array",
            "maxLength": 20,
            "items": { "type": "union", "refs": [
              "#weightParam", "#ratioParam", "#tempParam",
              "#timeParam", "#textParam", "#gearRef", "#ingredientRef"
            ]}
          },
          "steps": {
            "type": "array",
            "maxLength": 30,
            "items": { "type": "union", "refs": [
              "#pourStep", "#waitStep", "#stirStep",
              "#pressStep", "#steamStep", "#customStep"
            ]}
          },
          "pours": {
            "type": "array",
            "description": "[DEPRECATED] Use steps with #pourStep instead",
            "items": { "type": "ref", "ref": "#pour" }
          },
          "notes": { "type": "string", "maxLength": 2000 },
          "sourceRef": { "type": "string", "format": "at-uri" },
          "createdAt": { "type": "string", "format": "datetime" }
        }
      }
    },

    "pour": {
      "type": "object",
      "description": "[DEPRECATED] Legacy pour format",
      "required": ["waterAmount", "timeSeconds"],
      "properties": {
        "waterAmount": { "type": "integer", "minimum": 0 },
        "timeSeconds": { "type": "integer", "minimum": 0 }
      }
    },

    "weightParam": {
      "type": "object",
      "required": ["label", "grams"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "grams": { "type": "integer", "minimum": 0,
                    "description": "Weight in tenths of grams" }
      }
    },
    "ratioParam": {
      "type": "object",
      "required": ["label", "ratio"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "ratio": { "type": "integer", "minimum": 0,
                    "description": "Ratio in tenths (167 = 1:16.7)" }
      }
    },
    "tempParam": {
      "type": "object",
      "required": ["label", "celsius"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "celsius": { "type": "integer", "minimum": 0,
                      "description": "Temp in tenths of degrees C" }
      }
    },
    "timeParam": {
      "type": "object",
      "required": ["label", "seconds"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "seconds": { "type": "integer", "minimum": 0 }
      }
    },
    "textParam": {
      "type": "object",
      "required": ["label", "value"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "value": { "type": "string", "maxLength": 500 }
      }
    },
    "gearRef": {
      "type": "object",
      "required": ["label", "ref"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "ref": { "type": "string", "format": "at-uri" }
      }
    },
    "ingredientRef": {
      "type": "object",
      "required": ["label", "ref"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "ref": { "type": "string", "format": "at-uri" }
      }
    },

    "pourStep": {
      "type": "object",
      "required": ["grams", "seconds"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "grams": { "type": "integer", "minimum": 0 },
        "seconds": { "type": "integer", "minimum": 0 }
      }
    },
    "waitStep": {
      "type": "object",
      "required": ["seconds"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "seconds": { "type": "integer", "minimum": 0 }
      }
    },
    "stirStep": {
      "type": "object",
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "technique": { "type": "string", "maxLength": 200 }
      }
    },
    "pressStep": {
      "type": "object",
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "seconds": { "type": "integer", "minimum": 0 }
      }
    },
    "steamStep": {
      "type": "object",
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "milkType": { "type": "string", "maxLength": 100 },
        "tempCelsius": { "type": "integer", "minimum": 0,
                          "description": "Tenths of degrees C" }
      }
    },
    "customStep": {
      "type": "object",
      "required": ["label", "description"],
      "properties": {
        "label": { "type": "string", "maxLength": 50 },
        "description": { "type": "string", "maxLength": 500 }
      }
    }
  }
}
```

## Open Questions

1. **Should `waterAmount` stay in core?** It makes ratio filtering trivial but
   is awkward for espresso (where "yield" is the output measurement, not input
   water). Could keep it in core with the understanding that for espresso
   recipes, a `weightParam("Yield", ...)` is the meaningful number and
   `waterAmount` is omitted.

2. **Parameter ordering** — should the array order be meaningful (display
   order) or should the UI sort by type? Leaning toward array order = display
   order, since users will arrange their recipe builder intentionally.

3. **Step timing model** — current pours use absolute time (seconds from brew
   start). Steps could use relative time (duration of this step) or absolute.
   Relative is simpler for the user; absolute is easier for a timer UI.

4. **Preset templates** — should there be a `method` or `template` field that
   pre-populates parameters and steps? e.g., selecting "Espresso" auto-adds
   dose/yield/time/temp params. This is purely a UI concern, not a lexicon one.

5. **Brew integration** — brews currently reference a recipe. With freeform
   recipes, should a brew record capture the *snapshot* of parameters used, or
   just reference the recipe? Snapshot is more accurate (recipe might change),
   reference is lighter.
