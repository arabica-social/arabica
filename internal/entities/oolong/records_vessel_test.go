package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVesselRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Vessel{
		Name:        "500ml glass teapot",
		Style:       VesselStyleTeapot,
		CapacityMl:  500,
		Material:    "glass",
		Description: "Built-in strainer, side handle",
		CreatedAt:   createdAt,
	}
	rec, err := VesselToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "VesselToRecord/full vessel", rec)

	round, err := RecordToVessel(rec, "at://did:plc:test/social.oolong.alpha.vessel/v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	assert.Equal(t, original.CapacityMl, round.CapacityMl)
	assert.Equal(t, original.Material, round.Material)
}

func TestNormalizeVesselStyle(t *testing.T) {
	assert.Equal(t, VesselStyleTeapot, NormalizeVesselStyle("Yixing"))
	assert.Equal(t, VesselStyleMug, NormalizeVesselStyle("Cup"))
	assert.Equal(t, VesselStyleJar, NormalizeVesselStyle("Thermos"))
	assert.Equal(t, "unknown-thing", NormalizeVesselStyle("unknown-thing"))
}
