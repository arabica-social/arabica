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
		RKey: func(rec any) string {
			t, _ := rec.(*Tea)
			if t == nil {
				return ""
			}
			return t.RKey
		},
		DisplayTitle: func(rec any) string {
			t, _ := rec.(*Tea)
			if t == nil {
				return ""
			}
			return t.Name
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
		RKey: func(rec any) string {
			v, _ := rec.(*Vendor)
			if v == nil {
				return ""
			}
			return v.RKey
		},
		DisplayTitle: func(rec any) string {
			v, _ := rec.(*Vendor)
			if v == nil {
				return ""
			}
			return v.Name
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
		RKey: func(rec any) string {
			b, _ := rec.(*Brewer)
			if b == nil {
				return ""
			}
			return b.RKey
		},
		DisplayTitle: func(rec any) string {
			b, _ := rec.(*Brewer)
			if b == nil {
				return ""
			}
			return b.Name
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
		RKey: func(rec any) string {
			r, _ := rec.(*Recipe)
			if r == nil {
				return ""
			}
			return r.RKey
		},
		DisplayTitle: func(rec any) string {
			r, _ := rec.(*Recipe)
			if r == nil {
				return ""
			}
			return r.Name
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
		RKey: func(rec any) string {
			b, _ := rec.(*Brew)
			if b == nil {
				return ""
			}
			return b.RKey
		},
		DisplayTitle: func(rec any) string {
			b, _ := rec.(*Brew)
			if b == nil {
				return ""
			}
			// Brew has no name; fall back to the associated tea's name.
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
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
		RKey: func(rec any) string {
			c, _ := rec.(*Cafe)
			if c == nil {
				return ""
			}
			return c.RKey
		},
		DisplayTitle: func(rec any) string {
			c, _ := rec.(*Cafe)
			if c == nil {
				return ""
			}
			return c.Name
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
		RKey: func(rec any) string {
			d, _ := rec.(*Drink)
			if d == nil {
				return ""
			}
			return d.RKey
		},
		DisplayTitle: func(rec any) string {
			d, _ := rec.(*Drink)
			if d == nil {
				return ""
			}
			if d.Name != "" {
				return d.Name
			}
			return "Tea Drink"
		},
	})
	// Comment and Like are intentionally NOT registered.
	// App.NSIDs() in internal/atplatform/domain/app.go appends them
	// unconditionally — registering them as descriptors would produce
	// duplicates. Same convention as internal/entities/arabica/register.go.
}
