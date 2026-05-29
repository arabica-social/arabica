package handlers

import (
	"testing"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"

	"github.com/stretchr/testify/assert"
)

func TestNSIDForEntityUsesActiveAppDescriptors(t *testing.T) {
	h := &Handler{}
	h.SetApp(&domain.App{
		Name: "oolong",
		Descriptors: []*entities.Descriptor{
			{Type: "oolong-brewer", NSID: "social.oolong.alpha.brewer"},
		},
		EntityRoutes: []domain.EntityRoute{
			{Type: "oolong-brewer", Path: "brewers", Noun: "brewer"},
		},
	})

	assert.Equal(t, "social.oolong.alpha.brewer", h.nsidForEntity("brewers"))
}

func TestNSIDForEntityRejectsUnknownOrUnconfiguredPaths(t *testing.T) {
	h := &Handler{}

	assert.Empty(t, h.nsidForEntity("brewers"))

	h.SetApp(&domain.App{
		Name: "arabica",
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
		},
		EntityRoutes: []domain.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
		},
	})

	assert.Empty(t, h.nsidForEntity("vendors"))
}
