# Brewer Type Enum & Method-Specific Brew Params

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize brewer types as a known-values enum, then use the selected brewer's type to conditionally show method-specific parameter fields (espresso, pour-over) in the brew form.

**Architecture:** Add `knownValues` to the brewer lexicon's `brewerType` field. Add `espressoParams` and `pouroverParams` optional sub-objects to the brew lexicon. Update the Go models, AT Protocol record conversion, brew form template, and Alpine.js to show/hide method-specific fields based on the selected brewer's type. Leave TODOs for future method types (immersion, moka pot, cold brew, cupping).

**Tech Stack:** Go, templ, Alpine.js, HTMX, AT Protocol lexicons, Tailwind CSS

---

## Context

### Brewer Type Categories

| Canonical value | Examples | Extra form fields |
|---|---|---|
| `pourover` | V60, Chemex, Kalita Wave, Origami | Bloom water (g), bloom time (s), drawdown time (s), bypass water (g) |
| `espresso` | Machine, lever, manual | Yield weight (g), pressure (bar), pre-infusion time (s) |
| `immersion` | French press, Clever Dripper, siphon, Aeropress | TODO: future |
| `mokapot` | Moka pot, Bialetti | TODO: future (heat level) |
| `coldbrew` | Cold brew, cold drip | TODO: future |
| `cupping` | Cupping | TODO: future |
| `other` | Turkish, custom | No extra fields |

### Backwards Compatibility

- Existing brewer records with freeform strings (e.g. "Pour-Over") continue to work
- New optional fields on the brew lexicon are additive — old records simply lack them
- The app normalizes freeform values to canonical enum values on read for form logic
- Old clients ignore unknown fields per AT Protocol convention

### Key Files

- `lexicons/social.arabica.alpha.brewer.json` — brewer lexicon
- `lexicons/social.arabica.alpha.brew.json` — brew lexicon
- `internal/models/models.go` — Go domain models
- `internal/atproto/records.go` — AT Protocol record ↔ model conversion
- `internal/handlers/brew.go` — brew create/update handlers
- `internal/web/pages/brew_form.templ` — brew form template
- `static/js/brew-form.js` — Alpine.js brew form component
- `static/js/dropdown-manager.js` — brewer type lookup
- `internal/web/components/dialog_modals.templ` — brewer create/edit modal
- `internal/web/pages/profile.templ` — inline brewer form
- `internal/web/pages/brew_view.templ` — brew detail display
- `internal/web/components/icons.templ` — SVG icons

---

## Task 1: Update Brewer Lexicon with `knownValues`

**Files:**
- Modify: `lexicons/social.arabica.alpha.brewer.json`

**Step 1: Update the brewerType field to include knownValues**

The `brewerType` field is currently a free string. Add `knownValues` to guide clients while keeping it an open string (AT Protocol convention — `knownValues` is advisory, not enforced).

```json
"brewerType": {
  "type": "string",
  "maxLength": 100,
  "knownValues": [
    "pourover",
    "espresso",
    "immersion",
    "mokapot",
    "coldbrew",
    "cupping",
    "other"
  ],
  "description": "Category of brewer. Known values: pourover, espresso, immersion, mokapot, coldbrew, cupping, other"
}
```

**Step 2: Commit**

```
feat: add knownValues to brewer lexicon brewerType field
```

---

## Task 2: Update Brew Lexicon with Method-Specific Params

**Files:**
- Modify: `lexicons/social.arabica.alpha.brew.json`

**Step 1: Add espressoParams and pouroverParams sub-objects**

Add two new optional fields to the brew record, and two new `#defs` sub-objects:

In `properties` of the main record, add:

```json
"espressoParams": {
  "type": "ref",
  "ref": "#espressoParams",
  "description": "Espresso-specific brewing parameters (optional)"
},
"pouroverParams": {
  "type": "ref",
  "ref": "#pouroverParams",
  "description": "Pour-over-specific brewing parameters (optional)"
}
```

Add new defs alongside the existing `pour` def:

```json
"espressoParams": {
  "type": "object",
  "description": "Parameters specific to espresso brewing",
  "properties": {
    "yieldWeight": {
      "type": "integer",
      "minimum": 0,
      "description": "Espresso yield/output weight in tenths of a gram (e.g., 360 = 36.0g)"
    },
    "pressure": {
      "type": "integer",
      "minimum": 0,
      "description": "Brewing pressure in tenths of a bar (e.g., 90 = 9.0 bar)"
    },
    "preInfusionSeconds": {
      "type": "integer",
      "minimum": 0,
      "description": "Pre-infusion time in seconds"
    }
  }
},
"pouroverParams": {
  "type": "object",
  "description": "Parameters specific to pour-over brewing",
  "properties": {
    "bloomWater": {
      "type": "integer",
      "minimum": 0,
      "description": "Water used for bloom in grams"
    },
    "bloomSeconds": {
      "type": "integer",
      "minimum": 0,
      "description": "Bloom wait time in seconds"
    },
    "drawdownSeconds": {
      "type": "integer",
      "minimum": 0,
      "description": "Drawdown time in seconds (time after last pour until bed is dry)"
    },
    "bypassWater": {
      "type": "integer",
      "minimum": 0,
      "description": "Bypass water added after brewing in grams"
    }
  }
}
```

