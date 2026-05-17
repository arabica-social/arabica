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
		{lexicons.RecordTypeOolongVessel, NSIDVessel, "Vessel"},
		{lexicons.RecordTypeOolongInfuser, NSIDInfuser, "Infuser"},
		{lexicons.RecordTypeOolongVendor, NSIDVendor, "Tea Vendor"},
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

func TestOolongDeferredNotRegistered(t *testing.T) {
	// Cafe and Drink are deferred for v1 — defined in lexicons but
	// intentionally not registered as descriptors.
	assert.Nil(t, entities.Get(lexicons.RecordTypeOolongCafe), "cafe deferred for v1")
	assert.Nil(t, entities.Get(lexicons.RecordTypeOolongDrink), "drink deferred for v1")
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
		{lexicons.RecordTypeOolongVessel, &Vessel{RKey: "v1"}, "v1"},
		{lexicons.RecordTypeOolongInfuser, &Infuser{RKey: "i1"}, "i1"},
		{lexicons.RecordTypeOolongBrew, &Brew{RKey: "bw1"}, "bw1"},
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
		{"vessel name", lexicons.RecordTypeOolongVessel, &Vessel{Name: "Glass teapot"}, "Glass teapot"},
		{"infuser name", lexicons.RecordTypeOolongInfuser, &Infuser{Name: "Stainless basket"}, "Stainless basket"},
		{"brew uses tea name", lexicons.RecordTypeOolongBrew, &Brew{Tea: &Tea{Name: "Tieguanyin"}}, "Tieguanyin"},
		{"brew no tea falls back", lexicons.RecordTypeOolongBrew, &Brew{}, "Tea Brew"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := entities.Get(c.rt)
			assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", c.rt)
			assert.Equal(t, c.want, d.DisplayTitle(c.rec))
		})
	}
}
