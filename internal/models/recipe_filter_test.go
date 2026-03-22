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
	small := &Recipe{Name: "Small Dose", CoffeeAmount: 12}
	borderline := &Recipe{Name: "Borderline", CoffeeAmount: 20}
	large := &Recipe{Name: "Big Batch", CoffeeAmount: 35}

	assert.True(t, MatchesFilter(small, RecipeFilter{Category: "small"}))
	assert.True(t, MatchesFilter(borderline, RecipeFilter{Category: "small"}))
	assert.False(t, MatchesFilter(large, RecipeFilter{Category: "small"}))
}

func TestMatchesFilter_CategoryLarge(t *testing.T) {
	small := &Recipe{Name: "Small Dose", CoffeeAmount: 12}
	large := &Recipe{Name: "Big Batch", CoffeeAmount: 35}

	assert.False(t, MatchesFilter(small, RecipeFilter{Category: "large"}))
	assert.True(t, MatchesFilter(large, RecipeFilter{Category: "large"}))
}

func TestMatchesFilter_CategorySingle(t *testing.T) {
	single := &Recipe{Name: "One Cup", CoffeeAmount: 15, WaterAmount: 250}
	batch := &Recipe{Name: "Party Brew", CoffeeAmount: 15, WaterAmount: 500}

	assert.True(t, MatchesFilter(single, RecipeFilter{Category: "single"}))
	assert.False(t, MatchesFilter(batch, RecipeFilter{Category: "single"}))
}

func TestMatchesFilter_CategoryBatch(t *testing.T) {
	single := &Recipe{Name: "One Cup", WaterAmount: 250}
	batch := &Recipe{Name: "Party Brew", WaterAmount: 600}

	assert.False(t, MatchesFilter(single, RecipeFilter{Category: "batch"}))
	assert.True(t, MatchesFilter(batch, RecipeFilter{Category: "batch"}))
}

func TestMatchesFilter_CategoryExplicitOverride(t *testing.T) {
	// Category "small" sets MaxCoffee=20, but explicit MaxCoffee=25 overrides
	recipe := &Recipe{Name: "Medium", CoffeeAmount: 22}
	assert.False(t, MatchesFilter(recipe, RecipeFilter{Category: "small"}))
	assert.True(t, MatchesFilter(recipe, RecipeFilter{Category: "small", MaxCoffee: 25}))
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
