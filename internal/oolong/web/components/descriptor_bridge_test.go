package tea

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestOolongFeedViews_AllEntitiesHaveFeedRenderer(t *testing.T) {
	views := FeedViews()
	want := []lexicons.RecordType{
		lexicons.RecordTypeOolongTea,
		lexicons.RecordTypeOolongVendor,
		lexicons.RecordTypeOolongVessel,
		lexicons.RecordTypeOolongInfuser,
		lexicons.RecordTypeOolongBrew,
	}
	for _, rt := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.NotNil(t, views[rt].Render, "feed renderer not wired for %s", rt)
		assert.NotNil(t, d.RKey, "RKey not wired for %s", rt)
		assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", rt)
	}
}

func TestOolongFeedViews_BrewDescriptorKeepsEditURL(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongBrew)
	assert.NotNil(t, d)
	assert.NotNil(t, d.EditURL, "Oolong Brew should have EditURL wired")
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
