package oolong

import "time"

const (
	StyleLongSteep = "longSteep"
	StyleColdBrew  = "coldBrew"

	// Deprecated: dropped from the lexicon in the alpha trim-down. Kept
	// only so existing references and historical tests still compile.
	StyleGongfu  = "gongfu"
	StyleMatcha  = "matcha"
	StyleMilkTea = "milkTea"
)

var StyleKnownValues = []string{StyleLongSteep, StyleColdBrew}

var StyleLabels = map[string]string{
	StyleLongSteep: "Long Steep",
	StyleColdBrew:  "Cold Brew",
}

const (
	MatchaPrepUsucha = "usucha"
	MatchaPrepKoicha = "koicha"
	MatchaPrepIced   = "iced"
	MatchaPrepOther  = "other"
)

var MatchaPreparationKnownValues = []string{
	MatchaPrepUsucha, MatchaPrepKoicha, MatchaPrepIced, MatchaPrepOther,
}

const (
	IngredientUnitG     = "g"
	IngredientUnitMl    = "ml"
	IngredientUnitTsp   = "tsp"
	IngredientUnitTbsp  = "tbsp"
	IngredientUnitCup   = "cup"
	IngredientUnitPcs   = "pcs"
	IngredientUnitOther = "other"
)

var IngredientUnitKnownValues = []string{
	IngredientUnitG, IngredientUnitMl, IngredientUnitTsp, IngredientUnitTbsp,
	IngredientUnitCup, IngredientUnitPcs, IngredientUnitOther,
}

// MethodParams is a sealed union of style-specific brewing parameters. One of
// *GongfuParams, *MatchaParams, *MilkTeaParams. Nil for longSteep style.
type MethodParams interface {
	isMethodParams()
	atprotoTypeName() string
}

type GongfuParams struct {
	Rinse        bool    `json:"rinse"`
	RinseSeconds int     `json:"rinse_seconds"`
	Steeps       []Steep `json:"steeps,omitempty"`
	TotalSteeps  int     `json:"total_steeps,omitempty"`
}

func (*GongfuParams) isMethodParams()         {}
func (*GongfuParams) atprotoTypeName() string { return NSIDBrew + "#gongfuParams" }

// Steep represents one steep in a gong fu session. Temperature in tenths of °C
// (0 means "use brew.Temperature default"). Rating 0 means "no rating given".
type Steep struct {
	Number       int    `json:"number"`
	TimeSeconds  int    `json:"time_seconds"`
	Temperature  int    `json:"temperature,omitempty"`
	TastingNotes string `json:"tasting_notes,omitempty"`
	Rating       int    `json:"rating,omitempty"`
}

type MatchaParams struct {
	Preparation string `json:"preparation"`
	Sieved      bool   `json:"sieved"`
	WhiskType   string `json:"whisk_type"`
	WaterMl     int    `json:"water_ml"`
}

func (*MatchaParams) isMethodParams()         {}
func (*MatchaParams) atprotoTypeName() string { return NSIDBrew + "#matchaParams" }

type MilkTeaParams struct {
	Preparation string       `json:"preparation"`
	Ingredients []Ingredient `json:"ingredients,omitempty"`
	Iced        bool         `json:"iced"`
}

func (*MilkTeaParams) isMethodParams()         {}
func (*MilkTeaParams) atprotoTypeName() string { return NSIDBrew + "#milkTeaParams" }

// Ingredient on a milk tea / tea-based beverage. Amount is in float units;
// the lexicon stores tenths of `unit` as integer; conversion happens at the
// record boundary.
type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	Notes  string  `json:"notes,omitempty"`
}

type Brew struct {
	RKey         string       `json:"rkey"`
	TeaRKey      string       `json:"tea_rkey"`
	Style        string       `json:"style"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	RecipeRKey   string       `json:"recipe_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	TastingNotes string       `json:"tasting_notes,omitempty"`
	Rating       int          `json:"rating,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`

	// Joined data for display
	Tea       *Tea    `json:"tea,omitempty"`
	BrewerObj *Brewer `json:"brewer_obj,omitempty"`
	RecipeObj *Recipe `json:"recipe_obj,omitempty"`
}

type CreateBrewRequest struct {
	TeaRKey      string       `json:"tea_rkey"`
	Style        string       `json:"style"`
	BrewerRKey   string       `json:"brewer_rkey,omitempty"`
	RecipeRKey   string       `json:"recipe_rkey,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	LeafGrams    float64      `json:"leaf_grams,omitempty"`
	VesselMl     int          `json:"vessel_ml,omitempty"`
	TimeSeconds  int          `json:"time_seconds,omitempty"`
	TastingNotes string       `json:"tasting_notes,omitempty"`
	Rating       int          `json:"rating,omitempty"`
	MethodParams MethodParams `json:"method_params,omitempty"`
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
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	if r.Rating != 0 && (r.Rating < 1 || r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	// methodParams alignment with style
	switch r.Style {
	case StyleGongfu:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*GongfuParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleMatcha:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*MatchaParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleMilkTea:
		if r.MethodParams != nil {
			if _, ok := r.MethodParams.(*MilkTeaParams); !ok {
				return ErrStyleInvalid
			}
		}
	case StyleLongSteep:
		if r.MethodParams != nil {
			return ErrStyleInvalid // longSteep takes no params
		}
	}
	return nil
}
