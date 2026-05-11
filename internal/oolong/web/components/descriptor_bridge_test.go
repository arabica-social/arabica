package tea

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestOolongDescriptorBridge_AllEntitiesHaveFeedRenderer(t *testing.T) {
	want := []lexicons.RecordType{
		lexicons.RecordTypeOolongTea,
		lexicons.RecordTypeOolongVendor,
		lexicons.RecordTypeOolongBrewer,
		lexicons.RecordTypeOolongRecipe,
		lexicons.RecordTypeOolongBrew,
		lexicons.RecordTypeOolongCafe,
		lexicons.RecordTypeOolongDrink,
	}
	for _, rt := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.NotNil(t, d.RenderFeedContent, "RenderFeedContent not wired for %s", rt)
		assert.NotNil(t, d.RKey, "RKey not wired for %s", rt)
		assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", rt)
	}
}

func TestOolongDescriptorBridge_BrewHasEditURL(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongBrew)
	assert.NotNil(t, d)
	assert.NotNil(t, d.EditURL, "Oolong Brew should have EditURL wired")
}

func TestOolongDescriptorBridge_CompactEntities(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeOolongVendor,
		lexicons.RecordTypeOolongBrewer,
	} {
		d := entities.Get(rt)
		assert.NotNil(t, d)
		if d == nil {
			continue
		}
		assert.True(t, d.FeedCardCompact, "%s should be FeedCardCompact", rt)
	}
}

func TestOolongDescriptorBridge_NonCompactEntities(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeOolongTea,
		lexicons.RecordTypeOolongRecipe,
		lexicons.RecordTypeOolongBrew,
		lexicons.RecordTypeOolongCafe,
		lexicons.RecordTypeOolongDrink,
	} {
		d := entities.Get(rt)
		assert.NotNil(t, d)
		if d == nil {
			continue
		}
		assert.False(t, d.FeedCardCompact, "%s should not be FeedCardCompact", rt)
	}
}
