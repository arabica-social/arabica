package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	t.Run("full cafe", func(t *testing.T) {
		c := &Cafe{
			Name:        "Floating Mountain",
			Location:    "New York, NY",
			Address:     "243 W 72nd St",
			Website:     "https://floatingmountain.tea",
			Description: "Specialty teahouse",
			CreatedAt:   createdAt,
		}
		vendorURI := "at://did:plc:test/social.oolong.alpha.vendor/v1"
		rec, err := CafeToRecord(c, vendorURI)
		require.NoError(t, err)
		shutter.Snap(t, "CafeToRecord/full cafe", rec)
	})
	t.Run("minimal cafe", func(t *testing.T) {
		c := &Cafe{Name: "Tea Spot", CreatedAt: createdAt}
		rec, err := CafeToRecord(c, "")
		require.NoError(t, err)
		shutter.Snap(t, "CafeToRecord/minimal cafe", rec)
	})
}

func TestRecordToCafe(t *testing.T) {
	rec := map[string]any{
		"$type":     NSIDCafe,
		"name":      "Floating Mountain",
		"location":  "New York, NY",
		"vendorRef": "at://did:plc:test/social.oolong.alpha.vendor/v1",
		"createdAt": "2026-05-10T12:00:00Z",
	}
	c, err := RecordToCafe(rec, "at://did:plc:test/social.oolong.alpha.cafe/cafe1")
	require.NoError(t, err)
	assert.Equal(t, "cafe1", c.RKey)
	assert.Equal(t, "Floating Mountain", c.Name)
	assert.NotEmpty(t, c.VendorRKey)
}
