package oolong

import "time"

type Recipe struct {
	RKey         string       `json:"rkey"`
	Name         string       `json:"name"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	Style        string       `json:"style,omitempty"`
	TeaRKey      string       `json:"tea_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	SourceRef    string       `json:"source_ref,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`

	// Joined data for display
	BrewerObj *Brewer `json:"brewer_obj,omitempty"`
	Tea       *Tea    `json:"tea,omitempty"`

	// Derived display fields, populated by handlers
	AuthorDID     string `json:"author_did,omitempty"`
	AuthorHandle  string `json:"author_handle,omitempty"`
	AuthorAvatar  string `json:"author_avatar,omitempty"`
	AuthorDisplay string `json:"author_display,omitempty"`

	SourceAuthorHandle  string `json:"source_author_handle,omitempty"`
	SourceAuthorAvatar  string `json:"source_author_avatar,omitempty"`
	SourceAuthorDisplay string `json:"source_author_display,omitempty"`

	ForkCount     int      `json:"fork_count,omitempty"`
	BrewCount     int      `json:"brew_count,omitempty"`
	ForkerAvatars []string `json:"forker_avatars,omitempty"`
}

type CreateRecipeRequest struct {
	Name         string       `json:"name"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	Style        string       `json:"style,omitempty"`
	TeaRKey      string       `json:"tea_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	SourceRef    string       `json:"source_ref,omitempty"`
}

type UpdateRecipeRequest CreateRecipeRequest

func (r *CreateRecipeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if r.Style != "" && !isKnownValue(r.Style, StyleKnownValues) {
		return ErrStyleInvalid
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrFieldTooLong
	}
	return nil
}

func (r *UpdateRecipeRequest) Validate() error {
	c := CreateRecipeRequest(*r)
	return c.Validate()
}
