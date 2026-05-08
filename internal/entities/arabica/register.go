package arabica

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBean, NSID: NSIDBean,
		DisplayName: "Bean", Noun: "bean", URLPath: "beans",
		FeedFilterLabel: "Beans",
		GetField:        beanField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBean(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeRoaster, NSID: NSIDRoaster,
		DisplayName: "Roaster", Noun: "roaster", URLPath: "roasters",
		// FeedFilterLabel intentionally empty — roasters are reference
		// entities that rarely warrant a dedicated feed filter tab.
		GetField: roasterField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRoaster(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeGrinder, NSID: NSIDGrinder,
		DisplayName: "Grinder", Noun: "grinder", URLPath: "grinders",
		FeedFilterLabel: "Grinders",
		GetField:        grinderField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToGrinder(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBrewer, NSID: NSIDBrewer,
		DisplayName: "Brewer", Noun: "brewer", URLPath: "brewers",
		FeedFilterLabel: "Brewers",
		GetField:        brewerField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrewer(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeRecipe, NSID: NSIDRecipe,
		DisplayName: "Recipe", Noun: "recipe", URLPath: "recipes",
		FeedFilterLabel: "Recipes",
		GetField:        recipeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRecipe(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBrew, NSID: NSIDBrew,
		DisplayName: "Brew", Noun: "brew", URLPath: "brews",
		FeedFilterLabel: "Brews",
		GetField:        nil, // brew has no edit modal that needs prefill
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrew(rec, uri)
		},
	})
	// Like is intentionally omitted — has no entity page or modal.
}
