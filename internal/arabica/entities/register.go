package arabica

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	registerBean()
	registerRoaster()
	registerGrinder()
	registerBrewer()
	registerRecipe()
	registerBrew()
	// Like is intentionally omitted — has no entity page or modal.
}

func registerBean() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeBean, NSID: NSIDBean, DisplayName: "Bean"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeBean, &entities.RecordBehavior{
		GetField: beanField,
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
		ResolveRefs: resolveBeanFeedRef,
	})
}

func registerRoaster() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeRoaster, NSID: NSIDRoaster, DisplayName: "Roaster"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeRoaster, &entities.RecordBehavior{
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
	})
}

func registerGrinder() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeGrinder, NSID: NSIDGrinder, DisplayName: "Grinder"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeGrinder, &entities.RecordBehavior{
		GetField: grinderField,
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
	})
}

func registerBrewer() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeBrewer, NSID: NSIDBrewer, DisplayName: "Brewer"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeBrewer, &entities.RecordBehavior{
		GetField: brewerField,
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
}

func registerRecipe() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeRecipe, NSID: NSIDRecipe, DisplayName: "Recipe"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeRecipe, &entities.RecordBehavior{
		GetField: recipeField,
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
		ResolveRefs: resolveRecipeFeedRef,
	})
}

func registerBrew() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeBrew, NSID: NSIDBrew, DisplayName: "Brew"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeBrew, &entities.RecordBehavior{
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
		ResolveRefs: resolveBrewFeedRefs,
	})
}
