package oolong

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNSIDs(t *testing.T) {
	assert.Equal(t, "social.oolong.alpha", NSIDBase)
	assert.Equal(t, "social.oolong.alpha.tea", NSIDTea)
	assert.Equal(t, "social.oolong.alpha.brew", NSIDBrew)
	assert.Equal(t, "social.oolong.alpha.brewer", NSIDBrewer)
	assert.Equal(t, "social.oolong.alpha.recipe", NSIDRecipe)
	assert.Equal(t, "social.oolong.alpha.vendor", NSIDVendor)
	assert.Equal(t, "social.oolong.alpha.cafe", NSIDCafe)
	assert.Equal(t, "social.oolong.alpha.drink", NSIDDrink)
	assert.Equal(t, "social.oolong.alpha.comment", NSIDComment)
	assert.Equal(t, "social.oolong.alpha.like", NSIDLike)
}
