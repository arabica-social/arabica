package oolong

import "time"

const (
	StyleLongSteep = "longSteep"
	StyleColdBrew  = "coldBrew"
)

var StyleKnownValues = []string{StyleLongSteep, StyleColdBrew}

var StyleLabels = map[string]string{
	StyleLongSteep: "Long Steep",
	StyleColdBrew:  "Cold Brew",
}

const (
	InfusionMethodTeaBag    = "tea-bag"
	InfusionMethodLooseLeaf = "loose-leaf"
	InfusionMethodInfuser   = "infuser"
)

var InfusionMethodKnownValues = []string{
	InfusionMethodTeaBag, InfusionMethodLooseLeaf, InfusionMethodInfuser,
}

var InfusionMethodLabels = map[string]string{
	InfusionMethodTeaBag:    "Tea Bag",
	InfusionMethodLooseLeaf: "Loose Leaf",
	InfusionMethodInfuser:   "Infuser",
}

type Brew struct {
	RKey            string    `json:"rkey"`
	TeaRKey         string    `json:"tea_rkey"`
	Style           string    `json:"style"`
	VesselRKey      string    `json:"vessel_rkey,omitempty"`
	InfusionMethod  string    `json:"infusion_method,omitempty"`
	InfuserRKey     string    `json:"infuser_rkey,omitempty"`
	Temperature     float64   `json:"temperature,omitempty"`
	LeafGrams       float64   `json:"leaf_grams,omitempty"`
	WaterAmount     int       `json:"water_amount,omitempty"`
	TimeSeconds     int       `json:"time_seconds,omitempty"`
	TastingNotes    string    `json:"tasting_notes,omitempty"`
	Rating          int       `json:"rating,omitempty"`
	CreatedAt       time.Time `json:"created_at"`

	// Joined data for display
	Tea     *Tea     `json:"tea,omitempty"`
	Vessel  *Vessel  `json:"vessel_obj,omitempty"`
	Infuser *Infuser `json:"infuser_obj,omitempty"`
}

type CreateBrewRequest struct {
	TeaRKey        string  `json:"tea_rkey"`
	Style          string  `json:"style"`
	VesselRKey     string  `json:"vessel_rkey,omitempty"`
	InfusionMethod string  `json:"infusion_method,omitempty"`
	InfuserRKey    string  `json:"infuser_rkey,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	LeafGrams      float64 `json:"leaf_grams,omitempty"`
	WaterAmount    int     `json:"water_amount,omitempty"`
	TimeSeconds    int     `json:"time_seconds,omitempty"`
	TastingNotes   string  `json:"tasting_notes,omitempty"`
	Rating         int     `json:"rating,omitempty"`
}

func (r *CreateBrewRequest) Validate() error {
	if r.TeaRKey == "" {
		return ErrTeaRefRequired
	}
	if r.Style == "" {
		return ErrStyleRequired
	}
	if !isKnownValue(r.Style, StyleKnownValues) {
		return ErrStyleInvalid
	}
	if r.InfusionMethod != "" && !isKnownValue(r.InfusionMethod, InfusionMethodKnownValues) {
		return ErrInfusionMethodInvalid
	}
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	if r.Rating != 0 && (r.Rating < 1 || r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}
