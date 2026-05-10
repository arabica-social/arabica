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
