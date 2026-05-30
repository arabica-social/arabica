package arabica

import (
	"tangled.org/pdewey.com/atp"
)

// RecordLookup returns the raw record for an AT-URI reference.
// Callers bind it to the appropriate source (feed index, witness cache,
// authenticated PDS, public PDS). Hydration stays source-agnostic and treats
// missing refs as non-fatal.
type RecordLookup func(refURI string) (map[string]any, bool)

// resolveBrewFeedRefs hydrates bean/grinder/brewer/recipe references on a
// brew using already-fetched indexed records exposed via lookup. Missing
// refs are silently skipped — feed cards render fine with partial data.
func resolveBrewFeedRefs(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	brew, ok := model.(*Brew)
	if !ok || brew == nil {
		return
	}
	HydrateBrewRefs(brew, recordData, lookup)
}

// HydrateBrewRefs hydrates bean/grinder/brewer/recipe references on a brew.
// Nested Bean→Roaster and Recipe→Brewer refs are also hydrated when available.
func HydrateBrewRefs(brew *Brew, recordData map[string]any, lookup RecordLookup) {
	if brew == nil || recordData == nil || lookup == nil {
		return
	}

	if beanRef, ok := recordData["beanRef"].(string); ok && beanRef != "" {
		setStringIfEmpty(&brew.BeanRKey, atp.RKeyFromURI(beanRef))
		if beanData, found := lookup(beanRef); found {
			if bean, err := RecordToBean(beanData, beanRef); err == nil && brew.Bean == nil {
				brew.Bean = bean
			}
			HydrateBeanRefs(brew.Bean, beanData, lookup)
		}
	}

	if grinderRef, ok := recordData["grinderRef"].(string); ok && grinderRef != "" {
		setStringIfEmpty(&brew.GrinderRKey, atp.RKeyFromURI(grinderRef))
		if grinderData, found := lookup(grinderRef); found {
			if grinder, err := RecordToGrinder(grinderData, grinderRef); err == nil && brew.GrinderObj == nil {
				brew.GrinderObj = grinder
			}
		}
	}

	if brewerRef, ok := recordData["brewerRef"].(string); ok && brewerRef != "" {
		setStringIfEmpty(&brew.BrewerRKey, atp.RKeyFromURI(brewerRef))
		if brewerData, found := lookup(brewerRef); found {
			if brewer, err := RecordToBrewer(brewerData, brewerRef); err == nil && brew.BrewerObj == nil {
				brew.BrewerObj = brewer
			}
		}
	}

	if recipeRef, ok := recordData["recipeRef"].(string); ok && recipeRef != "" {
		setStringIfEmpty(&brew.RecipeRKey, atp.RKeyFromURI(recipeRef))
		if recipeData, found := lookup(recipeRef); found {
			if recipe, err := RecordToRecipe(recipeData, recipeRef); err == nil && brew.RecipeObj == nil {
				brew.RecipeObj = recipe
			}
			HydrateRecipeRefs(brew.RecipeObj, recipeData, lookup)
		}
	}
}

// resolveBeanFeedRef hydrates a bean's roaster reference.
func resolveBeanFeedRef(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	bean, ok := model.(*Bean)
	if !ok || bean == nil {
		return
	}
	HydrateBeanRefs(bean, recordData, lookup)
}

// HydrateBeanRefs hydrates a bean's roaster reference.
func HydrateBeanRefs(bean *Bean, recordData map[string]any, lookup RecordLookup) {
	if bean == nil || recordData == nil || lookup == nil {
		return
	}
	roasterRef, ok := recordData["roasterRef"].(string)
	if !ok || roasterRef == "" {
		return
	}
	setStringIfEmpty(&bean.RoasterRKey, atp.RKeyFromURI(roasterRef))
	if bean.Roaster != nil {
		return
	}
	roasterData, found := lookup(roasterRef)
	if !found {
		return
	}
	if roaster, err := RecordToRoaster(roasterData, roasterRef); err == nil {
		bean.Roaster = roaster
	}
}

// resolveRecipeFeedRef hydrates a recipe's brewer reference.
func resolveRecipeFeedRef(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	recipe, ok := model.(*Recipe)
	if !ok || recipe == nil {
		return
	}
	HydrateRecipeRefs(recipe, recordData, lookup)
}

// HydrateRecipeRefs hydrates a recipe's brewer reference.
func HydrateRecipeRefs(recipe *Recipe, recordData map[string]any, lookup RecordLookup) {
	if recipe == nil || recordData == nil || lookup == nil {
		return
	}
	brewerRef, ok := recordData["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	setStringIfEmpty(&recipe.BrewerRKey, atp.RKeyFromURI(brewerRef))
	if recipe.BrewerObj != nil {
		recipe.Interpolate()
		return
	}
	brewerData, found := lookup(brewerRef)
	if !found {
		return
	}
	if brewer, err := RecordToBrewer(brewerData, brewerRef); err == nil {
		recipe.BrewerObj = brewer
	}
	recipe.Interpolate()
}

func setStringIfEmpty(dst *string, value string) {
	if dst != nil && *dst == "" && value != "" {
		*dst = value
	}
}
