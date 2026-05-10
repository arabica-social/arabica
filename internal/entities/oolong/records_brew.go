package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// BrewToRecord converts a Brew to an atproto record map.
// teaURI is required; brewerURI and recipeURI are optional.
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
// a map carrying $type plus the params fields. Always emits the canonical
// brew#... $type even when used by recipe; the decoder accepts both forms.
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
// Accepts both brew#... and recipe#... forms (the canonical home is brew, but
// recipe declares the same shapes locally).
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
	// Accept both []any (post-JSON) and []map[string]any (in-memory) for steeps.
	switch raw := m["steeps"].(type) {
	case []any:
		p.Steeps = decodeSteepsFromAny(raw)
	case []map[string]any:
		p.Steeps = decodeSteepsFromMaps(raw)
	}
	if v, ok := toFloat64(m["totalSteeps"]); ok {
		p.TotalSteeps = int(v)
	}
	return p
}

func decodeSteepsFromAny(raw []any) []Steep {
	out := make([]Steep, 0, len(raw))
	for _, item := range raw {
		sm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, decodeSteep(sm))
	}
	return out
}

func decodeSteepsFromMaps(raw []map[string]any) []Steep {
	out := make([]Steep, 0, len(raw))
	for _, sm := range raw {
		out = append(out, decodeSteep(sm))
	}
	return out
}

func decodeSteep(sm map[string]any) Steep {
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
	return s
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
	switch raw := m["ingredients"].(type) {
	case []any:
		p.Ingredients = decodeIngredientsFromAny(raw)
	case []map[string]any:
		p.Ingredients = decodeIngredientsFromMaps(raw)
	}
	if b, ok := m["iced"].(bool); ok {
		p.Iced = b
	}
	return p
}

func decodeIngredientsFromAny(raw []any) []Ingredient {
	out := make([]Ingredient, 0, len(raw))
	for _, item := range raw {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if ing, ok := decodeIngredient(im); ok {
			out = append(out, ing)
		}
	}
	return out
}

func decodeIngredientsFromMaps(raw []map[string]any) []Ingredient {
	out := make([]Ingredient, 0, len(raw))
	for _, im := range raw {
		if ing, ok := decodeIngredient(im); ok {
			out = append(out, ing)
		}
	}
	return out
}

func decodeIngredient(im map[string]any) (Ingredient, bool) {
	name, _ := im["name"].(string)
	if name == "" {
		return Ingredient{}, false
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
	return ing, true
}