**Step 2: Commit**

```
feat: add espressoParams and pouroverParams to brew lexicon
```

---

## Task 3: Add Brewer Type Constants and Method Param Models

**Files:**
- Modify: `internal/models/models.go`
- Test: `internal/models/models_test.go`

**Step 1: Add brewer type constants**

Add after the existing field length constants block:

```go
// Brewer type categories (knownValues from lexicon)
const (
	BrewerTypePourover  = "pourover"
	BrewerTypeEspresso  = "espresso"
	BrewerTypeImmersion = "immersion"
	BrewerTypeMokaPot   = "mokapot"
	BrewerTypeColdBrew  = "coldbrew"
	BrewerTypeCupping   = "cupping"
	BrewerTypeOther     = "other"
)

// BrewerTypeLabels maps canonical brewer type values to display labels
var BrewerTypeLabels = map[string]string{
	BrewerTypePourover:  "Pour-over",
	BrewerTypeEspresso:  "Espresso",
	BrewerTypeImmersion: "Immersion",
	BrewerTypeMokaPot:   "Moka Pot",
	BrewerTypeColdBrew:  "Cold Brew",
	BrewerTypeCupping:   "Cupping",
	BrewerTypeOther:     "Other",
}

// BrewerTypeKnownValues is the ordered list for form dropdowns
var BrewerTypeKnownValues = []string{
	BrewerTypePourover,
	BrewerTypeEspresso,
	BrewerTypeImmersion,
	BrewerTypeMokaPot,
	BrewerTypeColdBrew,
	BrewerTypeCupping,
	BrewerTypeOther,
}
```

**Step 2: Add NormalizeBrewerType function**

This maps legacy freeform strings to canonical values:

```go
// NormalizeBrewerType maps freeform brewer type strings to canonical values.
// Returns the input unchanged if no mapping is found (preserves unknown values).
func NormalizeBrewerType(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lower == "pourover" || lower == "pour-over" || lower == "pour over" || lower == "dripper":
		return BrewerTypePourover
	case lower == "espresso" || lower == "espresso machine" || lower == "lever espresso" || lower == "lever espresso machine":
		return BrewerTypeEspresso
	case lower == "immersion" || lower == "french press" || lower == "aeropress" || lower == "siphon" || lower == "clever" || lower == "clever dripper":
		return BrewerTypeImmersion
	case lower == "mokapot" || lower == "moka pot" || lower == "moka" || lower == "bialetti":
		return BrewerTypeMokaPot
	case lower == "coldbrew" || lower == "cold brew" || lower == "cold drip":
		return BrewerTypeColdBrew
	case lower == "cupping":
		return BrewerTypeCupping
	case lower == "other":
		return BrewerTypeOther
	default:
		return raw // preserve unknown values
	}
}
```

Note: add `"strings"` to the import block in models.go.

**Step 3: Add EspressoParams and PouroverParams structs**

Add after the `Pour` struct:

```go
// EspressoParams holds espresso-specific brewing parameters
type EspressoParams struct {
	YieldWeight        float64 `json:"yield_weight"`         // Output weight in grams
	Pressure           float64 `json:"pressure"`             // Pressure in bar
	PreInfusionSeconds int     `json:"pre_infusion_seconds"` // Pre-infusion time
}

// PouroverParams holds pour-over-specific brewing parameters
type PouroverParams struct {
	BloomWater      int `json:"bloom_water"`      // Bloom water in grams
	BloomSeconds    int `json:"bloom_seconds"`    // Bloom wait time in seconds
	DrawdownSeconds int `json:"drawdown_seconds"` // Drawdown time in seconds
	BypassWater     int `json:"bypass_water"`     // Bypass water in grams
}
```

**Step 4: Add fields to Brew model**

Add to the `Brew` struct, after the `Pours` field:

```go
	EspressoParams *EspressoParams `json:"espresso_params,omitempty"`
	PouroverParams *PouroverParams `json:"pourover_params,omitempty"`
```

**Step 5: Add fields to CreateBrewRequest**

Add to `CreateBrewRequest`, after `Pours`:

```go
	EspressoParams *EspressoParams `json:"espresso_params,omitempty"`
	PouroverParams *PouroverParams `json:"pourover_params,omitempty"`
```

**Step 6: Write tests for NormalizeBrewerType**

In `internal/models/models_test.go`, add:

