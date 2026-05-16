package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func VesselToRecord(v *Vessel) (map[string]any, error) {
	if v.Name == "" {
		return nil, ErrNameRequired
	}
	rec := map[string]any{
		"$type":     NSIDVessel,
		"name":      v.Name,
		"createdAt": v.CreatedAt.Format(time.RFC3339),
	}
	if v.Style != "" {
		rec["style"] = v.Style
	}
	if v.CapacityMl > 0 {
		rec["capacityMl"] = v.CapacityMl
	}
	if v.Material != "" {
		rec["material"] = v.Material
	}
	if v.Description != "" {
		rec["description"] = v.Description
	}
	if v.SourceRef != "" {
		rec["sourceRef"] = v.SourceRef
	}
	return rec, nil
}

func RecordToVessel(record map[string]any, atURI string) (*Vessel, error) {
	v := &Vessel{}
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

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	v.CreatedAt = t

	if s, ok := record["style"].(string); ok {
		v.Style = s
	}
	if n, ok := toFloat64(record["capacityMl"]); ok {
		v.CapacityMl = int(n)
	}
	if s, ok := record["material"].(string); ok {
		v.Material = s
	}
	if s, ok := record["description"].(string); ok {
		v.Description = s
	}
	if s, ok := record["sourceRef"].(string); ok {
		v.SourceRef = s
	}
	return v, nil
}
