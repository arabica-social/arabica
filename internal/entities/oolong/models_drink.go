package oolong

import "time"

type Drink struct {
	RKey          string    `json:"rkey"`
	CafeRKey      string    `json:"cafe_rkey"`
	TeaRKey       string    `json:"tea_rkey,omitempty"`
	Name          string    `json:"name"`
	Style         string    `json:"style"`
	Description   string    `json:"description"`
	Rating        int       `json:"rating,omitempty"`
	TastingNotes  string    `json:"tasting_notes"`
	PriceUsdCents int       `json:"price_usd_cents,omitempty"`
	SourceRef     string    `json:"source_ref,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	// Joined data for display
	Cafe *Cafe `json:"cafe,omitempty"`
	Tea  *Tea  `json:"tea,omitempty"`
}

type CreateDrinkRequest struct {
	CafeRKey      string `json:"cafe_rkey"`
	TeaRKey       string `json:"tea_rkey,omitempty"`
	Name          string `json:"name"`
	Style         string `json:"style"`
	Description   string `json:"description"`
	Rating        int    `json:"rating,omitempty"`
	TastingNotes  string `json:"tasting_notes"`
	PriceUsdCents int    `json:"price_usd_cents,omitempty"`
	SourceRef     string `json:"source_ref,omitempty"`
}

type UpdateDrinkRequest CreateDrinkRequest

func (r *CreateDrinkRequest) Validate() error {
	if r.CafeRKey == "" {
		return ErrCafeRefRequired
	}
	if len(r.Name) > MaxMenuItemLength {
		return ErrFieldTooLong
	}
	if len(r.Style) > MaxStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > 500 {
		return ErrDescTooLong
	}
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	if r.Rating != 0 && (r.Rating < 1 || r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}

func (r *UpdateDrinkRequest) Validate() error {
	c := CreateDrinkRequest(*r)
	return c.Validate()
}
