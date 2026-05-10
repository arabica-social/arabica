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
