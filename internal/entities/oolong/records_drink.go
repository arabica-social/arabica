package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func DrinkToRecord(d *Drink, cafeURI, teaURI string) (map[string]any, error) {
	if cafeURI == "" {
		return nil, ErrCafeRefRequired
	}
	rec := map[string]any{
		"$type":     NSIDDrink,
		"cafeRef":   cafeURI,
		"createdAt": d.CreatedAt.Format(time.RFC3339),
	}
	if teaURI != "" {
		rec["teaRef"] = teaURI
	}
	if d.Name != "" {
		rec["name"] = d.Name
	}
	if d.Style != "" {
		rec["style"] = d.Style
	}
	if d.Description != "" {
		rec["description"] = d.Description
	}
	if d.Rating > 0 {
		rec["rating"] = d.Rating
	}
	if d.TastingNotes != "" {
		rec["tastingNotes"] = d.TastingNotes
	}
	if d.PriceUsdCents > 0 {
		rec["priceUsdCents"] = d.PriceUsdCents
	}
	if d.SourceRef != "" {
		rec["sourceRef"] = d.SourceRef
	}
	return rec, nil
}

func RecordToDrink(record map[string]any, atURI string) (*Drink, error) {
	d := &Drink{}
	if atURI != "" {
		parsed, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		d.RKey = parsed.RecordKey().String()
	}
	cafeRef, ok := record["cafeRef"].(string)
	if !ok || cafeRef == "" {
		return nil, ErrCafeRefRequired
	}
	cafeParsed, err := syntax.ParseATURI(cafeRef)
	if err != nil {
		return nil, fmt.Errorf("invalid cafeRef: %w", err)
	}
	d.CafeRKey = cafeParsed.RecordKey().String()

	createdStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt: %w", err)
	}
	d.CreatedAt = t

	if s, ok := record["teaRef"].(string); ok && s != "" {
		teaParsed, err := syntax.ParseATURI(s)
		if err == nil {
			d.TeaRKey = teaParsed.RecordKey().String()
		}
	}
	if s, ok := record["name"].(string); ok {
		d.Name = s
	}
	if s, ok := record["style"].(string); ok {
		d.Style = s
	}
	if s, ok := record["description"].(string); ok {
		d.Description = s
	}
	if v, ok := toFloat64(record["rating"]); ok {
		d.Rating = int(v)
	}
	if s, ok := record["tastingNotes"].(string); ok {
		d.TastingNotes = s
	}
	if v, ok := toFloat64(record["priceUsdCents"]); ok {
		d.PriceUsdCents = int(v)
	}
	if s, ok := record["sourceRef"].(string); ok {
		d.SourceRef = s
	}
	return d, nil
}
