package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVendorToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full vendor", func(t *testing.T) {
		v := &Vendor{
			Name:        "Spirit Tea",
			Location:    "Chicago, IL",
			Website:     "https://spirittea.co",
			Description: "Importer focused on Chinese and Taiwanese teas",
			CreatedAt:   createdAt,
		}
		record, err := VendorToRecord(v)
		require.NoError(t, err)
		shutter.Snap(t, "VendorToRecord/full vendor", record)
	})

	t.Run("minimal vendor", func(t *testing.T) {
		v := &Vendor{Name: "Tezumi", CreatedAt: createdAt}
		record, err := VendorToRecord(v)
		require.NoError(t, err)
		shutter.Snap(t, "VendorToRecord/minimal vendor", record)
	})

	t.Run("error without name", func(t *testing.T) {
		v := &Vendor{CreatedAt: createdAt}
		_, err := VendorToRecord(v)
		assert.ErrorIs(t, err, ErrNameRequired)
	})
}

func TestRecordToVendor(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":       NSIDVendor,
			"name":        "Spirit Tea",
			"location":    "Chicago, IL",
			"website":     "https://spirittea.co",
			"description": "Importer",
			"createdAt":   "2026-05-10T12:00:00Z",
		}
		v, err := RecordToVendor(record, "at://did:plc:test/social.oolong.alpha.vendor/abc123")
		require.NoError(t, err)
		assert.Equal(t, "abc123", v.RKey)
		assert.Equal(t, "Spirit Tea", v.Name)
		assert.Equal(t, "Chicago, IL", v.Location)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDVendor,
			"createdAt": "2026-05-10T12:00:00Z",
		}
		_, err := RecordToVendor(record, "")
		assert.Error(t, err)
	})
}

func TestVendorRoundTrip(t *testing.T) {
	original := &Vendor{
		Name:        "Yunnan Sourcing",
		Location:    "Kunming, Yunnan",
		Website:     "https://yunnansourcing.com",
		Description: "Direct from Yunnan",
		SourceRef:   "at://did:plc:other/social.oolong.alpha.vendor/source",
		CreatedAt:   time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	record, err := VendorToRecord(original)
	require.NoError(t, err)
	round, err := RecordToVendor(record, "at://did:plc:test/social.oolong.alpha.vendor/abc")
	require.NoError(t, err)
	round.RKey = ""
	assert.Equal(t, original, round)
}