```go
func TestNormalizeBrewerType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pourover", "pourover"},
		{"Pour-Over", "pourover"},
		{"pour over", "pourover"},
		{"Dripper", "pourover"},
		{"espresso", "espresso"},
		{"Espresso Machine", "espresso"},
		{"Lever Espresso Machine", "espresso"},
		{"immersion", "immersion"},
		{"French Press", "immersion"},
		{"Aeropress", "immersion"},
		{"Clever Dripper", "immersion"},
		{"mokapot", "mokapot"},
		{"Moka Pot", "mokapot"},
		{"coldbrew", "coldbrew"},
		{"Cold Brew", "coldbrew"},
		{"cupping", "cupping"},
		{"other", "other"},
		{"SomeUnknownType", "SomeUnknownType"}, // preserved
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeBrewerType(tt.input))
		})
	}
}
```

**Step 7: Run tests**

```bash
go test ./internal/models/... -v -run TestNormalizeBrewerType
```

**Step 8: Run vet and build**

```bash
go vet ./... && go build ./...
```

**Step 9: Commit**

```
feat: add brewer type constants, method param models, and NormalizeBrewerType
```

---

## Task 4: Update AT Protocol Record Conversion

**Files:**
- Modify: `internal/atproto/records.go`
- Test: `internal/atproto/records_test.go`

**Step 1: Update BrewToRecord to serialize new params**

In the `BrewToRecord` function, after the pours serialization block (after `record["pours"] = pours`), add:

```go
	// Espresso-specific params
	if brew.EspressoParams != nil {
		ep := map[string]interface{}{}
		if brew.EspressoParams.YieldWeight > 0 {
			ep["yieldWeight"] = int(brew.EspressoParams.YieldWeight * 10) // tenths of a gram
		}
		if brew.EspressoParams.Pressure > 0 {
			ep["pressure"] = int(brew.EspressoParams.Pressure * 10) // tenths of a bar
		}
		if brew.EspressoParams.PreInfusionSeconds > 0 {
			ep["preInfusionSeconds"] = brew.EspressoParams.PreInfusionSeconds
		}
		if len(ep) > 0 {
			record["espressoParams"] = ep
		}
	}

	// Pour-over-specific params
	if brew.PouroverParams != nil {
		pp := map[string]interface{}{}
		if brew.PouroverParams.BloomWater > 0 {
			pp["bloomWater"] = brew.PouroverParams.BloomWater
		}
		if brew.PouroverParams.BloomSeconds > 0 {
			pp["bloomSeconds"] = brew.PouroverParams.BloomSeconds
		}
		if brew.PouroverParams.DrawdownSeconds > 0 {
			pp["drawdownSeconds"] = brew.PouroverParams.DrawdownSeconds
		}
		if brew.PouroverParams.BypassWater > 0 {
			pp["bypassWater"] = brew.PouroverParams.BypassWater
		}
		if len(pp) > 0 {
			record["pouroverParams"] = pp
		}
	}
```

**Step 2: Update RecordToBrew to deserialize new params**

In the `RecordToBrew` function, after the pours deserialization block, add:

```go
	// Espresso params
	if epRaw, ok := record["espressoParams"].(map[string]interface{}); ok {
		ep := &models.EspressoParams{}
		if v, ok := epRaw["yieldWeight"].(float64); ok {
			ep.YieldWeight = v / 10.0 // tenths back to grams
		}
		if v, ok := epRaw["pressure"].(float64); ok {
			ep.Pressure = v / 10.0 // tenths back to bar
		}
		if v, ok := epRaw["preInfusionSeconds"].(float64); ok {
			ep.PreInfusionSeconds = int(v)
		}
		brew.EspressoParams = ep
	}

	// Pour-over params
	if ppRaw, ok := record["pouroverParams"].(map[string]interface{}); ok {
		pp := &models.PouroverParams{}
		if v, ok := ppRaw["bloomWater"].(float64); ok {
			pp.BloomWater = int(v)
		}
		if v, ok := ppRaw["bloomSeconds"].(float64); ok {
			pp.BloomSeconds = int(v)
		}
		if v, ok := ppRaw["drawdownSeconds"].(float64); ok {
			pp.DrawdownSeconds = int(v)
		}
		if v, ok := ppRaw["bypassWater"].(float64); ok {
			pp.BypassWater = int(v)
		}
		brew.PouroverParams = pp
	}
```

**Step 3: Add round-trip tests**

In `internal/atproto/records_test.go`, add tests for brew records with espresso and pourover params. Follow the existing `TestBrewRoundTrip` pattern:

