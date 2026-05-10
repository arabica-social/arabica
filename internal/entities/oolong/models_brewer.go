package oolong

import (
	"strings"
	"time"
)

const (
	BrewerStyleGaiwan     = "gaiwan"
	BrewerStyleKyusu      = "kyusu"
	BrewerStyleTeapot     = "teapot"
	BrewerStyleMatchaBowl = "matcha-bowl"
	BrewerStyleInfuser    = "infuser"
	BrewerStyleThermos    = "thermos"
	BrewerStyleOther      = "other"
)

var BrewerStyleKnownValues = []string{
	BrewerStyleGaiwan,
	BrewerStyleKyusu,
	BrewerStyleTeapot,
	BrewerStyleMatchaBowl,
	BrewerStyleInfuser,
	BrewerStyleThermos,
	BrewerStyleOther,
}

var BrewerStyleLabels = map[string]string{
	BrewerStyleGaiwan:     "Gaiwan",
	BrewerStyleKyusu:      "Kyusu",
	BrewerStyleTeapot:     "Teapot",
	BrewerStyleMatchaBowl: "Matcha Bowl",
	BrewerStyleInfuser:    "Infuser",
	BrewerStyleThermos:    "Thermos",
	BrewerStyleOther:      "Other",
}

// NormalizeBrewerStyle maps freeform strings to canonical values.
// Returns the input unchanged for unknown values.
func NormalizeBrewerStyle(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch lower {
	case "gaiwan":
		return BrewerStyleGaiwan
	case "kyusu", "kyuusu":
		return BrewerStyleKyusu
	case "teapot", "yixing", "zisha":
		return BrewerStyleTeapot
	case "matcha-bowl", "matcha bowl", "chawan":
		return BrewerStyleMatchaBowl
	case "infuser", "tea bag", "strainer":
		return BrewerStyleInfuser
	case "thermos", "grandpa", "bottle":
		return BrewerStyleThermos
	case "other":
		return BrewerStyleOther
	default:
		return raw
	}
}

type Brewer struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Style       string    `json:"style"`
	CapacityMl  int       `json:"capacity_ml"`
	Material    string    `json:"material"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateBrewerRequest struct {
	Name        string `json:"name"`
	Style       string `json:"style"`
	CapacityMl  int    `json:"capacity_ml"`
	Material    string `json:"material"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateBrewerRequest CreateBrewerRequest

func (r *CreateBrewerRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Style) > MaxBrewerStyleLength {
		return ErrFieldTooLong
	}
	if len(r.Material) > MaxMaterialLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateBrewerRequest) Validate() error {
	c := CreateBrewerRequest(*r)
	return c.Validate()
}
