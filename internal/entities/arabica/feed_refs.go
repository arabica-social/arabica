package arabica

import (
	"tangled.org/pdewey.com/atp"
)

// resolveBrewFeedRefs hydrates bean/grinder/brewer/recipe references on a
// brew using already-fetched indexed records exposed via lookup. Missing
// refs are silently skipped — feed cards render fine with partial data.
func resolveBrewFeedRefs(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	brew, ok := model.(*Brew)
	if !ok || brew == nil {
		return
	}

	if beanRef, ok := recordData["beanRef"].(string); ok && beanRef != "" {
		if beanData, found := lookup(beanRef); found {
			if bean, err := RecordToBean(beanData, beanRef); err == nil {
				brew.Bean = bean
				if roasterRef, ok := beanData["roasterRef"].(string); ok && roasterRef != "" {
					if roasterData, found := lookup(roasterRef); found {
						if roaster, err := RecordToRoaster(roasterData, roasterRef); err == nil {
							brew.Bean.Roaster = roaster
						}
					}
				}
			}
		}
	}

	if grinderRef, ok := recordData["grinderRef"].(string); ok && grinderRef != "" {
		if grinderData, found := lookup(grinderRef); found {
			if grinder, err := RecordToGrinder(grinderData, grinderRef); err == nil {
				brew.GrinderObj = grinder
			}
		}
	}

	if brewerRef, ok := recordData["brewerRef"].(string); ok && brewerRef != "" {
		if brewerData, found := lookup(brewerRef); found {
			if brewer, err := RecordToBrewer(brewerData, brewerRef); err == nil {
				brew.BrewerObj = brewer
			}
		}
	}

	if recipeRef, ok := recordData["recipeRef"].(string); ok && recipeRef != "" {
		if rkey := atp.RKeyFromURI(recipeRef); rkey != "" {
			brew.RecipeRKey = rkey
		}
		if recipeData, found := lookup(recipeRef); found {
			if recipe, err := RecordToRecipe(recipeData, recipeRef); err == nil {
				brew.RecipeObj = recipe
			}
		}
	}
}

// resolveBeanFeedRef hydrates a bean's roaster reference.
func resolveBeanFeedRef(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	bean, ok := model.(*Bean)
	if !ok || bean == nil {
		return
	}
	roasterRef, ok := recordData["roasterRef"].(string)
	if !ok || roasterRef == "" {
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
	brewerRef, ok := recordData["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	brewerData, found := lookup(brewerRef)
	if !found {
		return
	}
	if brewer, err := RecordToBrewer(brewerData, brewerRef); err == nil {
		recipe.BrewerObj = brewer
	}
}
