package oolong

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongTea,
		NSID:            NSIDTea,
		DisplayName:     "Tea",
		Noun:            "tea",
		URLPath:         "teas",
		FeedFilterLabel: "Teas",
		GetField:        teaField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToTea(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongVendor,
		NSID:            NSIDVendor,
		DisplayName:     "Tea Vendor",
		Noun:            "vendor",
		URLPath:         "vendors",
		FeedFilterLabel: "", // reference entity — no dedicated feed tab
		GetField:        vendorField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToVendor(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongBrewer,
		NSID:            NSIDBrewer,
		DisplayName:     "Tea Brewer",
		Noun:            "brewer",
		URLPath:         "brewers",
		FeedFilterLabel: "Brewers",
		GetField:        brewerField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrewer(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongRecipe,
		NSID:            NSIDRecipe,
		DisplayName:     "Tea Recipe",
		Noun:            "recipe",
		URLPath:         "recipes",
		FeedFilterLabel: "Recipes",
		GetField:        recipeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRecipe(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongBrew,
		NSID:            NSIDBrew,
		DisplayName:     "Tea Brew",
		Noun:            "brew",
		URLPath:         "brews",
		FeedFilterLabel: "Brews",
		GetField:        nil, // brew has no edit modal that needs prefill
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrew(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongCafe,
		NSID:            NSIDCafe,
		DisplayName:     "Tea Cafe",
		Noun:            "cafe",
		URLPath:         "cafes",
		FeedFilterLabel: "Cafes",
		GetField:        cafeField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToCafe(rec, uri)
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongDrink,
		NSID:            NSIDDrink,
		DisplayName:     "Tea Drink",
		Noun:            "drink",
		URLPath:         "drinks",
		FeedFilterLabel: "Drinks",
		GetField:        drinkField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToDrink(rec, uri)
		},
	})
	// Comment and Like are intentionally NOT registered.
	// App.NSIDs() in internal/atplatform/domain/app.go appends them
	// unconditionally — registering them as descriptors would produce
	// duplicates. Same convention as internal/entities/arabica/register.go.
}
