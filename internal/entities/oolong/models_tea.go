package oolong

import "time"

const (
	CategoryGreen    = "green"
	CategoryYellow   = "yellow"
	CategoryWhite    = "white"
	CategoryOolong   = "oolong"
	CategoryRed      = "red"
	CategoryBlack    = "black"
	CategoryDark     = "dark"
	CategoryFlavored = "flavored"
	CategoryBlend    = "blend"
	CategoryOther    = "other"
)

var CategoryKnownValues = []string{
	CategoryGreen, CategoryYellow, CategoryWhite, CategoryOolong,
	CategoryRed, CategoryBlack, CategoryDark, CategoryFlavored, CategoryBlend, CategoryOther,
}

var CategoryLabels = map[string]string{
	CategoryGreen:    "Green",
	CategoryYellow:   "Yellow",
	CategoryWhite:    "White",
	CategoryOolong:   "Oolong",
	CategoryRed:      "Red",
	CategoryBlack:    "Black",
	CategoryDark:     "Dark / Fermented",
	CategoryFlavored: "Flavored",
	CategoryBlend:    "Blend",
	CategoryOther:    "Other",
}

type Tea struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Origin      string    `json:"origin"`
	HarvestYear int       `json:"harvest_year,omitempty"`
	Description string    `json:"description"`
	VendorRKey  string    `json:"vendor_rkey,omitempty"`
	Rating      *int      `json:"rating,omitempty"`
	Closed      bool      `json:"closed"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Joined data for display
	Vendor *Vendor `json:"vendor,omitempty"`
}

type CreateTeaRequest struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Origin      string `json:"origin"`
	HarvestYear int    `json:"harvest_year,omitempty"`
	Description string `json:"description"`
	VendorRKey  string `json:"vendor_rkey,omitempty"`
	Rating      *int   `json:"rating,omitempty"`
	Closed      bool   `json:"closed"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateTeaRequest CreateTeaRequest

func (r *CreateTeaRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Category) > MaxCategoryLength {
		return ErrFieldTooLong
	}
	if len(r.Origin) > MaxOriginLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}

func (r *UpdateTeaRequest) Validate() error {
	c := CreateTeaRequest(*r)
	return c.Validate()
}

// IsIncomplete returns true if the tea is missing key classification fields.
func (t *Tea) IsIncomplete() bool {
	return t.Category == "" || t.Origin == ""
}

func (t *Tea) MissingFields() []string {
	var missing []string
	if t.Category == "" {
		missing = append(missing, "category")
	}
	if t.Origin == "" {
		missing = append(missing, "origin")
	}
	return missing
}

func isKnownValue(s string, known []string) bool {
	for _, k := range known {
		if s == k {
			return true
		}
	}
	return false
}
