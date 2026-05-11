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

func TestOolongRegistry_RKey(t *testing.T) {
	cases := []struct {
		rt   lexicons.RecordType
		rec  any
		want string
	}{
		{lexicons.RecordTypeOolongTea, &Tea{RKey: "t1"}, "t1"},
		{lexicons.RecordTypeOolongVendor, &Vendor{RKey: "v1"}, "v1"},
		{lexicons.RecordTypeOolongBrewer, &Brewer{RKey: "br1"}, "br1"},
		{lexicons.RecordTypeOolongRecipe, &Recipe{RKey: "re1"}, "re1"},
		{lexicons.RecordTypeOolongBrew, &Brew{RKey: "bw1"}, "bw1"},
		{lexicons.RecordTypeOolongCafe, &Cafe{RKey: "c1"}, "c1"},
		{lexicons.RecordTypeOolongDrink, &Drink{RKey: "d1"}, "d1"},
	}
	for _, c := range cases {
		d := entities.Get(c.rt)
		assert.NotNil(t, d.RKey, "RKey not wired for %s", c.rt)
		assert.Equal(t, c.want, d.RKey(c.rec), c.rt)
		assert.Equal(t, "", d.RKey(struct{}{}), "%s should reject wrong type", c.rt)
	}
}

func TestOolongRegistry_DisplayTitle(t *testing.T) {
	cases := []struct {
		name string
		rt   lexicons.RecordType
		rec  any
		want string
	}{
		{"tea name", lexicons.RecordTypeOolongTea, &Tea{Name: "Da Hong Pao"}, "Da Hong Pao"},
		{"vendor name", lexicons.RecordTypeOolongVendor, &Vendor{Name: "Yunnan Sourcing"}, "Yunnan Sourcing"},
		{"recipe name", lexicons.RecordTypeOolongRecipe, &Recipe{Name: "Gongfu 5g"}, "Gongfu 5g"},
		{"brew uses tea name", lexicons.RecordTypeOolongBrew, &Brew{Tea: &Tea{Name: "Tieguanyin"}}, "Tieguanyin"},
		{"brew no tea falls back", lexicons.RecordTypeOolongBrew, &Brew{}, "Tea Brew"},
		{"drink with name", lexicons.RecordTypeOolongDrink, &Drink{Name: "Matcha latte"}, "Matcha latte"},
		{"drink fallback", lexicons.RecordTypeOolongDrink, &Drink{}, "Tea Drink"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := entities.Get(c.rt)
			assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", c.rt)
			assert.Equal(t, c.want, d.DisplayTitle(c.rec))
		})
	}
}
