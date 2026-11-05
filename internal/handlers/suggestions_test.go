package handlers

import (
	"testing"

	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/entities"

	"github.com/stretchr/testify/assert"
)

func TestNSIDForEntityUsesActiveAppDescriptors(t *testing.T) {
	h := &Handler{}
	h.SetApp(&arabicaapp.App{
		Name: "arabica",
		Descriptors: []*entities.Descriptor{
			{Type: "brewer", NSID: "social.arabica.alpha.brewer"},
		},
		EntityRoutes: []arabicaapp.EntityRoute{
			{Type: "brewer", Path: "brewers", Noun: "brewer"},
		},
	})

	assert.Equal(t, "social.arabica.alpha.brewer", h.nsidForEntity("brewers"))
}

func TestNSIDForEntityRejectsUnknownOrUnconfiguredPaths(t *testing.T) {
	h := &Handler{}

	assert.Empty(t, h.nsidForEntity("brewers"))

	h.SetApp(&arabicaapp.App{
		Name: "arabica",
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
		},
		EntityRoutes: []arabicaapp.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
		},
	})

	assert.Empty(t, h.nsidForEntity("roasters"))
}
