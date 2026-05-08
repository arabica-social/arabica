package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestApp_NSIDs_includesDescriptorsAndSocialCollections(t *testing.T) {
	desc := &entities.Descriptor{
		Type:        lexicons.RecordType("test.bean"),
		NSID:        "test.example.bean",
		DisplayName: "Bean",
		Noun:        "bean",
		URLPath:     "beans",
	}
	app := &domain.App{
		Name:        "test",
		NSIDBase:    "test.example",
		Descriptors: []*entities.Descriptor{desc},
	}
	nsids := app.NSIDs()
	assert.Contains(t, nsids, "test.example.bean")
	assert.Contains(t, nsids, "test.example.like")
	assert.Contains(t, nsids, "test.example.comment")
	assert.Len(t, nsids, 3)
}

func TestApp_OAuthScopes_atprotoAndRepoPerNSID(t *testing.T) {
	desc := &entities.Descriptor{
		Type: lexicons.RecordType("test.bean"),
		NSID: "test.example.bean",
	}
	app := &domain.App{
		Name:        "test",
		NSIDBase:    "test.example",
		Descriptors: []*entities.Descriptor{desc},
	}
	scopes := app.OAuthScopes()
	assert.Equal(t, "atproto", scopes[0])
	assert.Contains(t, scopes, "repo:test.example.bean")
	assert.Contains(t, scopes, "repo:test.example.like")
	assert.Contains(t, scopes, "repo:test.example.comment")
	assert.Len(t, scopes, 4)
}

func TestApp_DescriptorByNSID(t *testing.T) {
	bean := &entities.Descriptor{
		Type: lexicons.RecordType("test.bean"),
		NSID: "test.example.bean",
	}
	app := &domain.App{
		NSIDBase:    "test.example",
		Descriptors: []*entities.Descriptor{bean},
	}
	got := app.DescriptorByNSID("test.example.bean")
	assert.Equal(t, bean, got)
	assert.Nil(t, app.DescriptorByNSID("test.example.unknown"))
}
