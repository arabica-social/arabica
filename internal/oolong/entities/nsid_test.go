package oolong

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNSIDs(t *testing.T) {
	assert.Equal(t, "social.oolong.alpha", NSIDBase)
	assert.Equal(t, "social.oolong.alpha.tea", NSIDTea)
	assert.Equal(t, "social.oolong.alpha.brew", NSIDBrew)
	assert.Equal(t, "social.oolong.alpha.vessel", NSIDVessel)
	assert.Equal(t, "social.oolong.alpha.infuser", NSIDInfuser)
	assert.Equal(t, "social.oolong.alpha.vendor", NSIDVendor)
	assert.Equal(t, "social.oolong.alpha.cafe", NSIDCafe)
	assert.Equal(t, "social.oolong.alpha.drink", NSIDDrink)
	assert.Equal(t, "social.oolong.alpha.comment", NSIDComment)
	assert.Equal(t, "social.oolong.alpha.like", NSIDLike)
}
