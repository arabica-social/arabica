package coffee

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestFeedViews_AllArabicaEntitiesHaveFeedRenderer(t *testing.T) {
	views := FeedViews()
	want := []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeRoaster,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeBrewer,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeBrew,
	}
	for _, rt := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.NotNil(t, views[rt].Render, "feed renderer not wired for %s", rt)
	}
}

func TestFeedViews_BrewDescriptorKeepsEditURL(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeBrew)
	assert.NotNil(t, d)
	assert.NotNil(t, d.EditURL, "Brew should have EditURL wired")
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
