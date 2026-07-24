package handlers

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// IncompleteRecord represents a single entity that needs attention (missing
// required fields). Populated by CollectIncompleteRecords and serialized by
// the onboarding/incomplete-records JSON handlers.
type IncompleteRecord struct {
	EntityType    string
	RKey          string
	Name          string
	MissingFields []string
}

// CollectIncompleteRecords scans all entities and returns incomplete ones
// (max limit). Used by the live JSON handlers that serve the onboarding and
// incomplete-records endpoints.
func CollectIncompleteRecords(beans []*arabica.Bean, grinders []*arabica.Grinder, brewers []*arabica.Brewer, limit int) []IncompleteRecord {
	var records []IncompleteRecord

	for _, b := range beans {
		if b.IsIncomplete() && !b.Closed {
			records = append(records, IncompleteRecord{
				EntityType:    "bean",
				RKey:          b.RKey,
				Name:          b.Name,
				MissingFields: b.MissingFields(),
			})
		}
	}
	for _, g := range grinders {
		if g.IsIncomplete() {
			records = append(records, IncompleteRecord{
				EntityType:    "grinder",
				RKey:          g.RKey,
				Name:          g.Name,
				MissingFields: g.MissingFields(),
			})
		}
	}
	for _, b := range brewers {
		if b.IsIncomplete() {
			records = append(records, IncompleteRecord{
				EntityType:    "brewer",
				RKey:          b.RKey,
				Name:          b.Name,
				MissingFields: b.MissingFields(),
			})
		}
	}

	if limit > 0 && len(records) > limit {
		return records[:limit]
	}
	return records
}
