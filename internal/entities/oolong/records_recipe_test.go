package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipeRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Recipe{
		Name:        "Standard gaiwan gongfu",
		Style:       StyleGongfu,
		Temperature: 95.0,
		LeafGrams:   5.5,
		VesselMl:    120,
		MethodParams: &GongfuParams{
			Rinse: true,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10},
				{Number: 2, TimeSeconds: 12},
				{Number: 3, TimeSeconds: 15},
			},
			TotalSteeps: 6,
		},
		Notes:     "Adjust steep times to taste",
		CreatedAt: createdAt,
	}
	rec, err := RecipeToRecord(original, "", "")
	require.NoError(t, err)
	shutter.Snap(t, "RecipeToRecord/gongfu standard", rec)

	round, err := RecordToRecipe(rec, "at://did:plc:test/social.oolong.alpha.recipe/r1")
	require.NoError(t, err)
	assert.Equal(t, "r1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	gp, ok := round.MethodParams.(*GongfuParams)
	require.True(t, ok)
	assert.Len(t, gp.Steeps, 3)
}

func TestRecipeMissingName(t *testing.T) {
	_, err := RecipeToRecord(&Recipe{CreatedAt: time.Now()}, "", "")
	assert.ErrorIs(t, err, ErrNameRequired)
}
