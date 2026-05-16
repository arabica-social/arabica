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
	if t.Origin != "" {
		rec["origin"] = t.Origin
	}
	if t.HarvestYear > 0 {
		rec["harvestYear"] = t.HarvestYear
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
	if s, ok := record["origin"].(string); ok {
		t.Origin = s
	}
	if v, ok := toFloat64(record["harvestYear"]); ok {
		t.HarvestYear = int(v)
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
