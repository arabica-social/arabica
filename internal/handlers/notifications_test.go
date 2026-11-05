package handlers

import (
	"testing"

	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/entities"

	"github.com/stretchr/testify/assert"
)

func TestResolveNotificationLinkUsesActiveAppDescriptor(t *testing.T) {
	app := &arabicaapp.App{
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
		},
		EntityRoutes: []arabicaapp.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
		},
	}

	link := resolveNotificationLink(app, "at://did:plc:alice/social.arabica.alpha.bean/3abc")

	assert.Equal(t, "/beans/did:plc:alice/3abc", link)
}

func TestResolveNotificationLinkRejectsUnknownCollections(t *testing.T) {
	app := &arabicaapp.App{
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
		},
		EntityRoutes: []arabicaapp.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
		},
	}

	assert.Empty(t, resolveNotificationLink(app, "at://did:plc:alice/social.other.alpha.tea/3abc"))
	assert.Empty(t, resolveNotificationLink(app, "not-an-at-uri"))
}

func TestResolveNotificationEntityNameUsesDescriptorNounWithFallback(t *testing.T) {
	app := &arabicaapp.App{
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
			{Type: "recipe", NSID: "social.arabica.alpha.recipe", DisplayName: "Recipe"},
		},
		EntityRoutes: []arabicaapp.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
			{Type: "recipe", Path: "recipes"},
		},
	}

	assert.Equal(t, "bean", resolveNotificationEntityName(app, "at://did:plc:alice/social.arabica.alpha.bean/3abc"))
	assert.Equal(t, "recipe", resolveNotificationEntityName(app, "at://did:plc:alice/social.arabica.alpha.recipe/3abc"))
	assert.Equal(t, "content", resolveNotificationEntityName(app, "at://did:plc:alice/social.arabica.alpha.comment/3abc"))
}
