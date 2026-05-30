// Package arabicaapp constructs the Arabica product configuration.
package arabicaapp

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/records"
)

func New() *domain.App {
	return &domain.App{
		Name:        "arabica",
		NSIDBase:    arabica.NSIDBase,
		Descriptors: entities.AllForApp(arabica.NSIDBase),
		EntityRoutes: []domain.EntityRoute{
			{Type: lexicons.RecordTypeBean, Path: "beans", Noun: "bean"},
			{Type: lexicons.RecordTypeRoaster, Path: "roasters", Noun: "roaster"},
			{Type: lexicons.RecordTypeGrinder, Path: "grinders", Noun: "grinder"},
			{Type: lexicons.RecordTypeBrewer, Path: "brewers", Noun: "brewer"},
			{Type: lexicons.RecordTypeRecipe, Path: "recipes", Noun: "recipe"},
			{Type: lexicons.RecordTypeBrew, Path: "brews", Noun: "brew"},
		},
		Brand: domain.BrandConfig{
			DisplayName: "Arabica",
			Tagline:     "Your brew, your data",
		},
		RecordStore: func(store records.Store) records.Store {
			if atpStore, ok := store.(*atproto.AtprotoStore); ok {
				return arabicastore.NewAtprotoStore(atpStore)
			}
			return store
		},
	}
}