```go
func TestBrewRoundTrip_EspressoParams(t *testing.T) {
	original := &models.Brew{
		BeanRKey:    "abc123",
		Temperature: 93.5,
		Rating:      8,
		CreatedAt:   time.Now().Truncate(time.Second),
		EspressoParams: &models.EspressoParams{
			YieldWeight:        36.0,
			Pressure:           9.0,
			PreInfusionSeconds: 5,
		},
	}

	record, err := BrewToRecord(original, "at://did:plc:test/social.arabica.alpha.bean/abc123", "", "", "")
	assert.NoError(t, err)

	// Verify espressoParams is in the record
	ep, ok := record["espressoParams"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 360, ep["yieldWeight"]) // 36.0 * 10
	assert.Equal(t, 90, ep["pressure"])     // 9.0 * 10
	assert.Equal(t, 5, ep["preInfusionSeconds"])

	restored, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/tid123")
	assert.NoError(t, err)
	assert.NotNil(t, restored.EspressoParams)
	assert.InDelta(t, 36.0, restored.EspressoParams.YieldWeight, 0.1)
	assert.InDelta(t, 9.0, restored.EspressoParams.Pressure, 0.1)
	assert.Equal(t, 5, restored.EspressoParams.PreInfusionSeconds)
}

func TestBrewRoundTrip_PouroverParams(t *testing.T) {
	original := &models.Brew{
		BeanRKey:  "abc123",
		CreatedAt: time.Now().Truncate(time.Second),
		PouroverParams: &models.PouroverParams{
			BloomWater:      50,
			BloomSeconds:    45,
			DrawdownSeconds: 30,
			BypassWater:     100,
		},
	}

	record, err := BrewToRecord(original, "at://did:plc:test/social.arabica.alpha.bean/abc123", "", "", "")
	assert.NoError(t, err)

	pp, ok := record["pouroverParams"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 50, pp["bloomWater"])
	assert.Equal(t, 45, pp["bloomSeconds"])
	assert.Equal(t, 30, pp["drawdownSeconds"])
	assert.Equal(t, 100, pp["bypassWater"])

	restored, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/tid123")
	assert.NoError(t, err)
	assert.NotNil(t, restored.PouroverParams)
	assert.Equal(t, 50, restored.PouroverParams.BloomWater)
	assert.Equal(t, 45, restored.PouroverParams.BloomSeconds)
	assert.Equal(t, 30, restored.PouroverParams.DrawdownSeconds)
	assert.Equal(t, 100, restored.PouroverParams.BypassWater)
}
```

**Step 4: Run tests**

```bash
go test ./internal/atproto/... -v -run TestBrewRoundTrip
```

**Step 5: Commit**

```
feat: serialize/deserialize espresso and pourover params in AT Protocol records
```

---

## Task 5: Update Brewer Form to Use Dropdown

**Files:**
- Modify: `internal/web/components/dialog_modals.templ`
- Modify: `internal/web/pages/profile.templ`

This changes the free-text `brewer_type` input to a `<select>` dropdown with known values plus an "Other" option that shows a text input for custom types.

**Step 1: Update the brewer modal in dialog_modals.templ**

Find the brewer type text input (around line 367-373):

```html
<input
    type="text"
    name="brewer_type"
    value={ getStringValue(brewer, "brewer_type") }
    placeholder="Type (e.g., Pour-Over, Immersion, Espresso)"
    class="w-full form-input"
/>
```

Replace with a select + conditional text input using Alpine.js:

```html
<div x-data="{ customType: false }" x-init="customType = !['', 'pourover', 'espresso', 'immersion', 'mokapot', 'coldbrew', 'cupping', 'other'].includes(document.querySelector('[name=brewer_type_select]')?.value)">
    <select
        name="brewer_type_select"
        @change="customType = ($event.target.value === '__custom__'); if (!customType) { $el.closest('form').querySelector('[name=brewer_type]').value = $event.target.value; }"
        class="w-full form-select"
    >
        <option value="">Select type...</option>
        for _, bt := range models.BrewerTypeKnownValues {
            <option
                value={ bt }
                if getStringValue(brewer, "brewer_type") == bt || models.NormalizeBrewerType(getStringValue(brewer, "brewer_type")) == bt {
                    selected
                }
            >
                { models.BrewerTypeLabels[bt] }
            </option>
        }
        <option value="__custom__"
            if v := getStringValue(brewer, "brewer_type"); v != "" && models.NormalizeBrewerType(v) == v && !isKnownBrewerType(v) {
                selected
            }
        >
            Custom...
        </option>
    </select>
    <input
        type="hidden"
        name="brewer_type"
        value={ getBrewerTypeValue(brewer) }
    />
    <input
        x-show="customType"
        x-cloak
        type="text"
        @input="$el.closest('form').querySelector('[name=brewer_type]').value = $event.target.value"
        value={ getCustomBrewerTypeValue(brewer) }
        placeholder="Enter custom brewer type..."
        class="w-full form-input mt-2"
    />
</div>
```

Note: You'll need to add helper functions to the `getStringValue` helper area at the bottom of `dialog_modals.templ`:

