package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func CafeToRecord(c *Cafe, vendorURI string) (map[string]any, error) {
	if c.Name == "" {
		return nil, ErrNameRequired
	}
	record := map[string]any{
		"$type":     NSIDCafe,
		"name":      c.Name,
		"createdAt": c.CreatedAt.Format(time.RFC3339),
	}
	if c.Location != "" {
		record["location"] = c.Location
	}
	if c.Address != "" {
		record["address"] = c.Address
	}
	if c.Website != "" {
		record["website"] = c.Website
	}
	if c.Description != "" {
		record["description"] = c.Description
	}
	if vendorURI != "" {
		record["vendorRef"] = vendorURI
	}
	if c.SourceRef != "" {
		record["sourceRef"] = c.SourceRef
	}
	return record, nil
}

func RecordToCafe(record map[string]any, atURI string) (*Cafe, error) {
	c := &Cafe{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		c.RKey = parsed.RecordKey().String()
	}
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, ErrNameRequired
	}
	c.Name = name

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	c.CreatedAt = t

	if s, ok := record["location"].(string); ok {
		c.Location = s
	}
	if s, ok := record["address"].(string); ok {
		c.Address = s
	}
	if s, ok := record["website"].(string); ok {
		c.Website = s
	}
	if s, ok := record["description"].(string); ok {
		c.Description = s
	}
	if s, ok := record["vendorRef"].(string); ok && s != "" {
		parsed, err := syntax.ParseATURI(s)
		if err == nil {
			c.VendorRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["sourceRef"].(string); ok {
		c.SourceRef = s
	}
	return c, nil
}
