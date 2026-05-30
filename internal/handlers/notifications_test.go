package handlers

import (
	"testing"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"

	"github.com/stretchr/testify/assert"
)

func TestResolveNotificationLinkUsesActiveAppDescriptor(t *testing.T) {
	app := &domain.App{
		Descriptors: []*entities.Descriptor{
			{Type: "oolong-tea", NSID: "social.oolong.alpha.tea"},
		},
		EntityRoutes: []domain.EntityRoute{
			{Type: "oolong-tea", Path: "teas", Noun: "tea"},
		},
	}

	link := resolveNotificationLink(app, "at://did:plc:alice/social.oolong.alpha.tea/3abc")

	assert.Equal(t, "/teas/did:plc:alice/3abc", link)
}

func TestResolveNotificationLinkRejectsUnknownCollections(t *testing.T) {
	app := &domain.App{
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
		},
		EntityRoutes: []domain.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
		},
	}

	assert.Empty(t, resolveNotificationLink(app, "at://did:plc:alice/social.oolong.alpha.tea/3abc"))
	assert.Empty(t, resolveNotificationLink(app, "not-an-at-uri"))
}

func TestResolveNotificationEntityNameUsesDescriptorNounWithFallback(t *testing.T) {
	app := &domain.App{
		Descriptors: []*entities.Descriptor{
			{Type: "bean", NSID: "social.arabica.alpha.bean"},
			{Type: "recipe", NSID: "social.arabica.alpha.recipe", DisplayName: "Recipe"},
		},
		EntityRoutes: []domain.EntityRoute{
			{Type: "bean", Path: "beans", Noun: "bean"},
			{Type: "recipe", Path: "recipes"},
		},
	}

	assert.Equal(t, "bean", resolveNotificationEntityName(app, "at://did:plc:alice/social.arabica.alpha.bean/3abc"))
	assert.Equal(t, "recipe", resolveNotificationEntityName(app, "at://did:plc:alice/social.arabica.alpha.recipe/3abc"))
	assert.Equal(t, "content", resolveNotificationEntityName(app, "at://did:plc:alice/social.oolong.alpha.tea/3abc"))
}
