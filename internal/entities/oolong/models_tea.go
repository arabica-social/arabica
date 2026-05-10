package oolong

import "time"

const (
	CategoryGreen    = "green"
	CategoryYellow   = "yellow"
	CategoryWhite    = "white"
	CategoryOolong   = "oolong"
	CategoryRed      = "red"
	CategoryDark     = "dark"
	CategoryFlavored = "flavored"
	CategoryBlend    = "blend"
	CategoryOther    = "other"
)

var CategoryKnownValues = []string{
	CategoryGreen, CategoryYellow, CategoryWhite, CategoryOolong,
	CategoryRed, CategoryDark, CategoryFlavored, CategoryBlend, CategoryOther,
}

var CategoryLabels = map[string]string{
	CategoryGreen:    "Green",
	CategoryYellow:   "Yellow",
	CategoryWhite:    "White",
	CategoryOolong:   "Oolong",
	CategoryRed:      "Red / Black",
	CategoryDark:     "Dark / Fermented",
	CategoryFlavored: "Flavored",
	CategoryBlend:    "Blend",
	CategoryOther:    "Other",
}

const (
	ProcessingSteamed    = "steamed"
	ProcessingPanFired   = "pan-fired"
	ProcessingRolled     = "rolled"
	ProcessingOxidized   = "oxidized"
	ProcessingShaded     = "shaded"
	ProcessingRoasted    = "roasted"
	ProcessingAged       = "aged"
	ProcessingFermented  = "fermented"
	ProcessingCompressed = "compressed"
	ProcessingScented    = "scented"
	ProcessingSmoked     = "smoked"
	ProcessingBlended    = "blended"
	ProcessingFlavored   = "flavored"
	ProcessingOther      = "other"
)

var ProcessingKnownValues = []string{
	ProcessingSteamed, ProcessingPanFired, ProcessingRolled, ProcessingOxidized,
	ProcessingShaded, ProcessingRoasted, ProcessingAged, ProcessingFermented,
	ProcessingCompressed, ProcessingScented, ProcessingSmoked, ProcessingBlended,
	ProcessingFlavored, ProcessingOther,
}

type ProcessingStep struct {
	Step   string `json:"step"`
	Detail string `json:"detail,omitempty"`
}

type Tea struct {
	RKey        string           `json:"rkey"`
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	SubStyle    string           `json:"sub_style"`
	Origin      string           `json:"origin"`
	Cultivar    string           `json:"cultivar"`
	CultivarRef string           `json:"cultivar_ref,omitempty"`
	Farm        string           `json:"farm"`
	FarmRef     string           `json:"farm_ref,omitempty"`
	HarvestYear int              `json:"harvest_year,omitempty"`
	Processing  []ProcessingStep `json:"processing,omitempty"`
	Description string           `json:"description"`
	VendorRKey  string           `json:"vendor_rkey,omitempty"`
	Rating      *int             `json:"rating,omitempty"`
	Closed      bool             `json:"closed"`
	SourceRef   string           `json:"source_ref,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`

	// Joined data for display
	Vendor *Vendor `json:"vendor,omitempty"`
}

type CreateTeaRequest struct {
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	SubStyle    string           `json:"sub_style"`
	Origin      string           `json:"origin"`
	Cultivar    string           `json:"cultivar"`
	Farm        string           `json:"farm"`
	HarvestYear int              `json:"harvest_year,omitempty"`
	Processing  []ProcessingStep `json:"processing,omitempty"`
	Description string           `json:"description"`
	VendorRKey  string           `json:"vendor_rkey,omitempty"`
	Rating      *int             `json:"rating,omitempty"`
	Closed      bool             `json:"closed"`
	SourceRef   string           `json:"source_ref,omitempty"`
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
	// Note: category is an open enum — unknown values are preserved (mirrors arabica's pattern).
	if len(r.SubStyle) > MaxSubStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Origin) > MaxOriginLength {
		return ErrFieldTooLong
	}
	if len(r.Cultivar) > MaxCultivarLength {
		return ErrFieldTooLong
	}
	if len(r.Farm) > MaxFarmLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	for _, p := range r.Processing {
		if len(p.Step) > MaxStepLength {
			return ErrFieldTooLong
		}
		if len(p.Detail) > MaxStepDetailLength {
			return ErrFieldTooLong
		}
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
