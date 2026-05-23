// Package oolongapp constructs the Oolong product configuration.
package oolongapp

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"

	// Blank import runs init() in Oolong web components, wiring templ render
	// hooks onto Oolong descriptors until descriptor/view registration is split.
	_ "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

func New() *domain.App {
	return &domain.App{
		Name:        "oolong",
		NSIDBase:    oolong.NSIDBase,
		Descriptors: entities.AllForApp(oolong.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: "Oolong",
			Tagline:     "Your tea, your data",
		},
	}
}
