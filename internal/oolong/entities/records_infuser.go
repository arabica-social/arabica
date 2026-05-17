package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func InfuserToRecord(i *Infuser) (map[string]any, error) {
	if i.Name == "" {
		return nil, ErrNameRequired
	}
	rec := map[string]any{
		"$type":     NSIDInfuser,
		"name":      i.Name,
		"createdAt": i.CreatedAt.Format(time.RFC3339),
	}
	if i.Style != "" {
		rec["style"] = i.Style
	}
	if i.Material != "" {
		rec["material"] = i.Material
	}
	if i.Description != "" {
		rec["description"] = i.Description
	}
	if i.SourceRef != "" {
		rec["sourceRef"] = i.SourceRef
	}
	return rec, nil
}

func RecordToInfuser(record map[string]any, atURI string) (*Infuser, error) {
	i := &Infuser{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		i.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	i.Name = name

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	i.CreatedAt = t

	if s, ok := record["style"].(string); ok {
		i.Style = s
	}
	if s, ok := record["material"].(string); ok {
		i.Material = s
	}
	if s, ok := record["description"].(string); ok {
		i.Description = s
	}
	if s, ok := record["sourceRef"].(string); ok {
		i.SourceRef = s
	}
	return i, nil
}