```go
func isKnownBrewerType(v string) bool {
	for _, bt := range models.BrewerTypeKnownValues {
		if v == bt {
			return true
		}
	}
	return false
}

func getBrewerTypeValue(entity interface{}) string {
	raw := ""
	switch e := entity.(type) {
	case *models.Brewer:
		if e != nil {
			raw = e.BrewerType
		}
	}
	if raw == "" {
		return ""
	}
	normalized := models.NormalizeBrewerType(raw)
	if isKnownBrewerType(normalized) {
		return normalized
	}
	return raw
}

func getCustomBrewerTypeValue(entity interface{}) string {
	raw := ""
	switch e := entity.(type) {
	case *models.Brewer:
		if e != nil {
			raw = e.BrewerType
		}
	}
	normalized := models.NormalizeBrewerType(raw)
	if isKnownBrewerType(normalized) {
		return ""
	}
	return raw
}
```

Also add `"arabica/internal/models"` to the imports if not already present.

**Step 2: Update the inline brewer form in profile.templ**

Find the brewer_type text input in profile.templ (around line 390):

```html
<input type="text" x-model="brewerForm.brewer_type" placeholder="Type (e.g., Pour-Over, Immersion, Espresso)" class="w-full form-input"/>
```

Replace with a similar select pattern. Since this form uses Alpine.js x-model, it's simpler:

```html
<select x-model="brewerForm.brewer_type" class="w-full form-select">
    <option value="">Select type...</option>
    <option value="pourover">Pour-over</option>
    <option value="espresso">Espresso</option>
    <option value="immersion">Immersion</option>
    <option value="mokapot">Moka Pot</option>
    <option value="coldbrew">Cold Brew</option>
    <option value="cupping">Cupping</option>
    <option value="other">Other</option>
</select>
```

Note: The profile page inline form doesn't need the "custom" escape hatch since it's a simpler create-only flow. Users needing custom types can edit via the modal.

**Step 3: Run templ generate and verify**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```
feat: replace brewer type free-text with enum dropdown in brewer forms
```

---

## Task 6: Update Brew Handler to Parse Method Params

**Files:**
- Modify: `internal/handlers/brew.go`

**Step 1: Add param parsing helper functions**

Add these helper functions near the existing `validateBrewRequest` function:

```go
// parseEspressoParams extracts espresso-specific params from form values.
// Returns nil if no espresso params were provided.
func parseEspressoParams(r *http.Request) *models.EspressoParams {
	yieldStr := r.FormValue("espresso_yield_weight")
	pressureStr := r.FormValue("espresso_pressure")
	preInfStr := r.FormValue("espresso_pre_infusion_seconds")

	if yieldStr == "" && pressureStr == "" && preInfStr == "" {
		return nil
	}

	ep := &models.EspressoParams{}
	if v, err := strconv.ParseFloat(yieldStr, 64); err == nil && v > 0 {
		ep.YieldWeight = v
	}
	if v, err := strconv.ParseFloat(pressureStr, 64); err == nil && v > 0 {
		ep.Pressure = v
	}
	if v, err := strconv.Atoi(preInfStr); err == nil && v > 0 {
		ep.PreInfusionSeconds = v
	}
	return ep
}

// parsePouroverParams extracts pour-over-specific params from form values.
// Returns nil if no pour-over params were provided.
func parsePouroverParams(r *http.Request) *models.PouroverParams {
	bloomWaterStr := r.FormValue("pourover_bloom_water")
	bloomSecsStr := r.FormValue("pourover_bloom_seconds")
	drawdownStr := r.FormValue("pourover_drawdown_seconds")
	bypassStr := r.FormValue("pourover_bypass_water")

	if bloomWaterStr == "" && bloomSecsStr == "" && drawdownStr == "" && bypassStr == "" {
		return nil
	}

	pp := &models.PouroverParams{}
	if v, err := strconv.Atoi(bloomWaterStr); err == nil && v > 0 {
		pp.BloomWater = v
	}
	if v, err := strconv.Atoi(bloomSecsStr); err == nil && v > 0 {
		pp.BloomSeconds = v
	}
	if v, err := strconv.Atoi(drawdownStr); err == nil && v > 0 {
		pp.DrawdownSeconds = v
	}
	if v, err := strconv.Atoi(bypassStr); err == nil && v > 0 {
		pp.BypassWater = v
	}
	return pp
}
```

**Step 2: Wire into HandleBrewCreate**

In `HandleBrewCreate`, after building the `CreateBrewRequest` (after `Pours: pours,`), add:

```go
	req.EspressoParams = parseEspressoParams(r)
	req.PouroverParams = parsePouroverParams(r)
```

**Step 3: Wire into HandleBrewUpdate**

Find the equivalent spot in `HandleBrewUpdate` and add the same two lines.

**Step 4: Ensure strconv is imported**

Check that `"strconv"` is in the imports of `brew.go`. It likely already is for existing number parsing.

