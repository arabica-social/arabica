package tea

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
)

func TestOolongFeedViews_AllEntitiesHaveFeedRenderer(t *testing.T) {
	views := FeedViews()
	want := map[lexicons.RecordType]string{
		lexicons.RecordTypeOolongTea:     "Teas",
		lexicons.RecordTypeOolongVendor:  "Vendors",
		lexicons.RecordTypeOolongVessel:  "Vessels",
		lexicons.RecordTypeOolongInfuser: "Infusers",
		lexicons.RecordTypeOolongBrew:    "Brews",
	}
	for rt, filterLabel := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.NotNil(t, views[rt].Render, "feed renderer not wired for %s", rt)
		assert.Equal(t, filterLabel, views.FilterLabel(rt), "feed filter label for %s", rt)
		assert.NotNil(t, d.RKey, "RKey not wired for %s", rt)
		assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", rt)
	}
}

func TestOolongFeedViews_ActionURLs(t *testing.T) {
	views := FeedViews()

	assert.Equal(t, "/teas/t1/edit", views.EditURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeOolongTea,
		Record:     &oolong.Tea{RKey: "t1"},
	}))
	assert.Equal(t, "/brews/b1/edit", views.EditURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeOolongBrew,
		Record:     &oolong.Brew{RKey: "b1"},
	}))
	assert.Equal(t, "/api/modals/vendor/v1", views.EditModalURL(&feed.FeedItem{
		RecordType: lexicons.RecordTypeOolongVendor,
		Record:     &oolong.Vendor{RKey: "v1"},
	}))
}

func TestOolongFeedViews_CompactEntities(t *testing.T) {
	views := FeedViews()
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeOolongVendor,
		lexicons.RecordTypeOolongVessel,
		lexicons.RecordTypeOolongInfuser,
	} {
		assert.True(t, views.Compact(rt), "%s should be compact", rt)
	}
}

func TestOolongFeedViews_NonCompactEntities(t *testing.T) {
	views := FeedViews()
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeOolongTea,
		lexicons.RecordTypeOolongBrew,
	} {
		assert.False(t, views.Compact(rt), "%s should not be compact", rt)
	}
}
