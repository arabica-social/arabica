package entities

import (
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	Register(&Descriptor{
		Type: lexicons.RecordTypeBean, NSID: atproto.NSIDBean,
		DisplayName: "Bean", Noun: "bean", URLPath: "beans",
		FeedFilterLabel: "Beans",
		GetField:        beanField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToBean(rec, uri)
		},
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRoaster, NSID: atproto.NSIDRoaster,
		DisplayName: "Roaster", Noun: "roaster", URLPath: "roasters",
		// FeedFilterLabel intentionally empty — roasters are reference
		// entities that rarely warrant a dedicated feed filter tab.
		GetField: roasterField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToRoaster(rec, uri)
		},
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeGrinder, NSID: atproto.NSIDGrinder,
		DisplayName: "Grinder", Noun: "grinder", URLPath: "grinders",
		FeedFilterLabel: "Grinders",
		GetField:        grinderField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToGrinder(rec, uri)
		},
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrewer, NSID: atproto.NSIDBrewer,
		DisplayName: "Brewer", Noun: "brewer", URLPath: "brewers",
		FeedFilterLabel: "Brewers",
		GetField:        brewerField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToBrewer(rec, uri)
		},
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRecipe, NSID: atproto.NSIDRecipe,
		DisplayName: "Recipe", Noun: "recipe", URLPath: "recipes",
		FeedFilterLabel: "Recipes",
		GetField:        recipeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToRecipe(rec, uri)
		},
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrew, NSID: atproto.NSIDBrew,
		DisplayName: "Brew", Noun: "brew", URLPath: "brews",
		FeedFilterLabel: "Brews",
		GetField:        nil, // brew has no edit modal that needs prefill
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return atproto.RecordToBrew(rec, uri)
		},
	})
	// Like is intentionally omitted — has no entity page or modal.
}
