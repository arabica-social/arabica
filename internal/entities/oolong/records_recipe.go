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
