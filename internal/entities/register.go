package entities

import (
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	Register(&Descriptor{
		Type: lexicons.RecordTypeBean, NSID: atproto.NSIDBean,
		DisplayName: "Bean", Noun: "bean", URLPath: "beans",
		GetField: beanField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRoaster, NSID: atproto.NSIDRoaster,
		DisplayName: "Roaster", Noun: "roaster", URLPath: "roasters",
		GetField: roasterField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeGrinder, NSID: atproto.NSIDGrinder,
		DisplayName: "Grinder", Noun: "grinder", URLPath: "grinders",
		GetField: grinderField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrewer, NSID: atproto.NSIDBrewer,
		DisplayName: "Brewer", Noun: "brewer", URLPath: "brewers",
		GetField: brewerField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeRecipe, NSID: atproto.NSIDRecipe,
		DisplayName: "Recipe", Noun: "recipe", URLPath: "recipes",
		GetField: recipeField,
	})
	Register(&Descriptor{
		Type: lexicons.RecordTypeBrew, NSID: atproto.NSIDBrew,
		DisplayName: "Brew", Noun: "brew", URLPath: "brews",
		GetField: nil, // brew has no edit modal that needs prefill
	})
	// Like is intentionally omitted — has no entity page or modal.
}
