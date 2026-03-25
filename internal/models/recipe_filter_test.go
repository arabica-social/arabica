package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesFilter_EmptyFilter(t *testing.T) {
	recipe := &Recipe{Name: "V60 Morning", CoffeeAmount: 18, WaterAmount: 300}
	assert.True(t, MatchesFilter(recipe, RecipeFilter{}))
}

func TestMatchesFilter_QueryMatch(t *testing.T) {
	recipe := &Recipe{Name: "V60 Morning Brew"}

	assert.True(t, MatchesFilter(recipe, RecipeFilter{Query: "morning"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Query: "V60"}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{Query: "espresso"}))
}

func TestMatchesFilter_QueryCaseInsensitive(t *testing.T) {
	recipe := &Recipe{Name: "French Press Bold"}
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Query: "FRENCH"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Query: "french press"}))
}

func TestMatchesFilter_BrewerType(t *testing.T) {
	recipe := &Recipe{Name: "Test", BrewerType: "Pour-Over"}

	assert.True(t, MatchesFilter(recipe, RecipeFilter{BrewerType: "Pour-Over"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{BrewerType: "pour-over"}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{BrewerType: "French Press"}))
}

func TestMatchesFilter_CoffeeRange(t *testing.T) {
	recipe := &Recipe{Name: "Test", CoffeeAmount: 18}

	assert.True(t, MatchesFilter(recipe, RecipeFilter{MinCoffee: 15}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{MaxCoffee: 20}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{MinCoffee: 15, MaxCoffee: 20}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{MinCoffee: 20}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{MaxCoffee: 15}))
}

func TestMatchesFilter_WaterRange(t *testing.T) {
	recipe := &Recipe{Name: "Test", WaterAmount: 300}

	assert.True(t, MatchesFilter(recipe, RecipeFilter{MinWater: 200}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{MaxWater: 400}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{MinWater: 400}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{MaxWater: 200}))
}

func TestMatchesFilter_MultipleFilters(t *testing.T) {
	recipe := &Recipe{
		Name:         "V60 Light",
		BrewerType:   "Pour-Over",
		CoffeeAmount: 15,
		WaterAmount:  250,
	}

	// All match
	assert.True(t, MatchesFilter(recipe, RecipeFilter{
		Query:      "V60",
		BrewerType: "Pour-Over",
		MinCoffee:  10,
		MaxWater:   300,
	}))

	// One fails
	assert.False(t, MatchesFilter(recipe, RecipeFilter{
		Query:      "V60",
		BrewerType: "French Press",
		MinCoffee:  10,
	}))
}

func TestMatchesFilter_CategorySmall(t *testing.T) {
	espresso := &Recipe{Name: "Espresso", CoffeeAmount: 9}
	borderline := &Recipe{Name: "Borderline", CoffeeAmount: 12}
	pourover := &Recipe{Name: "Pour Over", CoffeeAmount: 15}

	assert.True(t, MatchesFilter(espresso, RecipeFilter{Category: "small"}))
	assert.True(t, MatchesFilter(borderline, RecipeFilter{Category: "small"}))
	assert.False(t, MatchesFilter(pourover, RecipeFilter{Category: "small"}))
}

func TestMatchesFilter_CategoryLarge(t *testing.T) {
	single := &Recipe{Name: "Single Cup", CoffeeAmount: 15}
	borderline := &Recipe{Name: "Borderline", CoffeeAmount: 22}
	large := &Recipe{Name: "Big Batch", CoffeeAmount: 35}

	assert.False(t, MatchesFilter(single, RecipeFilter{Category: "large"}))
	assert.True(t, MatchesFilter(borderline, RecipeFilter{Category: "large"}))
	assert.True(t, MatchesFilter(large, RecipeFilter{Category: "large"}))
}

func TestMatchesFilter_CategorySingle(t *testing.T) {
	small := &Recipe{Name: "Espresso", CoffeeAmount: 9, WaterAmount: 40}
	single := &Recipe{Name: "One Cup", CoffeeAmount: 15, WaterAmount: 250}
	tooMuchWater := &Recipe{Name: "Party Brew", CoffeeAmount: 15, WaterAmount: 500}
	tooBigDose := &Recipe{Name: "Large", CoffeeAmount: 25, WaterAmount: 300}

	assert.False(t, MatchesFilter(small, RecipeFilter{Category: "single"}))        // coffee too low
	assert.True(t, MatchesFilter(single, RecipeFilter{Category: "single"}))        // perfect fit
	assert.False(t, MatchesFilter(tooMuchWater, RecipeFilter{Category: "single"})) // water too high
	assert.False(t, MatchesFilter(tooBigDose, RecipeFilter{Category: "single"}))   // coffee too high
}

func TestMatchesFilter_CategoriesNoOverlap(t *testing.T) {
	// A 15g dose should only match "single", not "small" or "large"
	recipe := &Recipe{Name: "V60", CoffeeAmount: 15, WaterAmount: 250}

	assert.False(t, MatchesFilter(recipe, RecipeFilter{Category: "small"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Category: "single"}))
	assert.False(t, MatchesFilter(recipe, RecipeFilter{Category: "large"}))
}

func TestMatchesFilter_CategoryBatch(t *testing.T) {
	single := &Recipe{Name: "One Cup", WaterAmount: 250}
	batch := &Recipe{Name: "Party Brew", WaterAmount: 600}

	assert.False(t, MatchesFilter(single, RecipeFilter{Category: "batch"}))
	assert.True(t, MatchesFilter(batch, RecipeFilter{Category: "batch"}))
}

func TestMatchesFilter_CategoryExplicitOverride(t *testing.T) {
	// Category "small" sets MaxCoffee=12, but explicit MaxCoffee=18 overrides
	recipe := &Recipe{Name: "Medium", CoffeeAmount: 15}
	assert.False(t, MatchesFilter(recipe, RecipeFilter{Category: "small"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Category: "small", MaxCoffee: 18}))
}

func TestMatchesFilter_UnknownCategory(t *testing.T) {
	recipe := &Recipe{Name: "Test", CoffeeAmount: 18}
	// Unknown category is ignored
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Category: "unknown"}))
}