**Step 5: Run vet and build**

```bash
go vet ./... && go build ./...
```

**Step 6: Commit**

```
feat: parse espresso and pourover params from brew form submissions
```

---

## Task 7: Update Store to Pass Method Params Through

**Files:**
- Modify: `internal/atproto/store.go`

The store's `CreateBrew` method builds a `Brew` model from the request and calls `BrewToRecord`. We need to pass the new params through.

**Step 1: Find where CreateBrewRequest is converted to Brew**

Search for where `CreateBrewRequest` fields are mapped to `Brew` fields in `store.go`. Add after pours mapping:

```go
	brew.EspressoParams = req.EspressoParams
	brew.PouroverParams = req.PouroverParams
```

Do the same for the update path.

**Step 2: Run vet and build**

```bash
go vet ./... && go build ./...
```

**Step 3: Commit**

```
feat: pass method-specific params through store layer
```

---

## Task 8: Add Method-Specific Form Sections to Brew Form

**Files:**
- Modify: `internal/web/pages/brew_form.templ`

**Step 1: Add new templ components for method-specific fields**

Add these after the existing `PoursSection` component:

```templ
// EspressoParamsSection renders espresso-specific fields (shown when brewer type is espresso)
templ EspressoParamsSection(props BrewFormProps) {
	<div x-show="brewerCategory === 'espresso'" x-cloak>
		<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
			<legend class="text-sm font-semibold text-brown-800 px-2">Espresso</legend>
			@components.FormField(
				components.FormFieldProps{
					Label:      "Yield Weight (grams)",
					HelperText: "Weight of espresso output",
				},
				components.NumberInput(components.NumberInputProps{
					Name:        "espresso_yield_weight",
					Value:       getEspressoYieldWeight(props),
					Placeholder: "e.g. 36",
					Step:        "0.1",
					Class:       "w-full form-input-lg",
				}),
			)
			@components.FormField(
				components.FormFieldProps{
					Label:      "Pressure (bar)",
					HelperText: "Brewing pressure",
				},
				components.NumberInput(components.NumberInputProps{
					Name:        "espresso_pressure",
					Value:       getEspressoPressure(props),
					Placeholder: "e.g. 9",
					Step:        "0.1",
					Class:       "w-full form-input-lg",
				}),
			)
			@components.FormField(
				components.FormFieldProps{Label: "Pre-infusion Time (seconds)"},
				components.NumberInput(components.NumberInputProps{
					Name:        "espresso_pre_infusion_seconds",
					Value:       getEspressoPreInfusion(props),
					Placeholder: "e.g. 5",
					Class:       "w-full form-input-lg",
				}),
			)
		</fieldset>
	</div>
}

// PouroverParamsSection renders pour-over-specific fields (shown when brewer type is pourover)
templ PouroverParamsSection(props BrewFormProps) {
	<div x-show="brewerCategory === 'pourover'" x-cloak>
		<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
			<legend class="text-sm font-semibold text-brown-800 px-2">Pour-over Details</legend>
			<div class="grid grid-cols-2 gap-4">
				@components.FormField(
					components.FormFieldProps{
						Label:      "Bloom Water (grams)",
						HelperText: "Water for bloom",
					},
					components.NumberInput(components.NumberInputProps{
						Name:        "pourover_bloom_water",
						Value:       getPouroverBloomWater(props),
						Placeholder: "e.g. 50",
						Class:       "w-full form-input-lg",
					}),
				)
				@components.FormField(
					components.FormFieldProps{
						Label:      "Bloom Time (seconds)",
						HelperText: "Bloom wait time",
					},
					components.NumberInput(components.NumberInputProps{
						Name:        "pourover_bloom_seconds",
						Value:       getPouroverBloomSeconds(props),
						Placeholder: "e.g. 45",
						Class:       "w-full form-input-lg",
					}),
				)
			</div>
			@components.FormField(
				components.FormFieldProps{
					Label:      "Drawdown Time (seconds)",
					HelperText: "Time after last pour until bed is dry",
				},
				components.NumberInput(components.NumberInputProps{
					Name:        "pourover_drawdown_seconds",
					Value:       getPouroverDrawdown(props),
					Placeholder: "e.g. 30",
					Class:       "w-full form-input-lg",
				}),
			)
			@components.FormField(
				components.FormFieldProps{
					Label:      "Bypass Water (grams)",
					HelperText: "Water added after brewing",
				},
				components.NumberInput(components.NumberInputProps{
					Name:        "pourover_bypass_water",
					Value:       getPouroverBypass(props),
					Placeholder: "e.g. 100",
					Class:       "w-full form-input-lg",
				}),
			)
		</fieldset>
	</div>
}
```

**Step 2: Add Go helper functions for reading existing values**

