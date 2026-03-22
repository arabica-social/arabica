package models

import "strings"

// RecipeFilter defines criteria for filtering recipes.
type RecipeFilter struct {
	Query      string  // case-insensitive substring match on name
	BrewerType string  // exact match on brewer_type
	MinCoffee  float64 // minimum coffee amount in grams
	MaxCoffee  float64 // maximum coffee amount in grams
	MinWater   float64 // minimum water amount in grams
	MaxWater   float64 // maximum water amount in grams
	Category   string  // predefined category key
}

// RecipeCategories maps category names to their filter criteria.
var RecipeCategories = map[string]RecipeFilter{
	"small":  {MaxCoffee: 20},
	"large":  {MinCoffee: 30},
	"single": {MaxCoffee: 20, MaxWater: 300},
	"batch":  {MinWater: 500},
}

// MatchesFilter returns true if the recipe satisfies all non-zero filter criteria.
// Criteria are combined with AND logic; zero-value fields are ignored.
func MatchesFilter(recipe *Recipe, filter RecipeFilter) bool {
	// Apply category defaults first (explicit fields override)
	f := resolveCategory(filter)

	if f.Query != "" && !strings.Contains(strings.ToLower(recipe.Name), strings.ToLower(f.Query)) {
		return false
	}
	if f.BrewerType != "" && !strings.EqualFold(recipe.BrewerType, f.BrewerType) {
		return false
	}
	if f.MinCoffee > 0 && recipe.CoffeeAmount < f.MinCoffee {
		return false
	}
	if f.MaxCoffee > 0 && recipe.CoffeeAmount > f.MaxCoffee {
		return false
	}
	if f.MinWater > 0 && recipe.WaterAmount < f.MinWater {
		return false
	}
	if f.MaxWater > 0 && recipe.WaterAmount > f.MaxWater {
		return false
	}
	return true
}

// FilterRecipes returns the subset of recipes matching the filter.
func FilterRecipes(recipes []*Recipe, filter RecipeFilter) []*Recipe {
	var result []*Recipe
	for _, r := range recipes {
		if MatchesFilter(r, filter) {
			result = append(result, r)
		}
	}
	return result
}

// resolveCategory merges category defaults with explicit filter fields.
// Explicit fields take precedence over category defaults.
func resolveCategory(f RecipeFilter) RecipeFilter {
	cat, ok := RecipeCategories[f.Category]
	if !ok {
		return f
	}
	if f.MinCoffee == 0 {
		f.MinCoffee = cat.MinCoffee
	}
	if f.MaxCoffee == 0 {
		f.MaxCoffee = cat.MaxCoffee
	}
	if f.MinWater == 0 {
		f.MinWater = cat.MinWater
	}
	if f.MaxWater == 0 {
		f.MaxWater = cat.MaxWater
	}
	return f
}
