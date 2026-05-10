package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrinkRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Drink{
		Name:          "Iced hojicha latte",
		Style:         "milkTea",
		Rating:        8,
		TastingNotes:  "Toasty, smooth",
		PriceUsdCents: 750,
		CreatedAt:     createdAt,
	}
	cafeURI := "at://did:plc:test/social.oolong.alpha.cafe/c1"
	teaURI := "at://did:plc:test/social.oolong.alpha.tea/t1"

	rec, err := DrinkToRecord(original, cafeURI, teaURI)
	require.NoError(t, err)
	shutter.Snap(t, "DrinkToRecord/full drink", rec)

	round, err := RecordToDrink(rec, "at://did:plc:test/social.oolong.alpha.drink/d1")
	require.NoError(t, err)
	assert.Equal(t, "d1", round.RKey)
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Style, round.Style)
	assert.Equal(t, original.PriceUsdCents, round.PriceUsdCents)
	assert.NotEmpty(t, round.CafeRKey)
	assert.NotEmpty(t, round.TeaRKey)
}

func TestDrinkRequiresCafeRef(t *testing.T) {
	d := &Drink{Name: "Tea", CreatedAt: time.Now()}
	_, err := DrinkToRecord(d, "", "")
	assert.ErrorIs(t, err, ErrCafeRefRequired)
}
