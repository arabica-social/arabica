package coffee

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestDescriptorBridge_AllArabicaEntitiesHaveFeedRenderer(t *testing.T) {
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
		assert.NotNil(t, d.RenderFeedContent, "RenderFeedContent not wired for %s", rt)
	}
}

func TestDescriptorBridge_BrewHasEditURL(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeBrew)
	assert.NotNil(t, d)
	assert.NotNil(t, d.EditURL, "Brew should have EditURL wired")
}

func TestDescriptorBridge_CompactEntities(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeRoaster,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeBrewer,
	} {
		d := entities.Get(rt)
		assert.NotNil(t, d)
		if d == nil {
			continue
		}
		assert.True(t, d.FeedCardCompact, "%s should be FeedCardCompact", rt)
	}
}

func TestDescriptorBridge_NonCompactEntities(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeBrew,
	} {
		d := entities.Get(rt)
		assert.NotNil(t, d)
		if d == nil {
			continue
		}
		assert.False(t, d.FeedCardCompact, "%s should not be FeedCardCompact", rt)
	}
}
