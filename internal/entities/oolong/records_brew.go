package oolong

import (
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// BrewToRecord converts a Brew to an atproto record map.
// teaURI is required; vesselURI and infuserURI are optional.
func BrewToRecord(b *Brew, teaURI, vesselURI, infuserURI string) (map[string]any, error) {
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
	if vesselURI != "" {
		rec["vesselRef"] = vesselURI
	}
	if b.InfusionMethod != "" {
		rec["infusionMethod"] = b.InfusionMethod
	}
	if infuserURI != "" {
		rec["infuserRef"] = infuserURI
	}
	if b.Temperature > 0 {
		rec["temperature"] = int(b.Temperature * 10)
	}
	if b.LeafGrams > 0 {
		rec["leafGrams"] = int(b.LeafGrams * 10)
	}
	if b.WaterAmount > 0 {
		rec["waterAmount"] = b.WaterAmount
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
	return rec, nil
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

	if s, ok := record["vesselRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			b.VesselRKey = parsed.RecordKey().String()
		}
	}
	if s, ok := record["infusionMethod"].(string); ok {
		b.InfusionMethod = s
	}
	if s, ok := record["infuserRef"].(string); ok && s != "" {
		if parsed, err := syntax.ParseATURI(s); err == nil {
			b.InfuserRKey = parsed.RecordKey().String()
		}
	}
	if v, ok := toFloat64(record["temperature"]); ok {
		b.Temperature = v / 10.0
	}
	if v, ok := toFloat64(record["leafGrams"]); ok {
		b.LeafGrams = v / 10.0
	}
	if v, ok := toFloat64(record["waterAmount"]); ok {
		b.WaterAmount = int(v)
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
	return b, nil
}