func TestMatchesFilter_ZeroCoffeeAmount(t *testing.T) {
	// Recipe with zero coffee amount should not match MinCoffee filter
	recipe := &Recipe{Name: "No Coffee Set", CoffeeAmount: 0}
	assert.False(t, MatchesFilter(recipe, RecipeFilter{MinCoffee: 10}))
	// But should match MaxCoffee (0 <= max)
	assert.True(t, MatchesFilter(recipe, RecipeFilter{MaxCoffee: 10}))
}

func TestRecipeInterpolate_WaterFromPours(t *testing.T) {
	recipe := &Recipe{
		Name:         "V60",
		CoffeeAmount: 15,
		WaterAmount:  0, // not set
		Pours: []*Pour{
			{WaterAmount: 50},
			{WaterAmount: 100},
			{WaterAmount: 100},
		},
	}
	recipe.Interpolate()
	assert.Equal(t, 250.0, recipe.WaterAmount)
	assert.InDelta(t, 16.67, recipe.Ratio, 0.01)
}

func TestRecipeInterpolate_WaterAlreadySet(t *testing.T) {
	recipe := &Recipe{
		Name:         "V60",
		CoffeeAmount: 15,
		WaterAmount:  300,
		Pours: []*Pour{
			{WaterAmount: 50},
			{WaterAmount: 100},
		},
	}
	recipe.Interpolate()
	// Should keep existing water amount, not sum pours
	assert.Equal(t, 300.0, recipe.WaterAmount)
	assert.InDelta(t, 20.0, recipe.Ratio, 0.01)
}

func TestRecipeInterpolate_RatioOnly(t *testing.T) {
	recipe := &Recipe{CoffeeAmount: 18, WaterAmount: 300}
	recipe.Interpolate()
	assert.InDelta(t, 16.67, recipe.Ratio, 0.01)
}

func TestRecipeInterpolate_NoCoffee(t *testing.T) {
	recipe := &Recipe{WaterAmount: 300}
	recipe.Interpolate()
	assert.Equal(t, 0.0, recipe.Ratio) // can't compute ratio without coffee
}

func TestMatchesFilter_InterpolatesWaterFromPours(t *testing.T) {
	// Recipe with no water_amount but pours that sum to 250g
	recipe := &Recipe{
		Name:         "Pour-over",
		CoffeeAmount: 15,
		Pours: []*Pour{
			{WaterAmount: 50},
			{WaterAmount: 100},
			{WaterAmount: 100},
		},
	}
	// Should match single cup (water 250 <= 400) after interpolation
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Category: "single"}))
	// Should not match batch (water 250 < 500)
	assert.False(t, MatchesFilter(recipe, RecipeFilter{Category: "batch"}))
}

func TestFilterRecipes(t *testing.T) {
	recipes := []*Recipe{
		{Name: "V60 Light", CoffeeAmount: 15, BrewerType: "Pour-Over"},
		{Name: "French Press Bold", CoffeeAmount: 30, BrewerType: "French Press"},
		{Name: "Espresso Shot", CoffeeAmount: 18, BrewerType: "Espresso"},
		{Name: "V60 Strong", CoffeeAmount: 20, BrewerType: "Pour-Over"},
	}

	result := FilterRecipes(recipes, RecipeFilter{Query: "V60"})
	assert.Len(t, result, 2)
	assert.Equal(t, "V60 Light", result[0].Name)
	assert.Equal(t, "V60 Strong", result[1].Name)

	result = FilterRecipes(recipes, RecipeFilter{BrewerType: "Pour-Over"})
	assert.Len(t, result, 2)

	result = FilterRecipes(recipes, RecipeFilter{MinCoffee: 20})
	assert.Len(t, result, 2)

	// Empty filter returns all
	result = FilterRecipes(recipes, RecipeFilter{})
	assert.Len(t, result, 4)
}
