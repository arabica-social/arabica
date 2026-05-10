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
	// processing may be []any (after JSON decode) or []map[string]any
	// (when constructed in-memory). Handle both.
	switch raw := record["processing"].(type) {
	case []any:
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
	case []map[string]any:
		t.Processing = make([]ProcessingStep, 0, len(raw))
		for _, m := range raw {
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
