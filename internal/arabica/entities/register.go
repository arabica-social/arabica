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
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBean(rec, uri)
		},
		ReferenceFields: []string{"roasterRef"},
		ResolveRefs:     resolveBeanFeedRef,
	})
}

func registerRoaster() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeRoaster, NSID: NSIDRoaster, DisplayName: "Roaster"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeRoaster, &entities.RecordBehavior{
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRoaster(rec, uri)
		},
	})
}

func registerGrinder() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeGrinder, NSID: NSIDGrinder, DisplayName: "Grinder"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeGrinder, &entities.RecordBehavior{
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToGrinder(rec, uri)
		},
	})
}

func registerBrewer() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeBrewer, NSID: NSIDBrewer, DisplayName: "Brewer"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeBrewer, &entities.RecordBehavior{
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrewer(rec, uri)
		},
	})
}

func registerRecipe() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeRecipe, NSID: NSIDRecipe, DisplayName: "Recipe"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeRecipe, &entities.RecordBehavior{
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToRecipe(rec, uri)
		},
		ReferenceFields: []string{"brewerRef"},
		ResolveRefs:     resolveRecipeFeedRef,
	})
}

func registerBrew() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeBrew, NSID: NSIDBrew, DisplayName: "Brew"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeBrew, &entities.RecordBehavior{
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrew(rec, uri)
		},
		ReferenceFields: []string{"beanRef", "grinderRef", "brewerRef", "recipeRef"},
		ResolveRefs:     resolveBrewFeedRefs,
	})
}
