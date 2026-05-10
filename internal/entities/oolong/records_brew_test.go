package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewToRecord_Gongfu(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleGongfu,
		Temperature:  95.0,
		LeafGrams:    5.5,
		VesselMl:     120,
		TastingNotes: "Sweet, nutty",
		Rating:       8,
		MethodParams: &GongfuParams{
			Rinse:        true,
			RinseSeconds: 5,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10, TastingNotes: "Bright, floral"},
				{Number: 2, TimeSeconds: 12, TastingNotes: "Deeper, nutty"},
				{Number: 3, TimeSeconds: 15, Temperature: 980},
			},
			TotalSteeps: 6,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/gongfu full", rec)
}

func TestBrewToRecord_Matcha(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleMatcha,
		Temperature:  75.0,
		LeafGrams:    2.0,
		VesselMl:     70,
		TastingNotes: "Umami forward",
		Rating:       9,
		MethodParams: &MatchaParams{
			Preparation: MatchaPrepUsucha,
			Sieved:      true,
			WhiskType:   "chasen 80-prong",
			WaterMl:     70,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/matcha full", rec)
}

func TestBrewToRecord_MilkTea(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:       StyleMilkTea,
		Temperature: 95.0,
		TimeSeconds: 600,
		Rating:      7,
		MethodParams: &MilkTeaParams{
			Preparation: "stovetop simmer",
			Ingredients: []Ingredient{
				{Name: "Whole milk", Amount: 200, Unit: IngredientUnitMl},
				{Name: "Cardamom pods", Amount: 4, Unit: IngredientUnitPcs},
				{Name: "Sugar", Amount: 1.5, Unit: IngredientUnitTsp, Notes: "Demerara"},
			},
			Iced: false,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/milkTea full", rec)
}

func TestBrewToRecord_LongSteep(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	brew := &Brew{
		Style:        StyleLongSteep,
		Temperature:  85.0,
		LeafGrams:    3.0,
		VesselMl:     500,
		TimeSeconds:  240,
		TastingNotes: "Western brewed, robust",
		Rating:       7,
		// MethodParams intentionally nil
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(brew, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/longSteep no params", rec)
}

func TestBrewRoundTrip_Gongfu(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:       StyleGongfu,
		Temperature: 95.0,
		LeafGrams:   5.5,
		VesselMl:    120,
		Rating:      8,
		MethodParams: &GongfuParams{
			Rinse:        true,
			RinseSeconds: 5,
			Steeps: []Steep{
				{Number: 1, TimeSeconds: 10, TastingNotes: "Bright"},
				{Number: 2, TimeSeconds: 12},
			},
		},
		CreatedAt: createdAt,
	}
	teaURI := "at://did:plc:test/social.oolong.alpha.tea/t1"
	rec, err := BrewToRecord(original, teaURI, "", "")
	require.NoError(t, err)

	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", round.RKey)
	assert.Equal(t, StyleGongfu, round.Style)
	assert.Equal(t, 95.0, round.Temperature)

	gp, ok := round.MethodParams.(*GongfuParams)
	require.True(t, ok, "expected gongfuParams")
	assert.True(t, gp.Rinse)
	assert.Len(t, gp.Steeps, 2)
	assert.Equal(t, "Bright", gp.Steeps[0].TastingNotes)
}

func TestBrewRoundTrip_Matcha(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style: StyleMatcha,
		MethodParams: &MatchaParams{
			Preparation: MatchaPrepKoicha,
			Sieved:      true,
			WhiskType:   "chasen 120-prong",
			WaterMl:     30,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b2")
	require.NoError(t, err)
	mp, ok := round.MethodParams.(*MatchaParams)
	require.True(t, ok)
	assert.Equal(t, MatchaPrepKoicha, mp.Preparation)
	assert.Equal(t, 30, mp.WaterMl)
}

func TestBrewRoundTrip_MilkTea(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style: StyleMilkTea,
		MethodParams: &MilkTeaParams{
			Preparation: "shaken iced",
			Ingredients: []Ingredient{
				{Name: "Whole milk", Amount: 200, Unit: "ml"},
				{Name: "Sugar", Amount: 1.5, Unit: "tsp", Notes: "Demerara"},
			},
			Iced: true,
		},
		CreatedAt: createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b3")
	require.NoError(t, err)
	mtp, ok := round.MethodParams.(*MilkTeaParams)
	require.True(t, ok)
	assert.True(t, mtp.Iced)
	require.Len(t, mtp.Ingredients, 2)
	assert.Equal(t, "Whole milk", mtp.Ingredients[0].Name)
	assert.Equal(t, 200.0, mtp.Ingredients[0].Amount)
	assert.Equal(t, 1.5, mtp.Ingredients[1].Amount)
	assert.Equal(t, "Demerara", mtp.Ingredients[1].Notes)
}

func TestBrewRoundTrip_LongSteepNoParams(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	original := &Brew{
		Style:       StyleLongSteep,
		Temperature: 85.0,
		LeafGrams:   3.0,
		VesselMl:    500,
		TimeSeconds: 240,
		CreatedAt:   createdAt,
	}
	rec, err := BrewToRecord(original, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	require.NoError(t, err)
	round, err := RecordToBrew(rec, "at://did:plc:test/social.oolong.alpha.brew/b4")
	require.NoError(t, err)
	assert.Equal(t, StyleLongSteep, round.Style)
	assert.Nil(t, round.MethodParams)
	assert.Equal(t, 85.0, round.Temperature)
	assert.Equal(t, 240, round.TimeSeconds)
}

func TestBrewRequiresTeaRef(t *testing.T) {
	_, err := BrewToRecord(&Brew{Style: StyleGongfu}, "", "", "")
	assert.ErrorIs(t, err, ErrTeaRefRequired)
}

func TestBrewRequiresStyle(t *testing.T) {
	_, err := BrewToRecord(&Brew{}, "at://did:plc:test/social.oolong.alpha.tea/t1", "", "")
	assert.ErrorIs(t, err, ErrStyleRequired)
}
