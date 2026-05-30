package oolong

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testDID = "did:plc:abcdefghijklmnopqrstuvwxyz"

func testURI(collection, rkey string) string {
	return "at://" + testDID + "/" + collection + "/" + rkey
}

func createdRecord(fields map[string]any) map[string]any {
	record := map[string]any{"createdAt": time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)}
	maps.Copy(record, fields)
	return record
}

func TestHydrateTeaRefsHydratesVendor(t *testing.T) {
	vendorURI := testURI(NSIDVendor, "vendor1")
	tea := &Tea{Name: "Tieguanyin"}

	HydrateTeaRefs(tea, createdRecord(map[string]any{"name": "Tieguanyin", "vendorRef": vendorURI}), func(ref string) (map[string]any, bool) {
		return createdRecord(map[string]any{"name": "Tea Habitat"}), ref == vendorURI
	})

	assert.NotNil(t, tea.Vendor)
	assert.Equal(t, "Tea Habitat", tea.Vendor.Name)
	assert.Equal(t, "vendor1", tea.Vendor.RKey)
}

func TestHydrateBrewRefsHydratesNestedReferences(t *testing.T) {
	teaURI := testURI(NSIDTea, "tea1")
	vendorURI := testURI(NSIDVendor, "vendor1")
	vesselURI := testURI(NSIDVessel, "vessel1")
	infuserURI := testURI(NSIDInfuser, "infuser1")
	lookupRecords := map[string]map[string]any{
		teaURI:     createdRecord(map[string]any{"name": "Tieguanyin", "vendorRef": vendorURI}),
		vendorURI:  createdRecord(map[string]any{"name": "Tea Habitat"}),
		vesselURI:  createdRecord(map[string]any{"name": "Gaiwan"}),
		infuserURI: createdRecord(map[string]any{"name": "Basket"}),
	}
	brew := &Brew{}

	HydrateBrewRefs(brew, createdRecord(map[string]any{
		"teaRef":     teaURI,
		"style":      StyleLongSteep,
		"vesselRef":  vesselURI,
		"infuserRef": infuserURI,
	}), func(ref string) (map[string]any, bool) {
		record, ok := lookupRecords[ref]
		return record, ok
	})

	assert.Equal(t, "Tieguanyin", brew.Tea.Name)
	assert.Equal(t, "Tea Habitat", brew.Tea.Vendor.Name)
	assert.Equal(t, "Gaiwan", brew.Vessel.Name)
	assert.Equal(t, "Basket", brew.Infuser.Name)
}

func TestHydrateBrewRefsSkipsMissingReferences(t *testing.T) {
	assert.NotPanics(t, func() {
		HydrateBrewRefs(&Brew{}, createdRecord(map[string]any{"teaRef": testURI(NSIDTea, "missing")}), func(string) (map[string]any, bool) {
			return nil, false
		})
	})
}
