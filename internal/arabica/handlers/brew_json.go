package coffeehandlers

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// BrewMutationJSONResponse is the JSON envelope returned by brew create/update
// handlers when the client requests JSON (Accept: application/json or no
// __redirect form value). The brew model carries json tags so it serializes
// directly; incomplete_nudge is populated when the referenced bean is missing
// fields, mirroring the X-Incomplete-Nudge header the HTMX path sets.
type BrewMutationJSONResponse struct {
	Brew            *arabica.Brew `json:"brew"`
	IncompleteNudge *BeanNudge    `json:"incomplete_nudge,omitempty"`
}

// BeanNudge describes an incomplete bean reference so the SPA can prompt the
// user to fill in missing fields after creating/updating a brew.
type BeanNudge struct {
	EntityType    string `json:"entity_type"`
	RKey          string `json:"rkey"`
	Name          string `json:"name"`
	MissingFields string `json:"missing"`
}
