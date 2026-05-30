package coffee

import (
	"testing"

	"github.com/stretchr/testify/assert"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestFeedViews_AllArabicaEntitiesHaveFeedRenderer(t *testing.T) {
	views := FeedViews()
	want := map[lexicons.RecordType]string{
		lexicons.RecordTypeBean:    "Beans",
		lexicons.RecordTypeRoaster: "",
		lexicons.RecordTypeGrinder: "Grinders",
		lexicons.RecordTypeBrewer:  "Brewers",
		lexicons.RecordTypeRecipe:  "Recipes",
		lexicons.RecordTypeBrew:    "Brews",
	}
	for rt, filterLabel := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.NotNil(t, views[rt].Render, "feed renderer not wired for %s", rt)
		assert.Equal(t, filterLabel, views.FilterLabel(rt), "feed filter label for %s", rt)
		assert.NotEmpty(t, views.CardClassNoun(rt), "feed card noun for %s", rt)
	}
}

func TestFeedViews_ActionURLs(t *testing.T) {
	views := FeedViews()

	assert.Equal(t, "/brews/b1/edit", views.EditURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBrew,
		Record:     &arabica.Brew{RKey: "b1"},
	}))
	assert.Equal(t, "/api/modals/bean/bean1", views.EditModalURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBean,
		Record:     &arabica.Bean{RKey: "bean1"},
	}))
}

func TestFeedViews_RecordURLs(t *testing.T) {
	views := FeedViews()

	assert.Equal(t, "/beans/patrick.test/bean1", views.ShareURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBean,
		Record:     &arabica.Bean{RKey: "bean1"},
		Author:     author("did:plc:alice", "patrick.test"),
	}))
	assert.Equal(t, "/profile/did:plc:alice", views.ShareURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBean,
		Record:     &arabica.Bean{},
		Author:     author("did:plc:alice", ""),
	}))
	assert.Equal(t, "/api/beans/bean1", views.DeleteURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBean,
		Record:     &arabica.Bean{RKey: "bean1"},
	}))
	assert.Equal(t, "/brews/b1", views.DeleteURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeBrew,
		Record:     &arabica.Brew{RKey: "b1"},
	}))
}

func TestFeedViews_CompactEntities(t *testing.T) {
	views := FeedViews()
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeRoaster,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeBrewer,
	} {
		assert.True(t, views.Compact(rt), "%s should be compact", rt)
	}
}

func author(did, handle string) *atproto.Profile {
	return &atproto.Profile{DID: did, Handle: handle}
}

func TestFeedViews_NonCompactEntities(t *testing.T) {
	views := FeedViews()
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeBrew,
	} {
		assert.False(t, views.Compact(rt), "%s should not be compact", rt)
	}
}
