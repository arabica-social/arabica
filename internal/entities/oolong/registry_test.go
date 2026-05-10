package oolong

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestOolongDescriptorsRegistered(t *testing.T) {
	cases := []struct {
		rt    lexicons.RecordType
		nsid  string
		label string
	}{
		{lexicons.RecordTypeOolongTea, NSIDTea, "Tea"},
		{lexicons.RecordTypeOolongBrew, NSIDBrew, "Tea Brew"},
		{lexicons.RecordTypeOolongBrewer, NSIDBrewer, "Tea Brewer"},
		{lexicons.RecordTypeOolongRecipe, NSIDRecipe, "Tea Recipe"},
		{lexicons.RecordTypeOolongVendor, NSIDVendor, "Tea Vendor"},
		{lexicons.RecordTypeOolongCafe, NSIDCafe, "Tea Cafe"},
		{lexicons.RecordTypeOolongDrink, NSIDDrink, "Tea Drink"},
	}
	for _, tc := range cases {
		t.Run(string(tc.rt), func(t *testing.T) {
			d := entities.Get(tc.rt)
			if assert.NotNil(t, d, "missing descriptor for %s", tc.rt) {
				assert.Equal(t, tc.nsid, d.NSID)
				assert.Equal(t, tc.label, d.DisplayName)
			}
		})
	}
}

func TestOolongLikeNotRegistered(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongLike)
	assert.Nil(t, d, "like is intentionally not registered (App.NSIDs() appends it)")
}

func TestOolongCommentNotRegistered(t *testing.T) {
	d := entities.Get(lexicons.RecordTypeOolongComment)
	assert.Nil(t, d, "comment is intentionally not registered (App.NSIDs() appends it)")
}

func TestOolongDescriptorsByNSID(t *testing.T) {
	d := entities.GetByNSID(NSIDTea)
	if assert.NotNil(t, d) {
		assert.Equal(t, lexicons.RecordTypeOolongTea, d.Type)
	}
}
