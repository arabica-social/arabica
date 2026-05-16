package oolong

import (
	"strings"
	"time"
)

const (
	InfuserStyleBasket = "basket"
	InfuserStyleBall   = "ball"
	InfuserStyleSock   = "sock"
	InfuserStyleOther  = "other"
)

var InfuserStyleKnownValues = []string{
	InfuserStyleBasket,
	InfuserStyleBall,
	InfuserStyleSock,
	InfuserStyleOther,
}

var InfuserStyleLabels = map[string]string{
	InfuserStyleBasket: "Basket",
	InfuserStyleBall:   "Ball",
	InfuserStyleSock:   "Sock",
	InfuserStyleOther:  "Other",
}

// NormalizeInfuserStyle maps freeform strings to canonical values.
// Returns the input unchanged for unknown values.
func NormalizeInfuserStyle(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch lower {
	case "basket", "strainer", "mesh":
		return InfuserStyleBasket
	case "ball", "tea ball", "tea-ball":
		return InfuserStyleBall
	case "sock", "bag", "cloth":
		return InfuserStyleSock
	case "other":
		return InfuserStyleOther
	default:
		return raw
	}
}

type Infuser struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Style       string    `json:"style"`
	Material    string    `json:"material"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInfuserRequest struct {
	Name        string `json:"name"`
	Style       string `json:"style"`
	Material    string `json:"material"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateInfuserRequest CreateInfuserRequest

func (r *CreateInfuserRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Style) > MaxInfuserStyleLength {
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

func (r *UpdateInfuserRequest) Validate() error {
	c := CreateInfuserRequest(*r)
	return c.Validate()
}