```go
func getEspressoYieldWeight(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.EspressoParams != nil && props.Brew.EspressoParams.YieldWeight > 0 {
		return fmt.Sprintf("%.1f", props.Brew.EspressoParams.YieldWeight)
	}
	return ""
}

func getEspressoPressure(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.EspressoParams != nil && props.Brew.EspressoParams.Pressure > 0 {
		return fmt.Sprintf("%.1f", props.Brew.EspressoParams.Pressure)
	}
	return ""
}

func getEspressoPreInfusion(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.EspressoParams != nil && props.Brew.EspressoParams.PreInfusionSeconds > 0 {
		return fmt.Sprintf("%d", props.Brew.EspressoParams.PreInfusionSeconds)
	}
	return ""
}

func getPouroverBloomWater(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.PouroverParams != nil && props.Brew.PouroverParams.BloomWater > 0 {
		return fmt.Sprintf("%d", props.Brew.PouroverParams.BloomWater)
	}
	return ""
}

func getPouroverBloomSeconds(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.PouroverParams != nil && props.Brew.PouroverParams.BloomSeconds > 0 {
		return fmt.Sprintf("%d", props.Brew.PouroverParams.BloomSeconds)
	}
	return ""
}

func getPouroverDrawdown(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.PouroverParams != nil && props.Brew.PouroverParams.DrawdownSeconds > 0 {
		return fmt.Sprintf("%d", props.Brew.PouroverParams.DrawdownSeconds)
	}
	return ""
}

func getPouroverBypass(props BrewFormProps) string {
	if props.Brew != nil && props.Brew.PouroverParams != nil && props.Brew.PouroverParams.BypassWater > 0 {
		return fmt.Sprintf("%d", props.Brew.PouroverParams.BypassWater)
	}
	return ""
}
```

**Step 3: Wire the sections into both form modes**

In both `RecipeModeSection` and `FreeformModeSection`, add the method-specific sections after the Brewing fieldset and before the Results fieldset:

```templ
		@EspressoParamsSection(props)
		@PouroverParamsSection(props)
```

**Step 4: Run templ generate and verify**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 5: Commit**

```
feat: add espresso and pourover parameter sections to brew form
```

---

## Task 9: Add Alpine.js Brewer Category Logic

**Files:**
- Modify: `static/js/brew-form.js`

**Step 1: Add brewerCategory state and logic**

In the Alpine.js `brewForm` data object, add a new reactive property:

```js
    brewerCategory: '', // 'pourover' | 'espresso' | 'immersion' | ... | ''
```

**Step 2: Update the `onBrewerChange` method**

Replace the existing `onBrewerChange` method:

```js
    onBrewerChange(rkey) {
      const brewerType = this.dropdownManager?.getBrewerType(rkey) || '';
      this.brewerCategory = this.normalizeBrewerCategory(brewerType);

      // Auto-show pours for pour-over brewers
      if (this.brewerCategory === 'pourover') {
        this.showPours = true;
      }
    },

    // Map brewer type strings to canonical categories
    normalizeBrewerCategory(raw) {
      if (!raw) return '';
      const lower = raw.toLowerCase().trim();

      // Pour-over variants
      if (['pourover', 'pour-over', 'pour over', 'dripper'].includes(lower)) return 'pourover';

      // Espresso variants
      if (['espresso', 'espresso machine', 'lever espresso', 'lever espresso machine'].includes(lower)) return 'espresso';

      // Immersion variants
      if (['immersion', 'french press', 'aeropress', 'siphon', 'clever', 'clever dripper'].includes(lower)) return 'immersion';

      // TODO: future method types
      // if (['mokapot', 'moka pot', 'moka', 'bialetti'].includes(lower)) return 'mokapot';
      // if (['coldbrew', 'cold brew', 'cold drip'].includes(lower)) return 'coldbrew';
      // if (lower === 'cupping') return 'cupping';

      // Direct match on canonical values
      if (['pourover', 'espresso', 'immersion', 'mokapot', 'coldbrew', 'cupping', 'other'].includes(lower)) return lower;

      return '';
    },
```

**Step 3: Update init to set brewerCategory on load**

In the `init()` method, where it already calls `this.onBrewerChange(sel.value)` in the `$nextTick`, this will now also set `brewerCategory`. No change needed — the existing code already calls `onBrewerChange` which now sets the category.

**Step 4: Update applyRecipe to set brewerCategory**

In the `applyRecipe` method, after `this.setFormField(form, 'brewer_rkey', recipe.brewer_rkey || '');`, add:

```js
        // Update brewer category from recipe's brewer
        if (recipe.brewer_rkey) {
          this.onBrewerChange(recipe.brewer_rkey);
        }
```

**Step 5: Bump the version query param**

In `internal/web/components/layout.templ`, find the brew-form.js script tag and bump its version:

```html
<script src="/static/js/brew-form.js?v=0.4.0" defer></script>
```

(Find the current version and increment it.)

**Step 6: Commit**

