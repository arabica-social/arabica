package oolong

import (
	"strings"
	"time"
)

const (
	VesselStyleTeapot     = "teapot"
	VesselStyleMug        = "mug"
	VesselStyleJar        = "jar"
	VesselStyleMatchaBowl = "matcha-bowl"
	VesselStyleOther      = "other"
)

var VesselStyleKnownValues = []string{
	VesselStyleTeapot,
	VesselStyleMug,
	VesselStyleJar,
	VesselStyleMatchaBowl,
	VesselStyleOther,
}

var VesselStyleLabels = map[string]string{
	VesselStyleTeapot:     "Teapot",
	VesselStyleMug:        "Mug",
	VesselStyleJar:        "Jar",
	VesselStyleMatchaBowl: "Matcha Bowl",
	VesselStyleOther:      "Other",
}

// NormalizeVesselStyle maps freeform strings to canonical values.
// Returns the input unchanged for unknown values.
func NormalizeVesselStyle(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch lower {
	case "teapot", "yixing", "zisha", "gaiwan", "kyusu", "kyuusu":
		return VesselStyleTeapot
	case "mug", "cup":
		return VesselStyleMug
	case "jar", "pitcher", "carafe", "bottle", "thermos":
		return VesselStyleJar
	case "matcha-bowl", "matcha bowl", "chawan":
		return VesselStyleMatchaBowl
	case "other":
		return VesselStyleOther
	default:
		return raw
	}
}

type Vessel struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Style       string    `json:"style"`
	CapacityMl  int       `json:"capacity_ml"`
	Material    string    `json:"material"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateVesselRequest struct {
	Name        string `json:"name"`
	Style       string `json:"style"`
	CapacityMl  int    `json:"capacity_ml"`
	Material    string `json:"material"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateVesselRequest CreateVesselRequest

func (r *CreateVesselRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Style) > MaxVesselStyleLength {
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

func (r *UpdateVesselRequest) Validate() error {
	c := CreateVesselRequest(*r)
	return c.Validate()
}
