package arabica

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

type rkeyer interface{ RKey() string }

func itemRKey(item any) string {
	if it, ok := item.(rkeyer); ok && it != nil {
		return it.RKey()
	}
	return ""
}

func modalEditURL(noun string) func(any) string {
	return func(item any) string {
		if r := itemRKey(item); r != "" {
			return "/api/modals/" + noun + "/" + r
		}
		return ""
	}
}

func init() {
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBean, NSID: NSIDBean,
		DisplayName: "Bean", Noun: "bean", URLPath: "beans",
		FeedFilterLabel: "Beans",
		GetField:        beanField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBean(rec, uri)
		},
		RKey: func(rec any) string {
			b, _ := rec.(*Bean)
			if b == nil {
				return ""
			}
			return b.RKey
		},
		DisplayTitle: func(rec any) string {
			b, _ := rec.(*Bean)
			if b == nil {
				return ""
			}
			if b.Name != "" {
				return b.Name
			}
			return b.Origin
		},
		EditModalURL: modalEditURL("bean"),
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
		RKey: func(rec any) string {
			r, _ := rec.(*Roaster)
			if r == nil {
				return ""
			}
			return r.RKey
		},
		DisplayTitle: func(rec any) string {
			r, _ := rec.(*Roaster)
			if r == nil {
				return ""
			}
			return r.Name
		},
		EditModalURL: modalEditURL("roaster"),
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeGrinder, NSID: NSIDGrinder,
		DisplayName: "Grinder", Noun: "grinder", URLPath: "grinders",
		FeedFilterLabel: "Grinders",
		GetField:        grinderField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToGrinder(rec, uri)
		},
		RKey: func(rec any) string {
			g, _ := rec.(*Grinder)
			if g == nil {
				return ""
			}
			return g.RKey
		},
		DisplayTitle: func(rec any) string {
			g, _ := rec.(*Grinder)
			if g == nil {
				return ""
			}
			return g.Name
		},
		EditModalURL: modalEditURL("grinder"),
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBrewer, NSID: NSIDBrewer,
		DisplayName: "Brewer", Noun: "brewer", URLPath: "brewers",
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
		EditModalURL: modalEditURL("brewer"),
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeRecipe, NSID: NSIDRecipe,
		DisplayName: "Recipe", Noun: "recipe", URLPath: "recipes",
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
		EditModalURL: modalEditURL("recipe"),
	})
	entities.Register(&entities.Descriptor{
		Type: lexicons.RecordTypeBrew, NSID: NSIDBrew,
		DisplayName: "Brew", Noun: "brew", URLPath: "brews",
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
			// Brew has no name field; fall back to bean name/origin.
			if b.Bean != nil {
				if b.Bean.Name != "" {
					return b.Bean.Name
				}
				return b.Bean.Origin
			}
			return "Coffee Brew"
		},
		EditURL: func(item any) string {
			if r := itemRKey(item); r != "" {
				return "/brews/" + r + "/edit"
			}
			return ""
		},
	})
	// Like is intentionally omitted — has no entity page or modal.
}