```
feat: add brewer category detection to Alpine.js brew form for conditional field display
```

---

## Task 10: Display Method Params on Brew View Page

**Files:**
- Modify: `internal/web/pages/brew_view.templ`
- Modify: `internal/web/components/icons.templ` (if new icons needed)

**Step 1: Add display sections for method params**

In `BrewParametersGrid`, after the brew time field and before the closing `</div>`, add conditional sections:

```templ
		if brew.EspressoParams != nil {
			if brew.EspressoParams.YieldWeight > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconScale(), Label: "Yield", Value: fmt.Sprintf("%.1fg", brew.EspressoParams.YieldWeight)})
			}
			if brew.EspressoParams.Pressure > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconBarChart(), Label: "Pressure", Value: fmt.Sprintf("%.1f bar", brew.EspressoParams.Pressure)})
			}
			if brew.EspressoParams.PreInfusionSeconds > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconClock(), Label: "Pre-infusion", Value: fmt.Sprintf("%ds", brew.EspressoParams.PreInfusionSeconds)})
			}
		}
		if brew.PouroverParams != nil {
			if brew.PouroverParams.BloomWater > 0 || brew.PouroverParams.BloomSeconds > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconDroplet(), Label: "Bloom", Value: formatBloom(brew.PouroverParams)})
			}
			if brew.PouroverParams.DrawdownSeconds > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconClock(), Label: "Drawdown", Value: fmt.Sprintf("%ds", brew.PouroverParams.DrawdownSeconds)})
			}
			if brew.PouroverParams.BypassWater > 0 {
				@components.DetailField(components.DetailFieldProps{Icon: components.IconDroplet(), Label: "Bypass Water", Value: fmt.Sprintf("%dg", brew.PouroverParams.BypassWater)})
			}
		}
```

**Step 2: Add formatBloom helper**

```go
func formatBloom(pp *models.PouroverParams) string {
	if pp.BloomWater > 0 && pp.BloomSeconds > 0 {
		return fmt.Sprintf("%dg for %ds", pp.BloomWater, pp.BloomSeconds)
	}
	if pp.BloomWater > 0 {
		return fmt.Sprintf("%dg", pp.BloomWater)
	}
	if pp.BloomSeconds > 0 {
		return fmt.Sprintf("%ds", pp.BloomSeconds)
	}
	return ""
}
```

**Step 3: Run templ generate and verify**

```bash
templ generate
go vet ./...
go build ./...
```

**Step 4: Commit**

```
feat: display espresso and pourover params on brew view page
```

---

## Task 11: Update Feed Display for Method Params (Optional)

**Files:**
- Modify: `internal/web/pages/feed.templ`

This is optional but nice to have — show a small badge or extra detail on feed cards when espresso/pourover params are present.

**Step 1: Check if feed cards already show brew params**

If feed cards show brew variables (coffee, water, temp), consider adding yield weight for espresso brews inline. This is a small cosmetic addition.

**Step 2: Commit if changed**

```
feat: show method-specific params in feed cards
```

---

## Task 12: Rebuild CSS and Final Verification

**Files:**
- None new

**Step 1: Rebuild Tailwind CSS**

```bash
just style
```

**Step 2: Run full test suite**

```bash
go test ./...
```

**Step 3: Run go vet**

```bash
go vet ./...
```

**Step 4: Bump CSS version for cache busting**

In `internal/web/components/layout.templ`, bump the CSS version query parameter.

**Step 5: Final commit**

```
chore: rebuild CSS and bump cache versions
```

---

## Summary of Changes

| Area | What changes |
|---|---|
| **Lexicons** | `brewer.json` gets `knownValues`, `brew.json` gets `espressoParams` and `pouroverParams` sub-objects |
| **Models** | Brewer type constants, `NormalizeBrewerType()`, `EspressoParams`/`PouroverParams` structs, new fields on `Brew` and `CreateBrewRequest` |
| **Records** | Serialization/deserialization for new params in `BrewToRecord`/`RecordToBrew` |
| **Handlers** | `parseEspressoParams()`/`parsePouroverParams()` helpers, wired into create/update |
| **Store** | Pass-through of new params |
| **Brew form** | New `EspressoParamsSection` and `PouroverParamsSection` templ components, conditional on `brewerCategory` |
| **Alpine.js** | `brewerCategory` state, `normalizeBrewerCategory()` method, updated `onBrewerChange()` |
| **Brewer form** | Free text → select dropdown with known values + "Custom..." option |
| **Brew view** | Conditional display of method-specific params in parameters grid |

### Future TODOs left in code

- `normalizeBrewerCategory()` in JS has commented-out cases for `mokapot`, `coldbrew`, `cupping`
- Each future method type needs: a form section component, a handler parser, and view display logic
- The pattern is established — copy `EspressoParamsSection`/`parseEspressoParams` as templates
