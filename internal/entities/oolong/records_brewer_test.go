package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewerRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brewer{
		Name:        "Tokoname kyusu",
		Style:       BrewerStyleKyusu,
		CapacityMl:  180,
		Material:    "stoneware",
		Description: "Side-handle, fine mesh",
		CreatedAt:   createdAt,
	}
	rec, err := BrewerToRecord(original)
	require.NoError(t, err)
	shutter.Snap(t, "BrewerToRecord/full brewer", rec)

	round, err := RecordToBrewer(rec, "at://did:plc:test/social.oolong.alpha.brewer/b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	assert.Equal(t, original.CapacityMl, round.CapacityMl)
	assert.Equal(t, original.Material, round.Material)
}

func TestNormalizeBrewerStyle(t *testing.T) {
	assert.Equal(t, BrewerStyleKyusu, NormalizeBrewerStyle("Kyuusu"))
	assert.Equal(t, BrewerStyleTeapot, NormalizeBrewerStyle("Yixing"))
	assert.Equal(t, "unknown-thing", NormalizeBrewerStyle("unknown-thing"))
}
