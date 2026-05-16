package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewToRecord_LongSteep(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:          StyleLongSteep,
		InfusionMethod: InfusionMethodLooseLeaf,
		Temperature:    85.0,
		LeafGrams:      3.0,
		WaterAmount:    500,
		TimeSeconds:    240,
		TastingNotes:   "Western brewed, robust",
		Rating:         7,
		CreatedAt:      createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/longSteep loose-leaf", rec)
}

func TestBrewToRecord_ColdBrew(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:          StyleColdBrew,
		InfusionMethod: InfusionMethodInfuser,
		Temperature:    4.0,
		LeafGrams:      10.0,
		WaterAmount:    1000,
		TimeSeconds:    43200, // 12h
		TastingNotes:   "Smooth, low astringency",
		Rating:         8,
		CreatedAt:      createdAt,
	}
	rec, err := BrewToRecord(
		brew,
		"at://did:plc:test/social.oolong.alpha.tea/t1",
		"at://did:plc:test/social.oolong.alpha.vessel/v1",
		"at://did:plc:test/social.oolong.alpha.infuser/i1",
	)
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/coldBrew with vessel+infuser", rec)
}

func TestBrewRoundTrip_LongSteep(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:          StyleLongSteep,
		InfusionMethod: InfusionMethodTeaBag,
		Temperature:    85.0,
		LeafGrams:      3.0,
		WaterAmount:    500,
		TimeSeconds:    240,
		CreatedAt:      createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", round.RKey)
	assert.Equal(t, StyleLongSteep, round.Style)
	assert.Equal(t, InfusionMethodTeaBag, round.InfusionMethod)
	assert.Equal(t, 85.0, round.Temperature)
	assert.Equal(t, 500, round.WaterAmount)
	assert.Equal(t, 240, round.TimeSeconds)
}

func TestBrewRoundTrip_ColdBrewWithRefs(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:          StyleColdBrew,
		InfusionMethod: InfusionMethodInfuser,
		LeafGrams:      10.0,
		WaterAmount:    1000,
		TimeSeconds:    43200,
		CreatedAt:      createdAt,
	}
	rec, err := BrewToRecord(
		original,
		"at://did:plc:test/social.oolong.alpha.tea/t1",
		"at://did:plc:test/social.oolong.alpha.vessel/v1",
		"at://did:plc:test/social.oolong.alpha.infuser/i1",
	)
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b2")
	require.NoError(t, err)
	assert.Equal(t, "v1", round.VesselRKey)
	assert.Equal(t, "i1", round.InfuserRKey)
	assert.Equal(t, InfusionMethodInfuser, round.InfusionMethod)
}

func TestBrewRequiresTeaRef(t *testing.T) {
	_, err := BrewToRecord(&Brew{Style: StyleLongSteep}, "", "", "")
	assert.ErrorIs(t, err, ErrTeaRefRequired)
}

func TestBrewRequiresStyle(t *testing.T) {
	_, err := BrewToRecord(&Brew{}, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	assert.ErrorIs(t, err, ErrStyleRequired)
}
